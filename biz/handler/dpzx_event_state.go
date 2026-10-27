package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/kingjxu/ddbaby/model"
)

const (
	maxDPZXSessions        = 512
	dpzxSessionTTL         = 30 * time.Minute
	maxDPZXDecisionResults = 1024
	dpzxDecisionResultTTL  = 30 * time.Minute
	maxDPZXSeats           = 10
	maxDPZXActionsPerHand  = 512
	maxDPZXChipAmount      = 1_000_000_000
)

type dpzxEventRegistry struct {
	mu       sync.Mutex
	sessions map[string]*dpzxTableState
	results  map[string]aapokerDecisionCacheEntry
}

type dpzxTableState struct {
	lastSeen time.Time
	userID   int64

	roomID       int64
	handNum      int
	buttonSeat   int
	heroSeat     int
	tableSize    int
	smallBlind   int
	bigBlind     int
	ante         int
	currentPot   int
	stage        string
	board        []string
	participants []int
	players      map[int]*aapokerPlayerState
	history      []model.TexasActionHistory

	ready               bool
	complete            bool
	expectedSeat        int
	operationSeat       int
	actionSequence      int
	lastPromptTimestamp int64
	turnTimestamp       int64
	inFlightTurn        string
	decidedTurn         string
}

type dpzxRoomSnapshot struct {
	BankIndex int   `json:"bankIndex"`
	MySeat    int   `json:"mySeat"`
	HandNum   int   `json:"HandNum"`
	RoomID    int64 `json:"roomID"`
	LittleBB  int   `json:"littleBB"`
	Ante      int   `json:"ante"`
	TableSize int   `json:"stisNumMax"`
}

type dpzxPlayerSnapshot struct {
	Index      int `json:"index"`
	Value      int `json:"value"`
	Score      int `json:"score"`
	StartState int `json:"startState"`
	BetState   int `json:"betState"`
}

type dpzxActionUpdate struct {
	Index    int `json:"index"`
	BetState int `json:"betState"`
	Score    int `json:"score"`
	Betted   int `json:"betted"`
}

type dpzxCardsUpdate struct {
	Index    int `json:"index"`
	VecCards []struct {
		Value int `json:"value"`
	} `json:"vecCards"`
}

type dpzxPlayerJoin struct {
	Index  int   `json:"index"`
	SID    int64 `json:"sID"`
	IsSelf bool  `json:"isSelf"`
}

func newDPZXEventRegistry() *dpzxEventRegistry {
	return &dpzxEventRegistry{
		sessions: make(map[string]*dpzxTableState),
		results:  make(map[string]aapokerDecisionCacheEntry),
	}
}

func (r *dpzxEventRegistry) process(event *texasPokerEvent, now time.Time) (*aapokerDecisionCandidate, error) {
	if event == nil || event.UUID == "" || event.Direction != "ServerToClient" || event.Location != "TABLE" {
		return nil, nil
	}
	uid, ok := rawPositiveInt64(event.UID)
	if !ok {
		return nil, nil
	}
	sessionKey := fmt.Sprintf("%d:%s:%d", len(event.UUID), event.UUID, uid)

	r.mu.Lock()
	defer r.mu.Unlock()
	r.removeExpired(now)
	r.removeExpiredResults(now)
	state := r.sessions[sessionKey]
	if state == nil {
		if len(r.sessions) >= maxDPZXSessions {
			r.removeOldest()
		}
		state = &dpzxTableState{
			userID: uid, heroSeat: -1, expectedSeat: -1, operationSeat: -1,
		}
		r.sessions[sessionKey] = state
	}
	state.lastSeen = now
	if err := state.apply(event.Cmd, event.Data, event.Timestamp); err != nil {
		return nil, err
	}
	request, turnKey := state.decisionRequest()
	if request == nil {
		return nil, nil
	}
	cacheKey := decisionCacheKey(sessionKey, turnKey)
	if cached, exists := r.results[cacheKey]; exists {
		return &aapokerDecisionCandidate{
			SessionKey: sessionKey,
			TurnKey:    turnKey,
			Cached:     cloneAAPokerDecisionResult(cached.result),
		}, nil
	}
	if turnKey == state.decidedTurn || turnKey == state.inFlightTurn {
		return nil, nil
	}
	state.inFlightTurn = turnKey
	return &aapokerDecisionCandidate{SessionKey: sessionKey, TurnKey: turnKey, Request: request}, nil
}

