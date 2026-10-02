package model

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func makeTexasResult(t *testing.T, villainCount int) *TexasResult {
	t.Helper()
	villains := make([]map[string]interface{}, 0, villainCount)
	for seat := 1; seat <= villainCount; seat++ {
		villains = append(villains, map[string]interface{}{
			"seat":        seat,
			"status":      "active",
			"stack":       100,
			"current_bet": 0,
		})
	}
	return makeTexasResultWithSeats(t, villainCount+1, villainCount+1, villains)
}

func makeTexasResultWithSeats(t *testing.T, buttonSeat, heroSeat int, villains []map[string]interface{}) *TexasResult {
	t.Helper()
	payload := map[string]interface{}{
		"table_info": map[string]interface{}{
			"button_seat": buttonSeat,
			"stage":       "flop",
			"blind_size":  "1/2/4(2)",
		},
		"hero_info":     map[string]interface{}{"seat": heroSeat, "status": "active", "stack": 100},
		"villains_info": villains,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	var result TexasResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	return &result
}

func TestBuildPlayersSupportsEightAndNineSeatTables(t *testing.T) {
	tests := []struct {
		name          string
		villainCount  int
		wantPositions []string
	}{
		{
			name:          "eight seats",
			villainCount:  7,
			wantPositions: []string{"BTN-Hero", "SB", "BB", "straddle", "UTG", "UTG+1", "MP", "CO"},
		},
		{
			name:          "nine seats",
			villainCount:  8,
			wantPositions: []string{"BTN-Hero", "SB", "BB", "straddle", "UTG", "UTG+1", "MP", "MP+1", "CO"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			players, currentPosition := buildPlayers(makeTexasResult(t, tt.villainCount), 2)
			if len(players) != tt.villainCount+1 {
				t.Fatalf("players length = %d, want %d", len(players), tt.villainCount+1)
			}
			if currentPosition != "BTN-Hero" {
				t.Fatalf("current position = %q, want BTN-Hero", currentPosition)
			}
			for i, want := range tt.wantPositions {
				if players[i].Position != want {
					t.Errorf("players[%d].Position = %q, want %q", i, players[i].Position, want)
				}
			}
		})
	}
}

func TestBuildPlayersUsesActualPlayersForNineSeatTableWithEmptySeat(t *testing.T) {
	villains := make([]map[string]interface{}, 0, 8)
	for seat := 1; seat <= 7; seat++ {
		villains = append(villains, map[string]interface{}{
			"seat":        seat,
			"status":      "active",
			"stack":       100,
			"current_bet": 0,
		})
	}
	// Seat 9 is part of the nine-seat table but is not an actual participant.
	villains = append(villains, map[string]interface{}{
		"seat":   9,
		"status": "empty",
	})

	players, _ := buildPlayers(makeTexasResultWithSeats(t, 8, 8, villains), 2)
	wantPositions := []string{"BTN-Hero", "SB", "BB", "straddle", "UTG", "UTG+1", "MP", "CO"}
	if len(players) != len(wantPositions) {
		t.Fatalf("players length = %d, want %d", len(players), len(wantPositions))
	}
	for i, want := range wantPositions {
		if players[i].Position != want {
			t.Errorf("players[%d].Position = %q, want %q", i, players[i].Position, want)
		}
	}
}

func TestBuildActionHistorySkipsWaitingSeatWithoutShiftingPositions(t *testing.T) {
	villains := make([]map[string]interface{}, 0, 8)
	for seat := 1; seat <= 7; seat++ {
		villains = append(villains, map[string]interface{}{
			"seat":        seat,
			"status":      "active",
			"stack":       100,
			"current_bet": 0,
		})
	}
	villains = append(villains, map[string]interface{}{
		"seat":   9,
		"status": "waiting",
	})
	first := makeTexasResultWithSeats(t, 8, 8, villains)
	first.TableInfo.Stage = "flop"
	second := makeTexasResultWithSeats(t, 8, 8, villains)
	second.TableInfo.Stage = "turn"

	history := buildActionHistory(context.Background(), []*TexasResult{first, second})
	if len(history) != 8 {
		t.Fatalf("history length = %d, want 8", len(history))
	}
	wantPositions := []string{"SB", "BB", "straddle", "UTG", "UTG+1", "MP", "CO", "BTN-Hero"}
	for i, want := range wantPositions {
		if history[i].Position != want {
			t.Errorf("history[%d].Position = %q, want %q", i, history[i].Position, want)
		}
	}
}

func TestBuildPlayersAndActionHistorySupportMultipleEmptySeats(t *testing.T) {
	tests := []struct {
		name          string
		buttonSeat    int
		heroSeat      int
		villains      []map[string]interface{}
		wantPositions []string
	}{
		{
			name:       "nine seat table with two empty seats",
			buttonSeat: 8,
			heroSeat:   8,
			villains: []map[string]interface{}{
				{"seat": 1, "status": "active", "stack": 100},
				{"seat": 2, "status": "active", "stack": 100},
				{"seat": 3, "status": "active", "stack": 100},
				{"seat": 4, "status": "active", "stack": 100},
				{"seat": 5, "status": "active", "stack": 100},
				{"seat": 6, "status": "active", "stack": 100},
				{"seat": 7, "status": "empty"},
				{"seat": 9, "status": "empty"},
			},
			wantPositions: []string{"BTN-Hero", "SB", "BB", "straddle", "UTG", "MP", "CO"},
		},
		{
			name:       "eight seat table with one waiting seat",
			buttonSeat: 8,
			heroSeat:   8,
			villains: []map[string]interface{}{
				{"seat": 1, "status": "active", "stack": 100},
				{"seat": 2, "status": "active", "stack": 100},
				{"seat": 3, "status": "active", "stack": 100},
				{"seat": 4, "status": "active", "stack": 100},
				{"seat": 5, "status": "active", "stack": 100},
				{"seat": 6, "status": "active", "stack": 100},
				{"seat": 7, "status": "waiting"},
			},
			wantPositions: []string{"BTN-Hero", "SB", "BB", "straddle", "UTG", "MP", "CO"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first := makeTexasResultWithSeats(t, tt.buttonSeat, tt.heroSeat, tt.villains)
			first.TableInfo.Stage = "flop"
			second := makeTexasResultWithSeats(t, tt.buttonSeat, tt.heroSeat, tt.villains)
			second.TableInfo.Stage = "turn"

			players, _ := buildPlayers(first, 2)
			if len(players) != len(tt.wantPositions) {
				t.Fatalf("players length = %d, want %d", len(players), len(tt.wantPositions))
			}
			for i, want := range tt.wantPositions {
				if players[i].Position != want {
					t.Errorf("players[%d].Position = %q, want %q", i, players[i].Position, want)
				}
			}

			history := buildActionHistory(context.Background(), []*TexasResult{first, second})
			if len(history) != len(tt.wantPositions) {
				t.Fatalf("history length = %d, want %d", len(history), len(tt.wantPositions))
			}
			wantHistoryPositions := append(append([]string{}, tt.wantPositions[1:]...), tt.wantPositions[0])
			for i, want := range wantHistoryPositions {
				if history[i].Position != want {
					t.Errorf("history[%d].Position = %q, want %q", i, history[i].Position, want)
				}
			}
		})
	}
}

func TestFoldedPlayerKeepsSeatPosition(t *testing.T) {
	villains := []map[string]interface{}{
		{"seat": 1, "status": "folded", "stack": 100},
		{"seat": 2, "status": "active", "stack": 100},
		{"seat": 3, "status": "active", "stack": 100},
		{"seat": 4, "status": "active", "stack": 100},
		{"seat": 5, "status": "active", "stack": 100},
		{"seat": 6, "status": "active", "stack": 100},
		{"seat": 7, "status": "active", "stack": 100},
		{"seat": 9, "status": "empty"},
	}
	flop := makeTexasResultWithSeats(t, 8, 8, villains)
	flop.TableInfo.Stage = "flop"
	turn := makeTexasResultWithSeats(t, 8, 8, villains)
	turn.TableInfo.Stage = "turn"

	players, _ := buildPlayers(flop, 2)
	wantPlayerPositions := []string{"BTN-Hero", "SB", "BB", "straddle", "UTG", "UTG+1", "MP", "CO"}
	if len(players) != len(wantPlayerPositions) {
		t.Fatalf("players length = %d, want %d", len(players), len(wantPlayerPositions))
	}
	for i, want := range wantPlayerPositions {
		if players[i].Position != want {
			t.Errorf("players[%d].Position = %q, want %q", i, players[i].Position, want)
		}
	}
	if players[1].IsActive || players[1].ActionTaken != "fold" {
		t.Fatalf("folded player = %+v, want inactive SB with fold action", players[1])
	}

	history := buildActionHistory(context.Background(), []*TexasResult{flop, turn})
	wantHistoryPositions := []string{"SB", "BB", "straddle", "UTG", "UTG+1", "MP", "CO", "BTN-Hero"}
	if len(history) != len(wantHistoryPositions) {
		t.Fatalf("history length = %d, want %d", len(history), len(wantHistoryPositions))
	}
	for i, want := range wantHistoryPositions {
		if history[i].Position != want {
			t.Errorf("history[%d].Position = %q, want %q", i, history[i].Position, want)
		}
	}
	if history[0].Action != "fold" {
		t.Fatalf("folded player's history action = %q, want fold", history[0].Action)
	}
}

func TestPreflopActionHistoryStartsAfterStraddle(t *testing.T) {
	villains := make([]map[string]interface{}, 0, 7)
	for seat := 1; seat <= 7; seat++ {
		villains = append(villains, map[string]interface{}{
			"seat":        seat,
			"status":      "active",
			"stack":       100,
			"current_bet": 0,
		})
	}
	preflop := makeTexasResultWithSeats(t, 8, 8, villains)
	preflop.TableInfo.Stage = "preflop"
	flop := makeTexasResultWithSeats(t, 8, 8, villains)
	flop.TableInfo.Stage = "flop"

	history := buildActionHistory(context.Background(), []*TexasResult{preflop, flop})
	if len(history) == 0 {
		t.Fatal("history is empty")
	}
	if history[0].Position != "UTG" {
		t.Fatalf("first preflop position = %q, want UTG after straddle", history[0].Position)
	}
}

func setSeatBet(result *TexasResult, seat, bet int) {
	if result.HeroInfo.Seat == seat {
		result.HeroInfo.CurrentBet = bet
		return
	}
	for i := range result.VillainsInfo {
		if result.VillainsInfo[i].Seat == seat {
			result.VillainsInfo[i].CurrentBet = bet
			return
		}
	}
}

func setSeatStatus(result *TexasResult, seat int, status string) {
	if result.HeroInfo.Seat == seat {
		result.HeroInfo.Status = status
		return
	}
	for i := range result.VillainsInfo {
		if result.VillainsInfo[i].Seat == seat {
			result.VillainsInfo[i].Status = status
			return
		}
	}
}

func preflopSnapshot(t *testing.T, bets map[int]int) *TexasResult {
	t.Helper()
	villains := make([]map[string]interface{}, 0, 7)
	for seat := 1; seat <= 7; seat++ {
		villains = append(villains, map[string]interface{}{
			"seat": seat, "status": "active", "stack": 1000, "current_bet": bets[seat],
		})
	}
	result := makeTexasResultWithSeats(t, 8, 8, villains)
	result.TableInfo.Stage = "preflop"
	result.TableInfo.BlindSize = "1/2/4(2)"
	result.HeroInfo.CurrentBet = bets[8]
	return result
}

func TestPrepareActionHistoryPreservesMultiRaiseSequence(t *testing.T) {
	initial := preflopSnapshot(t, map[int]int{1: 1, 2: 2, 3: 4})
	PrepareActionHistory(nil, initial)
	if len(initial.ObservedActionHistory) != 0 {
		t.Fatalf("forced blinds produced actions: %+v", initial.ObservedActionHistory)
	}

	twoBet := preflopSnapshot(t, map[int]int{1: 1, 2: 2, 3: 4, 4: 12})
	PrepareActionHistory(initial, twoBet)
	threeBet := preflopSnapshot(t, map[int]int{1: 1, 2: 2, 3: 4, 4: 12, 5: 40})
	PrepareActionHistory(twoBet, threeBet)
	fourBet := preflopSnapshot(t, map[int]int{1: 1, 2: 2, 3: 4, 4: 100, 5: 40})
	PrepareActionHistory(threeBet, fourBet)

	want := []TexasActionHistory{
		{Stage: "preflop", Position: "UTG", Action: "raise", Amount: 12, Timestamp: 0, Status: "active"},
		{Stage: "preflop", Position: "UTG+1", Action: "raise", Amount: 40, Timestamp: 1, Status: "active"},
		{Stage: "preflop", Position: "UTG", Action: "raise", Amount: 100, Timestamp: 2, Status: "active"},
	}
	if !reflect.DeepEqual(fourBet.ObservedActionHistory, want) {
		t.Fatalf("history = %+v, want %+v", fourBet.ObservedActionHistory, want)
	}

	req := Conv2TexasGtoDecisionReq(context.Background(), []*TexasResult{fourBet})
	if !reflect.DeepEqual(req.ActionHistory, want) {
		t.Fatalf("request history = %+v, want %+v", req.ActionHistory, want)
	}
}

func TestPrepareActionHistoryInfersMinimumSequenceFromFirstHeroFrame(t *testing.T) {
	lateSnapshot := preflopSnapshot(t, map[int]int{1: 1, 2: 2, 3: 4, 4: 12, 5: 40})
	lateSnapshot.HeroInfo.IsHeroTurn = true
	PrepareActionHistory(nil, lateSnapshot)

	want := []TexasActionHistory{
		{Stage: "preflop", Position: "UTG", Action: "raise", Amount: 12, Timestamp: 0, Status: "active"},
		{Stage: "preflop", Position: "UTG+1", Action: "raise", Amount: 40, Timestamp: 1, Status: "active"},
	}
	if !reflect.DeepEqual(lateSnapshot.ObservedActionHistory, want) {
		t.Fatalf("history = %+v, want minimum sequence %+v", lateSnapshot.ObservedActionHistory, want)
	}
}

func TestPrepareActionHistoryDoesNotRewriteChecksOrForcedBlinds(t *testing.T) {
	preflop := preflopSnapshot(t, map[int]int{1: 1, 2: 2, 3: 4})
	PrepareActionHistory(nil, preflop)
	flop := preflopSnapshot(t, nil)
	flop.TableInfo.Stage = "flop"
	PrepareActionHistory(preflop, flop)

	for _, action := range flop.ObservedActionHistory {
		if action.Action == "check" && action.Amount != 0 {
			t.Fatalf("invalid check amount: %+v", action)
		}
	}
}

func TestPrepareActionHistoryIgnoresDuplicateSnapshot(t *testing.T) {
	first := preflopSnapshot(t, map[int]int{1: 1, 2: 2, 3: 4, 4: 12})
	PrepareActionHistory(nil, first)
	duplicate := preflopSnapshot(t, map[int]int{1: 1, 2: 2, 3: 4, 4: 12})
	PrepareActionHistory(first, duplicate)

	if !reflect.DeepEqual(duplicate.ObservedActionHistory, first.ObservedActionHistory) {
		t.Fatalf("duplicate snapshot changed history: before=%+v after=%+v", first.ObservedActionHistory, duplicate.ObservedActionHistory)
	}
}

func TestPrepareActionHistoryDoesNotRepeatFoldAcrossStages(t *testing.T) {
	preflop := preflopSnapshot(t, map[int]int{1: 1, 2: 2, 3: 4})
	setSeatStatus(preflop, 4, "folded")
	PrepareActionHistory(nil, preflop)

	flop := preflopSnapshot(t, nil)
	flop.TableInfo.Stage = "flop"
	setSeatStatus(flop, 4, "folded")
	PrepareActionHistory(preflop, flop)

	foldCount := 0
	for _, action := range flop.ObservedActionHistory {
		if action.Position == "UTG" && action.Action == "fold" {
			foldCount++
		}
	}
	if foldCount != 1 {
		t.Fatalf("UTG fold count = %d, want 1; history=%+v", foldCount, flop.ObservedActionHistory)
	}
}

func TestPrepareActionHistoryCompletesPreviousStreetWithMinimumLegalActions(t *testing.T) {
	preflop := preflopSnapshot(t, map[int]int{1: 1, 2: 2, 3: 4})
	PrepareActionHistory(nil, preflop)

	flop := preflopSnapshot(t, nil)
	flop.TableInfo.Stage = "flop"
	for _, seat := range []int{1, 2, 3, 5, 6, 7} {
		setSeatStatus(flop, seat, "folded")
	}
	PrepareActionHistory(preflop, flop)

	calls := make(map[string]int)
	for _, action := range flop.ObservedActionHistory {
		if action.Stage != "preflop" {
			t.Fatalf("first flop snapshot invented a flop action: %+v", action)
		}
		if action.Action == "check" {
			t.Fatalf("preflop completion emitted check: %+v", action)
		}
		if action.Action == "call" {
			calls[action.Position] = action.Amount
		}
	}
	if calls["UTG"] != 4 || calls["BTN-Hero"] != 4 {
		t.Fatalf("minimum completion calls = %+v, want UTG and BTN-Hero call 4", calls)
	}
}

func TestPrepareActionHistoryOrdersEqualTargetsAfterLastActor(t *testing.T) {
	initial := preflopSnapshot(t, map[int]int{1: 1, 2: 2, 3: 4})
	PrepareActionHistory(nil, initial)

	mpRaise := preflopSnapshot(t, map[int]int{1: 1, 2: 2, 3: 4, 6: 40})
	PrepareActionHistory(initial, mpRaise)

	called := preflopSnapshot(t, map[int]int{1: 1, 2: 2, 3: 4, 4: 100, 6: 40, 8: 100})
	PrepareActionHistory(mpRaise, called)

	wantTail := []TexasActionHistory{
		{Stage: "preflop", Position: "BTN-Hero", Action: "raise", Amount: 100, Timestamp: 1, Status: "active"},
		{Stage: "preflop", Position: "UTG", Action: "call", Amount: 100, Timestamp: 2, Status: "active"},
	}
	got := called.ObservedActionHistory[len(called.ObservedActionHistory)-2:]
	if !reflect.DeepEqual(got, wantTail) {
		t.Fatalf("equal-target actions = %+v, want %+v", got, wantTail)
	}
}

func decisionRequest(heroBet, heroStack, villainBet int) *TexasGtoDecisionReq {
	return &TexasGtoDecisionReq{
		CurrentPlayerPosition: "BTN-Hero",
		Players: []TexasPlayer{
			{Position: "BTN-Hero", Stack: heroStack, Bet: heroBet, IsActive: true},
			{Position: "BB", Stack: 300, Bet: villainBet, IsActive: true},
		},
	}
}

func TestValidateTexasGtoDecisionRequestRejectsUnreadHeroStack(t *testing.T) {
	req := decisionRequest(20, 0, 20)
	if err := ValidateTexasGtoDecisionRequest(req); err == nil {
		t.Fatal("zero current-player stack should be rejected")
	}
}

func TestValidateTexasGtoDecisionResponse(t *testing.T) {
	tests := []struct {
		name    string
		req     *TexasGtoDecisionReq
		resp    *TexasGtoDecisionResp
		wantErr bool
	}{
		{name: "legal check", req: decisionRequest(0, 200, 0), resp: &TexasGtoDecisionResp{Action: "check"}},
		{name: "fold without wager", req: decisionRequest(0, 200, 0), resp: &TexasGtoDecisionResp{Action: "fold"}},
		{name: "legal fold", req: decisionRequest(20, 200, 77), resp: &TexasGtoDecisionResp{Action: "fold"}},
		{name: "check facing bet", req: decisionRequest(20, 200, 77), resp: &TexasGtoDecisionResp{Action: "check"}, wantErr: true},
		{name: "legal bet", req: decisionRequest(0, 200, 0), resp: &TexasGtoDecisionResp{Action: "bet", RaiseSize: 50}},
		{name: "zero bet", req: decisionRequest(0, 200, 0), resp: &TexasGtoDecisionResp{Action: "bet"}, wantErr: true},
		{name: "bet facing wager", req: decisionRequest(0, 200, 20), resp: &TexasGtoDecisionResp{Action: "bet", RaiseSize: 50}, wantErr: true},
		{name: "legal call", req: decisionRequest(20, 200, 77), resp: &TexasGtoDecisionResp{Action: "call"}},
		{name: "call without wager", req: decisionRequest(20, 200, 20), resp: &TexasGtoDecisionResp{Action: "call"}, wantErr: true},
		{name: "legal raise", req: decisionRequest(20, 207, 99), resp: &TexasGtoDecisionResp{Action: "raise", RaiseSize: 220}},
		{name: "raise without wager", req: decisionRequest(0, 207, 0), resp: &TexasGtoDecisionResp{Action: "raise", RaiseSize: 50}, wantErr: true},
		{name: "raise beyond stack", req: decisionRequest(99, 207, 99), resp: &TexasGtoDecisionResp{Action: "raise", RaiseSize: 420}, wantErr: true},
		{name: "raise below current bet", req: decisionRequest(20, 207, 99), resp: &TexasGtoDecisionResp{Action: "raise", RaiseSize: 99}, wantErr: true},
		{name: "unknown action", req: decisionRequest(0, 200, 0), resp: &TexasGtoDecisionResp{Action: "jam"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTexasGtoDecisionResponse(tt.req, tt.resp)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeTexasGtoDecisionToStack(t *testing.T) {
	tests := []struct {
		name       string
		req        *TexasGtoDecisionReq
		resp       *TexasGtoDecisionResp
		wantCapped bool
		wantAction string
		wantSize   int
	}{
		{name: "oversized raise becomes all-in", req: decisionRequest(99, 207, 210), resp: &TexasGtoDecisionResp{Action: "raise", RaiseSize: 420}, wantCapped: true, wantAction: "raise", wantSize: 306},
		{name: "short all-in raise becomes call", req: decisionRequest(20, 50, 99), resp: &TexasGtoDecisionResp{Action: "raise", RaiseSize: 200}, wantCapped: true, wantAction: "call"},
		{name: "oversized bet becomes all-in", req: decisionRequest(0, 200, 0), resp: &TexasGtoDecisionResp{Action: "bet", RaiseSize: 250}, wantCapped: true, wantAction: "bet", wantSize: 200},
		{name: "legal raise unchanged", req: decisionRequest(20, 207, 99), resp: &TexasGtoDecisionResp{Action: "raise", RaiseSize: 220}, wantAction: "raise", wantSize: 220},
		{name: "call unchanged", req: decisionRequest(20, 50, 99), resp: &TexasGtoDecisionResp{Action: "call"}, wantAction: "call"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeTexasGtoDecisionToStack(tt.req, tt.resp); got != tt.wantCapped {
				t.Fatalf("capped = %v, want %v", got, tt.wantCapped)
			}
			if tt.resp.Action != tt.wantAction {
				t.Fatalf("action = %s, want %s", tt.resp.Action, tt.wantAction)
			}
			if tt.resp.RaiseSize != tt.wantSize {
				t.Fatalf("raise_size = %d, want %d", tt.resp.RaiseSize, tt.wantSize)
			}
			if err := ValidateTexasGtoDecisionResponse(tt.req, tt.resp); err != nil {
				t.Fatalf("normalized response is invalid: %v", err)
			}
		})
	}
}

func TestPrepareActionHistoryKeepsFoldInActionOrder(t *testing.T) {
	initial := preflopSnapshot(t, map[int]int{1: 1, 2: 2, 3: 4})
	PrepareActionHistory(nil, initial)

	next := preflopSnapshot(t, map[int]int{1: 1, 2: 2, 3: 4, 5: 12, 6: 12})
	setSeatStatus(next, 4, "folded")
	PrepareActionHistory(initial, next)

	want := []TexasActionHistory{
		{Stage: "preflop", Position: "UTG", Action: "fold", Amount: 0, Timestamp: 0, Status: "folded"},
		{Stage: "preflop", Position: "UTG+1", Action: "raise", Amount: 12, Timestamp: 1, Status: "active"},
		{Stage: "preflop", Position: "MP", Action: "call", Amount: 12, Timestamp: 2, Status: "active"},
	}
	if !reflect.DeepEqual(next.ObservedActionHistory, want) {
		t.Fatalf("history = %+v, want %+v", next.ObservedActionHistory, want)
	}
}

func TestPrepareActionHistoryKeepsFoldAfterPriorRaiser(t *testing.T) {
	initial := preflopSnapshot(t, map[int]int{1: 1, 2: 2, 3: 4})
	PrepareActionHistory(nil, initial)

	next := preflopSnapshot(t, map[int]int{1: 1, 2: 2, 3: 4, 5: 12})
	setSeatStatus(next, 6, "folded")
	PrepareActionHistory(initial, next)

	want := []TexasActionHistory{
		{Stage: "preflop", Position: "UTG+1", Action: "raise", Amount: 12, Timestamp: 0, Status: "active"},
		{Stage: "preflop", Position: "MP", Action: "fold", Amount: 0, Timestamp: 1, Status: "folded"},
	}
	if !reflect.DeepEqual(next.ObservedActionHistory, want) {
		t.Fatalf("history = %+v, want %+v", next.ObservedActionHistory, want)
	}
}

func TestPrepareActionHistoryTreatsAllInAsActiveAction(t *testing.T) {
	initial := preflopSnapshot(t, map[int]int{1: 1, 2: 2, 3: 4})
	PrepareActionHistory(nil, initial)

	next := preflopSnapshot(t, map[int]int{1: 1, 2: 2, 3: 4, 4: 12})
	setSeatStatus(next, 4, "all_in")
	PrepareActionHistory(initial, next)

	want := TexasActionHistory{Stage: "preflop", Position: "UTG", Action: "raise", Amount: 12, Timestamp: 0, Status: "all_in"}
	if len(next.ObservedActionHistory) != 1 || next.ObservedActionHistory[0] != want {
		t.Fatalf("all-in history = %+v, want %+v", next.ObservedActionHistory, want)
	}
}

func TestPrepareActionHistoryTreatsShortAllInAsCallAfterRaise(t *testing.T) {
	initial := preflopSnapshot(t, map[int]int{1: 1, 2: 2, 3: 4})
	PrepareActionHistory(nil, initial)

	next := preflopSnapshot(t, map[int]int{1: 1, 2: 2, 3: 4, 4: 12, 5: 8})
	setSeatStatus(next, 5, "all_in")
	PrepareActionHistory(initial, next)

	want := []TexasActionHistory{
		{Stage: "preflop", Position: "UTG", Action: "raise", Amount: 12, Timestamp: 0, Status: "active"},
		{Stage: "preflop", Position: "UTG+1", Action: "call", Amount: 8, Timestamp: 1, Status: "all_in"},
	}
	if !reflect.DeepEqual(next.ObservedActionHistory, want) {
		t.Fatalf("history = %+v, want %+v", next.ObservedActionHistory, want)
	}
}

func TestPrepareActionHistoryDoesNotCompleteAllInOnNextStreet(t *testing.T) {
	preflop := preflopSnapshot(t, map[int]int{1: 1, 2: 2, 3: 4, 4: 12, 5: 8})
	setSeatStatus(preflop, 5, "all_in")
	PrepareActionHistory(nil, preflop)

	flop := preflopSnapshot(t, nil)
	flop.TableInfo.Stage = "flop"
	setSeatStatus(flop, 5, "all_in")
	PrepareActionHistory(preflop, flop)

	allInActions := 0
	for _, action := range flop.ObservedActionHistory {
		if action.Position == "UTG+1" {
			allInActions++
			if action.Action != "call" || action.Amount != 8 {
				t.Fatalf("unexpected all-in action: %+v", action)
			}
		}
	}
	if allInActions != 1 {
		t.Fatalf("all-in actions = %d, want 1; history=%+v", allInActions, flop.ObservedActionHistory)
	}
}

func TestPrepareActionHistoryOrdersStaticFoldAndShortAllIn(t *testing.T) {
	result := preflopSnapshot(t, map[int]int{1: 1, 2: 2, 3: 4, 4: 12, 5: 8})
	setSeatStatus(result, 5, "all_in")
	setSeatStatus(result, 6, "folded")

	PrepareActionHistory(nil, result)

	want := []TexasActionHistory{
		{Stage: "preflop", Position: "UTG", Action: "raise", Amount: 12, Timestamp: 0, Status: "active"},
		{Stage: "preflop", Position: "UTG+1", Action: "call", Amount: 8, Timestamp: 1, Status: "all_in"},
		{Stage: "preflop", Position: "MP", Action: "fold", Amount: 0, Timestamp: 2, Status: "folded"},
	}
	if !reflect.DeepEqual(result.ObservedActionHistory, want) {
		t.Fatalf("static history = %+v, want %+v", result.ObservedActionHistory, want)
	}
}

func TestPrepareActionHistoryIgnoresPendingBetBelowCurrentTarget(t *testing.T) {
	result := preflopSnapshot(t, map[int]int{1: 1, 2: 2, 3: 4, 4: 2})

	PrepareActionHistory(nil, result)

	if len(result.ObservedActionHistory) != 0 {
		t.Fatalf("pending partial bet produced an action: %+v", result.ObservedActionHistory)
	}
}

func TestPrepareActionHistoryUsesActualBlindForShortTable(t *testing.T) {
	villains := []map[string]interface{}{
		{"seat": 1, "status": "active", "stack": 100, "current_bet": 1},
		{"seat": 2, "status": "active", "stack": 100, "current_bet": 2},
	}
	result := makeTexasResultWithSeats(t, 3, 3, villains)
	result.TableInfo.Stage = "preflop"
	result.TableInfo.BlindSize = "1/2/4(2)"
	result.HeroInfo.CurrentBet = 4

	PrepareActionHistory(nil, result)

	want := TexasActionHistory{Stage: "preflop", Position: "BTN-Hero", Action: "raise", Amount: 4, Timestamp: 0, Status: "active"}
	if len(result.ObservedActionHistory) != 1 || result.ObservedActionHistory[0] != want {
		t.Fatalf("short-table history = %+v, want %+v", result.ObservedActionHistory, want)
	}
}

func TestObservedActionHistoryIsNotSerialized(t *testing.T) {
	result := preflopSnapshot(t, map[int]int{1: 1, 2: 2, 3: 4, 4: 12})
	PrepareActionHistory(nil, result)

	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), "ObservedActionHistory") || strings.Contains(string(payload), "observed_action_history") {
		t.Fatalf("internal history leaked into recognition protocol: %s", payload)
	}
}
