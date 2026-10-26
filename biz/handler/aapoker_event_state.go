package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kingjxu/ddbaby/model"
)

const (
	maxAAPokerSessions = 512
	aapokerSessionTTL  = 30 * time.Minute
)

type aapokerDecisionCandidate struct {
	SessionKey string
	TurnKey    string
	Request    *model.TexasGtoDecisionReq
}

type aapokerEventRegistry struct {
	mu       sync.Mutex
	sessions map[string]*aapokerTableState
}

type aapokerTableState struct {
	lastSeen time.Time
	userID   int64
	heroSeat int

	handNum        int
	buttonSeat     int
	smallBlind     int
	bigBlind       int
	ante           int
	straddle       int
	currentPot     int
	mushroom       int
	operationSeat  int
	operationRound int
	lastCardsRound int
	stage          string
	tableSize      int
	complete       bool
	board          []string
	participants   []int
	players        map[int]*aapokerPlayerState
	history        []model.TexasActionHistory

	inFlightTurn string
	decidedTurn  string
}

type aapokerPlayerState struct {
	seat        int
	position    string
	holeCards   []string
	stack       int
	bet         int
	active      bool
	actionTaken string
}

type aapokerEnterPayload struct {
	MySeatID       int     `json:"mySeatID"`
	PlayerIDArray  []int64 `json:"playerIDArray"`
	FirstCards     []int   `json:"firstCardarray"`
	SecondCards    []int   `json:"SecondArray"`
	Cards          []int   `json:"cards"`
	OperationID    int     `json:"operationID"`
	OperationRound int     `json:"operationRound"`
}

type aapokerStartPayload struct {
	BankerID       int   `json:"bankerID"`
	OperationID    int   `json:"operationID"`
	FirstCards     []int `json:"firstCardArray"`
	SecondCards    []int `json:"SecondCardArray"`
	SmallChip      int   `json:"smallChip"`
	BigChip        int   `json:"bigChip"`
	Chips          []int `json:"chipsArray"`
	CanPlay        []int `json:"canPlayStatus"`
	SeatStatus     []int `json:"seatStatus"`
	HandNum        int   `json:"handNum"`
	Straddle       []int `json:"straddlePop"`
	OperationRound int   `json:"operationRound"`
	MushroomPool   int   `json:"mushroomPool"`
	PoolChips      int   `json:"pool_chips"`
	Antes          []int `json:"anteCountArray"`
}

type aapokerActionPayload struct {
	SeatID         int `json:"seatID"`
	Action         int `json:"action"`
	OperationID    int `json:"operationID"`
	AnteNumber     int `json:"anteNumber"`
	OperationRound int `json:"operationRound"`
	FirstCard      int `json:"firstCard"`
	SecondCard     int `json:"secondCard"`
}

type aapokerCardsPayload struct {
	SeatID         int   `json:"seatID"`
	Cards          []int `json:"systemIDArray"`
	OperationRound int   `json:"operationRound"`
}

type aapokerPotsPayload struct {
	Pots []int `json:"pots"`
}

type aapokerSeatDownPayload struct {
	SeatID int   `json:"seatID"`
	UserID int64 `json:"userID"`
}

func newAAPokerEventRegistry() *aapokerEventRegistry {
	return &aapokerEventRegistry{sessions: make(map[string]*aapokerTableState)}
}