func (r *dpzxEventRegistry) complete(
	candidate *aapokerDecisionCandidate,
	result *aapokerDecisionResult,
	now time.Time,
) {
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
	if result == nil {
		return
	}
	state.decidedTurn = candidate.TurnKey
	r.removeExpiredResults(now)
	if len(r.results) >= maxDPZXDecisionResults {
		r.removeOldestResult()
	}
	r.results[decisionCacheKey(candidate.SessionKey, candidate.TurnKey)] = aapokerDecisionCacheEntry{
		createdAt: now,
		result:    cloneAAPokerDecisionResult(result),
	}
}

func (r *dpzxEventRegistry) removeExpired(now time.Time) {
	for key, state := range r.sessions {
		if now.Sub(state.lastSeen) > dpzxSessionTTL {
			delete(r.sessions, key)
		}
	}
}

func (r *dpzxEventRegistry) removeOldest() {
	var oldestKey string
	var oldest time.Time
	for key, state := range r.sessions {
		if oldestKey == "" || state.lastSeen.Before(oldest) {
			oldestKey, oldest = key, state.lastSeen
		}
	}
	delete(r.sessions, oldestKey)
}

func (r *dpzxEventRegistry) removeExpiredResults(now time.Time) {
	for key, entry := range r.results {
		if now.Sub(entry.createdAt) > dpzxDecisionResultTTL {
			delete(r.results, key)
		}
	}
}

func (r *dpzxEventRegistry) removeOldestResult() {
	var oldestKey string
	var oldest time.Time
	for key, entry := range r.results {
		if oldestKey == "" || entry.createdAt.Before(oldest) {
			oldestKey, oldest = key, entry.createdAt
		}
	}
	delete(r.results, oldestKey)
}

func (s *dpzxTableState) apply(command string, raw json.RawMessage, timestamp int64) error {
	switch command {
	case "421":
		var payload []dpzxPlayerJoin
		if err := json.Unmarshal(raw, &payload); err != nil {
			return err
		}
		for _, player := range payload {
			if player.IsSelf && player.SID == s.userID && player.Index >= 0 {
				s.heroSeat = player.Index
			}
		}
	case "423":
		var payload []int
		if err := json.Unmarshal(raw, &payload); err != nil {
			return err
		}
		if len(payload) > 0 && payload[0] == s.heroSeat {
			s.heroSeat = -1
			s.ready = false
		}
	case "430":
		return s.applyRoomSnapshot(raw)
	case "431":
		return s.applyHandStart(raw)
	case "432":
		return s.applyPrompt(raw, timestamp)
	case "433":
		return s.applyAction(raw)
	case "403":
		var payload []int
		if err := json.Unmarshal(raw, &payload); err != nil {
			return err
		}
		if len(payload) > 0 && validDPZXChipAmount(payload[0]) && s.ready {
			s.currentPot = payload[0]
		} else if len(payload) > 0 && s.ready {
			s.complete = false
			return errors.New("invalid DPZX pot")
		}
	case "490":
		return s.applyBoard(raw)
	case "492":
		return s.applyCards(raw)
	case "435":
		s.ready = false
		s.operationSeat = -1
		s.expectedSeat = -1
		s.inFlightTurn = ""
	}
	return nil
}

