package service

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
	maxAAPokerSessions          = 512
	aapokerSessionTTL           = 30 * time.Minute
	maxAAPokerDecisionResults   = 1024
	PokerEventDecisionResultTTL = 30 * time.Minute
)

// PokerEventDecision is a reserved Hero turn or a previously completed result.
type PokerEventDecision struct {
	route      pokerEventRoute
	sessionKey string
	turnKey    string
	Request    *model.TexasGtoDecisionReq
	Cached     *PokerEventDecisionResult
}

type aapokerEventRegistry struct {
	mu       sync.Mutex
	sessions map[string]*aapokerTableState
	results  map[string]pokerDecisionCacheEntry
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
	return &aapokerEventRegistry{
		sessions: make(map[string]*aapokerTableState),
		results:  make(map[string]pokerDecisionCacheEntry),
	}
}

func (r *aapokerEventRegistry) process(event *texasPokerEvent, now time.Time) (*PokerEventDecision, error) {
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
	r.removeExpiredResults(now)
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
	if request == nil {
		return nil, nil
	}
	cacheKey := decisionCacheKey(sessionKey, turnKey)
	if cached, ok := r.results[cacheKey]; ok {
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

func (r *aapokerEventRegistry) complete(
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
	if result != nil {
		state.decidedTurn = candidate.turnKey
		r.removeExpiredResults(now)
		if len(r.results) >= maxAAPokerDecisionResults {
			r.removeOldestResult()
		}
		r.results[decisionCacheKey(candidate.sessionKey, candidate.turnKey)] =
			pokerDecisionCacheEntry{
				createdAt: now,
				result:    cloneAAPokerDecisionResult(result),
			}
	}
}

func decisionCacheKey(sessionKey, turnKey string) string {
	return sessionKey + "\x00" + turnKey
}

func cloneAAPokerDecisionResult(result *PokerEventDecisionResult) *PokerEventDecisionResult {
	if result == nil {
		return nil
	}
	cloned := *result
	cloned.HeroCard = append([]string(nil), result.HeroCard...)
	cloned.CommunityCard = append([]string(nil), result.CommunityCard...)
	return &cloned
}

func (r *aapokerEventRegistry) removeExpiredResults(now time.Time) {
	for key, entry := range r.results {
		if now.Sub(entry.createdAt) > PokerEventDecisionResultTTL {
			delete(r.results, key)
		}
	}
}

func (r *aapokerEventRegistry) removeOldestResult() {
	var oldestKey string
	var oldest time.Time
	for key, entry := range r.results {
		if oldestKey == "" || entry.createdAt.Before(oldest) {
			oldestKey, oldest = key, entry.createdAt
		}
	}
	delete(r.results, oldestKey)
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

// apply 按服务端事件的到达顺序更新当前牌桌；调用方 process 已持有会话锁。
// 常见顺序为入桌/坐下 → START_INFOR 建手 → ACTION 累计动作 → CARDS 换街 → 结束。
// 这里只维护牌局，不调用策略：process 会在 apply 成功后用 decisionRequest 检查
// 是否轮到 Hero、状态是否完整，再处理并发资格和结果回放。
func (s *aapokerTableState) apply(command string, payload []byte) error {
	switch command {
	case "REQ_GAME_ENTER_ROOM":
		// 入桌快照用于确认登录游戏账号对应的座位，并补充可见私牌和行动轮次。
		var data aapokerEnterPayload
		if err := json.Unmarshal(payload, &data); err != nil {
			return err
		}
		s.applyEnter(data)
	case "REQ_GAME_RECV_START_INFOR":
		// 新一手的基线：参与者、庄位、盲注、后手、强制投入和初始行动位置。
		var data aapokerStartPayload
		if err := json.Unmarshal(payload, &data); err != nil {
			return err
		}
		return s.applyStart(data)
	case "REQ_GAME_RECV_SEAT_DOWN":
		// 只更新当前游戏账号的 Hero 座位，不能把其他人的坐下事件当成 Hero。
		var data aapokerSeatDownPayload
		if err := json.Unmarshal(payload, &data); err != nil {
			return err
		}
		if data.UserID == s.userID && data.SeatID >= 0 {
			s.heroSeat = data.SeatID
		}
	case "REQ_GAME_RECV_ACTION":
		// 动作带递增 operationRound，负责更新筹码、底池、历史和下一行动者。
		var data aapokerActionPayload
		if err := json.Unmarshal(payload, &data); err != nil {
			return err
		}
		s.applyAction(data)
	case "REQ_GAME_RECV_CARDS":
		// 此协议携带新增公共牌，追加到已有牌面后进入 flop/turn/river。
		var data aapokerCardsPayload
		if err := json.Unmarshal(payload, &data); err != nil {
			return err
		}
		return s.applyCards(data)
	case "REQ_SHOW_SIDE_POTS":
		// 使用服务端给出的各个正数底池之和校正 currentPot，避免重复累计。
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
		// 清除当前行动位置和在途标记；此处不修改complete，
		// 也不阻止后续ACTION更新行动位置。新手状态由START_INFOR建立。
		s.operationSeat = -1
		s.inFlightTurn = ""
	}
	return nil
}

// applyEnter 用 userID 在座位列表中定位 Hero；MySeatID 若有值，必须与之相符。
// 没有匹配座位时按旁观处理。旧快照不能把当前行动位置退回较早的轮次。
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

// applyStart 以 handNum 区分新手与同一手重传。
// 新手清空上一手的公共牌、动作历史和决策标记；同一手重传只补 Hero 私牌，
// 不重置已累计的动作，也不恢复因缺动作而失效的 complete 标记。
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
	// anteCountArray同时含前注与盲注/straddle。对已观察的现金桌协议，
	// 当前将每人的前注按bigChip映射，与已有截图决策请求口径保持一致。
	s.ante = data.BigChip
	s.straddle = 0
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
	// Chips 是快照中的剩余后手；Antes 含前注与盲注/straddle，扣除前注后
	// 才是本街已投入的 bet。这里只纳入本手能参与且有合法筹码的座位。
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
		// 缺少直接底池值时，使用参与者的强制投入构建初始底池。
		for seat, amount := range data.Antes {
			if seat < len(data.CanPlay) && data.CanPlay[seat] == 1 && amount > 0 {
				s.currentPot += amount
			}
		}
	}
	if err := s.assignPositions(data.Straddle); err != nil {
		s.complete = false
		return err
	}
	s.setHeroCards(data.FirstCards, data.SecondCards)
	return nil
}

// applyAction 消费一条已完成动作。AnteNumber 是这次追加的筹码，
// 与玩家本街累计投入 bet 不同；历史中的下注/加注金额使用累计目标值。
func (s *aapokerTableState) applyAction(data aapokerActionPayload) {
	// 尚未建手、旧轮次/重传或负金额不更新状态，避免重复扣筹码。
	if s.players == nil || data.OperationRound <= s.operationRound || data.AnteNumber < 0 {
		return
	}
	if data.OperationRound != s.operationRound+1 {
		// 轮次跳号表示缺动作：继续保留后续可观察状态，但本手不再生成策略请求。
		s.complete = false
	}
	player := s.players[data.SeatID]
	if player == nil {
		return
	}
	// 金额仅更新筹码；动作直接使用协议action，限制本次扣除不超过后手。
	if data.AnteNumber > player.stack {
		data.AnteNumber = player.stack
	}
	player.stack -= data.AnteNumber
	player.bet += data.AnteNumber
	s.currentPot += data.AnteNumber
	action := aapokerActionName(data.Action)
	if action == "" {
		s.complete = false
	}
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

// applyCards 先用轮次去重，再追加本次新发的公共牌；不是替换整个牌面快照。
// 换街只清本街投入和动作展示，不清剩余后手、底池或跨街动作历史。
func (s *aapokerTableState) applyCards(data aapokerCardsPayload) error {
	if s.players == nil || data.OperationRound < s.operationRound {
		return nil
	}
	if data.OperationRound == s.lastCardsRound {
		return nil
	}
	if data.OperationRound > s.operationRound {
		// 换街帧携带最后一个已完成动作的轮次；比本地大说明中间有 ACTION 丢失，
		// 不能用不完整的行动路线继续决策。
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

// assignPositions 仅根据本手straddlePop的明确座位插入抓盲位。
// 当前已确认单级抓盲；多级、非连续或不属于本手的标记不猜测位置。
func (s *aapokerTableState) assignPositions(straddlePop []int) error {
	ordered := s.orderedParticipantSeats()
	if len(ordered) < 2 || len(ordered) > 9 || s.players[s.buttonSeat] == nil {
		return errors.New("unsupported AAPoker button or player count")
	}
	straddleSeat, amount := -1, 0
	for seat, value := range straddlePop {
		if value < 0 {
			return errors.New("invalid AAPoker straddle amount")
		}
		if value == 0 {
			continue
		}
		if straddleSeat >= 0 || s.players[seat] == nil {
			return errors.New("unsupported AAPoker straddle positions")
		}
		straddleSeat, amount = seat, value
	}
	if straddleSeat >= 0 && (len(ordered) < 4 || ordered[3] != straddleSeat || amount <= s.bigBlind) {
		return errors.New("invalid AAPoker straddle position or amount")
	}
	positions := aapokerPositions(len(ordered), straddleSeat >= 0)
	for index, seat := range ordered {
		position := positions[index]
		if seat == s.heroSeat {
			position += "-Hero"
		}
		s.players[seat].position = position
	}
	s.straddle = amount
	return nil
}

// decisionRequest 是 apply 之后的决策门槛：完整建手、轮到 Hero 且私牌/后手有效
// 才构造统一GTO请求；未知对手私牌用占位值，最终仍须通过模型校验。
// handNum + operationRound + stage 标识同一个行动点，供 process 去重/回放。
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

func decodeAAPokerPayload(raw []byte) ([]byte, error) {
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

func rawPositiveInt64(raw []byte) (int64, bool) {
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

// aapokerActionName 使用AAPoker自己的action枚举，与DPZX betState不同。
func aapokerActionName(action int) string {
	switch action {
	case 2:
		return "call"
	case 3:
		return "raise"
	case 4:
		return "allin"
	case 5:
		return "check"
	case 6:
		return "fold"
	default:
		return ""
	}
}

func actionAmount(action string, target int) int {
	if action == "bet" || action == "raise" || action == "call" || action == "allin" {
		return target
	}
	return 0
}

func aapokerPositions(playerCount int, hasStraddle bool) []string {
	if playerCount == 2 {
		return []string{"BTN", "BB"}
	}
	positions := []string{"BTN", "SB", "BB"}
	if hasStraddle {
		positions = append(positions, "straddle")
	}
	// BB（或抓盲）之后的普通位，按本手实际参与人数分配。
	remaining := [][]string{
		{}, {"CO"}, {"UTG", "CO"}, {"UTG", "MP", "CO"},
		{"UTG", "UTG+1", "MP", "CO"},
		{"UTG", "UTG+1", "MP", "MP+1", "CO"},
		{"UTG", "UTG+1", "UTG+2", "MP", "MP+1", "CO"},
	}
	return append(positions, remaining[playerCount-len(positions)]...)
}

func PokerEventHeroMaxTarget(request *model.TexasGtoDecisionReq) int {
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
