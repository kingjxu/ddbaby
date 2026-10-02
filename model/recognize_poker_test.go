package model

import (
	"context"
	"encoding/json"
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