func (s *dpzxTableState) applyRoomSnapshot(raw json.RawMessage) error {
	var payload []json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil || len(payload) < 2 {
		return errors.New("invalid DPZX room snapshot")
	}
	var room dpzxRoomSnapshot
	var players []dpzxPlayerSnapshot
	if err := json.Unmarshal(payload[0], &room); err != nil {
		return err
	}
	if err := json.Unmarshal(payload[1], &players); err != nil {
		return err
	}
	if room.RoomID <= 0 || room.HandNum <= 0 || !validDPZXChipAmount(room.LittleBB) ||
		!validDPZXChipAmount(room.Ante) || room.LittleBB == 0 || room.BankIndex < 0 ||
		room.LittleBB > maxDPZXChipAmount/2 || len(players) < 2 || len(players) > maxDPZXSeats {
		return errors.New("incomplete DPZX room snapshot")
	}
	tableSize := room.TableSize
	if tableSize <= room.BankIndex {
		tableSize = room.BankIndex + 1
	}
	for _, player := range players {
		if player.Index >= tableSize {
			tableSize = player.Index + 1
		}
	}
	if tableSize < 2 || tableSize > maxDPZXSeats {
		return errors.New("invalid DPZX table size")
	}

	s.roomID = room.RoomID
	s.handNum = room.HandNum
	s.buttonSeat = room.BankIndex
	s.heroSeat = room.MySeat
	s.tableSize = tableSize
	s.smallBlind = room.LittleBB
	s.bigBlind = room.LittleBB * 2
	s.ante = room.Ante
	s.currentPot = 0
	s.stage = "preflop"
	s.board = nil
	s.participants = nil
	s.players = nil
	s.history = nil
	s.ready = false
	s.complete = false
	s.expectedSeat = -1
	s.operationSeat = -1
	s.actionSequence = 0
	s.lastPromptTimestamp = 0
	s.turnTimestamp = 0
	s.inFlightTurn = ""
	s.decidedTurn = ""
	return nil
}

func (s *dpzxTableState) applyHandStart(raw json.RawMessage) error {
	if s.handNum <= 0 || s.tableSize < 2 {
		return nil
	}
	var payload [][]dpzxPlayerSnapshot
	if err := json.Unmarshal(raw, &payload); err != nil || len(payload) == 0 || len(payload[0]) < 2 {
		return errors.New("invalid DPZX hand start")
	}
	players := make(map[int]*aapokerPlayerState, len(payload[0]))
	participants := make([]int, 0, len(payload[0]))
	pot := s.ante * len(payload[0])
	for _, item := range payload[0] {
		if item.Index < 0 || item.Index >= s.tableSize ||
			!validDPZXChipAmount(item.Value) || !validDPZXChipAmount(item.Score) {
			return errors.New("invalid DPZX player snapshot")
		}
		if _, exists := players[item.Index]; exists {
			return errors.New("duplicate DPZX player seat")
		}
		players[item.Index] = &aapokerPlayerState{
			seat: item.Index, stack: item.Score, bet: item.Value, active: true,
		}
		participants = append(participants, item.Index)
		pot += item.Value
	}
	s.players = players
	s.participants = participants
	s.currentPot = pot
	s.ready = true
	s.complete = s.heroSeat >= 0 && s.heroSeat < s.tableSize && players[s.heroSeat] != nil
	s.expectedSeat = -1
	s.operationSeat = -1
	s.assignPositions()
	return nil
}

func (s *dpzxTableState) applyPrompt(raw json.RawMessage, timestamp int64) error {
	if !s.ready {
		return nil
	}
	if timestamp <= 0 {
		s.complete = false
		return errors.New("invalid DPZX action timestamp")
	}
	var payload []json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil || len(payload) < 1 {
		return errors.New("invalid DPZX action prompt")
	}
	var seat int
	if err := json.Unmarshal(payload[0], &seat); err != nil || s.players[seat] == nil {
		s.complete = false
		return errors.New("invalid DPZX action seat")
	}
	if timestamp > 0 && timestamp == s.lastPromptTimestamp && seat == s.operationSeat {
		return nil
	}
	if s.expectedSeat >= 0 {
		s.complete = false
	}
	s.expectedSeat = seat
	s.operationSeat = seat
	s.turnTimestamp = timestamp
	s.lastPromptTimestamp = timestamp
	if len(payload) >= 3 {
		var contribution int
		if err := json.Unmarshal(payload[2], &contribution); err != nil || !validDPZXChipAmount(contribution) {
			s.complete = false
			return errors.New("invalid DPZX action contribution")
		}
		s.players[seat].bet = contribution
	}
	return nil
}

