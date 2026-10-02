package model

import (
	"context"
	"encoding/json"
	"reflect"
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

func TestPrepareActionHistoryInfersMinimumSequenceAcrossMissedFrames(t *testing.T) {
	initial := preflopSnapshot(t, map[int]int{1: 1, 2: 2, 3: 4})
	PrepareActionHistory(nil, initial)
	lateSnapshot := preflopSnapshot(t, map[int]int{1: 1, 2: 2, 3: 4, 4: 12, 5: 40})
	PrepareActionHistory(initial, lateSnapshot)

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
