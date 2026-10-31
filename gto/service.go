package gto

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

// Options configures the decision service.
type Options struct {
	TablesDir       string
	SolverURL       string
	SolverTimeout   time.Duration
	DecisionTimeout time.Duration
	Samples         int
	Logger          *log.Logger
}

// Service routes one request through L1 (preflop tables), L2 (optional solver
// sidecar), and L4 (heuristic). Every path returns the same response contract.
type Service struct {
	opts      Options
	tables    []*PreflopTable
	heuristic *Heuristic
	solver    *SolverClient
	logger    *log.Logger
}

// New loads tables and prepares the router. A missing tables directory is not
// fatal: the service then runs on the heuristic fallback.
func New(opts Options) (*Service, error) {
	if opts.Logger == nil {
		opts.Logger = log.Default()
	}
	if opts.DecisionTimeout <= 0 {
		opts.DecisionTimeout = 4 * time.Second
	}
	if opts.SolverTimeout <= 0 {
		opts.SolverTimeout = 1500 * time.Millisecond
	}
	if opts.Samples <= 0 {
		opts.Samples = 2000
	}
	tables, err := LoadPreflopTables(opts.TablesDir)
	if err != nil {
		return nil, err
	}
	if len(tables) == 0 {
		opts.Logger.Printf("[gto] no preflop tables loaded from %q; using heuristic only", opts.TablesDir)
	} else {
		opts.Logger.Printf("[gto] loaded %d preflop table(s) from %q", len(tables), opts.TablesDir)
	}
	return &Service{
		opts:      opts,
		tables:    tables,
		heuristic: &Heuristic{Samples: opts.Samples},
		solver:    NewSolverClient(opts.SolverURL, opts.SolverTimeout),
		logger:    opts.Logger,
	}, nil
}

// Decide returns a legal action for the request. It never returns a nil
// response together with a nil error.
func (s *Service) Decide(ctx context.Context, req *TexasGtoDecisionReq) (*TexasGtoDecisionResp, error) {
	if err := ValidateRequest(req); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, s.opts.DecisionTimeout)
	defer cancel()

	stage := normalizeStage(req.GameStage)
	if stage == "preflop" {
		if resp, ok := lookupPreflop(s.tables, req); ok {
			return finalize(req, resp), nil
		}
	}

	if stage != "preflop" && req.ActivePlayers() == 2 && s.solver.Enabled() {
		resp, err := s.solver.Decide(ctx, req)
		if err != nil {
			s.logger.Printf("[gto] solver fallback: %v", err)
		} else if resp != nil {
			return finalize(req, resp), nil
		}
	}

	if resp, ok := s.heuristic.decide(req); ok {
		return finalize(req, resp), nil
	}
	return nil, errors.New("no engine could produce a decision")
}

// Status is the health payload.
func (s *Service) Status() map[string]interface{} {
	return map[string]interface{}{
		"status":   "ok",
		"tables":   len(s.tables),
		"solver":   s.solver.Enabled(),
		"samples":  s.opts.Samples,
		"timeouts": s.opts.DecisionTimeout.String(),
	}
}

// ValidateRequest rejects states that cannot produce a safe decision. It
// mirrors the completeness checks ddbaby already performs before calling the
// upstream, so an invalid request fails fast with 400 instead of a bad action.
func ValidateRequest(req *TexasGtoDecisionReq) error {
	if req == nil {
		return errors.New("request is nil")
	}
	stage := normalizeStage(req.GameStage)
	wantCommunity := -1
	switch stage {
	case "preflop":
		wantCommunity = 0
	case "flop":
		wantCommunity = 3
	case "turn":
		wantCommunity = 4
	case "river":
		wantCommunity = 5
	default:
		return fmt.Errorf("invalid game_stage %q", req.GameStage)
	}
	if len(req.CommunityCards) != wantCommunity {
		return fmt.Errorf("%s needs %d community cards, got %d", stage, wantCommunity, len(req.CommunityCards))
	}
	if len(req.Players) == 0 {
		return errors.New("players is empty")
	}
	if strings.TrimSpace(req.CurrentPlayerPosition) == "" {
		return errors.New("current_player_position is empty")
	}
	if req.BbSize <= 0 {
		return errors.New("bb_size must be positive")
	}
	hero, _ := req.Hero()
	if hero == nil {
		return fmt.Errorf("current player %q is missing", req.CurrentPlayerPosition)
	}
	if !hero.IsActive {
		return fmt.Errorf("current player %q is inactive", hero.Position)
	}
	if hero.Stack <= 0 {
		return fmt.Errorf("current player %q has invalid stack %d", hero.Position, hero.Stack)
	}
	if len(hero.HoleCards) != 2 {
		return fmt.Errorf("current player %q has %d hole cards, want 2", hero.Position, len(hero.HoleCards))
	}
	seen := make(map[int]bool, 2+len(req.CommunityCards))
	for _, card := range append(append([]string(nil), hero.HoleCards...), req.CommunityCards...) {
		c, ok := parseCard(card)
		if !ok {
			return fmt.Errorf("invalid card %q", card)
		}
		if seen[c] {
			return fmt.Errorf("duplicate card %q", card)
		}
		seen[c] = true
	}
	return nil
}

// finalize enforces the response contract independently of the engine:
// raise_size is a target total, check is impossible facing a bet, and bet is
// impossible when someone already bet.
func finalize(req *TexasGtoDecisionReq, resp *TexasGtoDecisionResp) *TexasGtoDecisionResp {
	if resp == nil {
		resp = &TexasGtoDecisionResp{}
	}
	hero, maxBet := req.Hero()
	if hero == nil {
		return resp
	}
	maxTarget := hero.Bet + hero.Stack
	facing := maxBet > hero.Bet
	if resp.ActionProbability < 0 {
		resp.ActionProbability = 0
	}
	if resp.ActionProbability > 1 {
		resp.ActionProbability = 1
	}
	switch resp.Action {
	case "fold":
		resp.RaiseSize = 0
		if !facing {
			resp.Action = "check"
		}
	case "check":
		resp.RaiseSize = 0
		if facing {
			resp.Action = "fold"
		}
	case "call":
		resp.RaiseSize = 0
		if !facing {
			resp.Action = "check"
		}
	case "bet":
		if facing {
			resp.Action, resp.RaiseSize = "call", 0
			break
		}
		if resp.RaiseSize <= 0 {
			resp.Action, resp.RaiseSize = "check", 0
			break
		}
		if resp.RaiseSize > maxTarget {
			resp.RaiseSize = maxTarget
		}
	case "raise":
		if resp.RaiseSize > maxTarget {
			resp.RaiseSize = maxTarget
		}
		if resp.RaiseSize <= maxBet {
			resp.Action, resp.RaiseSize = "call", 0
			if !facing {
				resp.Action = "check"
			}
		}
	default:
		if facing {
			resp.Action, resp.RaiseSize = "fold", 0
		} else {
			resp.Action, resp.RaiseSize = "check", 0
		}
	}
	if resp.Detail == "" {
		resp.Detail = "engine=unknown"
	}
	return resp
}