func (s *dpzxTableState) applyAction(raw json.RawMessage) error {
	if !s.ready {
		return nil
	}
	var payload []dpzxActionUpdate
	if err := json.Unmarshal(raw, &payload); err != nil || len(payload) != 1 {
		return errors.New("invalid DPZX action update")
	}
	update := payload[0]
	player := s.players[update.Index]
	if player == nil || !validDPZXChipAmount(update.Score) || !validDPZXChipAmount(update.Betted) {
		s.complete = false
		return errors.New("invalid DPZX action player")
	}
	if s.expectedSeat != update.Index {
		s.complete = false
	}
	s.expectedSeat = -1
	s.operationSeat = -1
	previousMax := s.maxBet()
	previousBet := player.bet
	if update.Betted >= previousBet {
		player.bet = update.Betted
	}
	player.stack = update.Score
	if player.bet > previousBet {
		s.currentPot += player.bet - previousBet
	}
	action := dpzxActionName(update.BetState, player.bet, previousMax)
	if action == "" {
		s.complete = false
		return errors.New("unknown DPZX action state")
	}
	if action == "fold" {
		player.active = false
	}
	player.actionTaken = action
	if len(s.history) >= maxDPZXActionsPerHand {
		s.complete = false
		return errors.New("too many DPZX actions in one hand")
	}
	s.actionSequence++
	s.history = append(s.history, model.TexasActionHistory{
		Stage: s.stage, Position: player.position, Action: action,
		Amount: actionAmount(action, player.bet), Timestamp: s.actionSequence,
	})
	return nil
}

func (s *dpzxTableState) applyBoard(raw json.RawMessage) error {
	if !s.ready {
		return nil
	}
	var payload []json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil || len(payload) < 2 {
		return errors.New("invalid DPZX board update")
	}
	var cardIDs []int
	if err := json.Unmarshal(payload[1], &cardIDs); err != nil {
		return err
	}
	if len(cardIDs) == len(s.board) && sameDPZXBoard(s.board, cardIDs) {
		return nil
	}
	if s.expectedSeat >= 0 || !validDPZXBoardProgression(len(s.board), len(cardIDs)) {
		s.complete = false
		return errors.New("incomplete DPZX street transition")
	}
	board := make([]string, 0, len(cardIDs))
	seen := make(map[string]struct{}, len(cardIDs)+2)
	if hero := s.players[s.heroSeat]; hero != nil {
		for _, card := range hero.holeCards {
			seen[card] = struct{}{}
		}
	}
	for _, cardID := range cardIDs {
		card := dpzxCard(cardID)
		if card == "" {
			s.complete = false
			return errors.New("invalid DPZX community card")
		}
		if _, exists := seen[card]; exists {
			s.complete = false
			return errors.New("duplicate DPZX card")
		}
		seen[card] = struct{}{}
		board = append(board, card)
	}
	for index := range s.board {
		if board[index] != s.board[index] {
			s.complete = false
			return errors.New("changed DPZX community card prefix")
		}
	}
	s.board = board
	switch len(board) {
	case 3:
		s.stage = "flop"
	case 4:
		s.stage = "turn"
	case 5:
		s.stage = "river"
	}
	for _, player := range s.players {
		player.bet = 0
		player.actionTaken = ""
	}
	s.expectedSeat = -1
	s.operationSeat = -1
	return nil
}

func validDPZXBoardProgression(current, next int) bool {
	return current == 0 && next == 3 || current == 3 && next == 4 || current == 4 && next == 5
}

func sameDPZXBoard(current []string, cardIDs []int) bool {
	if len(current) != len(cardIDs) {
		return false
	}
	for index, cardID := range cardIDs {
		if current[index] != dpzxCard(cardID) {
			return false
		}
	}
	return true
}

