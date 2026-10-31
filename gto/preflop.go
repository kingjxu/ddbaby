package gto

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ActionFreq is a mixed strategy for one 169-grid hand at one node.
// RaiseTo/BetTo are target totals in big blinds.
type ActionFreq struct {
	Fold    float64 `json:"fold"`
	Call    float64 `json:"call"`
	Check   float64 `json:"check"`
	Raise   float64 `json:"raise"`
	Bet     float64 `json:"bet"`
	RaiseTo float64 `json:"raise_to"`
	BetTo   float64 `json:"bet_to"`
}

// raiseWeight lets hand-written tables omit the explicit raise frequency and
// infer it from the remaining probability mass.
func (f ActionFreq) raiseWeight() float64 {
	if f.Raise > 0 {
		return f.Raise
	}
	if f.RaiseTo > 0 {
		return math.Max(0, 1-f.Fold-f.Call-f.Check-f.Bet)
	}
	return 0
}

func (f ActionFreq) betWeight() float64 {
	if f.Bet > 0 {
		return f.Bet
	}
	if f.BetTo > 0 {
		return math.Max(0, 1-f.Fold-f.Call-f.Check-f.Raise)
	}
	return 0
}

// PreflopTable is one machine-readable preflop strategy file. It is the L1
// input: no images, no manual lookup at runtime.
type PreflopTable struct {
	Format          string                           `json:"format"`
	Players         int                              `json:"players"`
	StackBB         float64                          `json:"stack_bb"`
	AnteBB          float64                          `json:"ante_bb"`
	StraddleBB      float64                          `json:"straddle_bb"`
	Positions       []string                         `json:"positions"`
	PositionAliases map[string]string                `json:"position_aliases"`
	Nodes           map[string]map[string]ActionFreq `json:"nodes"`
	Source          string                           `json:"source"`

	file string
}

const (
	tableFormat   = "preflop-table/v1"
	maxStackDelta = 40.0
	maxAnteDelta  = 3.0
	maxStradDelta = 5.0
)

var defaultPositionAliases = map[string]string{
	"LJ":     "MP",
	"HJ":     "MP+1",
	"UTG+2":  "MP+1",
	"SMALL":  "SB",
	"BIG":    "BB",
	"BUTTON": "BTN",
}

// LoadPreflopTables reads every *.json file under dir. A missing directory is
// not fatal: the service then serves the heuristic fallback only.
func LoadPreflopTables(dir string) ([]*PreflopTable, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var tables []*PreflopTable
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		var table PreflopTable
		if err := json.Unmarshal(data, &table); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		if err := table.validate(); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		table.file = entry.Name()
		tables = append(tables, &table)
	}
	sort.Slice(tables, func(i, j int) bool {
		if tables[i].Players != tables[j].Players {
			return tables[i].Players < tables[j].Players
		}
		if tables[i].StackBB != tables[j].StackBB {
			return tables[i].StackBB < tables[j].StackBB
		}
		if tables[i].AnteBB != tables[j].AnteBB {
			return tables[i].AnteBB < tables[j].AnteBB
		}
		return tables[i].StraddleBB < tables[j].StraddleBB
	})
	return tables, nil
}

func (t *PreflopTable) validate() error {
	if t.Players < 2 || t.Players > 9 {
		return fmt.Errorf("players must be 2..9, got %d", t.Players)
	}
	if t.StackBB <= 0 {
		return fmt.Errorf("stack_bb must be positive")
	}
	if len(t.Nodes) == 0 {
		return fmt.Errorf("nodes is empty")
	}
	for key, node := range t.Nodes {
		if !strings.Contains(key, "|") {
			return fmt.Errorf("node key %q must be \"HERO|ACTION_LINE\"", key)
		}
		if len(node) == 0 {
			return fmt.Errorf("node %q has no hands", key)
		}
	}
	return nil
}

func (t *PreflopTable) alias(position string) string {
	pos := strings.ToUpper(strings.TrimSpace(position))
	pos = strings.TrimSuffix(pos, "-HERO")
	if mapped, ok := t.PositionAliases[pos]; ok {
		return strings.ToUpper(mapped)
	}
	if mapped, ok := defaultPositionAliases[pos]; ok {
		return mapped
	}
	return pos
}