func (r *aapokerEventRegistry) process(event *texasPokerEvent, now time.Time) (*aapokerDecisionCandidate, error) {
	if event == nil || event.UUID == "" || event.Direction != "ServerToClient" || event.Location != "TABLE" {
		return nil, nil
	}
	uid, ok := rawPositiveInt64(event.UID)
	if !ok {
		return nil, nil
	}
	tableID, ok := rawPositiveInt64(event.DataExtra)
	if !ok {
		return nil, nil
	}
	payload, err := decodeAAPokerPayload(event.Data)
	if err != nil {
		return nil, err
	}

	sessionKey := fmt.Sprintf("%d:%s:%d:%d", len(event.UUID), event.UUID, uid, tableID)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.removeExpired(now)
	state := r.sessions[sessionKey]
	if state == nil {
		if len(r.sessions) >= maxAAPokerSessions {
			r.removeOldest()
		}
		state = &aapokerTableState{userID: uid, heroSeat: -1, operationSeat: -1}
		r.sessions[sessionKey] = state
	}
	state.lastSeen = now
	if err := state.apply(event.Cmd, payload); err != nil {
		return nil, err
	}
	request, turnKey := state.decisionRequest()
	if request == nil || turnKey == state.decidedTurn || turnKey == state.inFlightTurn {
		return nil, nil
	}
	state.inFlightTurn = turnKey
	return &aapokerDecisionCandidate{SessionKey: sessionKey, TurnKey: turnKey, Request: request}, nil
}

func (r *aapokerEventRegistry) complete(candidate *aapokerDecisionCandidate, success bool) {
	if candidate == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	state := r.sessions[candidate.SessionKey]
	if state == nil || state.inFlightTurn != candidate.TurnKey {
		return
	}
	state.inFlightTurn = ""
	if success {
		state.decidedTurn = candidate.TurnKey
	}
}

func (r *aapokerEventRegistry) removeExpired(now time.Time) {
	for key, state := range r.sessions {
		if now.Sub(state.lastSeen) > aapokerSessionTTL {
			delete(r.sessions, key)
		}
	}
}

func (r *aapokerEventRegistry) removeOldest() {
	var oldestKey string
	var oldest time.Time
	for key, state := range r.sessions {
		if oldestKey == "" || state.lastSeen.Before(oldest) {
			oldestKey, oldest = key, state.lastSeen
		}
	}
	delete(r.sessions, oldestKey)
}

func (s *aapokerTableState) apply(command string, payload []byte) error {
	switch command {
	case "REQ_GAME_ENTER_ROOM":
		var data aapokerEnterPayload
		if err := json.Unmarshal(payload, &data); err != nil {
			return err
		}
		s.applyEnter(data)
	case "REQ_GAME_RECV_START_INFOR":
		var data aapokerStartPayload
		if err := json.Unmarshal(payload, &data); err != nil {
			return err
		}
		return s.applyStart(data)
	case "REQ_GAME_RECV_SEAT_DOWN":
		var data aapokerSeatDownPayload
		if err := json.Unmarshal(payload, &data); err != nil {
			return err
		}
		if data.UserID == s.userID && data.SeatID >= 0 {
			s.heroSeat = data.SeatID
		}
	case "REQ_GAME_RECV_ACTION":
		var data aapokerActionPayload
		if err := json.Unmarshal(payload, &data); err != nil {
			return err
		}
		s.applyAction(data)
	case "REQ_GAME_RECV_CARDS":
		var data aapokerCardsPayload
		if err := json.Unmarshal(payload, &data); err != nil {
			return err
		}
		return s.applyCards(data)
	case "REQ_SHOW_SIDE_POTS":
		var data aapokerPotsPayload
		if err := json.Unmarshal(payload, &data); err != nil {
			return err
		}
		total := 0
		for _, pot := range data.Pots {
			if pot > 0 {
				total += pot
			}
		}
		if total > 0 {
			s.currentPot = total
		}
	case "REQ_CURRENT_ROUND_FINISH", "REQ_GAME_RECV_WINNER":
		s.operationSeat = -1
		s.inFlightTurn = ""
	}
	return nil
}

