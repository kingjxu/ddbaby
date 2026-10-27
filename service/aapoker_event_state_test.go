package service

import (
	"fmt"
	ddbaby "github.com/kingjxu/ddbaby/biz/model/ddbaby"
	"sync"
	"testing"
	"time"
)

func TestAAPokerEventStateReplaysHandAndDeduplicatesHeroTurn(t *testing.T) {
	registry := newAAPokerEventRegistry()
	now := time.Unix(1792526400, 0)

	apply := func(command string, payload string) *PokerEventDecision {
		t.Helper()
		event := testAAPokerEvent(command, payload)
		candidate, err := registry.process(event, now)
		if err != nil {
			t.Fatalf("process(%s) error = %v", command, err)
		}
		now = now.Add(time.Millisecond)
		return candidate
	}

	if got := apply("REQ_GAME_ENTER_ROOM", `{
		"mySeatID":0,"playerIDArray":[1762080,2836313,2646258,2838496,2859796,-1,2860755,1081977],
		"operationID":4,"operationRound":16}`); got != nil {
		t.Fatal("enter event unexpectedly produced a decision")
	}
	if got := apply("REQ_GAME_RECV_START_INFOR", `{
		"bankerID":0,"operationID":4,"firstCardArray":[12,-1,-1,-1,-1,-1,-1,-1],
		"SecondCardArray":[24,-1,-1,-1,-1,-1,-1,-1],"smallChip":2,"bigChip":4,
		"chipsArray":[762,996,392,1557,606,-1,768,1020],
		"canPlayStatus":[1,1,1,1,1,0,1,1],"seatStatus":[2,2,2,2,2,0,2,2],
		"handNum":3,"straddlePop":[0,0,0,8,0,0,0,0],"operationRound":17,
		"mushroomPool":40,"pool_chips":42,"anteCountArray":[4,6,8,12,4,0,4,4]}`); got != nil {
		t.Fatal("start event unexpectedly produced a decision")
	}
	apply("REQ_GAME_RECV_ACTION", `{"seatID":4,"action":6,"operationID":6,"anteNumber":0,"operationRound":18}`)
	apply("REQ_GAME_RECV_ACTION", `{"seatID":6,"action":2,"operationID":7,"anteNumber":8,"operationRound":19}`)
	candidate := apply("REQ_GAME_RECV_ACTION", `{"seatID":7,"action":2,"operationID":0,"anteNumber":8,"operationRound":20}`)
	if candidate == nil {
		t.Fatal("hero turn did not produce a decision request")
	}
	request := candidate.Request
	if request.CurrentPlayerPosition != "BTN-Hero" || request.GameStage != "preflop" {
		t.Fatalf("hero turn = %s/%s", request.CurrentPlayerPosition, request.GameStage)
	}
	if request.CurrentPot != 58 || request.Ante != 4 || request.Straddle != 8 || request.Mushroom != 10 {
		t.Fatalf("unexpected table amounts: %+v", request)
	}
	if len(request.ActionHistory) != 3 || request.ActionHistory[1].Amount != 8 ||
		request.ActionHistory[2].Amount != 8 {
		t.Fatalf("unexpected action history: %+v", request.ActionHistory)
	}
	if request.Players[0].HoleCards[0] != "A-D" || request.Players[0].HoleCards[1] != "K-C" {
		t.Fatalf("unexpected hero cards: %+v", request.Players[0].HoleCards)
	}

	registry.complete(candidate, nil, now)
	retry := apply("REQ_GAME_RECV_ACTION", `{"seatID":7,"action":2,"operationID":0,"anteNumber":8,"operationRound":20}`)
	if retry == nil || retry.turnKey != candidate.turnKey {
		t.Fatal("failed decision did not preserve retry eligibility")
	}
	want := &PokerEventDecisionResult{Result: "Call 8", HeroCard: []string{"A-D", "K-C"}}
	registry.complete(retry, want, now)
	duplicate := apply("REQ_GAME_RECV_ACTION", `{"seatID":7,"action":2,"operationID":0,"anteNumber":8,"operationRound":20}`)
	if duplicate == nil || duplicate.Cached == nil || duplicate.Cached.Result != want.Result {
		t.Fatal("successful hero turn result was not replayed")
	}
	if duplicate.Request != nil {
		t.Fatal("cached hero turn unexpectedly requested another GTO call")
	}
}

