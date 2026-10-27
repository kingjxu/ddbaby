package handler

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	ddbaby "github.com/kingjxu/ddbaby/biz/model/ddbaby"
	_const "github.com/kingjxu/ddbaby/const"
	"github.com/kingjxu/ddbaby/model"
	"github.com/kingjxu/ddbaby/util"
)

func TestInvalidGtoDecisionIsAuditedWithoutBillingOrResult(t *testing.T) {
	db := useEventActiveCodeDB(t)
	db.info.CodeType = _const.ActiveCodeTypeByInvokeCnt
	db.info.InvokeCnt, db.info.TotalCnt = 9, 10
	strategy := failEventDecision(t)
	strategy.success, strategy.action = true, "check"
	h := NewTexasPokerDecisionHandler(&ddbaby.TexasPokerDecisionReq{UUID: util.Ptr("audit-user")})
	if err := h.verifyActiveCode(context.Background()); err != nil {
		t.Fatal(err)
	}
	request := &model.TexasGtoDecisionReq{
		GameType: "no_limit_holdem", NumPlayers: 2, CurrentPot: 150,
		SbSize: 50, BbSize: 100, GameStage: "preflop", CurrentPlayerPosition: "BTN-Hero",
		Players: []model.TexasPlayer{
			{Position: "BTN-Hero", HoleCards: []string{"A-H", "K-D"}, Stack: 1000, IsActive: true},
			{Position: "BB", HoleCards: []string{"X-X", "X-X"}, Stack: 900, Bet: 100, IsActive: true},
		},
	}
	for call := 1; call <= 2; call++ {
		result, err := executePokerDecision(context.Background(), h, request)
		if err == nil || result != nil {
			t.Fatal("illegal check was returned to client")
		}
		select {
		case record := <-db.audits:
			if record.UID != "audit-user" || record.FinalResponse != "null" {
				t.Fatalf("invalid audit metadata: %+v", record)
			}
			var raw model.TexasGtoDecisionResp
			if err := json.Unmarshal([]byte(record.DecisionResponse), &raw); err != nil || raw.Action != "check" {
				t.Fatalf("raw illegal decision was lost: %s", record.DecisionResponse)
			}
			var saved model.TexasGtoDecisionReq
			if err := json.Unmarshal([]byte(record.DecisionRequest), &saved); err != nil || saved.Players[1].Bet != 100 {
				t.Fatal("actual GTO request was lost")
			}
		case <-time.After(time.Second):
			t.Fatal("illegal GTO decision was not audited")
		}
		if db.billings != 0 || db.info.InvokeCnt != 9 || strategy.calls.Load() != int32(call) {
			t.Fatal("illegal result was billed or request count changed")
		}
	}
	select {
	case <-db.audits:
		t.Fatal("duplicate audit for one decision")
	default:
	}
}
