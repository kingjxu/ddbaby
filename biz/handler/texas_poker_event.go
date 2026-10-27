package handler

import (
	"context"
	"errors"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	ddbaby "github.com/kingjxu/ddbaby/biz/model/ddbaby"
	"github.com/kingjxu/ddbaby/service"
	"github.com/kingjxu/ddbaby/util"
	"github.com/sirupsen/logrus"
)

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
	resp, _ := handler.Handle(ctx)
	status := consts.StatusOK
	if resp.BaseResp.StatusCode != 0 {
		status = int(resp.BaseResp.StatusCode)
	}
	c.JSON(status, resp)
}

type TexasPokerEventHandler struct {
	req             *ddbaby.TexasPokerEventReq
	rawBody         []byte
	decisionHandler *TexasPokerDecisionHandler
}

func NewTexasPokerEventHandler(req *ddbaby.TexasPokerEventReq) *TexasPokerEventHandler {
	return &TexasPokerEventHandler{
		req: req,
	}
}

func (h *TexasPokerEventHandler) check(ctx context.Context) (int, error) {
	if h.req == nil {
		return consts.StatusBadRequest, errors.New("event is missing")
	}
	if !service.IsSupportedPokerEventPackage(h.req.PackageName) {
		return consts.StatusBadRequest, errors.New("invalid packageName")
	}
	// 旧客户端无uuid时只采集日志，不建立决策状态。
	if h.req.UUID == "" {
		return 0, nil
	}
	h.decisionHandler = NewTexasPokerDecisionHandler(&ddbaby.TexasPokerDecisionReq{
		UUID: util.Ptr(h.req.UUID), VPIP: util.Ptr(h.req.VPIP), PFR: util.Ptr(h.req.PFR),
	})
	if err := h.decisionHandler.verifyActiveCode(ctx); err != nil {
		return consts.StatusServiceUnavailable, err
	}
	return 0, nil
}

func (h *TexasPokerEventHandler) Handle(ctx context.Context) (*ddbaby.TexasPokerEventResp, error) {
	// 1. 校验请求。
	if status, err := h.check(ctx); err != nil {
		message := "invalid_event"
		if status == consts.StatusServiceUnavailable {
			message = "active_code_unavailable"
		}
		return h.newResp(status, message), err
	}
	event := h.req

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
	candidate, stateErr := service.ProcessPokerEvent(h.req)
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
		decision, decisionErr = executePokerDecision(
			ctx, h.decisionHandler, candidate.Request,
		)
		if decisionErr != nil {
			service.CompletePokerEventDecision(candidate, nil)
			logger.Warnf("[TexasPokerEvent] decision retry required:%v", decisionErr)
			return h.newResp(consts.StatusServiceUnavailable, "decision_retry"), decisionErr
		}
		service.CompletePokerEventDecision(candidate, decision)
	}
	writeAAPokerDecisionResponse(resp, decision)
	return resp, nil
}

func (h *TexasPokerEventHandler) newResp(status int, message string) *ddbaby.TexasPokerEventResp {
	return &ddbaby.TexasPokerEventResp{
		BaseResp: &ddbaby.BaseResp{StatusCode: int32(status), StatusMessage: message},
	}
}

func writeAAPokerDecisionResponse(response *ddbaby.TexasPokerEventResp, decision *service.PokerEventDecisionResult) {
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