func TestAAPokerEventStateDoesNotTreatWatcherAsHero(t *testing.T) {
	registry := newAAPokerEventRegistry()
	now := time.Unix(1792526400, 0)
	for _, item := range []struct{ command, payload string }{
		{"REQ_GAME_ENTER_ROOM", `{"mySeatID":-1,"playerIDArray":[2574961,2836313],"operationID":0,"operationRound":1}`},
		{"REQ_GAME_RECV_START_INFOR", `{"bankerID":0,"operationID":0,"firstCardArray":[12,-1],"SecondCardArray":[24,-1],"smallChip":2,"bigChip":4,"chipsArray":[100,100],"canPlayStatus":[1,1],"seatStatus":[2,2],"handNum":1,"operationRound":2,"pool_chips":6,"anteCountArray":[2,4]}`},
	} {
		candidate, err := registry.process(testAAPokerEvent(item.command, item.payload), now)
		if err != nil {
			t.Fatalf("process(%s) error = %v", item.command, err)
		}
		if candidate != nil {
			t.Fatal("watcher event produced a hero decision")
		}
		now = now.Add(time.Millisecond)
	}
}

func TestAAPokerEventStateAdvancesStreetOnce(t *testing.T) {
	state := &aapokerTableState{
		heroSeat: 1, handNum: 1, buttonSeat: 0, smallBlind: 2, bigBlind: 4,
		operationRound: 20, stage: "preflop", tableSize: 2,
		players: map[int]*aapokerPlayerState{
			0: {seat: 0, position: "BTN", stack: 100, active: true},
			1: {seat: 1, position: "BB-Hero", stack: 100, active: true, holeCards: []string{"A-D", "K-C"}},
		},
		participants: []int{0, 1}, lastCardsRound: -1, complete: true,
	}
	payload := aapokerCardsPayload{SeatID: 1, Cards: []int{8, 44, 16}, OperationRound: 20}
	if err := state.applyCards(payload); err != nil {
		t.Fatalf("applyCards() error = %v", err)
	}
	if err := state.applyCards(payload); err != nil {
		t.Fatalf("duplicate applyCards() error = %v", err)
	}
	request, _ := state.decisionRequest()
	if request == nil || request.GameStage != "flop" || len(request.CommunityCards) != 3 {
		t.Fatalf("unexpected flop request: %+v", request)
	}
}

func TestAAPokerEventStateBlocksDecisionAfterActionGap(t *testing.T) {
	state := &aapokerTableState{
		heroSeat: 0, handNum: 3, buttonSeat: 0, smallBlind: 2, bigBlind: 4,
		operationSeat: 4, operationRound: 17, stage: "preflop", tableSize: 2,
		complete: true,
		players: map[int]*aapokerPlayerState{
			0: {seat: 0, position: "BTN-Hero", stack: 100, active: true, holeCards: []string{"A-D", "K-C"}},
			1: {seat: 1, position: "BB", stack: 100, active: true},
		},
		participants: []int{0, 1},
	}
	state.applyAction(aapokerActionPayload{
		SeatID: 1, Action: 5, OperationID: 0, OperationRound: 19,
	})
	if request, _ := state.decisionRequest(); request != nil {
		t.Fatal("state with a missing operationRound produced a decision")
	}
}

