package service

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
	results  map[string]pokerDecisionCacheEntry
}

type dpzxTableState struct {
	lastSeen time.Time
	userID   int64

	roomID          int64
	handNum         int
	buttonSeat      int
	heroSeat        int
	tableSize       int
	smallBlind      int
	bigBlind        int
	ante            int
	straddleEnabled bool
	straddle        int
	currentPot      int
	stage           string
	board           []string
	participants    []int
	players         map[int]*aapokerPlayerState
	history         []model.TexasActionHistory

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
	BankIndex  int   `json:"bankIndex"`
	MySeat     int   `json:"mySeat"`
	HandNum    int   `json:"HandNum"`
	RoomID     int64 `json:"roomID"`
	LittleBB   int   `json:"littleBB"`
	Ante       int   `json:"ante"`
	TableSize  int   `json:"stisNumMax"`
	IsStraddle bool  `json:"IsStraddle"`
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
		results:  make(map[string]pokerDecisionCacheEntry),
	}
}

func (r *dpzxEventRegistry) process(event *texasPokerEvent, now time.Time) (*PokerEventDecision, error) {
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
		return &PokerEventDecision{
			sessionKey: sessionKey,
			turnKey:    turnKey,
			Cached:     cloneAAPokerDecisionResult(cached.result),
		}, nil
	}
	if turnKey == state.decidedTurn || turnKey == state.inFlightTurn {
		return nil, nil
	}
	state.inFlightTurn = turnKey
	return &PokerEventDecision{sessionKey: sessionKey, turnKey: turnKey, Request: request}, nil
}

func (r *dpzxEventRegistry) complete(
	candidate *PokerEventDecision,
	result *PokerEventDecisionResult,
	now time.Time,
) {
	if candidate == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	state := r.sessions[candidate.sessionKey]
	if state == nil || state.inFlightTurn != candidate.turnKey {
		return
	}
	state.inFlightTurn = ""
	if result == nil {
		return
	}
	state.decidedTurn = candidate.turnKey
	r.removeExpiredResults(now)
	if len(r.results) >= maxDPZXDecisionResults {
		r.removeOldestResult()
	}
	r.results[decisionCacheKey(candidate.sessionKey, candidate.turnKey)] = pokerDecisionCacheEntry{
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

// apply 在 process 的会话锁内按到达顺序消费 DPZX 服务端事件。
// 430 先清理旧手并建立牌桌基线，431 建立本手参与者；492 补 Hero 私牌。
// 之后以 432（谁行动）→ 433（动作结果）配对累计历史，403 校正底池，
// 490 推进公共牌，435 结束。这里只改状态，决策资格由后续 decisionRequest 检查。
// ready 表示本手参与者已建立；complete 表示当前收集到的行动路线可用于决策。
func (s *dpzxTableState) apply(command string, raw []byte, timestamp int64) error {
	switch command {
	case "421":
		// 只有 IsSelf 和游戏SID同时匹配时，才认领该座位为 Hero。
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
		// Hero 离座后退回未就绪状态，旁观期间不再产生策略请求。
		var payload []int
		if err := json.Unmarshal(raw, &payload); err != nil {
			return err
		}
		if len(payload) > 0 && payload[0] == s.heroSeat {
			s.heroSeat = -1
			s.ready = false
		}
	case "430":
		// 房间/手号、庄位、盲注和Hero座位快照；实际玩家状态等待431建立。
		return s.applyRoomSnapshot(raw)
	case "431":
		// 将本手玩家的剩余后手与已投入筹码装入状态，并计算初始底池。
		return s.applyHandStart(raw)
	case "432":
		// 记录等待行动的座位；本事件本身不是一条已经完成的动作。
		return s.applyPrompt(raw, timestamp)
	case "433":
		// 核对等待座位并消费动作结果，更新筹码、底池与历史。
		return s.applyAction(raw)
	case "403":
		// 服务端底池快照直接覆盖本地估算值；非法金额会让本手停止决策。
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
		// DPZX 给出截至当前街的完整公共牌，须校验街道推进和已有前缀。
		return s.applyBoard(raw)
	case "492":
		// 仅提取 Hero 的两张私牌，并拒绝非法、重复或与公共牌冲突的牌。
		return s.applyCards(raw)
	case "435":
		// 结束后清除行动提示和进行中的资格，保留的旧状态不再可决策。
		s.ready = false
		s.operationSeat = -1
		s.expectedSeat = -1
		s.inFlightTurn = ""
	}
	return nil
}

// applyRoomSnapshot 解析 [房间信息, 玩家列表]。玩家列表在此只参与推导桌子座位数；
// 真正的本手玩家/筹码由431建立，避免仅凭房间快照就沿用旧手继续决策。
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

	// 房间基线到达后清空旧手牌面、动作和决策标记，等待完整的431建手。
	s.roomID = room.RoomID
	s.handNum = room.HandNum
	s.buttonSeat = room.BankIndex
	s.heroSeat = room.MySeat
	s.tableSize = tableSize
	s.smallBlind = room.LittleBB
	s.bigBlind = room.LittleBB * 2
	s.ante = room.Ante
	s.straddleEnabled = room.IsStraddle
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

// applyHandStart 解析431的玩家快照。Score 已是剩余后手，Value 是本街已投入，
// 无需再从后手扣除Value；初始底池为每人的前注加上各自Value。
// 只有已定位且参与本手的Hero，才将complete置为true。
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
	if err := s.assignPositions(payload[0]); err != nil {
		s.complete = false
		return err
	}
	return nil
}

// applyPrompt 解析432：[行动座位, ... , 该座位本街投入, ...]。
// expectedSeat 用来等待配对的433，operationSeat供后续判断是否轮到Hero；
// turnTimestamp标识这次行动提示，使同一帧重传能回放同一个决策结果。
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
		// 同座位、同时间戳属于重复提示，不把它当成上一条433缺失。
		return nil
	}
	if s.expectedSeat >= 0 {
		// 上一次432还没等到433就收到新提示：行动路线有缺口，停止本手决策。
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

// applyAction 解析433中唯一一名玩家的动作结果。Betted是本街累计目标，
// Score是动作后的剩余后手，直接覆盖stack，不再从Score扣下注；
// 只有投入的正增量会增加底池。
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
		// 动作座位与432不配对时标记不完整；仍保留随后可观察的状态供排查。
		s.complete = false
	}
	s.expectedSeat = -1
	s.operationSeat = -1
	// 消费完本次提示，等待下一条432；动作类型须结合更新前的全桌最高投入判断。
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

