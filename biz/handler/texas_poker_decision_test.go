package handler

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server/binding"
	"github.com/kingjxu/ddbaby/biz/model/ddbaby"
	"github.com/kingjxu/ddbaby/model"
	"github.com/kingjxu/ddbaby/util"
	"github.com/stretchr/testify/assert"
)

func TestTexasPokerDecisionRequestBindsLowercaseProfileFields(t *testing.T) {
	body := `{"images":["image"],"timestamp":1,"uuid":"account","vpip":42,"pfr":20}`
	var requestContext app.RequestContext
	requestContext.Request.Header.SetMethod("POST")
	requestContext.Request.Header.SetContentTypeBytes([]byte("application/json"))
	requestContext.Request.Header.SetContentLength(len(body))
	requestContext.Request.SetBodyString(body)

	var req ddbaby.TexasPokerDecisionReq
	assert.NoError(t, binding.DefaultBinder().BindAndValidate(
		&requestContext.Request, &req, nil))
	assert.Equal(t, int32(42), req.GetVPIP())
	assert.Equal(t, int32(20), req.GetPFR())
}

func TestTexasPokerDecisionHandler_Handle(t *testing.T) {
	if os.Getenv("DDBABY_INTEGRATION_TEST") != "1" {
		t.Skip("requires MySQL, recognition and decision services; set DDBABY_INTEGRATION_TEST=1 to run")
	}

	t.Run("全轮次的调用", func(t *testing.T) {
		reqs := []*ddbaby.TexasPokerDecisionReq{
			/*		{
						Images:    []string{"/Users/bytedance/go/src/ddbaby/conf/badcase/20.jpg"},
						ImageType: util.Ptr("file_url"),
						UUID:      util.Ptr("123456"),
						Timestamp: util.Ptr(time.Now().Unix() * 1000),
					},
					{
						Images:    []string{"/Users/bytedance/go/src/ddbaby/conf/badcase/22.jpg"},
						ImageType: util.Ptr("file_url"),
						UUID:      util.Ptr("123456"),
						Timestamp: util.Ptr(time.Now().Unix()*1000 + 5000),
					},*/
			{
				Images:    []string{"/Users/bytedance/go/src/ddbaby/conf/badcase/19.jpg"},
				ImageType: util.Ptr("file_url"),
				UUID:      util.Ptr("123456"),
				Timestamp: util.Ptr(time.Now().Unix()*1000 + 5000),
			},
		}
		for i, req := range reqs {
			handler := NewTexasPokerDecisionHandler(req)
			_, err := handler.Handle(context.Background())
			t.Logf("---------------- %d ----------------\n", i+1)
			t.Logf("-----------------%d ----------------\n", i+1)
			assert.NoError(t, err)
		}
	})

}

