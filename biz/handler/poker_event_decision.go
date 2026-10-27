package handler

import (
	"context"
	"errors"

	_const "github.com/kingjxu/ddbaby/const"
	poker "github.com/kingjxu/ddbaby/dal/mysql/texas_poker"
	"github.com/kingjxu/ddbaby/model"
	"github.com/kingjxu/ddbaby/service"
	"github.com/sirupsen/logrus"
)

func executePokerDecision(
	ctx context.Context,
	handler *TexasPokerDecisionHandler,
	request *model.TexasGtoDecisionReq,
) (*service.PokerEventDecisionResult, error) {
	if request == nil || handler == nil || handler.activeCodeInfo == nil {
		return nil, errors.New("decision identity or request is missing")
	}
	uuid := handler.req.GetUUID()
	vpip, pfr := handler.req.GetVPIP(), handler.req.GetPFR()
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
		handler.saveDecisionRecordAsync(request, &rawResponse, nil)
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
	return &service.PokerEventDecisionResult{
		Result:         handler.getFinalAction(request, response),
		HeroCard:       heroCards,
		CommunityCard:  append([]string(nil), request.CommunityCards...),
		DecisionAction: response.Action,
		DecisionAmount: response.RaiseSize,
		HeroMaxTarget:  service.PokerEventHeroMaxTarget(request),
	}, nil
}
