package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	ddbaby "github.com/kingjxu/ddbaby/biz/model/ddbaby"
	"github.com/kingjxu/ddbaby/util"
	"github.com/sirupsen/logrus"
)

const (
	maxTexasPokerEventBytes     = 512 * 1024
	aapokerPackageName          = "com.plusaa.amula"
	aapokerAlternatePackageName = "com.ty.aapk"
	dpzxPackageName             = "com.dpzx.pro"
)

type pokerEventRoute uint8

const (
	pokerEventRouteAAPoker pokerEventRoute = iota + 1
	pokerEventRouteDPZX
)

// pokerEventPackageRoutes is the single allowlist and dispatch table. Add new
// packages here only after choosing an implemented state machine or log-only.
var pokerEventPackageRoutes = map[string]pokerEventRoute{
	aapokerPackageName:          pokerEventRouteAAPoker,
	aapokerAlternatePackageName: pokerEventRouteAAPoker,
	dpzxPackageName:             pokerEventRouteDPZX,
}

var texasPokerEventStates = newAAPokerEventRegistry()
var dpzxPokerEventStates = newDPZXEventRegistry()
var executeAAPokerDecisionForEvent = executeAAPokerDecision

type texasPokerEvent struct {
	ddbaby.TexasPokerEventReq
	route pokerEventRoute
}

// TexasPokerEvent receives one normalized poker-app event and dispatches it by
// the explicit package allowlist after validation and full diagnostic logging.
// @router /lyxz/gto_event [POST]
func TexasPokerEvent(ctx context.Context, c *app.RequestContext) {
	texasPokerEventAt(ctx, c, time.Now())
}

func texasPokerEventAt(ctx context.Context, c *app.RequestContext, now time.Time) {
	body, err := readTexasPokerEventBody(c)
	if err != nil {
		writeTexasPokerEventError(c, consts.StatusBadRequest, "invalid_body_size")
		return
	}

	event, err := parseTexasPokerEvent(body)
	if err != nil {
		writeTexasPokerEventError(c, consts.StatusBadRequest, "invalid_event")
		return
	}
	handler := NewTexasPokerEventHandler(&event.TexasPokerEventReq, body, texasPokerEventBearerToken(c))
	handler.now = now
	response, _ := handler.Handle(ctx)
	status := consts.StatusOK
	if response.BaseResp.StatusCode != 0 {
		status = int(response.BaseResp.StatusCode)
	}
	c.JSON(status, response)
}

type TexasPokerEventHandler struct {
	req     *ddbaby.TexasPokerEventReq
	rawBody []byte
	token   string
	now     time.Time
	route   pokerEventRoute
}

func NewTexasPokerEventHandler(req *ddbaby.TexasPokerEventReq, rawBody []byte, token string) *TexasPokerEventHandler {
	return &TexasPokerEventHandler{req: req, rawBody: rawBody, token: token, now: time.Now()}
}

func (h *TexasPokerEventHandler) check() error {
	if h.req == nil {
		return errors.New("event is missing")
	}
	route, supported := pokerEventPackageRoutes[h.req.PackageName]
	if !supported {
		return errors.New("invalid packageName")
	}
	h.route = route
	return nil
}

func (h *TexasPokerEventHandler) Handle(ctx context.Context) (*ddbaby.TexasPokerEventResp, error) {
	if err := h.check(); err != nil {
		return h.newResp(consts.StatusBadRequest, "invalid_event"), err
	}
	event := &texasPokerEvent{TexasPokerEventReq: *h.req, route: h.route}

	fields := logrus.Fields{
		"cmd":               event.Cmd,
		"direction":         event.Direction,
		"event":             string(h.rawBody),
		"full_event_logged": true,
		"location":          event.Location,
		"package_name":      event.PackageName,
		"payload_size":      len(event.Data),
	}
	logrus.WithContext(ctx).WithFields(fields).Info("[TexasPokerEvent] accepted")

	response := h.newResp(0, "success")
	var candidate *aapokerDecisionCandidate
	var stateErr error
	switch event.route {
	case pokerEventRouteAAPoker:
		candidate, stateErr = texasPokerEventStates.process(event, h.now)
	case pokerEventRouteDPZX:
		candidate, stateErr = dpzxPokerEventStates.process(event, h.now)
	}
	if stateErr != nil {
		logrus.WithContext(ctx).WithFields(fields).
			Warnf("[TexasPokerEvent] state update ignored:%v", stateErr)
	} else if candidate != nil {
		decision := candidate.Cached
		if decision == nil {
			var decisionErr error
			decision, decisionErr = executeAAPokerDecisionForEvent(
				ctx, event.UUID, h.token, event.VPIP, event.PFR, candidate.Request,
			)
			if decisionErr != nil {
				completePokerEventDecision(event.route, candidate, nil, h.now)
				logrus.WithContext(ctx).WithFields(fields).
					Warnf("[TexasPokerEvent] decision retry required:%v", decisionErr)
				return h.newResp(consts.StatusServiceUnavailable, "decision_retry"), decisionErr
			}
			completePokerEventDecision(event.route, candidate, decision, h.now)
		}
		writeAAPokerDecisionResponse(response, decision)
	}
	return response, nil
}