// applyBoard 解析490的完整公共牌快照。合法推进仅为0→3→4→5，
// 旧前缀必须不变，公共牌内部及与Hero私牌之间都不能重复。
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
		// 完全相同的牌面是重传，先去重，避免误判跨街或清掉正在等待的动作。
		return nil
	}
	if s.expectedSeat >= 0 || !validDPZXBoardProgression(len(s.board), len(cardIDs)) {
		// 跨街前仍缺433或跳街时，不允许通过新牌面抹掉动作缺口。
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
	// 整个新快照校验通过后才替换牌面；换街清本街投入/动作，不清后手与历史。
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

// applyCards 解析492的玩家私牌列表，仅使用Hero条目。
// 命中Hero的条目必须唯一且恰好两张有效牌；收到错误更新后标记complete=false，
// 避免沿用之前合法的旧私牌继续生成策略请求。
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

func (s *dpzxTableState) assignPositions(snapshot []dpzxPlayerSnapshot) error {
	// 431的startState明确标识SB/BB；BTN是SB之前的本手参与者，
	// 3/4由用户确认分别为straddle/straddle+1；标记决定位置，Value提供实际强制投入。
	s.straddle = 0
	forcedSeats := map[int]int{}
	forcedAmounts := map[int]int{}
	sb, bb := -1, -1
	for _, player := range snapshot {
		switch player.StartState {
		case 0:
		case 1:
			if sb >= 0 {
				return errors.New("duplicate DPZX small blind")
			}
			sb = player.Index
		case 2:
			if bb >= 0 {
				return errors.New("duplicate DPZX big blind")
			}
			bb = player.Index
		case 3, 4:
			if _, exists := forcedSeats[player.StartState]; exists {
				return errors.New("duplicate DPZX straddle position")
			}
			forcedSeats[player.StartState] = player.Index
			forcedAmounts[player.StartState] = player.Value
		default:
			return errors.New("unsupported DPZX forced position")
		}
	}
	if s.straddleEnabled && len(forcedSeats) == 0 {
		return errors.New("missing DPZX straddle positions")
	}
	if _, hasSecond := forcedSeats[4]; hasSecond {
		if _, hasFirst := forcedSeats[3]; !hasFirst {
			return errors.New("missing DPZX first straddle")
		}
	}
	button := s.buttonSeat
	if sb >= 0 {
		// 单挑时BTN兼SB；多人时从SB逆时针找到上一名参与者。
		button = sb
		if len(snapshot) > 2 {
			for offset := 1; offset < s.tableSize; offset++ {
				seat := (sb - offset + s.tableSize) % s.tableSize
				if s.players[seat] != nil {
					button = seat
					break
				}
			}
		}
	}
	if bb < 0 || bb == button || s.players[button] == nil {
		return errors.New("ambiguous DPZX blind positions")
	}
	// SB缺位时只能使用仍在本手的bankIndex；不能保留一个无玩家的BTN。
	bbOffset := (bb - button + s.tableSize) % s.tableSize
	for _, seat := range s.participants {
		offset := (seat - button + s.tableSize) % s.tableSize
		if seat != button && seat != sb && offset < bbOffset {
			return errors.New("ambiguous DPZX player before blinds")
		}
	}
	if sb >= 0 && (sb-button+s.tableSize)%s.tableSize >= bbOffset {
		return errors.New("invalid DPZX blind order")
	}
	s.buttonSeat = button
	s.players[button].position = "BTN"
	if sb >= 0 && sb != button {
		s.players[sb].position = "SB"
	}
	s.players[bb].position = "BB"
	// 从BB之后依次核对参与者，强制位置必须连续且与标记顺序一致。
	lastForced := bb
	previousAmount := s.bigBlind
	for state := 3; state < 3+len(forcedSeats); state++ {
		next := -1
		for offset := 1; offset < s.tableSize; offset++ {
			seat := (lastForced + offset) % s.tableSize
			if s.players[seat] != nil {
				next = seat
				break
			}
		}
		seat := forcedSeats[state]
		if next != seat || seat == button || seat == sb || forcedAmounts[state] <= previousAmount {
			return errors.New("invalid DPZX straddle order or amount")
		}
		s.players[seat].position = "straddle"
		if state == 4 {
			s.players[seat].position = "straddle+1"
		}
		lastForced = seat
		previousAmount = forcedAmounts[state]
		s.straddle = previousAmount
	}
	var others []int
	for offset := 1; offset < s.tableSize; offset++ {
		seat := (bb + offset) % s.tableSize
		if s.players[seat] != nil && seat != s.buttonSeat && seat != sb {
			if forced, ok := forcedSeats[3]; ok && seat == forced {
				continue
			}
			if forced, ok := forcedSeats[4]; ok && seat == forced {
				continue
			}
			others = append(others, seat)
		}
	}
	positions := [][]string{
		{}, {"CO"}, {"UTG", "CO"}, {"UTG", "MP", "CO"},
		{"UTG", "UTG+1", "MP", "CO"},
		{"UTG", "UTG+1", "MP", "MP+1", "CO"},
		{"UTG", "UTG+1", "UTG+2", "MP", "MP+1", "CO"},
		{"UTG", "UTG+1", "UTG+2", "MP", "MP+1", "MP+2", "CO"},
	}
	if len(others) >= len(positions) {
		return errors.New("unsupported DPZX table positions")
	}
	for index, seat := range others {
		s.players[seat].position = positions[len(others)][index]
	}
	if hero := s.players[s.heroSeat]; hero != nil {
		hero.position += "-Hero"
	}
	return nil
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

// decisionRequest 只在本手ready/complete、Hero仍参与且当前432确实等待Hero时生成请求。
// roomID + handNum + turnTimestamp + stage区分行动点，随后由process处理并发和回放。
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
		Ante: s.ante, Straddle: s.straddle, SbSize: s.smallBlind, BbSize: s.bigBlind,
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
	// 截图与492实物对照：31/3=6H/4D，8/43=9D/5S。
	// 每13张为一个花色，点数从A开始；剩余花色组1对应C。
	ranks := [...]string{"A", "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K"}
	suits := [...]string{"D", "C", "H", "S"}
	return ranks[cardID%13] + "-" + suits[cardID/13]
}

func validDPZXChipAmount(amount int) bool {
	return amount >= 0 && amount <= maxDPZXChipAmount
}