func (s *aapokerTableState) applyEnter(data aapokerEnterPayload) {
	matchedSeat := -1
	for seat, playerID := range data.PlayerIDArray {
		if playerID == s.userID {
			matchedSeat = seat
			break
		}
	}
	if matchedSeat >= 0 && (data.MySeatID < 0 || data.MySeatID == matchedSeat) {
		s.heroSeat = matchedSeat
	} else {
		s.heroSeat = -1
	}
	if s.heroSeat >= 0 && s.players != nil {
		s.setHeroCards(data.FirstCards, data.SecondCards)
	}
	if data.OperationRound >= s.operationRound {
		s.operationSeat = data.OperationID
		s.operationRound = data.OperationRound
	}
}

func (s *aapokerTableState) applyStart(data aapokerStartPayload) error {
	if data.HandNum <= 0 || data.BigChip <= 0 || data.SmallChip <= 0 || len(data.Chips) == 0 {
		return errors.New("incomplete start event")
	}
	if s.handNum == data.HandNum && s.players != nil {
		s.setHeroCards(data.FirstCards, data.SecondCards)
		return nil
	}
	if data.BankerID < 0 || data.BankerID >= len(data.Chips) {
		return errors.New("invalid button seat")
	}
	s.handNum = data.HandNum
	s.buttonSeat = data.BankerID
	s.smallBlind = data.SmallChip
	s.bigBlind = data.BigChip
	// AAPoker's anteCountArray contains ante plus any blind/straddle. In the
	// observed cash-game protocol the per-player ante equals bigChip, matching
	// the existing screenshot decision contract.
	s.ante = data.BigChip
	s.straddle = 0
	for _, amount := range data.Straddle {
		if amount > s.straddle {
			s.straddle = amount
		}
	}
	s.currentPot = data.PoolChips
	s.mushroom = data.MushroomPool / data.BigChip
	s.operationSeat = data.OperationID
	s.operationRound = data.OperationRound
	s.lastCardsRound = -1
	s.stage = "preflop"
	s.complete = true
	s.board = nil
	s.history = nil
	s.inFlightTurn = ""
	s.decidedTurn = ""
	s.players = make(map[int]*aapokerPlayerState)
	s.tableSize = len(data.Chips)
	s.participants = s.participants[:0]
	for seat, stack := range data.Chips {
		if stack < 0 || seat >= len(data.CanPlay) || data.CanPlay[seat] != 1 {
			continue
		}
		bet := 0
		if seat < len(data.Antes) {
			bet = data.Antes[seat] - s.ante
			if bet < 0 {
				bet = 0
			}
		}
		s.participants = append(s.participants, seat)
		s.players[seat] = &aapokerPlayerState{seat: seat, stack: stack, bet: bet, active: true}
	}
	if s.currentPot <= 0 {
		for seat, amount := range data.Antes {
			if seat < len(data.CanPlay) && data.CanPlay[seat] == 1 && amount > 0 {
				s.currentPot += amount
			}
		}
	}
	s.assignPositions()
	s.setHeroCards(data.FirstCards, data.SecondCards)
	return nil
}

func (s *aapokerTableState) applyAction(data aapokerActionPayload) {
	if s.players == nil || data.OperationRound <= s.operationRound || data.AnteNumber < 0 {
		return
	}
	if data.OperationRound != s.operationRound+1 {
		s.complete = false
	}
	player := s.players[data.SeatID]
	if player == nil {
		return
	}
	previousMax := s.maxBet()
	if data.AnteNumber > player.stack {
		data.AnteNumber = player.stack
	}
	player.stack -= data.AnteNumber
	player.bet += data.AnteNumber
	s.currentPot += data.AnteNumber
	action := aapokerActionName(data.Action, player.bet, previousMax)
	if action == "fold" {
		player.active = false
	}
	if data.Action == 4 {
		player.stack = 0
	}
	player.actionTaken = action
	if action != "" {
		s.history = append(s.history, model.TexasActionHistory{
			Stage: s.stage, Position: player.position, Action: action,
			Amount: actionAmount(action, player.bet), Timestamp: data.OperationRound,
		})
	}
	if data.SeatID == s.heroSeat && data.FirstCard >= 0 && data.SecondCard >= 0 {
		player.holeCards = []string{aapokerCard(data.FirstCard), aapokerCard(data.SecondCard)}
	}
	s.operationSeat = data.OperationID
	s.operationRound = data.OperationRound
}

