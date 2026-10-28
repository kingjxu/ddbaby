package service

import (
	"fmt"
	"reflect"
	"testing"
)

func aapokerPositionStart(seats []int, button, hero int, straddles map[int]int) aapokerStartPayload {
	data := aapokerStartPayload{BankerID: button, HandNum: 1, SmallChip: 2, BigChip: 4, OperationID: hero, OperationRound: 1}
	data.Chips = make([]int, 9)
	data.CanPlay = make([]int, 9)
	data.FirstCards = make([]int, 9)
	data.SecondCards = make([]int, 9)
	data.Antes = make([]int, 9)
	data.Straddle = make([]int, 9)
	for seat := range data.Chips {
		data.Chips[seat], data.FirstCards[seat], data.SecondCards[seat] = -1, -1, -1
	}
	for _, seat := range seats {
		data.Chips[seat], data.CanPlay[seat], data.Antes[seat] = 1000, 1, 4
	}
	data.FirstCards[hero], data.SecondCards[hero] = 12, 24
	for seat, amount := range straddles {
		data.Straddle[seat] = amount
	}
	return data
}

func TestAAPokerPositionsFollowThisHandsStraddleMarkers(t *testing.T) {
	for _, tc := range []struct {
		name      string
		seats     []int
		button    int
		hero      int
		straddles map[int]int
		want      []string
	}{
		{"eight/no-straddle", []int{0, 1, 2, 3, 4, 5, 6, 7}, 0, 0, nil, []string{"BTN-Hero", "SB", "BB", "UTG", "UTG+1", "MP", "MP+1", "CO"}},
		{"eight/straddle", []int{0, 1, 2, 3, 4, 5, 6, 7}, 0, 0, map[int]int{3: 8}, []string{"BTN-Hero", "SB", "BB", "straddle", "UTG", "UTG+1", "MP", "CO"}},
		{"gaps-and-wrap/no-straddle", []int{0, 2, 4, 6, 8}, 4, 0, nil, []string{"BTN", "SB", "BB", "UTG-Hero", "CO"}},
		{"gaps-and-wrap/straddle", []int{0, 2, 4, 6, 8}, 4, 0, map[int]int{0: 8}, []string{"BTN", "SB", "BB", "straddle-Hero", "CO"}},
		{"heads-up", []int{2, 6}, 2, 2, nil, []string{"BTN-Hero", "BB"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := &aapokerTableState{heroSeat: tc.hero}
			if err := state.applyStart(aapokerPositionStart(tc.seats, tc.button, tc.hero, tc.straddles)); err != nil {
				t.Fatal(err)
			}
			request, _ := state.decisionRequest()
			if request == nil {
				t.Fatal("missing Hero decision request")
			}
			var positions []string
			for _, player := range request.Players {
				positions = append(positions, player.Position)
			}
			if !reflect.DeepEqual(positions, tc.want) {
				t.Fatalf("positions=%v, want %v", positions, tc.want)
			}
			wantStraddle := 0
			if len(tc.straddles) > 0 {
				wantStraddle = 8
			}
			if request.Straddle != wantStraddle || request.CurrentPlayerPosition != state.players[tc.hero].position {
				t.Fatalf("wrong Hero or straddle: %+v", request)
			}
			// An opponent folding must retain the same position in players and history.
			actor := state.orderedParticipantSeats()[1]
			state.applyAction(aapokerActionPayload{SeatID: actor, Action: 6, OperationID: tc.hero, OperationRound: 2, FirstCard: -1, SecondCard: -1})
			request, _ = state.decisionRequest()
			if request == nil || len(request.ActionHistory) != 1 || request.ActionHistory[0].Position != tc.want[1] || request.Players[1].Position != tc.want[1] || request.Players[1].IsActive {
				t.Fatalf("positions changed after fold: %+v", request)
			}
		})
	}
}

func TestAAPokerPositionCountsAndNextHand(t *testing.T) {
	for count := 2; count <= 9; count++ {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			seats := make([]int, count)
			for i := range seats {
				seats[i] = i
			}
			state := &aapokerTableState{heroSeat: 0}
			data := aapokerPositionStart(seats, 0, 0, nil)
			if count >= 4 {
				data.Straddle[3] = 8
				if err := state.applyStart(data); err != nil || state.players[3].position != "straddle" {
					t.Fatalf("straddle hand: err=%v", err)
				}
				data.HandNum++
				data.Straddle = nil
			}
			if err := state.applyStart(data); err != nil {
				t.Fatal(err)
			}
			if state.straddle != 0 || len(state.players) != count {
				t.Fatal("previous hand's straddle leaked")
			}
			seen := map[string]bool{}
			for _, player := range state.players {
				if player.position == "" || player.position == "straddle" || seen[player.position] {
					t.Fatalf("invalid ordinary positions: %+v", state.players)
				}
				seen[player.position] = true
			}
		})
	}
}

func TestAAPokerAmbiguousStraddlesBlockDecision(t *testing.T) {
	for name, markers := range map[string]map[int]int{
		"multiple": {3: 8, 4: 16}, "on-button": {0: 8}, "on-blind": {2: 8},
		"skipped-seat": {4: 8}, "nonparticipant": {8: 8}, "negative": {3: -8}, "not-above-bb": {3: 4},
	} {
		t.Run(name, func(t *testing.T) {
			state := &aapokerTableState{heroSeat: 0}
			if err := state.applyStart(aapokerPositionStart([]int{0, 1, 2, 3, 4}, 0, 0, markers)); err == nil || state.complete {
				t.Fatal("ambiguous markers accepted")
			}
			if request, _ := state.decisionRequest(); request != nil {
				t.Fatal("ambiguous positions reached GTO")
			}
		})
	}
}
