package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/kingjxu/ddbaby/model"
)

const dpzxRealRoomSnapshot = `[{"bankIndex":6,"mySeat":3,"HandNum":125,"roomID":28815717,"littleBB":50,"ante":25,"stisNumMax":8},[{"index":1,"value":25,"score":7100},{"index":2,"value":25,"score":6004},{"index":3,"value":25,"score":4975},{"index":4,"value":25,"score":17545},{"index":5,"value":25,"score":14975},{"index":6,"value":25,"score":4413},{"index":7,"value":25,"score":3681}]]`

const dpzxRealHandStart = `[[{"index":1,"value":100,"score":7000,"startState":2,"betState":10},{"index":2,"value":0,"score":6004,"startState":0,"betState":10},{"index":3,"value":0,"score":4975,"startState":0,"betState":10},{"index":4,"value":0,"score":17545,"startState":0,"betState":10},{"index":5,"value":0,"score":14975,"startState":0,"betState":10},{"index":6,"value":0,"score":4413,"startState":0,"betState":10},{"index":7,"value":50,"score":3631,"startState":1,"betState":10}]]`

func TestDPZXEventStateBuildsDecisionFromRealHeroTurn(t *testing.T) {
	registry := newDPZXEventRegistry()
	now := time.Unix(1793076600, 0)
	apply := func(command, payload string, timestamp int64) *aapokerDecisionCandidate {
		t.Helper()
		candidate, err := registry.process(testDPZXEvent(command, payload, timestamp), now)
		if err != nil {
			t.Fatalf("process(%s) error = %v", command, err)
		}
		now = now.Add(time.Millisecond)
		return candidate
	}

	if got := apply("430", dpzxRealRoomSnapshot, 1793076613000); got != nil {
		t.Fatal("room snapshot unexpectedly produced a decision")
	}
	apply("431", dpzxRealHandStart, 1793076615000)
	apply("432", `[2,15,0,100,true]`, 1793076615500)
	apply("492", `[false,[{"index":3,"cardType":0,"vecCards":[{"index":0,"value":8},{"index":1,"value":43}]}]]`, 1793076615600)
	apply("433", `[{"index":2,"betState":2,"score":5904,"betted":100}]`, 1793076617000)
	apply("403", `[425]`, 1793076617100)
	candidate := apply("432", `[3,15,0,100,true]`, 1793076617478)
	if candidate == nil || candidate.Request == nil {
		t.Fatal("real DPZX Hero prompt did not produce a decision")
	}
	request := candidate.Request
	if request.CurrentPlayerPosition != "UTG-Hero" || request.GameStage != "preflop" {
		t.Fatalf("Hero turn = %s/%s", request.CurrentPlayerPosition, request.GameStage)
	}
	if request.CurrentPot != 425 || request.Ante != 25 || request.SbSize != 50 || request.BbSize != 100 {
		t.Fatalf("unexpected table values: %+v", request)
	}
	if len(request.ActionHistory) != 1 || request.ActionHistory[0].Position != "straddle" ||
		request.ActionHistory[0].Action != "call" || request.ActionHistory[0].Amount != 100 {
		t.Fatalf("unexpected action history: %+v", request.ActionHistory)
	}
	hero := request.Players[4]
	if hero.Position != "UTG-Hero" || hero.Stack != 4975 || hero.Bet != 0 ||
		len(hero.HoleCards) != 2 || hero.HoleCards[0] != "10-H" || hero.HoleCards[1] != "6-S" {
		t.Fatalf("unexpected Hero: %+v", hero)
	}
	if err := model.ValidateTexasGtoDecisionRequest(request); err != nil {
		t.Fatalf("generated request is invalid: %v", err)
	}

	registry.complete(candidate, nil, now)
	retry := apply("432", `[3,15,0,100,true]`, 1793076617478)
	if retry == nil || retry.TurnKey != candidate.TurnKey {
		t.Fatal("failed DPZX decision did not retain retry eligibility")
	}
	want := &aapokerDecisionResult{Result: "Call 100", HeroCard: []string{"10-H", "6-S"}}
	registry.complete(retry, want, now)
	replayed := apply("432", `[3,15,0,100,true]`, 1793076617478)
	if replayed == nil || replayed.Cached == nil || replayed.Cached.Result != want.Result {
		t.Fatal("successful DPZX decision was not replayed")
	}
}