func TestTexasPokerDecisionHandlerRejectsUnsupportedDecisionProfile(t *testing.T) {
	now := time.Now().UnixMilli()
	tests := []struct {
		name    string
		vpip    int32
		pfr     int32
		wantErr bool
	}{
		{name: "legacy default"},
		{name: "42/20", vpip: 42, pfr: 20},
		{name: "33/15", vpip: 33, pfr: 15},
		{name: "54/11", vpip: 54, pfr: 11},
		{name: "38/14", vpip: 38, pfr: 14},
		{name: "40/10", vpip: 40, pfr: 10},
		{name: "55/25", vpip: 55, pfr: 25},
		{name: "only vpip", vpip: 55, wantErr: true},
		{name: "only pfr", pfr: 25, wantErr: true},
		{name: "unknown pair", vpip: 99, pfr: 99, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &ddbaby.TexasPokerDecisionReq{
				Images: []string{"image"}, Timestamp: util.Ptr(now),
				UUID: util.Ptr("account"), VPIP: util.Ptr(tt.vpip), PFR: util.Ptr(tt.pfr),
			}
			err := NewTexasPokerDecisionHandler(req).check()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBuildGtoDecisionRecord(t *testing.T) {
	decisionTime := time.Date(2026, 10, 13, 10, 30, 0, 123000000, time.Local)
	req := &model.TexasGtoDecisionReq{
		GameType:              "no_limit_holdem",
		GameStage:             "flop",
		CurrentPlayerPosition: "BTN-Hero",
		CommunityCards:        []string{"2-S", "3-H", "4-C"},
		Players: []model.TexasPlayer{
			{Position: "BTN-Hero", HoleCards: []string{"A-H", "J-D"}, Stack: 100},
			{Position: "BB", Stack: 100},
		},
		VVIP: 42,
		PFR:  20,
	}
	rawResp := &model.TexasGtoDecisionResp{Action: "fold", ActionProbability: 0.8}
	finalResp := &model.TexasGtoDecisionResp{Action: "check", ActionProbability: 0.8}

	record, err := buildGtoDecisionRecord("user-1", decisionTime, req, rawResp, finalResp)
	assert.NoError(t, err)
	assert.Equal(t, "user-1", record.UID)
	assert.Equal(t, decisionTime, record.DecisionTime)
	assert.Equal(t, "BTN-Hero", record.Position)
	assert.Equal(t, "flop", record.Stage)
	assert.JSONEq(t, `["A-H","J-D"]`, record.HeroCards)
	assert.JSONEq(t, `{"action":"fold","raise_size":0,"action_probability":0.8,"detail":""}`, record.DecisionResponse)
	assert.JSONEq(t, `{"action":"check","raise_size":0,"action_probability":0.8,"detail":""}`, record.FinalResponse)

	var storedReq model.TexasGtoDecisionReq
	assert.NoError(t, json.Unmarshal([]byte(record.DecisionRequest), &storedReq))
	assert.Equal(t, int32(42), storedReq.VVIP)
	assert.Equal(t, int32(20), storedReq.PFR)
	assert.Equal(t, []string{"A-H", "J-D"}, storedReq.Players[0].HoleCards)
}

func TestCloneDecisionRequestDoesNotShareSlices(t *testing.T) {
	req := &model.TexasGtoDecisionReq{
		CommunityCards: []string{"2-S"},
		Players: []model.TexasPlayer{{
			Position: "BTN-Hero", HoleCards: []string{"A-H", "J-D"},
		}},
		RaiseRange:    []int{20},
		ActionHistory: []model.TexasActionHistory{{Stage: "preflop", Action: "call"}},
	}
	cloned := cloneDecisionRequest(req)

	req.CommunityCards[0] = "9-S"
	req.Players[0].Position = "SB"
	req.Players[0].HoleCards[0] = "K-H"
	req.RaiseRange[0] = 99
	req.ActionHistory[0].Action = "fold"

	assert.Equal(t, []string{"2-S"}, cloned.CommunityCards)
	assert.Equal(t, "BTN-Hero", cloned.Players[0].Position)
	assert.Equal(t, []string{"A-H", "J-D"}, cloned.Players[0].HoleCards)
	assert.Equal(t, []int{20}, cloned.RaiseRange)
	assert.Equal(t, "call", cloned.ActionHistory[0].Action)
}

func TestBuildTexasGtoDecisionRequestRejectsPendingHeroAction(t *testing.T) {
	pending := &model.TexasResult{
		PendingHeroBefore:    &model.TexasResult{},
		PendingHeroSnapshots: []*model.TexasResult{{}},
	}

	req, err := buildTexasGtoDecisionRequest(context.Background(), []*model.TexasResult{pending})
	assert.Nil(t, req)
	assert.EqualError(t, err, "hero action is still pending")
}

func TestRequestTexasGtoDecisionDoesNotCallStrategyWhileHeroActionPending(t *testing.T) {
	pending := &model.TexasResult{
		PendingHeroBefore:    &model.TexasResult{},
		PendingHeroSnapshots: []*model.TexasResult{{}},
	}
	calls := 0
	decide := func(context.Context, *model.TexasGtoDecisionReq, int32, int32) (*model.TexasGtoDecisionResp, error) {
		calls++
		return &model.TexasGtoDecisionResp{Action: "check"}, nil
	}

	req, resp, err := requestTexasGtoDecision(
		context.Background(), []*model.TexasResult{pending}, 0, 0, decide,
	)
	assert.Nil(t, req)
	assert.Nil(t, resp)
	assert.EqualError(t, err, "hero action is still pending")
	assert.Zero(t, calls)
}
