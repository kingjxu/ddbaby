package handler

import (
	"context"
	"errors"

	ddbaby "github.com/kingjxu/ddbaby/biz/model/ddbaby"
	_const "github.com/kingjxu/ddbaby/const"
	poker "github.com/kingjxu/ddbaby/dal/mysql/texas_poker"
	"github.com/kingjxu/ddbaby/model"
	"github.com/kingjxu/ddbaby/service"
	"github.com/kingjxu/ddbaby/util"
	"github.com/sirupsen/logrus"
)

type aapokerDecisionResult struct {
	Result         string   `json:"Result"`
	HeroCard       []string `json:"HeroCard"`
	CommunityCard  []string `json:"CommunityCard"`
	DecisionAction string   `json:"DecisionAction"`
	DecisionAmount int      `json:"DecisionAmount"`
	HeroMaxTarget  int      `json:"HeroMaxTarget"`
}

func executeAAPokerDecision(
	ctx context.Context,
	uuid string,
	token string,
	vpip int32,
	pfr int32,
	request *model.TexasGtoDecisionReq,
) (*aapokerDecisionResult, error) {
	if request == nil || uuid == "" {
		return nil, errors.New("decision identity or request is missing")
	}
	if err := service.VerifyTaoyuanIdentity(ctx, token, uuid); err != nil {
		return nil, err
	}
	decisionReq := &ddbaby.TexasPokerDecisionReq{
		UUID: util.Ptr(uuid), VPIP: util.Ptr(vpip), PFR: util.Ptr(pfr),
	}
	handler := NewTexasPokerDecisionHandler(decisionReq)
	if err := handler.verifyActiveCode(ctx); err != nil {
		return nil, err
	}
	response, err := service.GtoDecisionByProfile(ctx, request, vpip, pfr)
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, errors.New("decision service returned an empty response")
	}
	rawResponse := *response
	model.NormalizeTexasGtoDecisionToStack(request, response)
	if err = model.ValidateTexasGtoDecisionResponse(request, response); err != nil {
		return nil, err
	}
	model.NormalizeTexasGtoFreeFoldToCheck(request, response)

	handler.Action = response.Action
	handler.saveDecisionRecordAsync(request, &rawResponse, response)
	if handler.activeCodeInfo != nil && handler.activeCodeInfo.CodeType == _const.ActiveCodeTypeByInvokeCnt {
		handler.activeCodeInfo.InvokeCnt++
		if err = poker.UpsertUserActiveCode(ctx, handler.activeCodeInfo); err != nil {
			logrus.WithContext(ctx).WithField("uuid", uuid).
				Errorf("[TexasPokerEvent] update invoke count err:%v", err)
		}
	}

	heroCards := make([]string, 0, 2)
	for _, player := range request.Players {
		if player.Position == request.CurrentPlayerPosition {
			heroCards = append(heroCards, player.HoleCards...)
			break
		}
	}
	return &aapokerDecisionResult{
		Result:         handler.getFinalAction(request, response),
		HeroCard:       heroCards,
		CommunityCard:  append([]string(nil), request.CommunityCards...),
		DecisionAction: response.Action,
		DecisionAmount: response.RaiseSize,
		HeroMaxTarget:  eventHeroMaxTarget(request),
	}, nil
}