func TestDPZXEventStateTracksStreetAndRejectsMissingAction(t *testing.T) {
	state := &dpzxTableState{heroSeat: -1, expectedSeat: -1, operationSeat: -1}
	if err := state.apply("430", json.RawMessage(`[{"bankIndex":0,"mySeat":3,"HandNum":7,"roomID":99,"littleBB":50,"ante":0,"stisNumMax":4},[{"index":0,"value":0,"score":1000},{"index":1,"value":0,"score":1000},{"index":3,"value":0,"score":1000}]]`), 1); err != nil {
		t.Fatal(err)
	}
	if err := state.apply("431", json.RawMessage(`[[{"index":0,"value":0,"score":1000},{"index":1,"value":50,"score":950},{"index":3,"value":100,"score":900}]]`), 2); err != nil {
		t.Fatal(err)
	}
	state.apply("492", json.RawMessage(`[false,[{"index":3,"vecCards":[{"value":12},{"value":24}]}]]`), 3)
	state.apply("432", json.RawMessage(`[0,15,0,100,true]`), 4)
	state.apply("433", json.RawMessage(`[{"index":0,"betState":2,"score":900,"betted":100}]`), 5)
	state.apply("403", json.RawMessage(`[250]`), 6)
	state.apply("432", json.RawMessage(`[1,15,50,50,true]`), 7)
	state.apply("433", json.RawMessage(`[{"index":1,"betState":6,"score":950,"betted":50}]`), 8)
	state.apply("432", json.RawMessage(`[3,15,100,0,true]`), 9)
	preflop, _ := state.decisionRequest()
	if preflop == nil || preflop.CurrentPlayerPosition != "BB-Hero" {
		t.Fatalf("unexpected preflop request: %+v", preflop)
	}
	state.apply("433", json.RawMessage(`[{"index":3,"betState":3,"score":900,"betted":100}]`), 10)
	if err := state.apply("490", json.RawMessage(`[true,[0,14,28]]`), 11); err != nil {
		t.Fatal(err)
	}
	state.apply("432", json.RawMessage(`[3,15,0,0,false]`), 12)
	flop, _ := state.decisionRequest()
	if flop == nil || flop.GameStage != "flop" || strings.Join(flop.CommunityCards, ",") != "2-H,3-D,4-C" {
		t.Fatalf("unexpected flop request: %+v", flop)
	}
	if len(flop.ActionHistory) != 3 || flop.ActionHistory[2].Action != "check" {
		t.Fatalf("unexpected history after street change: %+v", flop.ActionHistory)
	}

	state.apply("433", json.RawMessage(`[{"index":3,"betState":3,"score":900,"betted":0}]`), 13)
	state.apply("432", json.RawMessage(`[0,15,0,0,false]`), 14)
	state.apply("432", json.RawMessage(`[3,15,0,0,false]`), 15)
	if request, _ := state.decisionRequest(); request != nil {
		t.Fatal("state with a missing action produced a decision")
	}
}