func TestAAPokerDecisionCacheExpiresAndStaysBounded(t *testing.T) {
	registry := newAAPokerEventRegistry()
	now := time.Unix(1792526400, 0)
	registry.results["expired"] = aapokerDecisionCacheEntry{
		createdAt: now.Add(-PokerEventDecisionResultTTL - time.Second),
		result:    &PokerEventDecisionResult{Result: "expired"},
	}
	for index := 0; index < maxAAPokerDecisionResults; index++ {
		key := fmt.Sprintf("cached-%04d", index)
		registry.results[key] = aapokerDecisionCacheEntry{
			createdAt: now.Add(time.Duration(index) * time.Millisecond),
			result:    &PokerEventDecisionResult{Result: key},
		}
	}
	registry.sessions["new-session"] = &aapokerTableState{inFlightTurn: "new-turn"}
	registry.complete(&PokerEventDecision{
		sessionKey: "new-session",
		turnKey:    "new-turn",
	}, &PokerEventDecisionResult{Result: "new"}, now.Add(time.Minute))

	if len(registry.results) != maxAAPokerDecisionResults {
		t.Fatalf("cached result count = %d", len(registry.results))
	}
	if _, ok := registry.results[decisionCacheKey("new-session", "new-turn")]; !ok {
		t.Fatal("new result was evicted instead of the oldest result")
	}
	if _, ok := registry.results["expired"]; ok {
		t.Fatal("expired result remained cached")
	}
}

func TestAAPokerEventStateAllowsOnlyOneConcurrentDecisionPerTurn(t *testing.T) {
	registry := newAAPokerEventRegistry()
	now := time.Unix(1792526400, 0)
	for _, item := range []struct{ command, payload string }{
		{"REQ_GAME_ENTER_ROOM", `{"mySeatID":0,"playerIDArray":[1762080,2836313,2646258,2838496,2859796,-1,2860755,1081977],"operationID":4,"operationRound":16}`},
		{"REQ_GAME_RECV_START_INFOR", `{"bankerID":0,"operationID":4,"firstCardArray":[12,-1,-1,-1,-1,-1,-1,-1],"SecondCardArray":[24,-1,-1,-1,-1,-1,-1,-1],"smallChip":2,"bigChip":4,"chipsArray":[762,996,392,1557,606,-1,768,1020],"canPlayStatus":[1,1,1,1,1,0,1,1],"seatStatus":[2,2,2,2,2,0,2,2],"handNum":3,"straddlePop":[0,0,0,8,0,0,0,0],"operationRound":17,"mushroomPool":40,"pool_chips":42,"anteCountArray":[4,6,8,12,4,0,4,4]}`},
		{"REQ_GAME_RECV_ACTION", `{"seatID":4,"action":6,"operationID":6,"anteNumber":0,"operationRound":18}`},
		{"REQ_GAME_RECV_ACTION", `{"seatID":6,"action":2,"operationID":7,"anteNumber":8,"operationRound":19}`},
	} {
		if _, err := registry.process(testAAPokerEvent(item.command, item.payload), now); err != nil {
			t.Fatalf("process(%s) error = %v", item.command, err)
		}
		now = now.Add(time.Millisecond)
	}

	const callers = 16
	start := make(chan struct{})
	candidates := make(chan *PokerEventDecision, callers)
	errors := make(chan error, callers)
	var wait sync.WaitGroup
	for index := 0; index < callers; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			candidate, err := registry.process(testAAPokerEvent(
				"REQ_GAME_RECV_ACTION",
				`{"seatID":7,"action":2,"operationID":0,"anteNumber":8,"operationRound":20}`,
			), now)
			if err != nil {
				errors <- err
				return
			}
			candidates <- candidate
		}()
	}
	close(start)
	wait.Wait()
	close(candidates)
	close(errors)
	for err := range errors {
		t.Fatalf("concurrent process error = %v", err)
	}
	count := 0
	for candidate := range candidates {
		if candidate != nil {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("concurrent decision candidates = %d, want 1", count)
	}
}

func testAAPokerEvent(command, payload string) *texasPokerEvent {
	return &texasPokerEvent{
		TexasPokerEventReq: ddbaby.TexasPokerEventReq{
			UUID: "taoyuan-account", Timestamp: 1792526400000,
			UID: ddbaby.PokerEventJSON(`1762080`), DataExtra: ddbaby.PokerEventJSON(`64161112`),
			Cmd: command, Direction: "ServerToClient", Location: "TABLE",
			Data: ddbaby.PokerEventJSON(payload),
		},
	}
}
