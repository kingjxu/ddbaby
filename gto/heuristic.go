package gto

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
)

// Heuristic is the L4 fallback. It is intentionally simple and transparent:
// range-blind Monte Carlo equity against random hands, pot odds, and a small
// position/table-size adjustment. It is not GTO and must not be presented as
// such; its job is to always return a legal, reasonable action.
type Heuristic struct {
	Samples int
}

func (h *Heuristic) decide(req *TexasGtoDecisionReq) (*TexasGtoDecisionResp, bool) {
	hero, maxBet := req.Hero()
	if hero == nil || len(hero.HoleCards) != 2 {
		return nil, false
	}
	h0, ok0 := parseCard(hero.HoleCards[0])
	h1, ok1 := parseCard(hero.HoleCards[1])
	if !ok0 || !ok1 || h0 == h1 {
		return nil, false
	}
	heroCards := [2]int{h0, h1}

	board := make([]int, 0, 5)
	seen := map[int]bool{h0: true, h1: true}
	for _, card := range req.CommunityCards {
		c, ok := parseCard(card)
		if !ok || seen[c] {
			return nil, false
		}
		seen[c] = true
		board = append(board, c)
	}

	opponents := req.ActivePlayers() - 1
	if opponents < 1 {
		opponents = 1
	}
	samples := h.Samples
	if samples <= 0 {
		samples = 2000
	}
	switch {
	case opponents >= 6:
		samples /= 3
	case opponents >= 4:
		samples /= 2
	}
	if samples < 300 {
		samples = 300
	}

	equity := monteCarloEquity(heroCards, board, opponents, samples, stateSeed(heroCards, board, opponents))
	pot := req.CurrentPot
	if pot <= 0 {
		for _, p := range req.Players {
			pot += p.Bet
		}
	}
	toCall := maxBet - hero.Bet
	maxTarget := hero.Bet + hero.Stack

	tightness := positionTightness(hero.Position, req.GameStage)
	tightness += 0.02 * float64(opponents-1)
	bb := req.BbSize
	if bb <= 0 {
		bb = 1
	}

	action := "check"
	size := 0
	probability := math.Round(equity*1000) / 1000

	if toCall <= 0 {
		if equity >= 0.60+tightness*0.5 {
			action = "bet"
			size = int(math.Round(float64(pot) * 0.5))
			if size < bb {
				size = bb
			}
			if size > maxTarget {
				size = maxTarget
			}
			if size <= 0 {
				action, size = "check", 0
			}
		}
	} else {
		required := float64(toCall) / float64(pot+toCall)
		switch {
		case equity < required-0.03:
			action, size = "fold", 0
		case equity >= required+0.20+tightness && equity >= 0.52:
			action = "raise"
			size = int(math.Round(float64(maxBet) * 2.5))
			if size <= maxBet {
				size = maxBet + bb
			}
			if size > maxTarget {
				size = maxTarget
			}
			if size <= maxBet {
				action, size = "call", 0
			}
		default:
			action, size = "call", 0
		}
	}

	return &TexasGtoDecisionResp{
		Action:            action,
		RaiseSize:         size,
		ActionProbability: probability,
		Detail: fmt.Sprintf(
			"engine=heuristic equity=%.3f required=%.3f opponents=%d samples=%d approx=true",
			equity, requiredEquity(toCall, pot), opponents, samples,
		),
	}, true
}

func requiredEquity(toCall, pot int) float64 {
	if toCall <= 0 {
		return 0
	}
	return float64(toCall) / float64(pot+toCall)
}

func positionTightness(position, stage string) float64 {
	if strings.ToLower(strings.TrimSpace(stage)) != "preflop" {
		return 0
	}
	pos := strings.ToUpper(strings.TrimSuffix(strings.TrimSpace(position), "-HERO"))
	switch pos {
	case "UTG", "UTG+1":
		return 0.06
	case "MP", "MP+1":
		return 0.03
	case "CO", "BTN":
		return -0.02
	default:
		return 0
	}
}

func stateSeed(hero [2]int, board []int, opponents int) int64 {
	seed := int64(0x9e3779b9)
	mix := func(x int) {
		seed ^= int64(x) + 0x9e3779b9 + (seed << 6) + (seed >> 2)
	}
	mix(hero[0])
	mix(hero[1])
	for _, c := range board {
		mix(c)
	}
	mix(opponents)
	return seed
}

// monteCarloEquity samples random opponent hands and remaining board cards.
// Ties split the pot between all tied players.
func monteCarloEquity(hero [2]int, board []int, opponents, samples int, seed int64) float64 {
	var used [52]bool
	used[hero[0]] = true
	used[hero[1]] = true
	for _, c := range board {
		used[c] = true
	}
	deck := make([]int, 0, 52)
	for c := 0; c < 52; c++ {
		if !used[c] {
			deck = append(deck, c)
		}
	}
	needBoard := 5 - len(board)
	need := needBoard + 2*opponents
	if need <= 0 || need > len(deck) {
		return 0.5
	}

	rng := rand.New(rand.NewSource(seed))
	var full [5]int
	var hero7 [7]int
	var opp7 [7]int
	wins, ties := 0.0, 0.0
	for s := 0; s < samples; s++ {
		for i := 0; i < need; i++ {
			j := i + rng.Intn(len(deck)-i)
			deck[i], deck[j] = deck[j], deck[i]
		}
		copy(full[:], board)
		for i := 0; i < needBoard; i++ {
			full[len(board)+i] = deck[i]
		}
		hero7[0], hero7[1] = hero[0], hero[1]
		copy(hero7[2:], full[:])
		heroScore := eval7(hero7[:])

		idx := needBoard
		beaten := false
		tied := 0
		for o := 0; o < opponents; o++ {
			opp7[0], opp7[1] = deck[idx], deck[idx+1]
			idx += 2
			copy(opp7[2:], full[:])
			score := eval7(opp7[:])
			if score > heroScore {
				beaten = true
				break
			}
			if score == heroScore {
				tied++
			}
		}
		if beaten {
			continue
		}
		if tied == 0 {
			wins++
		} else {
			ties += 1 / float64(tied+1)
		}
	}
	return (wins + ties) / float64(samples)
}
