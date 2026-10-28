package service

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestDPZXActionsKeepProtocolMeaningAcrossAmounts(t *testing.T) {
	for _, action := range []struct {
		code int
		name string
	}{{2, "call"}, {3, "check"}, {4, "raise"}, {5, "allin"}, {6, "fold"}, {13, "bet"}} {
		for _, maximum := range []int{0, 100, 12780, 20000} {
			t.Run(fmt.Sprintf("%s/max%d", action.name, maximum), func(t *testing.T) {
				state := &dpzxTableState{heroSeat: -1, expectedSeat: -1, operationSeat: -1}
				for _, event := range []struct{ cmd, data string }{
					{"430", dpzxRealRoomSnapshot}, {"431", dpzxRealHandStart},
					{"492", `[false,[{"index":3,"vecCards":[{"value":8},{"value":43}]}]]`},
					{"432", `[2,15,0,100,true]`},
				} {
					if err := state.apply(event.cmd, json.RawMessage(event.data), 1); err != nil {
						t.Fatal(err)
					}
				}
				// Same protocol action with no bet, smaller bet, or larger bet elsewhere.
				for _, player := range state.players {
					player.bet = 0
				}
				state.players[1].bet = maximum
				score := 500
				if action.name == "allin" {
					score = 0
				}
				payload := fmt.Sprintf(`[{"index":2,"betState":%d,"score":%d,"betted":12780}]`, action.code, score)
				if err := state.apply("433", json.RawMessage(payload), 2); err != nil {
					t.Fatal(err)
				}
				if err := state.apply("432", json.RawMessage(`[3,15,0,100,true]`), 3); err != nil {
					t.Fatal(err)
				}
				request, _ := state.decisionRequest()
				if request == nil || len(request.ActionHistory) != 1 {
					t.Fatalf("missing GTO request: %+v", request)
				}
				history := request.ActionHistory[0]
				wantAmount := 12780
				if action.name == "fold" || action.name == "check" {
					wantAmount = 0
				}
				if history.Action != action.name || history.Amount != wantAmount {
					t.Fatalf("history = %+v; want %s/%d", history, action.name, wantAmount)
				}
				for _, player := range request.Players {
					if player.Position == history.Position && (player.ActionTaken != action.name || player.Stack != score || player.Bet != 12780 || player.IsActive != (action.name != "fold")) {
						t.Fatalf("protocol action lost in player: %+v", player)
					}
				}
			})
		}
	}
}

func TestAAPokerActionsKeepProtocolMeaningAcrossAmounts(t *testing.T) {
	for _, action := range []struct {
		code int
		name string
	}{{2, "call"}, {3, "raise"}, {4, "allin"}, {5, "check"}, {6, "fold"}, {99, ""}} {
		for _, maximum := range []int{0, 100, 12780, 20000} {
			t.Run(fmt.Sprintf("%s/max%d", action.name, maximum), func(t *testing.T) {
				state := &aapokerTableState{
					complete: true, handNum: 1, heroSeat: 0, operationRound: 1,
					stage: "preflop", tableSize: 3, smallBlind: 50, bigBlind: 100,
					participants: []int{0, 1, 2},
					players: map[int]*aapokerPlayerState{
						0: {seat: 0, position: "BTN-Hero", active: true, stack: 30000, holeCards: []string{"A-D", "K-C"}},
						1: {seat: 1, position: "SB", active: true, stack: 12780},
						2: {seat: 2, position: "BB", active: true, stack: 30000, bet: maximum},
					},
				}
				state.applyAction(aapokerActionPayload{SeatID: 1, Action: action.code, AnteNumber: 12780, OperationID: 0, OperationRound: 2, FirstCard: -1, SecondCard: -1})
				request, _ := state.decisionRequest()
				if action.name == "" {
					if request != nil || state.complete || len(state.history) != 0 {
						t.Fatal("unknown protocol action must block GTO, not guess from amount")
					}
					return
				}
				if request == nil || len(request.ActionHistory) != 1 {
					t.Fatalf("missing GTO request: %+v", request)
				}
				history := request.ActionHistory[0]
				wantAmount := 12780
				if action.name == "fold" || action.name == "check" {
					wantAmount = 0
				}
				if history.Action != action.name || history.Amount != wantAmount || request.Players[1].ActionTaken != action.name {
					t.Fatalf("protocol action lost: history=%+v players=%+v", history, request.Players)
				}
			})
		}
	}
}
