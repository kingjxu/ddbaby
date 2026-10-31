package gto

import (
	"context"
	"testing"
)

func mustCards(t *testing.T, values ...string) []int {
	t.Helper()
	out := make([]int, 0, len(values))
	for _, value := range values {
		card, ok := parseCard(value)
		if !ok {
			t.Fatalf("invalid card %q", value)
		}
		out = append(out, card)
	}
	return out
}

func TestEval7CategoryOrdering(t *testing.T) {
	royal := eval7(mustCards(t, "A-S", "K-S", "Q-S", "J-S", "10-S", "2-H", "3-D"))
	quads := eval7(mustCards(t, "A-H", "A-D", "A-C", "A-S", "K-H", "2-D", "3-C"))
	fullHouse := eval7(mustCards(t, "K-H", "K-D", "K-C", "2-S", "2-H", "7-D", "8-C"))
	flush := eval7(mustCards(t, "A-H", "J-H", "8-H", "5-H", "2-H", "K-S", "Q-D"))
	straight := eval7(mustCards(t, "9-S", "8-H", "7-D", "6-C", "5-S", "K-H", "Q-D"))
	trips := eval7(mustCards(t, "Q-H", "Q-D", "Q-C", "A-S", "K-H", "7-D", "2-C"))
	twoPair := eval7(mustCards(t, "J-H", "J-D", "8-C", "8-S", "A-H", "3-D", "2-C"))
	pair := eval7(mustCards(t, "10-H", "10-D", "A-S", "K-H", "7-D", "3-C", "2-S"))
	high := eval7(mustCards(t, "A-S", "K-H", "9-D", "7-C", "5-S", "3-H", "2-D"))

	ordered := []int32{royal, quads, fullHouse, flush, straight, trips, twoPair, pair, high}
	for i := 0; i+1 < len(ordered); i++ {
		if ordered[i] <= ordered[i+1] {
			t.Fatalf("category %d (%d) must beat category %d (%d)", i, ordered[i], i+1, ordered[i+1])
		}
	}
}

func TestEvalWheelStraight(t *testing.T) {
	wheel := eval7(mustCards(t, "A-S", "2-H", "3-D", "4-C", "5-S", "K-H", "Q-D"))
	sixHigh := eval7(mustCards(t, "2-S", "3-H", "4-D", "5-C", "6-S", "K-H", "Q-D"))
	if sixHigh <= wheel {
		t.Fatalf("6-high straight (%d) must beat wheel (%d)", sixHigh, wheel)
	}
}

func TestEquityStrongHandBeatsWeakHand(t *testing.T) {
	aa := [2]int{}
	aa[0], _ = parseCard("A-S")
	aa[1], _ = parseCard("A-H")
	weak := [2]int{}
	weak[0], _ = parseCard("7-S")
	weak[1], _ = parseCard("2-H")

	aaEquity := monteCarloEquity(aa, nil, 1, 2000, 7)
	weakEquity := monteCarloEquity(weak, nil, 1, 2000, 7)
	if aaEquity < weakEquity+0.3 {
		t.Fatalf("AA equity %.3f should clearly beat 72o equity %.3f", aaEquity, weakEquity)
	}
}

func TestHeuristicFreeCheckNeverFolds(t *testing.T) {
	service := newTestService(t)
	req := postflopRequest("river",
		[]string{"A-S", "K-H", "7-D", "2-C", "3-S"},
		[]string{"Q-S", "J-S"},
		500, 500, 1000,
	)
	resp, err := service.Decide(context.Background(), req)
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if resp.Action == "fold" {
		t.Fatalf("free check must not fold (detail=%s)", resp.Detail)
	}
	if resp.Action == "check" && resp.RaiseSize != 0 {
		t.Fatalf("check must have zero size, got %d", resp.RaiseSize)
	}
}

func TestHeuristicFacingBetIsLegal(t *testing.T) {
	service := newTestService(t)
	req := postflopRequest("river",
		[]string{"A-S", "K-H", "7-D", "2-C", "3-S"},
		[]string{"Q-S", "J-S"},
		0, 500, 1000,
	)
	resp, err := service.Decide(context.Background(), req)
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	switch resp.Action {
	case "fold", "call":
		if resp.RaiseSize != 0 {
			t.Fatalf("%s must have zero size", resp.Action)
		}
	case "raise":
		if resp.RaiseSize <= 500 || resp.RaiseSize > 5000 {
			t.Fatalf("raise size %d outside (500, 5000]", resp.RaiseSize)
		}
	default:
		t.Fatalf("unexpected action %q when facing a bet", resp.Action)
	}
}

func TestHeuristicStrongHandRaises(t *testing.T) {
	service := newTestService(t)
	req := postflopRequest("river",
		[]string{"A-S", "A-H", "7-D", "2-C", "3-S"},
		[]string{"A-C", "K-C"},
		0, 500, 1000,
	)
	resp, err := service.Decide(context.Background(), req)
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if resp.Action != "raise" && resp.Action != "call" {
		t.Fatalf("trips should raise or call, got %q (detail=%s)", resp.Action, resp.Detail)
	}
	if resp.Action == "raise" && resp.RaiseSize <= 500 {
		t.Fatalf("raise size %d must exceed the current bet 500", resp.RaiseSize)
	}
}

func TestFinalizeKeepsResponseLegal(t *testing.T) {
	facing := preflopRequest("BTN", []string{"A-S", "A-H"})
	if got := finalize(facing, &TexasGtoDecisionResp{Action: "raise", RaiseSize: 50}); got.Action != "call" {
		t.Fatalf("undersized raise should become call, got %q", got.Action)
	}
	if got := finalize(facing, &TexasGtoDecisionResp{Action: "check"}); got.Action != "fold" {
		t.Fatalf("check facing a bet should become fold, got %q", got.Action)
	}

	option := preflopRequest("BB", []string{"7-S", "2-H"})
	if got := finalize(option, &TexasGtoDecisionResp{Action: "fold"}); got.Action != "check" {
		t.Fatalf("fold without a bet should become check, got %q", got.Action)
	}
	if got := finalize(option, &TexasGtoDecisionResp{Action: "bet", RaiseSize: 200}); got.Action != "bet" || got.RaiseSize != 200 {
		t.Fatalf("legal bet changed: %+v", got)
	}
}