// nearestPreflopTable picks the closest table for the same player count.
// A large mismatch is rejected so the heuristic fallback takes over.
func nearestPreflopTable(tables []*PreflopTable, players int, stack, ante, straddle float64) *PreflopTable {
	var best *PreflopTable
	bestScore := math.Inf(1)
	for _, t := range tables {
		if t.Players != players {
			continue
		}
		ds := math.Abs(t.StackBB - stack)
		da := math.Abs(t.AnteBB - ante)
		dst := math.Abs(t.StraddleBB - straddle)
		if ds > maxStackDelta || da > maxAnteDelta || dst > maxStradDelta {
			continue
		}
		score := ds + 2*da + 2*dst
		if score < bestScore {
			best, bestScore = t, score
		}
	}
	return best
}

func roundTo(value, step float64) float64 {
	if step <= 0 {
		return value
	}
	return math.Round(value/step) * step
}

func formatAmount(value float64) string {
	return strconv.FormatFloat(math.Round(value*100)/100, 'f', -1, 64)
}

func normalizeAction(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "fold":
		return "fold"
	case "check":
		return "check"
	case "call":
		return "call"
	case "bet":
		return "bet"
	case "raise":
		return "raise"
	case "allin", "all-in", "all_in", "all in":
		return "raise"
	default:
		return ""
	}
}

func normalizeStage(stage string) string {
	return strings.ToLower(strings.TrimSpace(stage))
}

type lineAction struct {
	position string
	action   string
	amountBB float64
}

func preflopActions(t *PreflopTable, req *TexasGtoDecisionReq, bb float64) []lineAction {
	actions := make([]lineAction, 0, len(req.ActionHistory))
	for _, a := range req.ActionHistory {
		if normalizeStage(a.Stage) != "preflop" {
			continue
		}
		action := normalizeAction(a.Action)
		if action == "" {
			continue
		}
		actions = append(actions, lineAction{
			position: t.alias(a.Position),
			action:   action,
			amountBB: float64(a.Amount) / bb,
		})
	}
	return actions
}

func canonicalLine(actions []lineAction, step float64) string {
	parts := make([]string, 0, len(actions))
	for _, a := range actions {
		token := a.position + ":" + a.action
		if a.action == "raise" || a.action == "bet" || a.action == "call" {
			token += ":" + formatAmount(roundTo(a.amountBB, step))
		}
		parts = append(parts, token)
	}
	return strings.Join(parts, "|")
}

func canonicalNodeKey(hero, line string) string {
	return hero + "|" + line
}

func situationKey(hero string, actions []lineAction) string {
	raises := 0
	calls := 0
	firstRaiser := ""
	for _, a := range actions {
		switch a.action {
		case "raise", "bet":
			raises++
			if raises == 1 {
				firstRaiser = a.position
			}
		case "call":
			calls++
		}
	}
	switch {
	case raises == 0 && calls == 0:
		return hero + "|RFI"
	case raises == 0:
		return hero + "|VS_LIMP"
	case raises == 1 && calls == 0:
		return hero + "|VS_OPEN_" + firstRaiser
	case raises == 1:
		return hero + "|VS_OPEN_CALL_" + firstRaiser
	case raises == 2:
		return hero + "|VS_3BET_" + firstRaiser
	default:
		return hero + "|VS_4BET_" + firstRaiser
	}
}