func (s *aapokerTableState) applyCards(data aapokerCardsPayload) error {
	if s.players == nil || data.OperationRound < s.operationRound {
		return nil
	}
	if data.OperationRound == s.lastCardsRound {
		return nil
	}
	if data.OperationRound > s.operationRound {
		// A street transition carries the round of the last completed action.
		// A larger value means one or more action events never reached us, so
		// the action route is unsafe for the remainder of this hand.
		s.complete = false
	}
	for _, cardID := range data.Cards {
		card := aapokerCard(cardID)
		if card == "" {
			return errors.New("invalid community card")
		}
		s.board = append(s.board, card)
	}
	switch len(s.board) {
	case 3:
		s.stage = "flop"
	case 4:
		s.stage = "turn"
	case 5:
		s.stage = "river"
	default:
		return errors.New("invalid community card count")
	}
	for _, player := range s.players {
		player.bet = 0
		player.actionTaken = ""
	}
	s.operationSeat = data.SeatID
	s.operationRound = data.OperationRound
	s.lastCardsRound = data.OperationRound
	return nil
}

func (s *aapokerTableState) setHeroCards(first, second []int) {
	if s.heroSeat < 0 || s.players == nil || s.heroSeat >= len(first) || s.heroSeat >= len(second) {
		return
	}
	firstCard, secondCard := aapokerCard(first[s.heroSeat]), aapokerCard(second[s.heroSeat])
	player := s.players[s.heroSeat]
	if player != nil && firstCard != "" && secondCard != "" {
		player.holeCards = []string{firstCard, secondCard}
	}
}

func (s *aapokerTableState) assignPositions() {
	positions := aapokerPositions(len(s.participants))
	ordered := make([]int, 0, len(s.participants))
	for offset := 0; offset < s.tableSize; offset++ {
		seat := (s.buttonSeat + offset) % s.tableSize
		if _, ok := s.players[seat]; ok {
			ordered = append(ordered, seat)
		}
	}
	for index, seat := range ordered {
		position := positions[index]
		if seat == s.heroSeat {
			position += "-Hero"
		}
		s.players[seat].position = position
	}
}

func (s *aapokerTableState) decisionRequest() (*model.TexasGtoDecisionReq, string) {
	if !s.complete || s.handNum <= 0 || s.operationSeat < 0 || s.operationSeat != s.heroSeat || s.players == nil {
		return nil, ""
	}
	hero := s.players[s.heroSeat]
	if hero == nil || !hero.active || hero.stack <= 0 || len(hero.holeCards) != 2 {
		return nil, ""
	}
	players := make([]model.TexasPlayer, 0, len(s.participants))
	for _, seat := range s.orderedParticipantSeats() {
		player := s.players[seat]
		holeCards := []string{"X-X", "X-X"}
		if seat == s.heroSeat {
			holeCards = append([]string(nil), player.holeCards...)
		}
		players = append(players, model.TexasPlayer{
			Position: player.position, HoleCards: holeCards, Stack: player.stack,
			Bet: player.bet, IsActive: player.active, ActionTaken: player.actionTaken,
		})
	}
	request := &model.TexasGtoDecisionReq{
		GameType: "no_limit_holdem", NumPlayers: len(players), CurrentPot: s.currentPot,
		Ante: s.ante, Straddle: s.straddle, SbSize: s.smallBlind, BbSize: s.bigBlind,
		Mushroom:       s.mushroom,
		CommunityCards: append([]string(nil), s.board...), GameStage: s.stage,
		Players: players, CurrentPlayerPosition: hero.position,
		ActionHistory: append([]model.TexasActionHistory(nil), s.history...),
	}
	if err := model.ValidateTexasGtoDecisionRequest(request); err != nil {
		return nil, ""
	}
	turnKey := fmt.Sprintf("%d:%d:%s", s.handNum, s.operationRound, s.stage)
	return request, turnKey
}