func TestDPZXEventStateRejectsIncompleteStreetTransitions(t *testing.T) {
	newState := func(t *testing.T) *dpzxTableState {
		t.Helper()
		state := &dpzxTableState{heroSeat: -1, expectedSeat: -1, operationSeat: -1}
		if err := state.apply("430", json.RawMessage(`[{"bankIndex":0,"mySeat":3,"HandNum":7,"roomID":99,"littleBB":50,"ante":0,"stisNumMax":4},[{"index":0,"value":0,"score":1000},{"index":1,"value":0,"score":1000},{"index":3,"value":0,"score":1000}]]`), 1); err != nil {
			t.Fatal(err)
		}
		if err := state.apply("431", json.RawMessage(`[[{"index":0,"value":0,"score":1000},{"index":1,"value":50,"score":950},{"index":3,"value":100,"score":900}]]`), 2); err != nil {
			t.Fatal(err)
		}
		if err := state.apply("492", json.RawMessage(`[false,[{"index":3,"vecCards":[{"value":12},{"value":24}]}]]`), 3); err != nil {
			t.Fatal(err)
		}
		return state
	}

	t.Run("pending action cannot be erased by flop", func(t *testing.T) {
		state := newState(t)
		if err := state.apply("432", json.RawMessage(`[0,15,0,100,true]`), 4); err != nil {
			t.Fatal(err)
		}
		if err := state.apply("490", json.RawMessage(`[true,[0,14,28]]`), 5); err == nil {
			t.Fatal("street transition with a missing action was accepted")
		}
		state.apply("432", json.RawMessage(`[3,15,100,0,true]`), 6)
		if request, _ := state.decisionRequest(); request != nil {
			t.Fatal("missing action followed by a Hero prompt produced a decision")
		}
	})

	tests := []struct {
		name  string
		board string
	}{
		{name: "skip flop", board: `[true,[0,14,28,42]]`},
		{name: "duplicate public card", board: `[true,[0,14,14]]`},
		{name: "duplicate Hero card", board: `[true,[0,14,12]]`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := newState(t)
			if err := state.apply("490", json.RawMessage(test.board), 4); err == nil {
				t.Fatal("invalid first board transition was accepted")
			}
			state.apply("432", json.RawMessage(`[3,15,100,0,true]`), 5)
			if request, _ := state.decisionRequest(); request != nil {
				t.Fatal("invalid board transition produced a decision")
			}
		})
	}

	t.Run("skip turn", func(t *testing.T) {
		state := newState(t)
		if err := state.apply("490", json.RawMessage(`[true,[0,14,28]]`), 4); err != nil {
			t.Fatal(err)
		}
		if err := state.apply("490", json.RawMessage(`[true,[0,14,28,42,5]]`), 5); err == nil {
			t.Fatal("flop-to-river transition was accepted")
		}
		state.apply("432", json.RawMessage(`[3,15,0,0,false]`), 6)
		if request, _ := state.decisionRequest(); request != nil {
			t.Fatal("skipped turn produced a decision")
		}
	})

	t.Run("changed board prefix", func(t *testing.T) {
		state := newState(t)
		if err := state.apply("490", json.RawMessage(`[true,[0,14,28]]`), 4); err != nil {
			t.Fatal(err)
		}
		if err := state.apply("490", json.RawMessage(`[true,[1,14,28,42]]`), 5); err == nil {
			t.Fatal("changed board prefix was accepted")
		}
		state.apply("432", json.RawMessage(`[3,15,0,0,false]`), 6)
		if request, _ := state.decisionRequest(); request != nil {
			t.Fatal("changed board prefix produced a decision")
		}
	})
}

func TestDPZXEventStateRejectsWatcherAndUnknownAction(t *testing.T) {
	state := &dpzxTableState{heroSeat: -1, expectedSeat: -1, operationSeat: -1}
	room := json.RawMessage(`[{"bankIndex":0,"mySeat":9,"HandNum":8,"roomID":99,"littleBB":50,"ante":25,"stisNumMax":8},[{"index":0,"value":25,"score":975},{"index":1,"value":25,"score":975}]]`)
	start := json.RawMessage(`[[{"index":0,"value":50,"score":925},{"index":1,"value":100,"score":875}]]`)
	if err := state.apply("430", room, 1); err != nil {
		t.Fatal(err)
	}
	if err := state.apply("431", start, 2); err != nil {
		t.Fatal(err)
	}
	state.apply("432", json.RawMessage(`[0,15,50,50,true]`), 3)
	if request, _ := state.decisionRequest(); request != nil {
		t.Fatal("watcher state produced a decision")
	}

	state.heroSeat = 0
	state.complete = true
	state.operationSeat = 0
	state.expectedSeat = 0
	state.players[0].holeCards = []string{"A-H", "K-D"}
	if err := state.apply("433", json.RawMessage(`[{"index":0,"betState":99,"score":925,"betted":50}]`), 4); err == nil {
		t.Fatal("unknown action state was accepted")
	}
	if request, _ := state.decisionRequest(); request != nil {
		t.Fatal("unknown action state left the hand eligible for a decision")
	}
}

func TestDPZXEventStateRejectsInvalidHeroCardsAfterValidCards(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{name: "incomplete", payload: `[false,[{"index":3,"vecCards":[{"value":12}]}]]`},
		{name: "out of range", payload: `[false,[{"index":3,"vecCards":[{"value":12},{"value":52}]}]]`},
		{name: "duplicate", payload: `[false,[{"index":3,"vecCards":[{"value":12},{"value":12}]}]]`},
		{name: "malformed", payload: `[false,`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registry := newDPZXEventRegistry()
			now := time.Unix(1793076600, 0)
			apply := func(command, payload string, timestamp int64) (*aapokerDecisionCandidate, error) {
				candidate, err := registry.process(testDPZXEvent(command, payload, timestamp), now)
				now = now.Add(time.Millisecond)
				return candidate, err
			}
			if _, err := apply("430", dpzxRealRoomSnapshot, 1793076613000); err != nil {
				t.Fatal(err)
			}
			if _, err := apply("431", dpzxRealHandStart, 1793076615000); err != nil {
				t.Fatal(err)
			}
			if _, err := apply("492", `[false,[{"index":3,"vecCards":[{"value":8},{"value":43}]}]]`, 1793076615600); err != nil {
				t.Fatal(err)
			}
			if _, err := apply("492", test.payload, 1793076615700); err == nil {
				t.Fatal("invalid Hero cards were accepted")
			}
			candidate, err := apply("432", `[3,15,0,100,true]`, 1793076615800)
			if err != nil {
				t.Fatal(err)
			}
			if candidate != nil {
				t.Fatal("Hero prompt after invalid cards produced a decision candidate")
			}
		})
	}
}

