package service

import (
	"encoding/json"
	"testing"
)

func TestDPZXPositionsUseProtocolBlindsAndPreserveDeadButton(t *testing.T) {
	for _, test := range []struct {
		name                 string
		button, hero, sb, bb int
		seats                []int
		want                 map[int]string
	}{
		{"production dead button", 1, 5, 2, 3, []int{0, 2, 3, 4, 5, 6, 7}, map[int]string{2: "SB", 3: "BB", 4: "UTG", 5: "UTG+1-Hero", 6: "MP", 7: "MP+1", 0: "CO"}},
		{"eight players", 0, 5, 1, 2, []int{0, 1, 2, 3, 4, 5, 6, 7}, map[int]string{0: "BTN", 1: "SB", 2: "BB", 3: "UTG", 4: "UTG+1", 5: "MP-Hero", 6: "MP+1", 7: "CO"}},
		{"dead small blind", 0, 4, -1, 2, []int{0, 2, 4, 5}, map[int]string{0: "BTN", 2: "BB", 4: "UTG-Hero", 5: "CO"}},
		{"heads up", 2, 2, 2, 6, []int{2, 6}, map[int]string{2: "BTN-Hero", 6: "BB"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := &dpzxTableState{buttonSeat: test.button, heroSeat: test.hero, tableSize: 8, handNum: 1}
			var players []dpzxPlayerSnapshot
			for _, seat := range test.seats {
				role := 0
				if seat == test.sb {
					role = 1
				}
				if seat == test.bb {
					role = 2
				}
				// Deliberately unrelated chip values: roles must come from startState.
				players = append(players, dpzxPlayerSnapshot{Index: seat, StartState: role, Score: 1000, Value: seat * 73})
			}
			raw, _ := json.Marshal([][]dpzxPlayerSnapshot{players})
			if err := state.applyHandStart(raw); err != nil {
				t.Fatal(err)
			}
			for seat, want := range test.want {
				if got := state.players[seat].position; got != want {
					t.Fatalf("seat%d=%s want%s", seat, got, want)
				}
			}
		})
	}
}

func TestDPZXAmbiguousForcedPositionsStopDecision(t *testing.T) {
	for _, test := range []struct {
		name     string
		straddle bool
		players  []dpzxPlayerSnapshot
	}{
		{"missing big blind", false, []dpzxPlayerSnapshot{{Index: 0}, {Index: 1, StartState: 1}, {Index: 2}}},
		{"duplicate big blind", false, []dpzxPlayerSnapshot{{Index: 0}, {Index: 1, StartState: 2}, {Index: 2, StartState: 2}}},
		{"unknown forced role", false, []dpzxPlayerSnapshot{{Index: 0}, {Index: 1, StartState: 2}, {Index: 2, StartState: 3}}},
		{"enabled straddle", true, []dpzxPlayerSnapshot{{Index: 0}, {Index: 1, StartState: 1}, {Index: 2, StartState: 2}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := &dpzxTableState{buttonSeat: 0, heroSeat: 1, tableSize: 8, handNum: 1, straddleEnabled: test.straddle, complete: true}
			raw, _ := json.Marshal([][]dpzxPlayerSnapshot{test.players})
			if err := state.applyHandStart(raw); err == nil || state.complete {
				t.Fatalf("ambiguous position accepted: %v", err)
			}
			if req, _ := state.decisionRequest(); req != nil {
				t.Fatal("ambiguous positions produced GTO request")
			}
		})
	}
}
