package service

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestDPZXStraddleRealHandsReachHeroDecision(t *testing.T) {
	for _, tc := range []struct {
		hand, amount int
		position     string
		cards        [2]string
	}{
		{450, 200, "CO-Hero", [2]string{"A-S", "K-D"}},
		{519, 400, "SB-Hero", [2]string{"J-D", "6-C"}},
	} {
		t.Run(fmt.Sprint(tc.hand), func(t *testing.T) {
			raw, err := os.ReadFile(fmt.Sprintf("testdata/dpzx-straddle-%d.json", tc.hand))
			if err != nil {
				t.Fatal(err)
			}
			var steps []struct {
				Cmd       string
				Data      json.RawMessage
				Timestamp int64
			}
			if err = json.Unmarshal(raw, &steps); err != nil {
				t.Fatal(err)
			}
			r := newDPZXEventRegistry()
			var candidate *PokerEventDecision
			for _, step := range steps {
				candidate, err = r.process(testDPZXEvent(step.Cmd, string(step.Data), step.Timestamp), time.Now())
				if err != nil {
					t.Fatalf("cmd %s: %v", step.Cmd, err)
				}
			}
			if candidate == nil {
				t.Fatal("Hero prompt did not produce decision")
			}
			req := candidate.Request
			encoded, _ := json.Marshal(req)
			t.Logf("GTO_REQUEST_%d=%s", tc.hand, encoded)
			if req.Straddle != tc.amount || req.CurrentPlayerPosition != tc.position {
				t.Fatalf("unexpected positions/straddle: %+v", req)
			}
			roles := map[string]int{}
			for _, p := range req.Players {
				roles[p.Position] = p.Bet
				if p.Position == tc.position && (len(p.HoleCards) != 2 || p.HoleCards[0] != tc.cards[0] || p.HoleCards[1] != tc.cards[1]) {
					t.Fatalf("wrong cards: %+v", p)
				}
			}
			if roles["straddle+1"] != 200 {
				t.Fatalf("first straddle missing: %+v", roles)
			}
			if tc.hand == 519 && roles["straddle+2"] != 400 {
				t.Fatalf("second straddle missing: %+v", roles)
			}
		})
	}
}

func TestDPZXStraddleRejectsAmbiguousMarkers(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func([]dpzxPlayerSnapshot)
	}{
		{"duplicate", func(p []dpzxPlayerSnapshot) { p[3].StartState = 3 }},
		{"missing first", func(p []dpzxPlayerSnapshot) { p[1].StartState = 0 }},
		{"wrong seat order", func(p []dpzxPlayerSnapshot) { p[1].StartState, p[2].StartState = p[2].StartState, p[1].StartState }},
		{"unknown", func(p []dpzxPlayerSnapshot) { p[1].StartState = 5 }},
		{"invalid amount", func(p []dpzxPlayerSnapshot) { p[1].Value = 100 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := []dpzxPlayerSnapshot{{Index: 0, StartState: 2, Value: 100}, {Index: 1, StartState: 3, Value: 200}, {Index: 2, StartState: 4, Value: 400}, {Index: 3}, {Index: 6}, {Index: 7, StartState: 1, Value: 50}}
			tc.change(p)
			s := &dpzxTableState{buttonSeat: 6, tableSize: 8, bigBlind: 100, players: map[int]*aapokerPlayerState{}}
			for _, v := range p {
				s.players[v.Index] = &aapokerPlayerState{}
				s.participants = append(s.participants, v.Index)
			}
			if err := s.assignPositions(p); err == nil {
				t.Fatal("ambiguous straddle accepted")
			}
		})
	}
}
