package model

import (
	"context"
	"encoding/json"
	"fmt"
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

func TestForcedPositionsAdvancePastEmptyAndWaitingSeats(t *testing.T) {
	villains := []map[string]interface{}{
		{"seat": 1, "status": "active", "stack": 100},
		{"seat": 2, "status": "empty"},
		{"seat": 3, "status": "waiting"},
		{"seat": 4, "status": "active", "stack": 100},
		{"seat": 5, "status": "active", "stack": 100},
		{"seat": 6, "status": "active", "stack": 100},
		{"seat": 8, "status": "empty"},
	}
	flop := makeTexasResultWithSeats(t, 8, 7, villains)
	turn := makeTexasResultWithSeats(t, 8, 7, villains)
	turn.TableInfo.Stage = "turn"

	players, currentPosition := buildPlayers(flop, 2)
	wantPositions := []string{"BTN", "SB", "BB", "straddle", "CO-Hero"}
	if currentPosition != "CO-Hero" {
		t.Fatalf("current position = %q, want CO-Hero", currentPosition)
	}
	if len(players) != len(wantPositions) {
		t.Fatalf("players length = %d, want %d", len(players), len(wantPositions))
	}
	for i, want := range wantPositions {
		if players[i].Position != want {
			t.Errorf("players[%d].Position = %q, want %q", i, players[i].Position, want)
		}
	}

	seats := actionSeats(flop)
	if len(seats) != len(wantPositions) {
		t.Fatalf("action seats length = %d, want %d", len(seats), len(wantPositions))
	}
	for i, want := range wantPositions {
		if seats[i].position != want {
			t.Errorf("actionSeats[%d].position = %q, want %q", i, seats[i].position, want)
		}
	}

	history := buildActionHistory(context.Background(), []*TexasResult{flop, turn})
	wantHistoryPositions := []string{"SB", "BB", "straddle", "CO-Hero", "BTN"}
	if len(history) != len(wantHistoryPositions) {
		t.Fatalf("history length = %d, want %d", len(history), len(wantHistoryPositions))
	}
	for i, want := range wantHistoryPositions {
		if history[i].Position != want {
			t.Errorf("history[%d].Position = %q, want %q", i, history[i].Position, want)
		}
	}
}

func TestWaitingHeroDoesNotCountAsParticipant(t *testing.T) {
	villains := []map[string]interface{}{
		{"seat": 1, "status": "active", "stack": 100},
		{"seat": 2, "status": "active", "stack": 100},
		{"seat": 3, "status": "active", "stack": 100},
		{"seat": 4, "status": "active", "stack": 100},
	}
	result := makeTexasResultWithSeats(t, 5, 5, villains)
	result.HeroInfo.Status = "waiting"

	players, currentPosition := buildPlayers(result, 2)
	if currentPosition != "" {
		t.Fatalf("current position = %q, want empty for waiting hero", currentPosition)
	}
	wantPositions := []string{"BTN", "SB", "BB", "straddle"}
	if len(players) != len(wantPositions) {
		t.Fatalf("players length = %d, want %d", len(players), len(wantPositions))
	}
	for i, want := range wantPositions {
		if players[i].Position != want {
			t.Errorf("players[%d].Position = %q, want %q", i, players[i].Position, want)
		}
	}

	seats := actionSeats(result)
	if len(seats) != len(wantPositions) {
		t.Fatalf("action seats length = %d, want %d", len(seats), len(wantPositions))
	}
	for i, want := range wantPositions {
		if seats[i].position != want {
			t.Errorf("actionSeats[%d].position = %q, want %q", i, seats[i].position, want)
		}
	}

	next := makeTexasResultWithSeats(t, 5, 5, villains)
	next.HeroInfo.Status = "waiting"
	next.TableInfo.Stage = "turn"
	history := buildActionHistory(context.Background(), []*TexasResult{result, next})
	wantHistoryPositions := []string{"SB", "BB", "straddle", "BTN"}
	if len(history) != len(wantHistoryPositions) {
		t.Fatalf("history length = %d, want %d", len(history), len(wantHistoryPositions))
	}
	for i, want := range wantHistoryPositions {
		if history[i].Position != want {
			t.Errorf("history[%d].Position = %q, want %q", i, history[i].Position, want)
		}
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

func historyContainsAction(history []TexasActionHistory, stage, position, action string, amount int) bool {
	for _, item := range history {
		if item.Stage == stage && item.Position == position && item.Action == action && item.Amount == amount {
			return true
		}
	}
	return false
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

func TestPrepareActionHistoryKeepsRecognizedHeroOpen(t *testing.T) {
	initial := makeTexasResultWithSeats(t, 2, 1, []map[string]interface{}{
		{"seat": 2, "status": "active", "stack": 126, "current_bet": 0},
		{"seat": 3, "status": "active", "stack": 119, "current_bet": 1},
		{"seat": 4, "status": "active", "stack": 244, "current_bet": 2},
		{"seat": 5, "status": "active", "stack": 271, "current_bet": 4},
		{"seat": 6, "status": "empty", "stack": 0, "current_bet": 0},
		{"seat": 7, "status": "folded", "stack": 219, "current_bet": 0},
		{"seat": 8, "status": "active", "stack": 245, "current_bet": 4},
	})
	initial.TableInfo.Stage = "preflop"
	initial.TableInfo.BlindSize = "1/2/4(2)"
	initial.HeroInfo.CurrentBet = 0
	PrepareActionHistory(nil, initial)

	afterOpen := makeTexasResultWithSeats(t, 2, 1, []map[string]interface{}{
		{"seat": 2, "status": "active", "stack": 126, "current_bet": 0},
		{"seat": 3, "status": "active", "stack": 119, "current_bet": 1},
		{"seat": 4, "status": "active", "stack": 244, "current_bet": 2},
		{"seat": 5, "status": "active", "stack": 271, "current_bet": 4},
		{"seat": 6, "status": "empty", "stack": 0, "current_bet": 0},
		{"seat": 7, "status": "folded", "stack": 219, "current_bet": 0},
		{"seat": 8, "status": "active", "stack": 245, "current_bet": 4},
	})
	afterOpen.TableInfo.Stage = "preflop"
	afterOpen.TableInfo.BlindSize = "1/2/4(2)"
	afterOpen.HeroInfo.CurrentBet = 33
	PrepareActionHistory(initial, afterOpen)

	hasHeroOpen := func(history []TexasActionHistory) bool {
		for _, action := range history {
			if action.Stage == "preflop" && action.Position == "CO-Hero" &&
				action.Action == "raise" && action.Amount == 33 && action.Status == "active" {
				return true
			}
		}
		return false
	}
	if !hasHeroOpen(afterOpen.ObservedActionHistory) {
		t.Fatalf("history does not contain recognized Hero open 33: %+v", afterOpen.ObservedActionHistory)
	}

	request := Conv2TexasGtoDecisionReq(context.Background(), []*TexasResult{afterOpen})
	if !hasHeroOpen(request.ActionHistory) {
		t.Fatalf("request history lost recognized Hero open 33: %+v", request.ActionHistory)
	}
	foundHero := false
	for _, player := range request.Players {
		if player.Position != "CO-Hero" {
			continue
		}
		foundHero = true
		if player.Bet != 33 {
			t.Fatalf("request Hero bet = %d, want recognized 33", player.Bet)
		}
	}
	if !foundHero {
		t.Fatal("request players do not contain CO-Hero")
	}
}

func fourPlayerPreflopSnapshot(t *testing.T, heroBet, sbBet, bbBet, straddleBet int) *TexasResult {
	t.Helper()
	result := makeTexasResultWithSeats(t, 1, 1, []map[string]interface{}{
		{"seat": 2, "status": "active", "stack": 500, "current_bet": sbBet},
		{"seat": 3, "status": "active", "stack": 500, "current_bet": bbBet},
		{"seat": 4, "status": "active", "stack": 500, "current_bet": straddleBet},
		{"seat": 5, "status": "empty", "stack": 0, "current_bet": 0},
		{"seat": 6, "status": "empty", "stack": 0, "current_bet": 0},
		{"seat": 7, "status": "empty", "stack": 0, "current_bet": 0},
		{"seat": 8, "status": "empty", "stack": 0, "current_bet": 0},
	})
	result.TableInfo.Stage = "preflop"
	result.TableInfo.BlindSize = "1/2/4(2)"
	result.HeroInfo.CurrentBet = heroBet
	return result
}

func fourPlayerFlopSnapshot(t *testing.T, heroTurn bool, btnBet, sbBet int, bbStatus string) *TexasResult {
	t.Helper()
	result := makeTexasResultWithSeats(t, 1, 4, []map[string]interface{}{
		{"seat": 1, "status": "active", "stack": 500, "current_bet": btnBet},
		{"seat": 2, "status": "active", "stack": 500, "current_bet": sbBet},
		{"seat": 3, "status": bbStatus, "stack": 500, "current_bet": 0},
	})
	result.TableInfo.Stage = "flop"
	result.HeroInfo.IsHeroTurn = heroTurn
	return result
}

func k9FlopSnapshot(t *testing.T, heroTurn bool, btnBet, sbBet int, bbStatus string) *TexasResult {
	t.Helper()
	result := makeTexasResultWithSeats(t, 7, 1, []map[string]interface{}{
		{"seat": 2, "status": "folded", "stack": 1572, "current_bet": 0},
		{"seat": 3, "status": "active", "stack": 712, "current_bet": 0},
		{"seat": 4, "status": "active", "stack": 2382, "current_bet": 0},
		{"seat": 5, "status": "folded", "stack": 1553, "current_bet": 0},
		{"seat": 6, "status": "folded", "stack": 5328, "current_bet": 0},
		{"seat": 7, "status": "active", "stack": 3859 - btnBet, "current_bet": btnBet},
		{"seat": 8, "status": "active", "stack": 1836 - sbBet, "current_bet": sbBet},
		{"seat": 9, "status": bbStatus, "stack": 3623, "current_bet": 0},
	})
	result.TableInfo.Stage = "flop"
	result.TableInfo.BlindSize = "3/6/12(6)"
	result.TableInfo.CommunityCards = []string{"Q 红桃", "K 红桃", "4 梅花"}
	result.HeroInfo.Stack = 4918
	result.HeroInfo.HeroCards = []string{"9 红桃", "K 梅花"}
	result.HeroInfo.IsHeroTurn = heroTurn
	return result
}

func TestPrepareActionHistoryDefersHeroCheckFromDecidedFrame(t *testing.T) {
	beforeHero := fourPlayerFlopSnapshot(t, true, 0, 0, "active")
	PrepareActionHistory(nil, beforeHero)

	decided := fourPlayerFlopSnapshot(t, false, 0, 0, "active")
	PrepareActionHistory(beforeHero, decided)

	want := []TexasActionHistory{
		{Stage: "flop", Position: "SB", Action: "check", Amount: 0, Timestamp: 0, Status: "active"},
		{Stage: "flop", Position: "BB", Action: "check", Amount: 0, Timestamp: 1, Status: "active"},
	}
	if !reflect.DeepEqual(decided.ObservedActionHistory, want) {
		t.Fatalf("decided-frame history = %+v, want %+v", decided.ObservedActionHistory, want)
	}
	if decided.PendingHeroBefore == nil {
		t.Fatal("decided frame should leave Hero action pending")
	}
}

func TestPrepareActionHistoryResolvesPendingHeroAsBetInsteadOfCheck(t *testing.T) {
	beforeHero := fourPlayerFlopSnapshot(t, true, 0, 0, "active")
	PrepareActionHistory(nil, beforeHero)

	decided := fourPlayerFlopSnapshot(t, false, 0, 0, "active")
	PrepareActionHistory(beforeHero, decided)

	later := fourPlayerFlopSnapshot(t, false, 121, 0, "active")
	later.HeroInfo.CurrentBet = 37
	later.HeroInfo.Stack = decided.HeroInfo.Stack - 37
	PrepareActionHistory(decided, later)

	wantTail := []TexasActionHistory{
		{Stage: "flop", Position: "straddle-Hero", Action: "bet", Amount: 37, Timestamp: 2, Status: "active"},
		{Stage: "flop", Position: "BTN", Action: "raise", Amount: 121, Timestamp: 3, Status: "active"},
	}
	gotTail := later.ObservedActionHistory[len(later.ObservedActionHistory)-len(wantTail):]
	if !reflect.DeepEqual(gotTail, wantTail) {
		t.Fatalf("pending Hero action tail = %+v, want %+v; full=%+v", gotTail, wantTail, later.ObservedActionHistory)
	}
	for _, action := range later.ObservedActionHistory {
		if action.Position == "straddle-Hero" && action.Action == "check" {
			t.Fatalf("decided frame invented Hero check before visible bet: %+v", later.ObservedActionHistory)
		}
	}
	if later.PendingHeroBefore != nil || len(later.PendingHeroSnapshots) != 0 {
		t.Fatal("visible Hero bet should resolve pending action")
	}
}

func TestPrepareActionHistoryDefersPostHeroFoldUntilBetIsVisible(t *testing.T) {
	beforeHero := fourPlayerFlopSnapshot(t, true, 0, 0, "active")
	PrepareActionHistory(nil, beforeHero)

	decided := fourPlayerFlopSnapshot(t, false, 0, 0, "active")
	setSeatStatus(decided, 1, "folded")
	PrepareActionHistory(beforeHero, decided)
	if len(decided.ObservedActionHistory) != 2 || decided.PendingHeroBefore == nil {
		t.Fatalf("decided frame committed unresolved events: history=%+v pending=%v", decided.ObservedActionHistory, decided.PendingHeroBefore != nil)
	}

	later := fourPlayerFlopSnapshot(t, false, 0, 0, "active")
	setSeatStatus(later, 1, "folded")
	later.HeroInfo.CurrentBet = 37
	PrepareActionHistory(decided, later)

	want := []TexasActionHistory{
		{Stage: "flop", Position: "SB", Action: "check", Amount: 0, Timestamp: 0, Status: "active"},
		{Stage: "flop", Position: "BB", Action: "check", Amount: 0, Timestamp: 1, Status: "active"},
		{Stage: "flop", Position: "straddle-Hero", Action: "bet", Amount: 37, Timestamp: 2, Status: "active"},
		{Stage: "flop", Position: "BTN", Action: "fold", Amount: 0, Timestamp: 3, Status: "folded"},
	}
	if !reflect.DeepEqual(later.ObservedActionHistory, want) {
		t.Fatalf("resolved history = %+v, want %+v", later.ObservedActionHistory, want)
	}
}

func TestPrepareActionHistoryKeepsWholeDeltaPendingAcrossSnapshots(t *testing.T) {
	beforeHero := fourPlayerFlopSnapshot(t, true, 0, 0, "active")
	PrepareActionHistory(nil, beforeHero)

	firstPending := fourPlayerFlopSnapshot(t, false, 0, 0, "active")
	setSeatStatus(firstPending, 1, "folded")
	PrepareActionHistory(beforeHero, firstPending)

	secondPending := fourPlayerFlopSnapshot(t, false, 0, 0, "active")
	setSeatStatus(secondPending, 1, "folded")
	PrepareActionHistory(firstPending, secondPending)
	if len(secondPending.ObservedActionHistory) != 2 || secondPending.PendingHeroBefore == nil {
		t.Fatalf("intermediate frame committed pending events: history=%+v pending=%v", secondPending.ObservedActionHistory, secondPending.PendingHeroBefore != nil)
	}

	confirmed := fourPlayerFlopSnapshot(t, false, 0, 0, "active")
	setSeatStatus(confirmed, 1, "folded")
	confirmed.HeroInfo.CurrentBet = 37
	PrepareActionHistory(secondPending, confirmed)
	if confirmed.PendingHeroBefore != nil || len(confirmed.PendingHeroSnapshots) != 0 {
		t.Fatal("confirmed Hero bet did not clear pending baseline")
	}
	wantTail := []TexasActionHistory{
		{Stage: "flop", Position: "straddle-Hero", Action: "bet", Amount: 37, Timestamp: 2, Status: "active"},
		{Stage: "flop", Position: "BTN", Action: "fold", Amount: 0, Timestamp: 3, Status: "folded"},
	}
	gotTail := confirmed.ObservedActionHistory[len(confirmed.ObservedActionHistory)-len(wantTail):]
	if !reflect.DeepEqual(gotTail, wantTail) {
		t.Fatalf("multi-frame pending tail = %+v, want %+v", gotTail, wantTail)
	}
}

func TestPrepareActionHistoryKeepsOpponentWagerPending(t *testing.T) {
	beforeHero := fourPlayerFlopSnapshot(t, true, 0, 0, "active")
	PrepareActionHistory(nil, beforeHero)

	decided := fourPlayerFlopSnapshot(t, false, 0, 0, "active")
	PrepareActionHistory(beforeHero, decided)

	opponentOnly := fourPlayerFlopSnapshot(t, false, 20, 0, "active")
	PrepareActionHistory(decided, opponentOnly)
	if opponentOnly.PendingHeroBefore == nil {
		t.Fatal("opponent wager incorrectly resolved pending Hero action")
	}
	if len(opponentOnly.ObservedActionHistory) != 2 {
		t.Fatalf("opponent wager committed an ambiguous delta: %+v", opponentOnly.ObservedActionHistory)
	}
	for _, action := range opponentOnly.ObservedActionHistory {
		if action.Position == "straddle-Hero" && action.Action == "check" {
			t.Fatalf("opponent wager invented Hero check: %+v", opponentOnly.ObservedActionHistory)
		}
	}
}

func TestPrepareActionHistoryResolvesEqualPendingWagersAsHeroBetThenCall(t *testing.T) {
	beforeHero := fourPlayerFlopSnapshot(t, true, 0, 0, "active")
	PrepareActionHistory(nil, beforeHero)
	decided := fourPlayerFlopSnapshot(t, false, 0, 0, "active")
	PrepareActionHistory(beforeHero, decided)

	opponentFirst := fourPlayerFlopSnapshot(t, false, 20, 0, "active")
	PrepareActionHistory(decided, opponentFirst)

	heroVisible := fourPlayerFlopSnapshot(t, false, 20, 0, "active")
	heroVisible.HeroInfo.CurrentBet = 20
	PrepareActionHistory(opponentFirst, heroVisible)

	wantTail := []TexasActionHistory{
		{Stage: "flop", Position: "straddle-Hero", Action: "bet", Amount: 20, Timestamp: 2, Status: "active"},
		{Stage: "flop", Position: "BTN", Action: "call", Amount: 20, Timestamp: 3, Status: "active"},
	}
	gotTail := heroVisible.ObservedActionHistory[len(heroVisible.ObservedActionHistory)-len(wantTail):]
	if !reflect.DeepEqual(gotTail, wantTail) {
		t.Fatalf("equal pending wagers tail = %+v, want %+v", gotTail, wantTail)
	}
}

func TestPrepareActionHistoryKeepsAmbiguousPartialWagerPending(t *testing.T) {
	beforeHero := fourPlayerFlopSnapshot(t, true, 0, 0, "active")
	PrepareActionHistory(nil, beforeHero)
	decided := fourPlayerFlopSnapshot(t, false, 0, 0, "active")
	PrepareActionHistory(beforeHero, decided)

	opponentFirst := fourPlayerFlopSnapshot(t, true, 60, 0, "active")
	PrepareActionHistory(decided, opponentFirst)
	if opponentFirst.PendingHeroBefore == nil || len(opponentFirst.ObservedActionHistory) != 2 {
		t.Fatalf("returning Hero turn guessed an unresolved action: history=%+v pending=%v", opponentFirst.ObservedActionHistory, opponentFirst.PendingHeroBefore != nil)
	}

	heroVisible := fourPlayerFlopSnapshot(t, true, 60, 0, "active")
	heroVisible.HeroInfo.CurrentBet = 20
	PrepareActionHistory(opponentFirst, heroVisible)

	if heroVisible.PendingHeroBefore == nil {
		t.Fatal("partial Hero wager after action returned should remain pending")
	}
	if !reflect.DeepEqual(heroVisible.ObservedActionHistory, beforeHero.ObservedActionHistory) {
		t.Fatalf("ambiguous partial wager committed history: %+v", heroVisible.ObservedActionHistory)
	}
}

func TestPrepareActionHistoryResolvesSecondOrbitHeroCall(t *testing.T) {
	beforeHero := fourPlayerFlopSnapshot(t, true, 0, 0, "active")
	PrepareActionHistory(nil, beforeHero)
	decided := fourPlayerFlopSnapshot(t, false, 0, 0, "active")
	PrepareActionHistory(beforeHero, decided)

	backToHero := fourPlayerFlopSnapshot(t, true, 60, 0, "active")
	PrepareActionHistory(decided, backToHero)
	heroCall := fourPlayerFlopSnapshot(t, false, 60, 0, "active")
	heroCall.HeroInfo.CurrentBet = 60
	PrepareActionHistory(backToHero, heroCall)

	wantTail := []TexasActionHistory{
		{Stage: "flop", Position: "straddle-Hero", Action: "check", Amount: 0, Timestamp: 2, Status: "active"},
		{Stage: "flop", Position: "BTN", Action: "bet", Amount: 60, Timestamp: 3, Status: "active"},
		{Stage: "flop", Position: "straddle-Hero", Action: "call", Amount: 60, Timestamp: 4, Status: "active"},
	}
	gotTail := heroCall.ObservedActionHistory[len(heroCall.ObservedActionHistory)-len(wantTail):]
	if !reflect.DeepEqual(gotTail, wantTail) {
		t.Fatalf("second-orbit call tail = %+v, want %+v; history=%+v", gotTail, wantTail, heroCall.ObservedActionHistory)
	}
}

func TestPrepareActionHistoryResolvesSecondOrbitHeroRaise(t *testing.T) {
	beforeHero := fourPlayerFlopSnapshot(t, true, 0, 0, "active")
	PrepareActionHistory(nil, beforeHero)
	decided := fourPlayerFlopSnapshot(t, false, 0, 0, "active")
	PrepareActionHistory(beforeHero, decided)

	backToHero := fourPlayerFlopSnapshot(t, true, 60, 0, "active")
	PrepareActionHistory(decided, backToHero)
	heroRaise := fourPlayerFlopSnapshot(t, false, 60, 0, "active")
	heroRaise.HeroInfo.CurrentBet = 180
	PrepareActionHistory(backToHero, heroRaise)

	wantTail := []TexasActionHistory{
		{Stage: "flop", Position: "straddle-Hero", Action: "check", Amount: 0, Timestamp: 2, Status: "active"},
		{Stage: "flop", Position: "BTN", Action: "bet", Amount: 60, Timestamp: 3, Status: "active"},
		{Stage: "flop", Position: "straddle-Hero", Action: "raise", Amount: 180, Timestamp: 4, Status: "active"},
	}
	gotTail := heroRaise.ObservedActionHistory[len(heroRaise.ObservedActionHistory)-len(wantTail):]
	if !reflect.DeepEqual(gotTail, wantTail) {
		t.Fatalf("second-orbit raise tail = %+v, want %+v; history=%+v", gotTail, wantTail, heroRaise.ObservedActionHistory)
	}
}

func TestPrepareActionHistoryKeepsFirstBetBeforeSecondOrbitCall(t *testing.T) {
	beforeHero := fourPlayerFlopSnapshot(t, true, 0, 0, "active")
	PrepareActionHistory(nil, beforeHero)
	decided := fourPlayerFlopSnapshot(t, false, 0, 0, "active")
	PrepareActionHistory(beforeHero, decided)

	heroBet := fourPlayerFlopSnapshot(t, false, 0, 0, "active")
	heroBet.HeroInfo.CurrentBet = 20
	PrepareActionHistory(decided, heroBet)
	opponentRaise := fourPlayerFlopSnapshot(t, true, 60, 0, "active")
	opponentRaise.HeroInfo.CurrentBet = 20
	PrepareActionHistory(heroBet, opponentRaise)
	heroCall := fourPlayerFlopSnapshot(t, false, 60, 0, "active")
	heroCall.HeroInfo.CurrentBet = 60
	PrepareActionHistory(opponentRaise, heroCall)

	wantTail := []TexasActionHistory{
		{Stage: "flop", Position: "straddle-Hero", Action: "bet", Amount: 20, Timestamp: 2, Status: "active"},
		{Stage: "flop", Position: "BTN", Action: "raise", Amount: 60, Timestamp: 3, Status: "active"},
		{Stage: "flop", Position: "straddle-Hero", Action: "call", Amount: 60, Timestamp: 4, Status: "active"},
	}
	gotTail := heroCall.ObservedActionHistory[len(heroCall.ObservedActionHistory)-len(wantTail):]
	if !reflect.DeepEqual(gotTail, wantTail) {
		t.Fatalf("two-orbit history tail = %+v, want %+v; history=%+v", gotTail, wantTail, heroCall.ObservedActionHistory)
	}
}

func TestPrepareActionHistoryOrdersPendingHeroFoldAfterOpponentBet(t *testing.T) {
	beforeHero := fourPlayerFlopSnapshot(t, true, 0, 0, "active")
	PrepareActionHistory(nil, beforeHero)
	decided := fourPlayerFlopSnapshot(t, false, 0, 0, "active")
	PrepareActionHistory(beforeHero, decided)

	opponentBet := fourPlayerFlopSnapshot(t, true, 20, 0, "active")
	PrepareActionHistory(decided, opponentBet)
	heroFolded := fourPlayerFlopSnapshot(t, false, 20, 0, "active")
	heroFolded.HeroInfo.Status = "folded"
	PrepareActionHistory(opponentBet, heroFolded)

	wantTail := []TexasActionHistory{
		{Stage: "flop", Position: "straddle-Hero", Action: "check", Amount: 0, Timestamp: 2, Status: "active"},
		{Stage: "flop", Position: "BTN", Action: "bet", Amount: 20, Timestamp: 3, Status: "active"},
		{Stage: "flop", Position: "straddle-Hero", Action: "fold", Amount: 0, Timestamp: 4, Status: "folded"},
	}
	gotTail := heroFolded.ObservedActionHistory[len(heroFolded.ObservedActionHistory)-len(wantTail):]
	if !reflect.DeepEqual(gotTail, wantTail) {
		t.Fatalf("pending Hero fold tail = %+v, want %+v; history=%+v", gotTail, wantTail, heroFolded.ObservedActionHistory)
	}
}

func TestPrepareActionHistoryCompletesPendingOpponentWagerAcrossStreet(t *testing.T) {
	beforeHero := fourPlayerFlopSnapshot(t, true, 0, 0, "active")
	PrepareActionHistory(nil, beforeHero)
	decided := fourPlayerFlopSnapshot(t, false, 0, 0, "active")
	PrepareActionHistory(beforeHero, decided)

	opponentWager := fourPlayerFlopSnapshot(t, false, 20, 0, "active")
	PrepareActionHistory(decided, opponentWager)

	turn := fourPlayerFlopSnapshot(t, true, 0, 0, "active")
	turn.TableInfo.Stage = "turn"
	PrepareActionHistory(opponentWager, turn)

	counts := make(map[string]int)
	for _, action := range turn.ObservedActionHistory {
		if action.Stage != "flop" {
			continue
		}
		counts[action.Position+":"+action.Action]++
	}
	if counts["straddle-Hero:check"] != 1 || counts["straddle-Hero:call"] != 1 ||
		counts["BTN:bet"] != 1 {
		t.Fatalf("pending cross-street completion is not one legal sequence: counts=%v history=%+v", counts, turn.ObservedActionHistory)
	}
}

func TestPrepareActionHistoryKeepsTransientPendingWagerUntilStreetEnd(t *testing.T) {
	beforeHero := fourPlayerFlopSnapshot(t, true, 0, 0, "active")
	PrepareActionHistory(nil, beforeHero)
	decided := fourPlayerFlopSnapshot(t, false, 0, 0, "active")
	PrepareActionHistory(beforeHero, decided)

	wagerVisible := fourPlayerFlopSnapshot(t, true, 20, 0, "active")
	PrepareActionHistory(decided, wagerVisible)
	wagerCleared := fourPlayerFlopSnapshot(t, false, 0, 0, "active")
	PrepareActionHistory(wagerVisible, wagerCleared)

	turn := fourPlayerFlopSnapshot(t, true, 0, 0, "active")
	turn.TableInfo.Stage = "turn"
	PrepareActionHistory(wagerCleared, turn)

	wantSequence := []string{
		"straddle-Hero:check:0",
		"BTN:bet:20",
		"SB:call:20",
		"BB:call:20",
		"straddle-Hero:call:20",
	}
	gotSequence := make([]string, 0, len(wantSequence))
	for _, action := range turn.ObservedActionHistory {
		if action.Stage != "flop" {
			continue
		}
		key := fmt.Sprintf("%s:%s:%d", action.Position, action.Action, action.Amount)
		for _, want := range wantSequence {
			if key == want {
				gotSequence = append(gotSequence, key)
			}
		}
	}
	if !reflect.DeepEqual(gotSequence, wantSequence) {
		t.Fatalf("transient wager sequence = %v, want %v; history=%+v", gotSequence, wantSequence, turn.ObservedActionHistory)
	}
}

func TestPrepareActionHistoryClearsPendingAcrossStreetOnce(t *testing.T) {
	flopHero := fourPlayerFlopSnapshot(t, true, 0, 0, "active")
	PrepareActionHistory(nil, flopHero)
	decided := fourPlayerFlopSnapshot(t, false, 0, 0, "active")
	PrepareActionHistory(flopHero, decided)

	turn := fourPlayerFlopSnapshot(t, true, 0, 0, "active")
	turn.TableInfo.Stage = "turn"
	PrepareActionHistory(decided, turn)
	if turn.PendingHeroBefore != nil || len(turn.PendingHeroSnapshots) != 0 {
		t.Fatal("street transition did not clear pending baseline")
	}
	checks := 0
	for _, action := range turn.ObservedActionHistory {
		if action.Stage == "flop" && action.Position == "straddle-Hero" && action.Action == "check" {
			checks++
		}
	}
	if checks != 1 {
		t.Fatalf("flop Hero check count = %d, want 1; history=%+v", checks, turn.ObservedActionHistory)
	}
}

func TestPrepareActionHistoryClearsPendingOnNewHand(t *testing.T) {
	beforeHero := fourPlayerFlopSnapshot(t, true, 0, 0, "active")
	PrepareActionHistory(nil, beforeHero)
	decided := fourPlayerFlopSnapshot(t, false, 0, 0, "active")
	PrepareActionHistory(beforeHero, decided)

	newHand := fourPlayerPreflopSnapshot(t, 0, 1, 2, 4)
	newHand.TableInfo.ButtonSeat = 2
	PrepareActionHistory(decided, newHand)
	if newHand.PendingHeroBefore != nil || len(newHand.PendingHeroSnapshots) != 0 {
		t.Fatal("new hand retained pending Hero baseline")
	}
	for _, action := range newHand.ObservedActionHistory {
		if action.Stage == "flop" {
			t.Fatalf("new hand retained prior street action: %+v", newHand.ObservedActionHistory)
		}
	}
}

func TestPrepareActionHistoryKeepsReturnedHeroTurnPending(t *testing.T) {
	beforeHero := makeTexasResultWithSeats(t, 1, 3, []map[string]interface{}{
		{"seat": 1, "status": "active", "stack": 500, "current_bet": 0},
		{"seat": 2, "status": "folded", "stack": 500, "current_bet": 0},
		{"seat": 4, "status": "active", "stack": 500, "current_bet": 0},
		{"seat": 5, "status": "active", "stack": 500, "current_bet": 0},
		{"seat": 6, "status": "folded", "stack": 500, "current_bet": 0},
		{"seat": 7, "status": "folded", "stack": 500, "current_bet": 0},
		{"seat": 8, "status": "folded", "stack": 500, "current_bet": 0},
	})
	beforeHero.TableInfo.Stage = "flop"
	beforeHero.HeroInfo.IsHeroTurn = true
	PrepareActionHistory(nil, beforeHero)

	afterHero := *beforeHero
	afterHero.VillainsInfo = append(afterHero.VillainsInfo[:0:0], afterHero.VillainsInfo...)
	afterHero.HeroInfo.IsHeroTurn = false
	PrepareActionHistory(beforeHero, &afterHero)

	backToHero := afterHero
	backToHero.VillainsInfo = append(backToHero.VillainsInfo[:0:0], backToHero.VillainsInfo...)
	backToHero.HeroInfo.IsHeroTurn = true
	setSeatBet(&backToHero, 1, 20)
	PrepareActionHistory(&afterHero, &backToHero)

	if backToHero.PendingHeroBefore == nil {
		t.Fatal("returning Hero turn incorrectly resolved pending action")
	}
	if !reflect.DeepEqual(backToHero.ObservedActionHistory, afterHero.ObservedActionHistory) {
		t.Fatalf("returning Hero turn committed ambiguous actions: before=%+v after=%+v", afterHero.ObservedActionHistory, backToHero.ObservedActionHistory)
	}
}

func TestInferSnapshotActionsDoesNotInventCheckBeforeFirstActor(t *testing.T) {
	firstActor := makeTexasResultWithSeats(t, 1, 2, []map[string]interface{}{
		{"seat": 1, "status": "active", "stack": 500, "current_bet": 0},
		{"seat": 3, "status": "active", "stack": 500, "current_bet": 0},
		{"seat": 4, "status": "active", "stack": 500, "current_bet": 0},
		{"seat": 5, "status": "empty", "stack": 0, "current_bet": 0},
		{"seat": 6, "status": "empty", "stack": 0, "current_bet": 0},
		{"seat": 7, "status": "empty", "stack": 0, "current_bet": 0},
		{"seat": 8, "status": "empty", "stack": 0, "current_bet": 0},
	})
	firstActor.TableInfo.Stage = "flop"
	firstActor.HeroInfo.IsHeroTurn = true

	if history := inferSnapshotActions(firstActor); len(history) != 0 {
		t.Fatalf("first actor received invented prior actions: %+v", history)
	}
}

func TestInferSnapshotActionsRecordsChecksOnlyBeforeMiddleActor(t *testing.T) {
	middleActor := makeTexasResultWithSeats(t, 1, 3, []map[string]interface{}{
		{"seat": 1, "status": "active", "stack": 500, "current_bet": 0},
		{"seat": 2, "status": "active", "stack": 500, "current_bet": 0},
		{"seat": 4, "status": "active", "stack": 500, "current_bet": 0},
		{"seat": 5, "status": "empty", "stack": 0, "current_bet": 0},
		{"seat": 6, "status": "empty", "stack": 0, "current_bet": 0},
		{"seat": 7, "status": "empty", "stack": 0, "current_bet": 0},
		{"seat": 8, "status": "empty", "stack": 0, "current_bet": 0},
	})
	middleActor.TableInfo.Stage = "flop"
	middleActor.HeroInfo.IsHeroTurn = true

	want := []TexasActionHistory{
		{Stage: "flop", Position: "SB", Action: "check", Amount: 0, Timestamp: 0, Status: "active"},
	}
	if history := inferSnapshotActions(middleActor); !reflect.DeepEqual(history, want) {
		t.Fatalf("middle actor history = %+v, want %+v", history, want)
	}
}

func TestPrepareActionHistoryKeepsPriorCheckBeforeHeroWithoutDuplicate(t *testing.T) {
	preflop := makeTexasResultWithSeats(t, 1, 1, []map[string]interface{}{
		{"seat": 2, "status": "active", "stack": 110, "current_bet": 24},
		{"seat": 3, "status": "folded", "stack": 1972, "current_bet": 0},
		{"seat": 4, "status": "waiting", "stack": 0, "current_bet": 0},
		{"seat": 5, "status": "folded", "stack": 892, "current_bet": 0},
		{"seat": 6, "status": "folded", "stack": 203, "current_bet": 0},
		{"seat": 7, "status": "folded", "stack": 352, "current_bet": 0},
		{"seat": 8, "status": "folded", "stack": 507, "current_bet": 0},
	})
	preflop.TableInfo.Stage = "preflop"
	preflop.TableInfo.BlindSize = "1/2/4(2)"
	preflop.HeroInfo.CurrentBet = 24
	preflop.ObservedActionHistory = []TexasActionHistory{
		{Stage: "preflop", Position: "BTN-Hero", Action: "raise", Amount: 24, Timestamp: 0, Status: "active"},
		{Stage: "preflop", Position: "SB", Action: "call", Amount: 24, Timestamp: 1, Status: "active"},
	}

	flopHero := makeTexasResultWithSeats(t, 1, 1, []map[string]interface{}{
		{"seat": 2, "status": "active", "stack": 110, "current_bet": 0},
		{"seat": 3, "status": "folded", "stack": 1972, "current_bet": 0},
		{"seat": 4, "status": "waiting", "stack": 0, "current_bet": 0},
		{"seat": 5, "status": "folded", "stack": 892, "current_bet": 0},
		{"seat": 6, "status": "folded", "stack": 203, "current_bet": 0},
		{"seat": 7, "status": "folded", "stack": 352, "current_bet": 0},
		{"seat": 8, "status": "folded", "stack": 507, "current_bet": 0},
	})
	flopHero.TableInfo.Stage = "flop"
	flopHero.TableInfo.BlindSize = "1/2/4(2)"
	flopHero.HeroInfo.IsHeroTurn = true
	PrepareActionHistory(preflop, flopHero)

	decided := *flopHero
	decided.HeroInfo.IsHeroTurn = false
	PrepareActionHistory(flopHero, &decided)

	turn := decided
	turn.VillainsInfo = append(turn.VillainsInfo[:0:0], turn.VillainsInfo...)
	turn.TableInfo.Stage = "turn"
	turn.HeroInfo.IsHeroTurn = true
	setSeatBet(&turn, 2, 36)
	PrepareActionHistory(&decided, &turn)

	checks := make([]TexasActionHistory, 0, 2)
	for _, action := range turn.ObservedActionHistory {
		if action.Stage == "flop" && action.Action == "check" {
			checks = append(checks, action)
		}
	}
	want := []TexasActionHistory{
		{Stage: "flop", Position: "SB", Action: "check", Amount: 0, Timestamp: 2, Status: "active"},
		{Stage: "flop", Position: "BTN-Hero", Action: "check", Amount: 0, Timestamp: 3, Status: "active"},
	}
	if !reflect.DeepEqual(checks, want) {
		t.Fatalf("flop checks = %+v, want %+v; history=%+v", checks, want, turn.ObservedActionHistory)
	}
}

func TestPrepareActionHistoryKeepsFoldFirstSeenOnNewStreet(t *testing.T) {
	flop := fourPlayerFlopSnapshot(t, false, 0, 0, "active")
	flop.ObservedActionHistory = []TexasActionHistory{
		{Stage: "flop", Position: "SB", Action: "check", Timestamp: 0, Status: "active"},
		{Stage: "flop", Position: "BB", Action: "check", Timestamp: 1, Status: "active"},
		{Stage: "flop", Position: "straddle-Hero", Action: "check", Timestamp: 2, Status: "active"},
		{Stage: "flop", Position: "BTN", Action: "check", Timestamp: 3, Status: "active"},
	}

	turn := fourPlayerFlopSnapshot(t, true, 0, 20, "folded")
	turn.TableInfo.Stage = "turn"
	PrepareActionHistory(flop, turn)

	wantTail := []TexasActionHistory{
		{Stage: "turn", Position: "SB", Action: "bet", Amount: 20, Timestamp: 4, Status: "active"},
		{Stage: "turn", Position: "BB", Action: "fold", Amount: 0, Timestamp: 5, Status: "folded"},
	}
	gotTail := turn.ObservedActionHistory[len(turn.ObservedActionHistory)-len(wantTail):]
	if !reflect.DeepEqual(gotTail, wantTail) {
		t.Fatalf("new-street history tail = %+v, want %+v; full=%+v", gotTail, wantTail, turn.ObservedActionHistory)
	}
}

func TestPrepareActionHistoryKeepsPriorStreetAggressorFoldOnNewStreet(t *testing.T) {
	flop := fourPlayerFlopSnapshot(t, false, 20, 20, "active")
	setSeatBet(flop, 3, 20)
	flop.HeroInfo.CurrentBet = 20
	flop.ObservedActionHistory = []TexasActionHistory{
		{Stage: "flop", Position: "BTN", Action: "bet", Amount: 20, Timestamp: 0, Status: "active"},
		{Stage: "flop", Position: "SB", Action: "call", Amount: 20, Timestamp: 1, Status: "active"},
		{Stage: "flop", Position: "BB", Action: "call", Amount: 20, Timestamp: 2, Status: "active"},
		{Stage: "flop", Position: "straddle-Hero", Action: "call", Amount: 20, Timestamp: 3, Status: "active"},
	}

	turn := makeTexasResultWithSeats(t, 1, 4, []map[string]interface{}{
		{"seat": 1, "status": "folded", "stack": 480, "current_bet": 0},
		{"seat": 2, "status": "active", "stack": 460, "current_bet": 20},
		{"seat": 3, "status": "active", "stack": 480, "current_bet": 0},
	})
	turn.TableInfo.Stage = "turn"
	turn.HeroInfo.IsHeroTurn = true
	PrepareActionHistory(flop, turn)

	for _, action := range turn.ObservedActionHistory {
		if action.Position == "BTN" && action.Action == "fold" && action.Stage != "turn" {
			t.Fatalf("prior aggressor fold assigned to %s, want turn: %+v", action.Stage, turn.ObservedActionHistory)
		}
	}
	if !historyContainsAction(turn.ObservedActionHistory, "turn", "BTN", "fold", 0) {
		t.Fatalf("turn does not contain prior aggressor fold: %+v", turn.ObservedActionHistory)
	}
}

func TestPrepareActionHistoryKeepsUnansweredFoldOnPriorStreet(t *testing.T) {
	flop := fourPlayerFlopSnapshot(t, false, 20, 20, "active")
	setSeatBet(flop, 3, 0)
	flop.HeroInfo.CurrentBet = 20
	flop.ObservedActionHistory = []TexasActionHistory{
		{Stage: "flop", Position: "BTN", Action: "bet", Amount: 20, Timestamp: 0, Status: "active"},
		{Stage: "flop", Position: "SB", Action: "call", Amount: 20, Timestamp: 1, Status: "active"},
	}

	turn := makeTexasResultWithSeats(t, 1, 4, []map[string]interface{}{
		{"seat": 1, "status": "active", "stack": 480, "current_bet": 0},
		{"seat": 2, "status": "active", "stack": 480, "current_bet": 0},
		{"seat": 3, "status": "folded", "stack": 500, "current_bet": 0},
	})
	turn.TableInfo.Stage = "turn"
	turn.HeroInfo.IsHeroTurn = true
	PrepareActionHistory(flop, turn)

	if !historyContainsAction(turn.ObservedActionHistory, "flop", "BB", "fold", 0) {
		t.Fatalf("unanswered fold was not assigned to flop: %+v", turn.ObservedActionHistory)
	}
	if historyContainsAction(turn.ObservedActionHistory, "turn", "BB", "fold", 0) {
		t.Fatalf("unanswered fold was duplicated on turn: %+v", turn.ObservedActionHistory)
	}
}

func TestInferStageCompletionAllowsActionAfterNewBet(t *testing.T) {
	previous := fourPlayerFlopSnapshot(t, false, 0, 0, "active")
	previous.ObservedActionHistory = []TexasActionHistory{
		{Stage: "flop", Position: "BTN", Action: "check", Timestamp: 0, Status: "active"},
		{Stage: "flop", Position: "SB", Action: "bet", Amount: 20, Timestamp: 1, Status: "active"},
	}
	setSeatBet(previous, 2, 20)
	nextStreet := fourPlayerFlopSnapshot(t, false, 0, 0, "active")
	nextStreet.TableInfo.Stage = "turn"
	setSeatBet(nextStreet, 2, 0)

	completion := inferStageCompletion(previous, nextStreet)
	foundCall := false
	for _, action := range completion {
		if action.Position == "BTN" && action.Action == "call" && action.Amount == 20 {
			foundCall = true
		}
	}
	if !foundCall {
		t.Fatalf("prior check prevented required call after a new bet: %+v", completion)
	}
}

func TestInferStageCompletionDoesNotFoldPlayerTwiceWithoutNewBet(t *testing.T) {
	previous := fourPlayerFlopSnapshot(t, false, 0, 0, "active")
	previous.ObservedActionHistory = []TexasActionHistory{
		{Stage: "flop", Position: "BB", Action: "check", Timestamp: 0, Status: "active"},
	}
	nextStreet := fourPlayerFlopSnapshot(t, false, 0, 0, "folded")
	nextStreet.TableInfo.Stage = "turn"

	for _, action := range inferStageCompletion(previous, nextStreet) {
		if action.Position == "BB" {
			t.Fatalf("player acted twice without a new bet: %+v", action)
		}
	}
}

func TestPrepareActionHistoryDoesNotTurnUnmatchedBetIntoHeroCheck(t *testing.T) {
	beforeHero := fourPlayerFlopSnapshot(t, true, 84, 84, "folded")
	PrepareActionHistory(nil, beforeHero)
	decided := fourPlayerFlopSnapshot(t, false, 84, 84, "folded")
	PrepareActionHistory(beforeHero, decided)

	for _, action := range decided.ObservedActionHistory {
		if action.Position == "straddle-Hero" && action.Action == "check" {
			t.Fatalf("Hero facing 84 was incorrectly recorded as check: %+v", decided.ObservedActionHistory)
		}
	}
}

func TestPrepareActionHistoryKeepsSecondOrbitPendingWithoutVisibleHeroAction(t *testing.T) {
	beforeHero := k9FlopSnapshot(t, true, 0, 0, "active")
	beforeHero.ObservedActionHistory = []TexasActionHistory{
		{Stage: "preflop", Position: "BB", Action: "call", Amount: 12, Timestamp: 7, Status: "active"},
	}
	decided := k9FlopSnapshot(t, false, 0, 0, "active")
	PrepareActionHistory(beforeHero, decided)

	backToHero := k9FlopSnapshot(t, true, 84, 84, "folded")
	PrepareActionHistory(decided, backToHero)

	if backToHero.PendingHeroBefore == nil {
		t.Fatal("K9 returning Hero turn incorrectly resolved pending action")
	}
	for _, action := range backToHero.ObservedActionHistory {
		if action.Stage == "flop" {
			t.Fatalf("K9 pending segment committed ambiguous flop action: %+v", backToHero.ObservedActionHistory)
		}
	}
}

func TestPrepareActionHistoryReconstructsHeroABCThreeBetFourBetOrbit(t *testing.T) {
	beforeHero := fourPlayerPreflopSnapshot(t, 0, 1, 2, 4)
	PrepareActionHistory(nil, beforeHero)

	afterHero := fourPlayerPreflopSnapshot(t, 12, 1, 2, 4)
	PrepareActionHistory(beforeHero, afterHero)

	backToHero := fourPlayerPreflopSnapshot(t, 12, 40, 40, 100)
	PrepareActionHistory(afterHero, backToHero)

	want := []TexasActionHistory{
		{Stage: "preflop", Position: "BTN-Hero", Action: "raise", Amount: 12, Timestamp: 0, Status: "active"},
		{Stage: "preflop", Position: "SB", Action: "raise", Amount: 40, Timestamp: 1, Status: "active"},
		{Stage: "preflop", Position: "BB", Action: "call", Amount: 40, Timestamp: 2, Status: "active"},
		{Stage: "preflop", Position: "straddle", Action: "raise", Amount: 100, Timestamp: 3, Status: "active"},
	}
	if !reflect.DeepEqual(backToHero.ObservedActionHistory, want) {
		t.Fatalf("Hero-A-B-C-Hero history = %+v, want %+v", backToHero.ObservedActionHistory, want)
	}

	request := Conv2TexasGtoDecisionReq(context.Background(), []*TexasResult{backToHero})
	if !reflect.DeepEqual(request.ActionHistory, want) {
		t.Fatalf("Hero-A-B-C-Hero request history = %+v, want %+v", request.ActionHistory, want)
	}
}

func TestPrepareActionHistoryOrdersFoldAndShortAllInWithinRaiseOrbit(t *testing.T) {
	beforeHero := fourPlayerPreflopSnapshot(t, 0, 1, 2, 4)
	PrepareActionHistory(nil, beforeHero)
	afterHero := fourPlayerPreflopSnapshot(t, 12, 1, 2, 4)
	PrepareActionHistory(beforeHero, afterHero)

	backToHero := fourPlayerPreflopSnapshot(t, 12, 40, 25, 100)
	setSeatStatus(backToHero, 3, "all_in")
	PrepareActionHistory(afterHero, backToHero)

	want := []TexasActionHistory{
		{Stage: "preflop", Position: "BTN-Hero", Action: "raise", Amount: 12, Timestamp: 0, Status: "active"},
		{Stage: "preflop", Position: "SB", Action: "raise", Amount: 40, Timestamp: 1, Status: "active"},
		{Stage: "preflop", Position: "BB", Action: "call", Amount: 25, Timestamp: 2, Status: "all_in"},
		{Stage: "preflop", Position: "straddle", Action: "raise", Amount: 100, Timestamp: 3, Status: "active"},
	}
	if !reflect.DeepEqual(backToHero.ObservedActionHistory, want) {
		t.Fatalf("short-all-in orbit history = %+v, want %+v", backToHero.ObservedActionHistory, want)
	}
}

func TestPrepareActionHistoryKeepsMultipleHeroOrbitsOnSameStreet(t *testing.T) {
	beforeHero := fourPlayerPreflopSnapshot(t, 0, 1, 2, 4)
	PrepareActionHistory(nil, beforeHero)
	afterHeroOpen := fourPlayerPreflopSnapshot(t, 12, 1, 2, 4)
	PrepareActionHistory(beforeHero, afterHeroOpen)
	firstReturn := fourPlayerPreflopSnapshot(t, 12, 40, 40, 100)
	PrepareActionHistory(afterHeroOpen, firstReturn)

	afterHeroCall := fourPlayerPreflopSnapshot(t, 100, 40, 40, 100)
	PrepareActionHistory(firstReturn, afterHeroCall)
	secondReturn := fourPlayerPreflopSnapshot(t, 100, 200, 40, 200)
	setSeatStatus(secondReturn, 3, "folded")
	PrepareActionHistory(afterHeroCall, secondReturn)

	want := []TexasActionHistory{
		{Stage: "preflop", Position: "BTN-Hero", Action: "raise", Amount: 12, Timestamp: 0, Status: "active"},
		{Stage: "preflop", Position: "SB", Action: "raise", Amount: 40, Timestamp: 1, Status: "active"},
		{Stage: "preflop", Position: "BB", Action: "call", Amount: 40, Timestamp: 2, Status: "active"},
		{Stage: "preflop", Position: "straddle", Action: "raise", Amount: 100, Timestamp: 3, Status: "active"},
		{Stage: "preflop", Position: "BTN-Hero", Action: "call", Amount: 100, Timestamp: 4, Status: "active"},
		{Stage: "preflop", Position: "SB", Action: "raise", Amount: 200, Timestamp: 5, Status: "active"},
		{Stage: "preflop", Position: "BB", Action: "fold", Amount: 0, Timestamp: 6, Status: "folded"},
		{Stage: "preflop", Position: "straddle", Action: "call", Amount: 200, Timestamp: 7, Status: "active"},
	}
	if !reflect.DeepEqual(secondReturn.ObservedActionHistory, want) {
		t.Fatalf("multi-orbit history = %+v, want %+v", secondReturn.ObservedActionHistory, want)
	}
}

func TestPrepareActionHistoryCompletesMultiRaiseOrbitOnNextStreet(t *testing.T) {
	beforeHero := fourPlayerPreflopSnapshot(t, 0, 1, 2, 4)
	PrepareActionHistory(nil, beforeHero)
	afterHeroOpen := fourPlayerPreflopSnapshot(t, 12, 1, 2, 4)
	PrepareActionHistory(beforeHero, afterHeroOpen)
	firstReturn := fourPlayerPreflopSnapshot(t, 12, 40, 40, 100)
	PrepareActionHistory(afterHeroOpen, firstReturn)
	afterHeroCall := fourPlayerPreflopSnapshot(t, 100, 40, 40, 100)
	PrepareActionHistory(firstReturn, afterHeroCall)

	flop := fourPlayerPreflopSnapshot(t, 0, 0, 0, 0)
	flop.TableInfo.Stage = "flop"
	setSeatStatus(flop, 3, "folded")
	PrepareActionHistory(afterHeroCall, flop)

	wantTail := []TexasActionHistory{
		{Stage: "preflop", Position: "SB", Action: "call", Amount: 100, Timestamp: 5, Status: "active"},
		{Stage: "preflop", Position: "BB", Action: "fold", Amount: 0, Timestamp: 6, Status: "folded"},
	}
	gotTail := flop.ObservedActionHistory[len(flop.ObservedActionHistory)-len(wantTail):]
	if !reflect.DeepEqual(gotTail, wantTail) {
		t.Fatalf("completed multi-raise orbit tail = %+v, want %+v; history=%+v", gotTail, wantTail, flop.ObservedActionHistory)
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

func TestPrepareActionHistoryRecordsStraddleOptionCheck(t *testing.T) {
	preflop := preflopSnapshot(t, map[int]int{1: 1, 2: 2, 3: 4})
	PrepareActionHistory(nil, preflop)
	flop := preflopSnapshot(t, nil)
	flop.TableInfo.Stage = "flop"
	PrepareActionHistory(preflop, flop)

	straddleChecks := 0
	for _, action := range flop.ObservedActionHistory {
		if action.Action == "check" && action.Amount != 0 {
			t.Fatalf("invalid check amount: %+v", action)
		}
		if action.Stage == "preflop" && action.Position == "straddle" && action.Action == "check" {
			straddleChecks++
		}
	}
	if straddleChecks != 1 {
		t.Fatalf("straddle option check count = %d, want 1; history=%+v", straddleChecks, flop.ObservedActionHistory)
	}
}

func TestPrepareActionHistoryRecordsStraddleCheckForTJFlopRegression(t *testing.T) {
	preflop := makeTexasResultWithSeats(t, 7, 1, []map[string]interface{}{
		{"seat": 2, "status": "active", "stack": 459, "current_bet": 4},
		{"seat": 3, "status": "active", "stack": 194, "current_bet": 4},
		{"seat": 4, "status": "folded", "stack": 171, "current_bet": 0},
		{"seat": 5, "status": "folded", "stack": 601, "current_bet": 0},
		{"seat": 6, "status": "folded", "stack": 907, "current_bet": 0},
		{"seat": 7, "status": "folded", "stack": 838, "current_bet": 0},
		{"seat": 8, "status": "active", "stack": 96, "current_bet": 4},
	})
	preflop.TableInfo.Stage = "preflop"
	preflop.TableInfo.BlindSize = "1/2/4(2)"
	preflop.HeroInfo.Stack = 251
	preflop.HeroInfo.CurrentBet = 4
	preflop.HeroInfo.HeroCards = []string{"J 方块", "10 梅花"}
	PrepareActionHistory(nil, preflop)

	flop := makeTexasResultWithSeats(t, 7, 1, []map[string]interface{}{
		{"seat": 2, "status": "active", "stack": 459, "current_bet": 0},
		{"seat": 3, "status": "active", "stack": 194, "current_bet": 0},
		{"seat": 4, "status": "folded", "stack": 171, "current_bet": 0},
		{"seat": 5, "status": "folded", "stack": 601, "current_bet": 0},
		{"seat": 6, "status": "folded", "stack": 907, "current_bet": 0},
		{"seat": 7, "status": "folded", "stack": 838, "current_bet": 0},
		{"seat": 8, "status": "active", "stack": 96, "current_bet": 0},
	})
	flop.TableInfo.Stage = "flop"
	flop.TableInfo.BlindSize = "1/2/4(2)"
	flop.TableInfo.CommunityCards = []string{"3 黑桃", "10 方块", "8 梅花"}
	flop.HeroInfo.Stack = 251
	flop.HeroInfo.HeroCards = []string{"J 方块", "10 梅花"}
	PrepareActionHistory(preflop, flop)

	request := Conv2TexasGtoDecisionReq(context.Background(), []*TexasResult{flop})
	for _, action := range request.ActionHistory {
		if action.Stage == "preflop" && action.Position == "straddle" && action.Action == "check" {
			return
		}
	}
	t.Fatalf("TJ flop request omitted preflop straddle check: %+v", request.ActionHistory)
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
		GameStage:             "preflop",
		CurrentPlayerPosition: "BTN-Hero",
		Players: []TexasPlayer{
			{Position: "BTN-Hero", HoleCards: []string{"A-S", "K-H"}, Stack: heroStack, Bet: heroBet, IsActive: true},
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

func TestValidateTexasGtoDecisionRequestRejectsUnreadableHeroCards(t *testing.T) {
	tests := []struct {
		name  string
		cards []string
	}{
		{name: "missing", cards: nil},
		{name: "one card", cards: []string{"A-S"}},
		{name: "invalid card", cards: []string{"A-S", "?-H"}},
		{name: "duplicate", cards: []string{"A-S", "A-S"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := decisionRequest(2, 160, 219)
			req.Players[0].HoleCards = tt.cards
			if err := ValidateTexasGtoDecisionRequest(req); err == nil {
				t.Fatalf("cards %v should be rejected", tt.cards)
			}
		})
	}
}

func TestValidateTexasGtoDecisionRequestRejectsHeroBoardDuplicate(t *testing.T) {
	req := decisionRequest(0, 200, 0)
	req.GameStage = "flop"
	req.CommunityCards = []string{"A-S", "7-D", "2-C"}
	if err := ValidateTexasGtoDecisionRequest(req); err == nil {
		t.Fatal("hero card duplicated on board should be rejected")
	}
}

func TestValidateTexasGtoDecisionRequestRejectsWrongBoardCardCount(t *testing.T) {
	tests := []struct {
		name      string
		stage     string
		community []string
	}{
		{name: "unknown stage", stage: "showdown"},
		{name: "flop missing card", stage: "flop", community: []string{"2-S", "3-H"}},
		{name: "turn missing card", stage: "turn", community: []string{"2-S", "3-H", "4-C"}},
		{name: "river missing card", stage: "river", community: []string{"2-S", "3-H", "4-C", "5-D"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := decisionRequest(0, 200, 0)
			req.GameStage = tt.stage
			req.CommunityCards = tt.community
			if err := ValidateTexasGtoDecisionRequest(req); err == nil {
				t.Fatalf("stage %q with cards %v should be rejected", tt.stage, tt.community)
			}
		})
	}
}

func TestValidateTexasGtoDecisionRequestRejectsCrossHandHistory(t *testing.T) {
	req := decisionRequest(0, 200, 0)
	req.GameStage = "river"
	req.CommunityCards = []string{"2-S", "J-H", "K-C", "9-S", "A-D"}
	req.ActionHistory = []TexasActionHistory{
		{Stage: "river", Position: "BB-Hero", Action: "fold", Timestamp: 0},
		{Stage: "preflop", Position: "BTN", Action: "call", Amount: 2, Timestamp: 1},
	}
	if err := ValidateTexasGtoDecisionRequest(req); err == nil {
		t.Fatal("history that regresses from river to preflop should be rejected")
	}
}

func TestValidateTexasGtoDecisionRequestAcceptsMonotonicHistory(t *testing.T) {
	req := decisionRequest(0, 200, 0)
	req.GameStage = "turn"
	req.CommunityCards = []string{"2-S", "J-H", "K-C", "9-S"}
	req.ActionHistory = []TexasActionHistory{
		{Stage: "preflop", Position: "BTN", Action: "call", Amount: 2, Timestamp: 0},
		{Stage: "flop", Position: "BTN", Action: "check", Timestamp: 1},
		{Stage: "flop", Position: "BB-Hero", Action: "check", Timestamp: 2},
	}
	if err := ValidateTexasGtoDecisionRequest(req); err != nil {
		t.Fatalf("valid monotonic history was rejected: %v", err)
	}
}

func TestValidateTexasGtoDecisionRequestRejectsFutureStageHistory(t *testing.T) {
	req := decisionRequest(0, 200, 0)
	req.GameStage = "flop"
	req.CommunityCards = []string{"2-S", "J-H", "K-C"}
	req.ActionHistory = []TexasActionHistory{
		{Stage: "preflop", Position: "BTN", Action: "call", Amount: 2, Timestamp: 0},
		{Stage: "turn", Position: "BTN", Action: "check", Timestamp: 1},
	}
	if err := ValidateTexasGtoDecisionRequest(req); err == nil {
		t.Fatal("history from a future street should be rejected")
	}
}

func TestValidateTexasGtoDecisionRequestRejectsNonIncreasingTimestamp(t *testing.T) {
	req := decisionRequest(0, 200, 0)
	req.GameStage = "flop"
	req.CommunityCards = []string{"2-S", "J-H", "K-C"}
	req.ActionHistory = []TexasActionHistory{
		{Stage: "preflop", Position: "BTN", Action: "call", Amount: 2, Timestamp: 0},
		{Stage: "flop", Position: "BTN", Action: "check", Timestamp: 0},
	}
	if err := ValidateTexasGtoDecisionRequest(req); err == nil {
		t.Fatal("history with a repeated timestamp should be rejected")
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
		{name: "legal big blind option raise", req: decisionRequest(6, 207, 6), resp: &TexasGtoDecisionResp{Action: "raise", RaiseSize: 30}},
		{name: "legal straddle option raise", req: decisionRequest(12, 207, 12), resp: &TexasGtoDecisionResp{Action: "raise", RaiseSize: 84}},
		{name: "open raise without recorded wager", req: decisionRequest(0, 207, 0), resp: &TexasGtoDecisionResp{Action: "raise", RaiseSize: 50}},
		{name: "zero open raise", req: decisionRequest(0, 207, 0), resp: &TexasGtoDecisionResp{Action: "raise"}, wantErr: true},
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

func TestValidateTexasGtoDecisionResponseAcceptsPocketNinesOpenRaise(t *testing.T) {
	req := decisionRequest(0, 207, 0)
	req.BbSize = 2
	req.Straddle = 4
	req.Players[0].HoleCards = []string{"9-C", "9-H"}
	resp := &TexasGtoDecisionResp{Action: "raise", RaiseSize: 24}

	if err := ValidateTexasGtoDecisionResponse(req, resp); err != nil {
		t.Fatalf("preflop open raise was rejected: %v", err)
	}
}

func TestNormalizeTexasGtoFreeFoldToCheck(t *testing.T) {
	tests := []struct {
		name       string
		req        *TexasGtoDecisionReq
		resp       *TexasGtoDecisionResp
		wantChange bool
		wantAction string
	}{
		{name: "preflop straddle option", req: func() *TexasGtoDecisionReq {
			req := decisionRequest(12, 200, 12)
			req.CurrentPlayerPosition = "straddle-Hero"
			req.Players[0].Position = "straddle-Hero"
			return req
		}(), resp: &TexasGtoDecisionResp{Action: "fold"}, wantChange: true, wantAction: "check"},
		{name: "preflop big blind option", req: func() *TexasGtoDecisionReq {
			req := decisionRequest(6, 200, 6)
			req.CurrentPlayerPosition = "BB-Hero"
			req.Players[0].Position = "BB-Hero"
			return req
		}(), resp: &TexasGtoDecisionResp{Action: "fold"}, wantChange: true, wantAction: "check"},
		{name: "postflop first to act without wager", req: func() *TexasGtoDecisionReq { req := decisionRequest(0, 200, 0); req.GameStage = "flop"; return req }(), resp: &TexasGtoDecisionResp{Action: "fold"}, wantChange: true, wantAction: "check"},
		{name: "preflop non-blind matched snapshot", req: decisionRequest(49, 200, 49), resp: &TexasGtoDecisionResp{Action: "fold"}, wantAction: "fold"},
		{name: "facing wager", req: decisionRequest(12, 200, 49), resp: &TexasGtoDecisionResp{Action: "fold"}, wantAction: "fold"},
		{name: "non-fold unchanged", req: decisionRequest(0, 200, 0), resp: &TexasGtoDecisionResp{Action: "call"}, wantAction: "call"},
		{name: "missing Hero", req: &TexasGtoDecisionReq{CurrentPlayerPosition: "BTN-Hero", Players: []TexasPlayer{{Position: "SB", Bet: 1}}}, resp: &TexasGtoDecisionResp{Action: "fold"}, wantAction: "fold"},
		{name: "nil request", resp: &TexasGtoDecisionResp{Action: "fold"}, wantAction: "fold"},
		{name: "nil response", req: decisionRequest(0, 200, 0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeTexasGtoFreeFoldToCheck(tt.req, tt.resp); got != tt.wantChange {
				t.Fatalf("NormalizeTexasGtoFreeFoldToCheck() = %v, want %v", got, tt.wantChange)
			}
			if tt.resp != nil && tt.resp.Action != tt.wantAction {
				t.Fatalf("action = %q, want %q", tt.resp.Action, tt.wantAction)
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

func TestPrepareActionHistoryResetsWhenFixedButtonStartsNewHand(t *testing.T) {
	previous := &TexasResult{}
	previous.TableInfo.ButtonSeat = 5
	previous.TableInfo.Stage = "river"
	previous.HeroInfo.HeroCards = []string{"J 方块", "2 方块"}
	previous.ObservedActionHistory = []TexasActionHistory{
		{Stage: "river", Position: "BB-Hero", Action: "bet", Amount: 14},
		{Stage: "river", Position: "BTN", Action: "raise", Amount: 205, Timestamp: 1},
	}

	current := &TexasResult{}
	current.TableInfo.ButtonSeat = 5
	current.TableInfo.Stage = "preflop"
	current.HeroInfo.HeroCards = []string{"10 梅花", "J 梅花"}

	PrepareActionHistory(previous, current)

	if len(current.ObservedActionHistory) != 0 {
		t.Fatalf("new hand retained previous history: %+v", current.ObservedActionHistory)
	}
}

func TestIsNewHandUsesHeroCardsWhenFirstObservedStreetDoesNotRegress(t *testing.T) {
	previous := &TexasResult{}
	previous.TableInfo.ButtonSeat = 5
	previous.TableInfo.Stage = "preflop"
	previous.HeroInfo.HeroCards = []string{"10 梅花", "2 红桃"}

	current := &TexasResult{}
	current.TableInfo.ButtonSeat = 5
	current.TableInfo.Stage = "flop"
	current.HeroInfo.HeroCards = []string{"A 方块", "K 黑桃"}

	if !IsNewHand(previous, current) {
		t.Fatal("changed complete Hero cards with a fixed button must start a new hand")
	}
}

func TestIsNewHandIgnoresHeroCardOrder(t *testing.T) {
	previous := &TexasResult{}
	previous.TableInfo.ButtonSeat = 5
	previous.TableInfo.Stage = "flop"
	previous.HeroInfo.HeroCards = []string{"A 方块", "K 黑桃"}

	current := &TexasResult{}
	current.TableInfo.ButtonSeat = 5
	current.TableInfo.Stage = "turn"
	current.HeroInfo.HeroCards = []string{"K 黑桃", "A 方块"}

	if IsNewHand(previous, current) {
		t.Fatal("reordered Hero cards must not start a new hand")
	}
}