func (h *TexasPokerEventHandler) newResp(status int, message string) *ddbaby.TexasPokerEventResp {
	return &ddbaby.TexasPokerEventResp{
		BaseResp: &ddbaby.BaseResp{StatusCode: int32(status), StatusMessage: message},
	}
}

func completePokerEventDecision(
	route pokerEventRoute,
	candidate *aapokerDecisionCandidate,
	result *aapokerDecisionResult,
	now time.Time,
) {
	switch route {
	case pokerEventRouteAAPoker:
		texasPokerEventStates.complete(candidate, result, now)
	case pokerEventRouteDPZX:
		dpzxPokerEventStates.complete(candidate, result, now)
	}
}

func writeAAPokerDecisionResponse(response *ddbaby.TexasPokerEventResp, decision *aapokerDecisionResult) {
	if decision == nil {
		return
	}
	response.Result = util.Ptr(decision.Result)
	response.HeroCard = decision.HeroCard
	response.CommunityCard = decision.CommunityCard
	response.DecisionAction = util.Ptr(decision.DecisionAction)
	response.DecisionAmount = util.Ptr(int64(decision.DecisionAmount))
	response.HeroMaxTarget = util.Ptr(int64(decision.HeroMaxTarget))
}

func texasPokerEventBearerToken(c *app.RequestContext) string {
	header := strings.TrimSpace(string(c.Request.Header.Peek("Authorization")))
	if len(header) < 8 || !strings.EqualFold(header[:7], "Bearer ") {
		return ""
	}
	token := strings.TrimSpace(header[7:])
	if token == "" || strings.ContainsAny(token, " \t\r\n") || len(token) > 4096 {
		return ""
	}
	return token
}

func readTexasPokerEventBody(c *app.RequestContext) ([]byte, error) {
	if !c.Request.IsBodyStream() {
		body := c.Request.Body()
		if len(body) == 0 || len(body) > maxTexasPokerEventBytes {
			return nil, errors.New("invalid body size")
		}
		return body, nil
	}

	contentLength := c.Request.Header.ContentLength()
	if contentLength == 0 || contentLength > maxTexasPokerEventBytes {
		return nil, errors.New("invalid body size")
	}

	body, err := io.ReadAll(io.LimitReader(
		c.RequestBodyStream(), int64(maxTexasPokerEventBytes)+1,
	))
	if err != nil || len(body) == 0 || len(body) > maxTexasPokerEventBytes {
		return nil, errors.New("invalid body size")
	}
	return body, nil
}

func parseTexasPokerEvent(body []byte) (*texasPokerEvent, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	var fields map[string]json.RawMessage
	if err := decoder.Decode(&fields); err != nil || fields == nil {
		return nil, errors.New("event must be a JSON object")
	}
	var packageName string
	if err := json.Unmarshal(fields["packageName"], &packageName); err != nil {
		return nil, errors.New("invalid packageName")
	}
	route, supported := pokerEventPackageRoutes[packageName]
	if !supported {
		return nil, errors.New("invalid packageName")
	}

	var event texasPokerEvent
	// Only packageName controls routing. Other fields are best-effort decoded so
	// protocol analysis can retain new or malformed shapes in the full raw log.
	_ = json.Unmarshal(body, &event.TexasPokerEventReq)
	event.route = route
	event.PackageName = packageName

	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, errors.New("multiple JSON values")
	}
	return &event, nil
}

func writeTexasPokerEventError(c *app.RequestContext, status int, code string) {
	c.JSON(status, &ddbaby.TexasPokerEventResp{
		BaseResp: &ddbaby.BaseResp{StatusCode: int32(status), StatusMessage: code},
	})
}
