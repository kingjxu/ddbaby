package service

import (
	"errors"
	"time"

	ddbaby "github.com/kingjxu/ddbaby/biz/model/ddbaby"
)

const (
	aapokerPackageName          = "com.plusaa.amula"
	aapokerAlternatePackageName = "com.ty.aapk"
	dpzxPackageName             = "com.dpzx.pro"
)

type pokerEventRoute uint8

const (
	pokerEventRouteAAPoker pokerEventRoute = iota + 1
	pokerEventRouteDPZX
)

// pokerEventPackageRoutes is the single allowlist and dispatch table. Add new
// packages here only after choosing an implemented state machine or log-only.
var pokerEventPackageRoutes = map[string]pokerEventRoute{
	aapokerPackageName:          pokerEventRouteAAPoker,
	aapokerAlternatePackageName: pokerEventRouteAAPoker,
	dpzxPackageName:             pokerEventRouteDPZX,
}

var texasPokerEventStates = newAAPokerEventRegistry()
var dpzxPokerEventStates = newDPZXEventRegistry()

type texasPokerEvent struct {
	ddbaby.TexasPokerEventReq
	route pokerEventRoute
}

// PokerEventDecisionResult is the complete client response cached for replay.
type PokerEventDecisionResult struct {
	Result         string   `json:"Result"`
	HeroCard       []string `json:"HeroCard"`
	CommunityCard  []string `json:"CommunityCard"`
	DecisionAction string   `json:"DecisionAction"`
	DecisionAmount int      `json:"DecisionAmount"`
	HeroMaxTarget  int      `json:"HeroMaxTarget"`
}

type pokerDecisionCacheEntry struct {
	createdAt time.Time
	result    *PokerEventDecisionResult
}

// IsSupportedPokerEventPackage reports whether the service handles this source.
func IsSupportedPokerEventPackage(packageName string) bool {
	_, ok := pokerEventPackageRoutes[packageName]
	return ok
}

// ProcessPokerEvent rebuilds a source-isolated hand and reserves a Hero turn.
func ProcessPokerEvent(req *ddbaby.TexasPokerEventReq) (*PokerEventDecision, error) {
	if req == nil {
		return nil, errors.New("event is missing")
	}
	route, ok := pokerEventPackageRoutes[req.PackageName]
	if !ok {
		return nil, errors.New("invalid packageName")
	}
	event := &texasPokerEvent{TexasPokerEventReq: *req, route: route}
	var candidate *PokerEventDecision
	var err error
	switch route {
	case pokerEventRouteAAPoker:
		candidate, err = texasPokerEventStates.process(event, time.Now())
	case pokerEventRouteDPZX:
		candidate, err = dpzxPokerEventStates.process(event, time.Now())
	}
	if candidate != nil {
		candidate.route = route
	}
	return candidate, err
}

// CompletePokerEventDecision caches success or releases the turn for retry.
func CompletePokerEventDecision(candidate *PokerEventDecision, result *PokerEventDecisionResult) {
	if candidate == nil {
		return
	}
	switch candidate.route {
	case pokerEventRouteAAPoker:
		texasPokerEventStates.complete(candidate, result, time.Now())
	case pokerEventRouteDPZX:
		dpzxPokerEventStates.complete(candidate, result, time.Now())
	}
}