func (s *dpzxTableState) applyCards(raw json.RawMessage) error {
	if !s.ready || s.heroSeat < 0 {
		return nil
	}
	var payload []json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil || len(payload) < 2 {
		s.complete = false
		return errors.New("invalid DPZX cards update")
	}
	var players []dpzxCardsUpdate
	if err := json.Unmarshal(payload[1], &players); err != nil {
		s.complete = false
		return err
	}
	heroUpdates := 0
	for _, item := range players {
		if item.Index != s.heroSeat {
			continue
		}
		heroUpdates++
		if heroUpdates > 1 || len(item.VecCards) != 2 {
			s.complete = false
			return errors.New("invalid DPZX Hero cards")
		}
		first, second := dpzxCard(item.VecCards[0].Value), dpzxCard(item.VecCards[1].Value)
		if first == "" || second == "" || first == second {
			s.complete = false
			return errors.New("invalid DPZX Hero cards")
		}
		for _, boardCard := range s.board {
			if first == boardCard || second == boardCard {
				s.complete = false
				return errors.New("duplicate DPZX Hero card")
			}
		}
		if hero := s.players[s.heroSeat]; hero != nil {
			hero.holeCards = []string{first, second}
		}
	}
	return nil
}

func (s *dpzxTableState) assignPositions() {
	positions := aapokerPositions(len(s.participants))
	for index, seat := range s.orderedParticipantSeats() {
		position := positions[index]
		if seat == s.heroSeat {
			position += "-Hero"
		}
		s.players[seat].position = position
	}
}

func (s *dpzxTableState) orderedParticipantSeats() []int {
	result := make([]int, 0, len(s.participants))
	for offset := 0; offset < s.tableSize; offset++ {
		seat := (s.buttonSeat + offset) % s.tableSize
		if _, exists := s.players[seat]; exists {
			result = append(result, seat)
		}
	}
	return result
}

func (s *dpzxTableState) decisionRequest() (*model.TexasGtoDecisionReq, string) {
	if !s.ready || !s.complete || s.roomID <= 0 || s.handNum <= 0 ||
		s.operationSeat != s.heroSeat || s.expectedSeat != s.heroSeat || s.players == nil {
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
		Ante: s.ante, Straddle: 0, SbSize: s.smallBlind, BbSize: s.bigBlind,
		CommunityCards: append([]string(nil), s.board...), GameStage: s.stage,
		Players: players, CurrentPlayerPosition: hero.position,
		ActionHistory: append([]model.TexasActionHistory(nil), s.history...),
	}
	if err := model.ValidateTexasGtoDecisionRequest(request); err != nil {
		return nil, ""
	}
	turnKey := fmt.Sprintf("%d:%d:%d:%s", s.roomID, s.handNum, s.turnTimestamp, s.stage)
	return request, turnKey
}

func (s *dpzxTableState) maxBet() int {
	max := 0
	for _, player := range s.players {
		if player.bet > max {
			max = player.bet
		}
	}
	return max
}

func dpzxActionName(betState, target, previousMax int) string {
	switch betState {
	case 2:
		return "call"
	case 3:
		return "check"
	case 4, 13:
		if previousMax == 0 {
			return "bet"
		}
		if target <= previousMax {
			return "call"
		}
		return "raise"
	case 5:
		if target <= previousMax {
			return "call"
		}
		if previousMax == 0 {
			return "bet"
		}
		return "raise"
	case 6:
		return "fold"
	default:
		return ""
	}
}

func dpzxCard(cardID int) string {
	if cardID < 0 || cardID >= 52 {
		return ""
	}
	ranks := [...]string{"2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K", "A"}
	suits := [...]string{"H", "D", "C", "S"}
	return ranks[cardID%13] + "-" + suits[cardID/13]
}

func validDPZXChipAmount(amount int) bool {
	return amount >= 0 && amount <= maxDPZXChipAmount
}
