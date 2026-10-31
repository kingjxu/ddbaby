package gto

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var testPositions = []string{"UTG", "UTG+1", "MP", "MP+1", "CO", "BTN", "SB", "BB"}

func preflopRequest(heroPos string, holeCards []string) *TexasGtoDecisionReq {
	players := make([]TexasPlayer, 0, len(testPositions))
	for _, pos := range testPositions {
		player := TexasPlayer{Position: pos, Stack: 10000, IsActive: true}
		switch pos {
		case "SB":
			player.Bet = 50
		case "BB":
			player.Bet = 100
		}
		if pos == heroPos {
			player.Position = heroPos + "-Hero"
			player.HoleCards = holeCards
		}
		players = append(players, player)
	}
	return &TexasGtoDecisionReq{
		GameType:              "no_limit_holdem",
		NumPlayers:            len(players),
		CurrentPot:            150,
		Ante:                  100,
		Straddle:              0,
		SbSize:                50,
		BbSize:                100,
		CommunityCards:        nil,
		GameStage:             "preflop",
		Players:               players,
		CurrentPlayerPosition: heroPos + "-Hero",
	}
}

func postflopRequest(stage string, board, holeCards []string, heroBet, villainBet, pot int) *TexasGtoDecisionReq {
	players := []TexasPlayer{
		{Position: "BTN-Hero", HoleCards: holeCards, Stack: 5000, Bet: heroBet, IsActive: true},
		{Position: "CO", Stack: 5000, Bet: villainBet, IsActive: true},
		{Position: "UTG", Stack: 5000, IsActive: false},
		{Position: "UTG+1", Stack: 5000, IsActive: false},
		{Position: "MP", Stack: 5000, IsActive: false},
		{Position: "MP+1", Stack: 5000, IsActive: false},
		{Position: "SB", Stack: 5000, IsActive: false},
		{Position: "BB", Stack: 5000, IsActive: false},
	}
	return &TexasGtoDecisionReq{
		GameType:              "no_limit_holdem",
		NumPlayers:            len(players),
		CurrentPot:            pot,
		SbSize:                50,
		BbSize:                100,
		CommunityCards:        board,
		GameStage:             stage,
		Players:               players,
		CurrentPlayerPosition: "BTN-Hero",
	}
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	service, err := New(Options{
		TablesDir:       filepath.Join("testdata"),
		DecisionTimeout: 2 * time.Second,
		Samples:         300,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return service
}

func TestPreflopRfiUsesTable(t *testing.T) {
	service := newTestService(t)
	req := preflopRequest("BTN", []string{"A-S", "A-H"})
	resp, err := service.Decide(context.Background(), req)
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if resp.Action != "raise" {
		t.Fatalf("action = %q, want raise (detail=%s)", resp.Action, resp.Detail)
	}
	if resp.RaiseSize != 250 {
		t.Fatalf("raise_size = %d, want 250 (2.5bb)", resp.RaiseSize)
	}
	if resp.ActionProbability != 1 {
		t.Fatalf("probability = %v, want 1", resp.ActionProbability)
	}
}

func TestPreflopExactLine(t *testing.T) {
	service := newTestService(t)
	req := preflopRequest("BTN", []string{"A-S", "A-H"})
	req.ActionHistory = []TexasActionHistory{{Stage: "preflop", Position: "UTG", Action: "raise", Amount: 250}}
	req.CurrentPot = 400
	resp, err := service.Decide(context.Background(), req)
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if resp.Action != "raise" || resp.RaiseSize != 800 {
		t.Fatalf("action/size = %s/%d, want raise/800 (detail=%s)", resp.Action, resp.RaiseSize, resp.Detail)
	}
}

func TestPreflopSituationFallback(t *testing.T) {
	service := newTestService(t)
	req := preflopRequest("BTN", []string{"A-S", "A-H"})
	req.ActionHistory = []TexasActionHistory{
		{Stage: "preflop", Position: "UTG", Action: "raise", Amount: 250},
		{Stage: "preflop", Position: "MP", Action: "call", Amount: 250},
	}
	resp, err := service.Decide(context.Background(), req)
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if resp.Action != "raise" || resp.RaiseSize != 1000 {
		t.Fatalf("action/size = %s/%d, want raise/1000 (detail=%s)", resp.Action, resp.RaiseSize, resp.Detail)
	}
}

func TestPreflopNearestStackBucket(t *testing.T) {
	service := newTestService(t)
	req := preflopRequest("BTN", []string{"A-S", "A-H"})
	for i := range req.Players {
		if req.Players[i].Position == "BTN-Hero" {
			req.Players[i].Stack = 9000
		}
	}
	resp, err := service.Decide(context.Background(), req)
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if resp.Action != "raise" {
		t.Fatalf("action = %q, want raise from nearest 100bb table", resp.Action)
	}
}

func TestPreflopBigBlindOption(t *testing.T) {
	service := newTestService(t)
	req := preflopRequest("BB", []string{"7-S", "2-H"})
	resp, err := service.Decide(context.Background(), req)
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if resp.Action != "check" {
		t.Fatalf("action = %q, want check for BB option (detail=%s)", resp.Action, resp.Detail)
	}
}

func TestMissingTablesFallsBackToHeuristic(t *testing.T) {
	service, err := New(Options{TablesDir: filepath.Join("testdata", "does-not-exist"), Samples: 300})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	req := preflopRequest("UTG", []string{"7-S", "2-H"})
	resp, err := service.Decide(context.Background(), req)
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if resp.Action != "fold" && resp.Action != "raise" && resp.Action != "call" {
		t.Fatalf("unexpected preflop action %q", resp.Action)
	}
	if resp.Detail == "" {
		t.Fatal("detail must not be empty")
	}
}

func TestLoadTablesRejectsBadSchema(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte(`{"players": 8}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPreflopTables(dir); err == nil {
		t.Fatal("expected error for table without nodes/stack")
	}
}