// lookupPreflop returns a legal response when a table covers this state.
func lookupPreflop(tables []*PreflopTable, req *TexasGtoDecisionReq) (*TexasGtoDecisionResp, bool) {
	hero, maxBet := req.Hero()
	if hero == nil || len(hero.HoleCards) != 2 {
		return nil, false
	}
	bb := float64(req.BbSize)
	if bb <= 0 {
		bb = 1
	}
	players := req.NumPlayers
	if players <= 0 {
		players = len(req.Players)
	}
	table := nearestPreflopTable(tables,
		players,
		float64(hero.Stack+hero.Bet)/bb,
		float64(req.Ante)/bb,
		float64(req.Straddle)/bb,
	)
	if table == nil {
		return nil, false
	}
	heroPos := table.alias(hero.Position)
	actions := preflopActions(table, req, bb)
	line := canonicalLine(actions, 0.1)
	hand := handClass(hero.HoleCards[0], hero.HoleCards[1])

	keys := []string{
		canonicalNodeKey(heroPos, line),
		canonicalNodeKey(heroPos, canonicalLine(actions, 0.5)),
		situationKey(heroPos, actions),
	}
	var (
		freq ActionFreq
		key  string
		ok   bool
	)
	for _, k := range keys {
		node, exists := table.Nodes[k]
		if !exists {
			continue
		}
		if f, found := node[hand]; found {
			freq, key, ok = f, k, true
			break
		}
		if f, found := node["*"]; found {
			freq, key, ok = f, k, true
			break
		}
	}
	if !ok {
		return nil, false
	}

	action, size, probability := chooseAction(freq, req.FacingBet(), hero, maxBet, bb)
	if action == "" {
		return nil, false
	}
	approx := key != canonicalNodeKey(heroPos, line)
	return &TexasGtoDecisionResp{
		Action:            action,
		RaiseSize:         size,
		ActionProbability: probability,
		Detail: fmt.Sprintf(
			"engine=preflop-table table=%s key=%s hand=%s freqs(fold=%.2f,call=%.2f,check=%.2f,raise=%.2f) raise_to=%.2fbb approx=%t",
			table.file, key, hand, freq.Fold, freq.Call, freq.Check, freq.raiseWeight(), freq.RaiseTo, approx,
		),
	}, true
}

// chooseAction converts a mixed strategy into a single legal action. It is
// deterministic: the most frequent legal action wins, with ties preferring the
// non-folding option.
func chooseAction(f ActionFreq, facingBet bool, hero *TexasPlayer, maxBet int, bb float64) (string, int, float64) {
	if hero == nil {
		return "", 0, 0
	}
	maxTarget := hero.Bet + hero.Stack
	type candidate struct {
		action string
		weight float64
		size   int
	}
	candidates := make([]candidate, 0, 3)
	add := func(action string, weight float64, size int) {
		if weight <= 0 {
			return
		}
		if size < 0 {
			size = 0
		}
		candidates = append(candidates, candidate{action, weight, size})
	}
	if facingBet {
		add("call", f.Call, 0)
		if f.RaiseTo > 0 {
			target := int(math.Round(f.RaiseTo * bb))
			if target > maxTarget {
				target = maxTarget
			}
			if target > maxBet {
				add("raise", f.raiseWeight(), target)
			} else {
				add("call", f.raiseWeight(), 0)
			}
		}
		add("fold", f.Fold, 0)
	} else {
		add("check", f.Check, 0)
		if f.BetTo > 0 && maxBet == 0 {
			target := int(math.Round(f.BetTo * bb))
			if target > maxTarget {
				target = maxTarget
			}
			if target > 0 {
				add("bet", f.betWeight(), target)
			}
		}
		if f.RaiseTo > 0 && maxBet > 0 {
			target := int(math.Round(f.RaiseTo * bb))
			if target > maxTarget {
				target = maxTarget
			}
			if target > maxBet {
				add("raise", f.raiseWeight(), target)
			}
		}
	}
	if len(candidates) == 0 {
		return "", 0, 0
	}
	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.weight > best.weight {
			best = c
			continue
		}
		if c.weight == best.weight {
			// Prefer check/call over fold, then raise/bet over call/check.
			if best.action == "fold" && c.action != "fold" {
				best = c
			} else if (best.action == "check" && c.action == "bet") ||
				(best.action == "call" && c.action == "raise") {
				best = c
			}
		}
	}
	total := 0.0
	for _, c := range candidates {
		total += c.weight
	}
	probability := 1.0
	if total > 0 {
		probability = best.weight / total
	}
	return best.action, best.size, math.Round(probability*1000) / 1000
}
