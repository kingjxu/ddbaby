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
		"table_info":    map[string]interface{}{"button_seat": buttonSeat, "stage": "flop"},
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
			wantPositions: []string{"BTN-Hero", "SB", "BB", "UTG", "UTG+1", "UTG+2", "HJ", "CO"},
		},
		{
			name:          "nine seats",
			villainCount:  8,
			wantPositions: []string{"BTN-Hero", "SB", "BB", "UTG", "UTG+1", "UTG+2", "LJ", "HJ", "CO"},
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
	wantPositions := []string{"BTN-Hero", "SB", "BB", "UTG", "UTG+1", "UTG+2", "HJ", "CO"}
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
	wantPositions := []string{"SB", "BB", "UTG", "UTG+1", "UTG+2", "HJ", "CO", "BTN-Hero"}
	for i, want := range wantPositions {
		if history[i].Position != want {
			t.Errorf("history[%d].Position = %q, want %q", i, history[i].Position, want)
		}
	}
}
