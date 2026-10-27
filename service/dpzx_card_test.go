package service

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDPZXCardScreenshotAnchorsAndDeckBoundaries(t *testing.T) {
	for _, test := range []struct {
		id   int
		want string
	}{
		// Two independent screenshot/log pairs, hands486 and125.
		{31, "6-H"}, {3, "4-D"}, {8, "9-D"}, {43, "5-S"},
		{0, "A-D"}, {12, "K-D"}, {13, "A-C"}, {25, "K-C"},
		{26, "A-H"}, {38, "K-H"}, {39, "A-S"}, {51, "K-S"},
		{-1, ""}, {52, ""}, {65, ""}, {106, ""},
	} {
		if got := dpzxCard(test.id); got != test.want {
			t.Errorf("card%d=%s want%s", test.id, got, test.want)
		}
	}
	seen := map[string]bool{}
	for id := 0; id < 52; id++ {
		card := dpzxCard(id)
		if !strings.Contains(card, "-") || seen[card] {
			t.Fatalf("invalid/duplicate deck card%d=%s", id, card)
		}
		seen[card] = true
	}
}

func TestDPZXPrivateAndCommunityCardsUseSameEncoding(t *testing.T) {
	state := &dpzxTableState{heroSeat: -1, expectedSeat: -1, operationSeat: -1}
	for _, step := range []struct{ command, data string }{
		{"430", dpzxRealRoomSnapshot}, {"431", dpzxRealHandStart},
		{"492", `[false,[{"index":3,"vecCards":[{"value":8},{"value":43}]}]]`},
		{"490", `[true,[0,13,26]]`},
		{"490", `[true,[0,13,26,39]]`},
		{"490", `[true,[0,13,26,39,12]]`},
		{"432", `[3,15,0,0,false]`},
	} {
		if err := state.apply(step.command, json.RawMessage(step.data), 1000); err != nil {
			t.Fatal(err)
		}
	}
	req, _ := state.decisionRequest()
	if req == nil {
		t.Fatal("complete hand did not produce request")
	}
	if got := strings.Join(req.CommunityCards, ","); got != "A-D,A-C,A-H,A-S,K-D" {
		t.Fatalf("board=%s", got)
	}
	for _, player := range req.Players {
		if player.Position == req.CurrentPlayerPosition && strings.Join(player.HoleCards, ",") != "9-D,5-S" {
			t.Fatalf("Hero cards=%v", player.HoleCards)
		}
	}
}
