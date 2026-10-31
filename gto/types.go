// Package gto implements the decision service behind the existing
// POST /gto/decision and POST /gto/decision_v2 contracts.
//
// The request and response types below mirror model.TexasGtoDecisionReq and
// model.TexasGtoDecisionResp in the ddbaby repository. They are duplicated on
// purpose: this service has no dependency on ddbaby internals and can be
// built/deployed on its own.
package gto

// TexasPlayer is one seat in the request snapshot.
type TexasPlayer struct {
	Position    string   `json:"position"`
	HoleCards   []string `json:"hole_cards"`
	Stack       int      `json:"stack"`
	Bet         int      `json:"bet"`
	IsActive    bool     `json:"is_active"`
	ActionTaken string   `json:"action_taken"`
}

// TexasActionHistory is one observed action. Amount is the target total for
// bet/raise, the call amount for call, and zero for fold/check.
type TexasActionHistory struct {
	Stage     string `json:"stage"`
	Position  string `json:"position"`
	Action    string `json:"action"`
	Amount    int    `json:"amount"`
	Timestamp int    `json:"timestamp"`
}

// TexasGtoDecisionReq is the exact request body accepted by the existing
// upstream /gto/decision and /gto/decision_v2 endpoints.
type TexasGtoDecisionReq struct {
	GameType              string               `json:"game_type"`
	NumPlayers            int                  `json:"num_players"`
	CurrentPot            int                  `json:"current_pot"`
	Ante                  int                  `json:"ante"`
	Straddle              int                  `json:"straddle"`
	SbSize                int                  `json:"sb_size"`
	BbSize                int                  `json:"bb_size"`
	Mushroom              int                  `json:"mushroom"`
	CommunityCards        []string             `json:"community_cards"`
	GameStage             string               `json:"game_stage"`
	Players               []TexasPlayer        `json:"players"`
	CurrentPlayerPosition string               `json:"current_player_position"`
	RaiseRange            []int                `json:"raise_range"`
	ActionHistory         []TexasActionHistory `json:"action_history"`
	VVIP                  int32                `json:"vvip,omitempty"`
	PFR                   int32                `json:"pfr,omitempty"`
}

// TexasGtoDecisionResp is the exact response body returned to ddbaby.
// RaiseSize is a target total (not an increment) for bet/raise.
type TexasGtoDecisionResp struct {
	Action            string  `json:"action"`
	RaiseSize         int     `json:"raise_size"`
	ActionProbability float64 `json:"action_probability"`
	Detail            string  `json:"detail"`
}

// Hero returns the acting player and the current maximum committed bet.
func (r *TexasGtoDecisionReq) Hero() (*TexasPlayer, int) {
	if r == nil {
		return nil, 0
	}
	maxBet := 0
	var hero *TexasPlayer
	for i := range r.Players {
		p := &r.Players[i]
		if p.Bet > maxBet {
			maxBet = p.Bet
		}
		if p.Position == r.CurrentPlayerPosition {
			hero = p
		}
	}
	return hero, maxBet
}

// ActivePlayers counts players who have not folded.
func (r *TexasGtoDecisionReq) ActivePlayers() int {
	if r == nil {
		return 0
	}
	n := 0
	for _, p := range r.Players {
		if p.IsActive {
			n++
		}
	}
	return n
}

// FacingBet reports whether Hero has to match chips to continue.
func (r *TexasGtoDecisionReq) FacingBet() bool {
	hero, maxBet := r.Hero()
	return hero != nil && maxBet > hero.Bet
}