func TestTexasPokerEventRoutesDPZXThroughDecisionResponse(t *testing.T) {
	previousRegistry := dpzxPokerEventStates
	previousExecute := executeAAPokerDecisionForEvent
	dpzxPokerEventStates = newDPZXEventRegistry()
	calls := 0
	executeAAPokerDecisionForEvent = func(
		_ context.Context, uuid, token string, _, _ int32, request *model.TexasGtoDecisionReq,
	) (*aapokerDecisionResult, error) {
		calls++
		if uuid != "hero@example.com" || token != "session-token" || request.CurrentPlayerPosition != "UTG-Hero" {
			t.Fatalf("unexpected decision input: uuid=%q token=%q request=%+v", uuid, token, request)
		}
		if calls == 1 {
			return nil, errors.New("temporary GTO failure")
		}
		return &aapokerDecisionResult{
			Result: "Call 100", HeroCard: []string{"10-H", "6-S"},
			DecisionAction: "call", DecisionAmount: 0, HeroMaxTarget: 4975,
		}, nil
	}
	t.Cleanup(func() {
		dpzxPokerEventStates = previousRegistry
		executeAAPokerDecisionForEvent = previousExecute
	})

	now := time.Unix(1793076600, 0)
	post := func(command, payload string, timestamp int64) *app.RequestContext {
		t.Helper()
		body := fmt.Sprintf(`{"packageName":"com.dpzx.pro","uuid":"hero@example.com","vpip":0,"pfr":0,"timestamp":%d,"uid":2773921,"cmd":%q,"direction":"ServerToClient","location":"TABLE","data":%s}`,
			timestamp, command, payload)
		requestContext := &app.RequestContext{}
		requestContext.Request.SetBodyString(body)
		requestContext.Request.Header.Set("Authorization", "Bearer session-token")
		texasPokerEventAt(context.Background(), requestContext, now)
		now = now.Add(time.Millisecond)
		return requestContext
	}

	post("430", dpzxRealRoomSnapshot, 1793076613000)
	post("431", dpzxRealHandStart, 1793076615000)
	post("432", `[2,15,0,100,true]`, 1793076615500)
	post("492", `[false,[{"index":3,"vecCards":[{"value":8},{"value":43}]}]]`, 1793076615600)
	post("433", `[{"index":2,"betState":2,"score":5904,"betted":100}]`, 1793076617000)
	post("403", `[425]`, 1793076617100)
	failed := post("432", `[3,15,0,100,true]`, 1793076617478)
	if failed.Response.StatusCode() != consts.StatusServiceUnavailable || calls != 1 {
		t.Fatalf("failed response/calls = %s/%d", failed.Response.Body(), calls)
	}
	decision := post("432", `[3,15,0,100,true]`, 1793076617478)
	if decision.Response.StatusCode() != consts.StatusOK ||
		!strings.Contains(string(decision.Response.Body()), `"Result":"Call 100"`) || calls != 2 {
		t.Fatalf("decision response/calls = %s/%d", decision.Response.Body(), calls)
	}
	replayed := post("432", `[3,15,0,100,true]`, 1793076617478)
	if replayed.Response.StatusCode() != consts.StatusOK ||
		!strings.Contains(string(replayed.Response.Body()), `"Result":"Call 100"`) || calls != 2 {
		t.Fatalf("replay response/calls = %s/%d", replayed.Response.Body(), calls)
	}
}

func testDPZXEvent(command, payload string, timestamp int64) *texasPokerEvent {
	return &texasPokerEvent{
		route: pokerEventRouteDPZX, PackageName: dpzxPackageName,
		UUID: "hero@example.com", Timestamp: timestamp,
		UID: json.RawMessage(`2773921`), Cmd: command,
		Direction: "ServerToClient", Location: "TABLE", Data: json.RawMessage(payload),
	}
}