func (s *aapokerTableState) orderedParticipantSeats() []int {
	result := make([]int, 0, len(s.participants))
	for offset := 0; offset < s.tableSize; offset++ {
		seat := (s.buttonSeat + offset) % s.tableSize
		if _, ok := s.players[seat]; ok {
			result = append(result, seat)
		}
	}
	return result
}

func (s *aapokerTableState) maxBet() int {
	max := 0
	for _, player := range s.players {
		if player.bet > max {
			max = player.bet
		}
	}
	return max
}

func decodeAAPokerPayload(raw json.RawMessage) ([]byte, error) {
	payload := bytes.TrimSpace(raw)
	if len(payload) == 0 {
		return nil, errors.New("empty event payload")
	}
	if payload[0] == '"' {
		var encoded string
		if err := json.Unmarshal(payload, &encoded); err != nil {
			return nil, err
		}
		payload = []byte(encoded)
	}
	if !json.Valid(payload) || len(payload) == 0 || payload[0] != '{' {
		return nil, errors.New("event payload must be an object")
	}
	return payload, nil
}

func rawPositiveInt64(raw json.RawMessage) (int64, bool) {
	if len(raw) == 0 {
		return 0, false
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return 0, false
	}
	var token string
	switch typed := value.(type) {
	case string:
		token = typed
	case json.Number:
		token = typed.String()
	default:
		return 0, false
	}
	parsed, err := strconv.ParseInt(token, 10, 64)
	return parsed, err == nil && parsed > 0
}

func aapokerCard(cardID int) string {
	if cardID < 0 || cardID >= 52 {
		return ""
	}
	ranks := [...]string{"2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K", "A"}
	suits := [...]string{"D", "C", "H", "S"}
	return ranks[cardID%13] + "-" + suits[cardID/13]
}

func aapokerActionName(action, target, previousMax int) string {
	switch action {
	case 2:
		return "call"
	case 3:
		if previousMax == 0 {
			return "bet"
		}
		return "raise"
	case 4:
		if target <= previousMax {
			return "call"
		}
		if previousMax == 0 {
			return "bet"
		}
		return "raise"
	case 5:
		return "check"
	case 6:
		return "fold"
	default:
		return ""
	}
}

func actionAmount(action string, target int) int {
	if action == "bet" || action == "raise" || action == "call" {
		return target
	}
	return 0
}

func aapokerPositions(playerCount int) []string {
	switch playerCount {
	case 9:
		return []string{"BTN", "SB", "BB", "straddle", "UTG", "UTG+1", "MP", "MP+1", "CO"}
	case 8:
		return []string{"BTN", "SB", "BB", "straddle", "UTG", "UTG+1", "MP", "CO"}
	case 7:
		return []string{"BTN", "SB", "BB", "straddle", "UTG", "MP", "CO"}
	case 6:
		return []string{"BTN", "SB", "BB", "straddle", "UTG", "CO"}
	case 5:
		return []string{"BTN", "SB", "BB", "straddle", "CO"}
	case 4:
		return []string{"BTN", "SB", "BB", "straddle"}
	case 3:
		return []string{"BTN", "SB", "BB"}
	case 2:
		return []string{"BTN", "BB"}
	case 1:
		return []string{"BTN"}
	default:
		return []string{"BTN"}
	}
}

func eventHeroMaxTarget(request *model.TexasGtoDecisionReq) int {
	if request == nil {
		return 0
	}
	for _, player := range request.Players {
		if strings.EqualFold(player.Position, request.CurrentPlayerPosition) {
			return player.Bet + player.Stack
		}
	}
	return 0
}
