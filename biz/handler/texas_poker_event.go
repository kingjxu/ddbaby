package handler

import (
	"context"
	"errors"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	ddbaby "github.com/kingjxu/ddbaby/biz/model/ddbaby"
	"github.com/kingjxu/ddbaby/util"
	"github.com/sirupsen/logrus"
)

const (
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

// TexasPokerEvent .
// @router /lyxz/gto_event [POST]
func TexasPokerEvent(ctx context.Context, c *app.RequestContext) {
	var err error
	var req ddbaby.TexasPokerEventReq
	err = c.BindAndValidate(&req)
	if err != nil {
		writeTexasPokerEventError(c, consts.StatusBadRequest, "invalid_event")
		return
	}

	handler := NewTexasPokerEventHandler(&req)
	handler.rawBody = c.Request.Body()
	handler.token = texasPokerEventBearerToken(c)
	handler.now = c.GetTime(pokerEventReceivedAtKey)
	resp, _ := handler.Handle(ctx)
	status := consts.StatusOK
	if resp.BaseResp.StatusCode != 0 {
		status = int(resp.BaseResp.StatusCode)
	}
	c.JSON(status, resp)
}

type TexasPokerEventHandler struct {
	req     *ddbaby.TexasPokerEventReq
	rawBody []byte
	token   string
	now     time.Time
	route   pokerEventRoute
}

func NewTexasPokerEventHandler(req *ddbaby.TexasPokerEventReq) *TexasPokerEventHandler {
	return &TexasPokerEventHandler{
		req: req,
		now: time.Now(),
	}
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
	// 1. 校验请求。
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
	logger := logrus.WithContext(ctx).WithFields(fields)
	logger.Info("[TexasPokerEvent] accepted")

	// 2. 按应用来源重建牌局。
	resp := h.newResp(0, "success")
	var candidate *aapokerDecisionCandidate
	var stateErr error
	switch event.route {
	case pokerEventRouteAAPoker:
		candidate, stateErr = texasPokerEventStates.process(event, h.now)
	case pokerEventRouteDPZX:
		candidate, stateErr = dpzxPokerEventStates.process(event, h.now)
	}
	if stateErr != nil {
		logger.Warnf("[TexasPokerEvent] state update ignored:%v", stateErr)
		return resp, nil
	}
	if candidate == nil {
		return resp, nil
	}

	// 3. 回放已有结果，或调用策略并保存可重放的结果。
	decision := candidate.Cached
	if decision == nil {
		var decisionErr error
		decision, decisionErr = executeAAPokerDecisionForEvent(
			ctx, event.UUID, h.token, event.VPIP, event.PFR, candidate.Request,
		)
		if decisionErr != nil {
			completePokerEventDecision(event.route, candidate, nil, h.now)
			logger.Warnf("[TexasPokerEvent] decision retry required:%v", decisionErr)
			return h.newResp(consts.StatusServiceUnavailable, "decision_retry"), decisionErr
		}
		completePokerEventDecision(event.route, candidate, decision, h.now)
	}
	writeAAPokerDecisionResponse(resp, decision)
	return resp, nil
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
