package model

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

type RecognizeRequest struct {
	ImageBase64 string `json:"image_base64"`
	App         string `json:"app"`
	ParseAll    bool   `json:"parse_all"`
}
type TexasResult struct {
	ReceivedAt string `json:"received_at"`
	ParsedAt   string `json:"parsed_at"`
	ElapsedMs  int    `json:"elapsed_ms"`
	App        string `json:"app"`
	TableInfo  struct {
		Stage          string   `json:"stage"`
		CommunityCards []string `json:"community_cards"`
		MainPot        int      `json:"main_pot"`
		ButtonSeat     int      `json:"button_seat"`
		BlindSize      string   `json:"blind_size"`
		MushroomCount  int      `json:"mushroom_count"`
	} `json:"table_info"`
	HeroInfo struct {
		Seat       int      `json:"seat"`
		Status     string   `json:"status"`
		Stack      int      `json:"stack"`
		CurrentBet int      `json:"current_bet"`
		IsHeroTurn bool     `json:"is_hero_turn"`
		HeroCards  []string `json:"hero_cards"`
	} `json:"hero_info"`
	VillainsInfo []struct {
		Seat       int    `json:"seat"`
		Status     string `json:"status"`
		Stack      int    `json:"stack"`
		CurrentBet int    `json:"current_bet"`
	} `json:"villains_info"`
	// ObservedActionHistory is kept only in the in-memory hand cache. It lets us
	// preserve actions observed between consecutive screenshots without changing
	// the recognition or decision protocols.
	ObservedActionHistory []TexasActionHistory `json:"-"`
}

type TexasPlayer struct {
	Position    string   `json:"position"`
	HoleCards   []string `json:"hole_cards"`
	Stack       int      `json:"stack"`
	Bet         int      `json:"bet"`
	IsActive    bool     `json:"is_active"`
	ActionTaken string   `json:"action_taken"`
}
type TexasActionHistory struct {
	Stage     string `json:"stage"`
	Position  string `json:"position"`
	Action    string `json:"action"`
	Amount    int    `json:"amount"`
	Timestamp int    `json:"timestamp"`
	Status    string `json:"-"`
}
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
}

type TexasGtoDecisionResp struct {
	Action            string  `json:"action"`
	RaiseSize         int     `json:"raise_size"`
	ActionProbability float64 `json:"action_probability"`
	Detail            string  `json:"detail"`
}

// CanTexasGtoDecisionCheck reports whether Hero has matched the largest
// current bet and can therefore continue without committing more chips.
// Invalid requests return false; callers must still validate the full request
// before treating this as a decision fact.
func CanTexasGtoDecisionCheck(req *TexasGtoDecisionReq) bool {
	if req == nil {
		return false
	}
	heroBet := 0
	heroFound := false
	maxBet := 0
	for _, player := range req.Players {
		if player.Bet > maxBet {
			maxBet = player.Bet
		}
		if player.Position == req.CurrentPlayerPosition {
			heroBet = player.Bet
			heroFound = true
		}
	}
	return heroFound && heroBet >= maxBet
}

// ValidateTexasGtoDecisionRequest rejects recognition results that cannot
// represent a player who is currently allowed to act. Sending such data to
// the decision service can turn an OCR miss into an unsafe automated action.
func ValidateTexasGtoDecisionRequest(req *TexasGtoDecisionReq) error {
	if req == nil {
		return errors.New("decision request is nil")
	}
	expectedCommunityCards := -1
	switch req.GameStage {
	case "preflop":
		expectedCommunityCards = 0
	case "flop":
		expectedCommunityCards = 3
	case "turn":
		expectedCommunityCards = 4
	case "river":
		expectedCommunityCards = 5
	default:
		return fmt.Errorf("decision request has invalid game stage %q", req.GameStage)
	}
	if len(req.CommunityCards) != expectedCommunityCards {
		return fmt.Errorf("%s decision request has %d community cards, want %d", req.GameStage, len(req.CommunityCards), expectedCommunityCards)
	}
	previousStage := -1
	previousTimestamp := -1
	for index, action := range req.ActionHistory {
		actionStage := stageOrder(action.Stage)
		if actionStage < 0 || actionStage > expectedCommunityCardsStage(expectedCommunityCards) {
			return fmt.Errorf("action history item %d has invalid stage %q for %s", index, action.Stage, req.GameStage)
		}
		if actionStage < previousStage {
			return fmt.Errorf("action history item %d regresses from %s to %s", index, req.ActionHistory[index-1].Stage, action.Stage)
		}
		if action.Timestamp <= previousTimestamp {
			return fmt.Errorf("action history item %d has non-increasing timestamp %d", index, action.Timestamp)
		}
		previousStage = actionStage
		previousTimestamp = action.Timestamp
	}
	for _, player := range req.Players {
		if player.Position != req.CurrentPlayerPosition {
			continue
		}
		if !player.IsActive {
			return fmt.Errorf("current player %s is inactive", player.Position)
		}
		if player.Stack <= 0 {
			return fmt.Errorf("current player %s has invalid stack %d", player.Position, player.Stack)
		}
		if len(player.HoleCards) != 2 {
			return fmt.Errorf("current player %s has %d hole cards, want 2", player.Position, len(player.HoleCards))
		}
		seenCards := make(map[string]struct{}, 2+len(req.CommunityCards))
		for _, card := range append(append([]string(nil), player.HoleCards...), req.CommunityCards...) {
			if !isValidTexasCard(card) {
				return fmt.Errorf("decision request has invalid card %q", card)
			}
			if _, exists := seenCards[card]; exists {
				return fmt.Errorf("decision request has duplicate card %q", card)
			}
			seenCards[card] = struct{}{}
		}
		return nil
	}
	return fmt.Errorf("current player %q is missing", req.CurrentPlayerPosition)
}

func expectedCommunityCardsStage(cardCount int) int {
	switch cardCount {
	case 0:
		return 0
	case 3:
		return 1
	case 4:
		return 2
	case 5:
		return 3
	default:
		return -1
	}
}

func isValidTexasCard(card string) bool {
	parts := strings.Split(card, "-")
	if len(parts) != 2 {
		return false
	}
	switch parts[0] {
	case "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K", "A":
	default:
		return false
	}
	switch parts[1] {
	case "S", "H", "C", "D":
		return true
	default:
		return false
	}
}

// NormalizeTexasGtoDecisionToStack converts an oversized bet/raise target to
// the maximum total Hero can put in. An undersized all-in raise is represented
// as call because it cannot reopen betting under poker rules.
func NormalizeTexasGtoDecisionToStack(req *TexasGtoDecisionReq, resp *TexasGtoDecisionResp) bool {
	if req == nil || resp == nil || (resp.Action != "bet" && resp.Action != "raise") {
		return false
	}
	maxBet := 0
	for _, player := range req.Players {
		if player.Bet > maxBet {
			maxBet = player.Bet
		}
	}
	for _, player := range req.Players {
		if player.Position != req.CurrentPlayerPosition {
			continue
		}
		maxTarget := player.Bet + player.Stack
		if resp.RaiseSize > maxTarget {
			if resp.Action == "raise" && maxTarget <= maxBet {
				resp.Action = "call"
				resp.RaiseSize = 0
				return true
			}
			resp.RaiseSize = maxTarget
			return true
		}
		return false
	}
	return false
}

// ValidateTexasGtoDecisionResponse checks that the returned action is legal
// for the current snapshot after any stack cap has been applied. RaiseSize is
// interpreted as the target total bet, matching the existing protocol.
func ValidateTexasGtoDecisionResponse(req *TexasGtoDecisionReq, resp *TexasGtoDecisionResp) error {
	if err := ValidateTexasGtoDecisionRequest(req); err != nil {
		return err
	}
	if resp == nil {
		return errors.New("decision response is nil")
	}

	var hero *TexasPlayer
	maxBet := 0
	for i := range req.Players {
		player := &req.Players[i]
		if player.Bet > maxBet {
			maxBet = player.Bet
		}
		if player.Position == req.CurrentPlayerPosition {
			hero = player
		}
	}
	maxTarget := hero.Bet + hero.Stack
	facingBet := maxBet > hero.Bet

	switch resp.Action {
	case "fold":
		if resp.RaiseSize != 0 {
			return fmt.Errorf("fold has non-zero raise_size %d", resp.RaiseSize)
		}
	case "check":
		if facingBet {
			return fmt.Errorf("cannot check facing bet %d with %d committed", maxBet, hero.Bet)
		}
		if resp.RaiseSize != 0 {
			return fmt.Errorf("check has non-zero raise_size %d", resp.RaiseSize)
		}
	case "call":
		if !facingBet {
			return errors.New("cannot call without an outstanding bet")
		}
		if resp.RaiseSize != 0 {
			return fmt.Errorf("call has non-zero raise_size %d", resp.RaiseSize)
		}
	case "bet":
		if maxBet != 0 {
			return fmt.Errorf("cannot bet when current maximum bet is %d", maxBet)
		}
		if resp.RaiseSize <= 0 || resp.RaiseSize > maxTarget {
			return fmt.Errorf("bet target %d is outside (0, %d]", resp.RaiseSize, maxTarget)
		}
	case "raise":
		if !facingBet {
			return errors.New("cannot raise without an outstanding bet")
		}
		if resp.RaiseSize <= maxBet || resp.RaiseSize > maxTarget {
			return fmt.Errorf("raise target %d must be in (%d, %d]", resp.RaiseSize, maxBet, maxTarget)
		}
	default:
		return fmt.Errorf("unknown decision action %q", resp.Action)
	}
	return nil
}

func Conv2TexasGtoDecisionReq(ctx context.Context, recResult []*TexasResult) *TexasGtoDecisionReq {
	if len(recResult) == 0 {
		return nil
	}

	currentResult := recResult[len(recResult)-1]

	sbSize, bbSize := parseBlindSize(currentResult.TableInfo.BlindSize)

	players, currentPlayerPos := buildPlayers(currentResult, bbSize)
	actionHistory := buildActionHistory(ctx, recResult)

	return &TexasGtoDecisionReq{
		GameType:              "no_limit_holdem",
		NumPlayers:            len(players),
		CurrentPot:            currentResult.TableInfo.MainPot,
		SbSize:                sbSize,
		BbSize:                bbSize,
		Ante:                  bbSize,
		Straddle:              bbSize * 2,
		Mushroom:              currentResult.TableInfo.MushroomCount / bbSize, // 蘑菇要换算成多少个bb表示
		CommunityCards:        parseCards(currentResult.TableInfo.CommunityCards),
		GameStage:             currentResult.TableInfo.Stage,
		Players:               players,
		CurrentPlayerPosition: currentPlayerPos,
		ActionHistory:         actionHistory,
	}
}

// PrepareActionHistory carries the current hand's accumulated action events to
// result and appends the minimum actions observable between two snapshots. A
// nil previous result starts a new hand and falls back to a minimum legal
// reconstruction of the first snapshot.
func PrepareActionHistory(previous, result *TexasResult) {
	if result == nil {
		return
	}

	if IsNewHand(previous, result) {
		result.ObservedActionHistory = inferSnapshotActions(result)
		return
	}

	result.ObservedActionHistory = append(
		make([]TexasActionHistory, 0, len(previous.ObservedActionHistory)),
		previous.ObservedActionHistory...,
	)
	if normalizeStage(previous.TableInfo.Stage) == normalizeStage(result.TableInfo.Stage) {
		result.ObservedActionHistory = append(result.ObservedActionHistory, inferSnapshotDelta(previous, result)...)
		return
	}

	result.ObservedActionHistory = append(result.ObservedActionHistory, inferStageCompletion(previous, result)...)
	stageActions := inferSnapshotActions(result)
	for _, action := range stageActions {
		// A player already folded in the first snapshot of a new street must
		// have folded while completing the previous street.
		if action.Action == "fold" {
			continue
		}
		action.Timestamp = nextTimestamp(result.ObservedActionHistory)
		result.ObservedActionHistory = append(result.ObservedActionHistory, action)
	}
}

// IsNewHand reports whether two consecutive full recognitions cannot belong to
// the same poker hand. Button movement is the usual signal, but some clients
// keep reporting a fixed button seat. In that case a street regression or a
// change of the two visible Hero cards is sufficient evidence of a new hand.
func IsNewHand(previous, current *TexasResult) bool {
	if previous == nil || current == nil {
		return true
	}
	if previous.TableInfo.ButtonSeat != current.TableInfo.ButtonSeat {
		return true
	}
	if stageOrder(current.TableInfo.Stage) < stageOrder(previous.TableInfo.Stage) {
		return true
	}
	return completeHeroCardsChanged(previous.HeroInfo.HeroCards, current.HeroInfo.HeroCards)
}

func stageOrder(stage string) int {
	switch normalizeStage(stage) {
	case "preflop":
		return 0
	case "flop":
		return 1
	case "turn":
		return 2
	case "river":
		return 3
	default:
		return -1
	}
}

func completeHeroCardsChanged(previous, current []string) bool {
	if len(previous) != 2 || len(current) != 2 {
		return false
	}
	previousCards := append([]string(nil), previous...)
	currentCards := append([]string(nil), current...)
	sort.Strings(previousCards)
	sort.Strings(currentCards)
	return previousCards[0] != currentCards[0] || previousCards[1] != currentCards[1]
}

func parseCards(cards []string) []string {
	newCards := make([]string, 0)
	for _, card := range cards {
		faceAndSuit := strings.Split(card, " ")
		if len(faceAndSuit) >= 2 {
			newCards = append(newCards, faceAndSuit[0]+"-"+convSuit(faceAndSuit[1]))
		}
	}
	return newCards
}
func convSuit(suit string) string {
	switch suit {
	case "黑桃":
		return "S"
	case "红桃":
		return "H"
	case "梅花":
		return "C"
	case "方块":
		return "D"
	default:
		return ""
	}
}

func tableSeatCount(villainCount int) int {
	return villainCount + 1
}

func tablePositions(playerCount int) []string {
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
		// 牌桌容量目前只支持 2～9 人；保留安全兜底，避免异常识别结果导致除零。
		return []string{"BTN"}
	}
}

func isParticipatingStatus(status string) bool {
	return status != "empty" && status != "waiting"
}

func parseBlindSize(blindSizeStr string) (int, int) {
	sliceBindSize := strings.Split(blindSizeStr, "(")
	sliceBindSizeV2 := strings.Split(sliceBindSize[0], "/")
	if len(sliceBindSizeV2) < 2 {
		return 1, 2
	}
	sbSize, _ := strconv.ParseInt(sliceBindSizeV2[0], 10, 0)
	bbSize, _ := strconv.ParseInt(sliceBindSizeV2[1], 10, 0)
	return int(sbSize), int(bbSize)
}

func buildPlayers(result *TexasResult, bbSize int) ([]TexasPlayer, string) {
	type seatInfo struct {
		seat       int
		isHero     bool
		status     string
		stack      int
		currentBet int
		heroCards  []string
		isHeroTurn bool
	}

	// 收集所有有玩家的座位
	allSeats := make([]seatInfo, 0)
	if isParticipatingStatus(result.HeroInfo.Status) {
		allSeats = append(allSeats, seatInfo{
			seat:       result.HeroInfo.Seat,
			isHero:     true,
			status:     result.HeroInfo.Status,
			stack:      result.HeroInfo.Stack,
			currentBet: result.HeroInfo.CurrentBet,
			heroCards:  result.HeroInfo.HeroCards,
			isHeroTurn: result.HeroInfo.IsHeroTurn,
		})
	}

	for _, villain := range result.VillainsInfo {
		if !isParticipatingStatus(villain.Status) {
			continue
		}
		allSeats = append(allSeats, seatInfo{
			seat:       villain.Seat,
			isHero:     false,
			status:     villain.Status,
			stack:      villain.Stack,
			currentBet: villain.CurrentBet,
		})
	}

	// 从按钮位开始，按顺时针顺序找到有玩家的座位。
	// 识别服务会为每个非 hero 座位返回一项（包括空座/等待座位），因此
	// VillainsInfo 的长度加上 hero 就是当前牌桌的座位数。
	totalSeats := tableSeatCount(len(result.VillainsInfo))
	orderedSeats := make([]seatInfo, 0)

	for offset := 0; offset < totalSeats; offset++ {
		seat := ((result.TableInfo.ButtonSeat - 1 + offset) % totalSeats) + 1

		for _, si := range allSeats {
			if si.seat == seat {
				orderedSeats = append(orderedSeats, si)
				break
			}
		}
	}

	// 空座/等待座位不进入 orderedSeats，因此位置表必须按实际参与玩家数选择，
	// 不能按桌容量选择，否则九人桌少一个空座时会把后位整体错移一位。
	positions := tablePositions(len(orderedSeats))
	players := make([]TexasPlayer, 0)
	currentPlayerPos := ""

	// 先创建所有玩家，不带ActionTaken
	type playerWithInfo struct {
		player TexasPlayer
		info   seatInfo
		pos    string
	}
	playerList := make([]playerWithInfo, 0)

	for i, si := range orderedSeats {
		position := positions[i%len(positions)]
		var player TexasPlayer
		if si.isHero {
			player = TexasPlayer{
				Position:  position + "-Hero",
				HoleCards: parseCards(si.heroCards),
				Stack:     si.stack,
				Bet:       si.currentBet,
				IsActive:  si.status == "active" || si.status == "all_in",
			}
			currentPlayerPos = position + "-Hero"
		} else {
			player = TexasPlayer{
				Position:    position,
				HoleCards:   []string{"X-X", "X-X"},
				Stack:       si.stack,
				Bet:         si.currentBet,
				IsActive:    si.status == "active" || si.status == "all_in",
				ActionTaken: "",
			}
		}
		playerList = append(playerList, playerWithInfo{player, si, position})
	}

	// 确定行动开始位置
	var startIndex int
	if result.TableInfo.Stage == "preflop" {
		// 有 straddle 时从其下一位开始；短桌没有 straddle 时从 BTN 开始。
		for i, pi := range playerList {
			if pi.pos == "straddle" {
				startIndex = (i + 1) % len(playerList)
				break
			}
		}
	} else {
		// 其他阶段从SB开始
		for i, pi := range playerList {
			if pi.pos == "SB" {
				startIndex = i
				break
			}
		}
	}

	// 按行动顺序处理每个玩家
	actionOrder := make([]int, 0)
	for i := 0; i < len(playerList); i++ {
		actionOrder = append(actionOrder, (startIndex+i)%len(playerList))
	}

	// 处理ActionTaken
	prevBet := 0
	isFirst := true
	for _, idx := range actionOrder {
		pi := &playerList[idx]

		// 检查是否是hero
		if pi.info.isHero { // hero之后的不再给action_taken
			break
		}

		// 判断ActionTaken
		if !pi.player.IsActive {
			pi.player.ActionTaken = "fold"
		} else if pi.player.Bet == 0 {
			pi.player.ActionTaken = "check"
		} else if isFirst || (prevBet == 0 && pi.player.Bet > 0) {
			pi.player.ActionTaken = "bet"
		} else if pi.player.Bet == prevBet && pi.player.Bet > 0 {
			pi.player.ActionTaken = "call"
		} else if pi.player.Bet >= int(1.6*float64(prevBet)) { // 大于1.6倍就算raise了
			pi.player.ActionTaken = "raise"
		} else {
			pi.player.ActionTaken = "call"
		}

		if result.TableInfo.Stage == "preflop" && pi.player.Bet == bbSize*2 && pi.player.ActionTaken == "raise" {
			pi.player.ActionTaken = "call"
		}

		if result.TableInfo.Stage == "preflop" && pi.player.ActionTaken == "bet" { // preflop下注时，实际是call
			pi.player.ActionTaken = "call"
		}

		// 更新前位bet
		if pi.player.Bet > prevBet {
			prevBet = pi.player.Bet
		}
		isFirst = false
	}

	// 构建最终的players数组
	for _, pi := range playerList {
		players = append(players, pi.player)
	}

	return players, currentPlayerPos
}

type actionSeat struct {
	seat       int
	position   string
	status     string
	currentBet int
	order      int
}

func normalizeStage(stage string) string {
	if stage == "" {
		return "preflop"
	}
	return stage
}

func actionSeats(result *TexasResult) []actionSeat {
	if result == nil {
		return nil
	}

	type rawSeat struct {
		seat       int
		isHero     bool
		status     string
		currentBet int
	}
	raw := make([]rawSeat, 0, len(result.VillainsInfo)+1)
	if isParticipatingStatus(result.HeroInfo.Status) {
		raw = append(raw, rawSeat{
			seat:       result.HeroInfo.Seat,
			isHero:     true,
			status:     result.HeroInfo.Status,
			currentBet: result.HeroInfo.CurrentBet,
		})
	}
	for _, villain := range result.VillainsInfo {
		if !isParticipatingStatus(villain.Status) {
			continue
		}
		raw = append(raw, rawSeat{
			seat:       villain.Seat,
			status:     villain.Status,
			currentBet: villain.CurrentBet,
		})
	}

	totalSeats := tableSeatCount(len(result.VillainsInfo))
	ordered := make([]rawSeat, 0, len(raw))
	for offset := 0; offset < totalSeats; offset++ {
		seat := ((result.TableInfo.ButtonSeat - 1 + offset) % totalSeats) + 1
		for _, player := range raw {
			if player.seat == seat {
				ordered = append(ordered, player)
				break
			}
		}
	}
	if len(ordered) == 0 {
		return nil
	}

	positions := tablePositions(len(ordered))
	startIndex := 0
	stage := normalizeStage(result.TableInfo.Stage)
	if stage == "preflop" {
		for i, position := range positions {
			if position == "straddle" {
				startIndex = (i + 1) % len(ordered)
				break
			}
		}
	} else {
		for i, position := range positions {
			if position == "SB" {
				startIndex = i
				break
			}
		}
	}

	resultSeats := make([]actionSeat, 0, len(ordered))
	for i, player := range ordered {
		position := positions[i%len(positions)]
		if player.isHero {
			position += "-Hero"
		}
		order := (i - startIndex + len(ordered)) % len(ordered)
		resultSeats = append(resultSeats, actionSeat{
			seat:       player.seat,
			position:   position,
			status:     player.status,
			currentBet: player.currentBet,
			order:      order,
		})
	}
	return resultSeats
}

func forcedBlind(position string, sbSize, bbSize int) int {
	position = strings.TrimSuffix(position, "-Hero")
	switch position {
	case "SB":
		return sbSize
	case "BB":
		return bbSize
	case "straddle":
		return bbSize * 2
	default:
		return 0
	}
}

func isActiveStatus(status string) bool {
	return status == "active" || status == "all_in"
}

func nextTimestamp(history []TexasActionHistory) int {
	if len(history) == 0 {
		return 0
	}
	return history[len(history)-1].Timestamp + 1
}

func initialStreetBet(stage string, seats []actionSeat, sbSize, bbSize int) int {
	if stage != "preflop" {
		return 0
	}
	maxBlind := 0
	for _, player := range seats {
		if blind := forcedBlind(player.position, sbSize, bbSize); blind > maxBlind {
			maxBlind = blind
		}
	}
	return maxBlind
}

type inferredActionEvent struct {
	player actionSeat
	target int
}

func circularDistance(order, afterOrder, playerCount int) int {
	return (order - afterOrder - 1 + playerCount) % playerCount
}

// orderInferredEvents keeps observed active bets in ascending target order so
// missing raises form the shortest legal sequence. Folds and all-ins are then
// inserted by seat order: their amount alone cannot tell whether they happened
// before a later raise or were a short call after it.
func orderInferredEvents(events []inferredActionEvent, lastOrder, playerCount int) []inferredActionEvent {
	regular := make([]inferredActionEvent, 0, len(events))
	seatOrdered := make([]inferredActionEvent, 0, len(events))
	for _, event := range events {
		if event.player.status == "active" {
			regular = append(regular, event)
		} else {
			seatOrdered = append(seatOrdered, event)
		}
	}
	sort.SliceStable(regular, func(i, j int) bool {
		if regular[i].target == regular[j].target {
			return circularDistance(regular[i].player.order, lastOrder, playerCount) <
				circularDistance(regular[j].player.order, lastOrder, playerCount)
		}
		return regular[i].target < regular[j].target
	})

	ordered := make([]inferredActionEvent, 0, len(events))
	cursor := lastOrder
	for _, bet := range regular {
		betDistance := circularDistance(bet.player.order, cursor, playerCount)
		before := make([]inferredActionEvent, 0)
		remaining := make([]inferredActionEvent, 0, len(seatOrdered))
		for _, event := range seatOrdered {
			if circularDistance(event.player.order, cursor, playerCount) <= betDistance {
				before = append(before, event)
			} else {
				remaining = append(remaining, event)
			}
		}
		sort.SliceStable(before, func(i, j int) bool {
			return circularDistance(before[i].player.order, cursor, playerCount) <
				circularDistance(before[j].player.order, cursor, playerCount)
		})
		ordered = append(ordered, before...)
		ordered = append(ordered, bet)
		seatOrdered = remaining
		cursor = bet.player.order
	}
	sort.SliceStable(seatOrdered, func(i, j int) bool {
		return circularDistance(seatOrdered[i].player.order, cursor, playerCount) <
			circularDistance(seatOrdered[j].player.order, cursor, playerCount)
	})
	return append(ordered, seatOrdered...)
}

// inferSnapshotActions builds the shortest legal history consistent with one
// snapshot. SB/BB/straddle are forced bets rather than voluntary actions.
func inferSnapshotActions(result *TexasResult) []TexasActionHistory {
	seats := actionSeats(result)
	if len(seats) == 0 {
		return []TexasActionHistory{}
	}
	stage := normalizeStage(result.TableInfo.Stage)
	sbSize, bbSize := parseBlindSize(result.TableInfo.BlindSize)
	currentMax := initialStreetBet(stage, seats, sbSize, bbSize)

	events := make([]inferredActionEvent, 0, len(seats))
	for _, player := range seats {
		if !isActiveStatus(player.status) {
			// A static snapshot cannot reveal which raise caused a fold. Place
			// it at the earliest legal betting level instead of claiming that
			// it happened after every visible raise.
			events = append(events, inferredActionEvent{player: player, target: currentMax})
			continue
		}
		baseline := 0
		if stage == "preflop" {
			baseline = forcedBlind(player.position, sbSize, bbSize)
		}
		if player.currentBet > baseline && (player.currentBet >= currentMax || player.status == "all_in") {
			events = append(events, inferredActionEvent{player: player, target: player.currentBet})
		}
	}

	events = orderInferredEvents(events, -1, len(seats))

	history := make([]TexasActionHistory, 0, len(events))
	for _, event := range events {
		player := event.player
		if !isActiveStatus(player.status) {
			history = append(history, TexasActionHistory{
				Stage: stage, Position: player.position, Action: "fold",
				Amount: 0, Timestamp: len(history), Status: player.status,
			})
			continue
		}
		action := "call"
		if player.currentBet > currentMax {
			if stage != "preflop" && currentMax == 0 {
				action = "bet"
			} else {
				action = "raise"
			}
			currentMax = player.currentBet
		}
		history = append(history, TexasActionHistory{
			Stage: stage, Position: player.position, Action: action,
			Amount: player.currentBet, Timestamp: len(history), Status: player.status,
		})
	}
	return history
}

func inferSnapshotDelta(previous, current *TexasResult) []TexasActionHistory {
	previousSeats := actionSeats(previous)
	currentSeats := actionSeats(current)
	if len(previousSeats) == 0 || len(currentSeats) == 0 {
		return nil
	}

	previousBySeat := make(map[int]actionSeat, len(previousSeats))
	sbSize, bbSize := parseBlindSize(previous.TableInfo.BlindSize)
	stage := normalizeStage(current.TableInfo.Stage)
	currentMax := initialStreetBet(stage, previousSeats, sbSize, bbSize)
	for _, player := range previousSeats {
		previousBySeat[player.seat] = player
		if player.currentBet > currentMax {
			currentMax = player.currentBet
		}
	}

	events := make([]inferredActionEvent, 0)
	for _, player := range currentSeats {
		old, ok := previousBySeat[player.seat]
		if !ok {
			continue
		}
		if isActiveStatus(old.status) && !isActiveStatus(player.status) {
			events = append(events, inferredActionEvent{player: player, target: currentMax})
			continue
		}
		if isActiveStatus(player.status) && player.currentBet > old.currentBet &&
			(player.currentBet >= currentMax || player.status == "all_in") {
			events = append(events, inferredActionEvent{player: player, target: player.currentBet})
		}
	}

	lastOrder := -1
	orderByPosition := make(map[string]int, len(previousSeats))
	for _, player := range previousSeats {
		orderByPosition[player.position] = player.order
	}
	for _, action := range previous.ObservedActionHistory {
		if normalizeStage(action.Stage) != stage {
			continue
		}
		if order, ok := orderByPosition[action.Position]; ok {
			lastOrder = order
		}
	}
	events = orderInferredEvents(events, lastOrder, len(previousSeats))

	timestamp := nextTimestamp(previous.ObservedActionHistory)
	history := make([]TexasActionHistory, 0, len(events))
	for _, event := range events {
		player := event.player
		if !isActiveStatus(player.status) {
			history = append(history, TexasActionHistory{
				Stage: stage, Position: player.position, Action: "fold",
				Amount: 0, Timestamp: timestamp, Status: player.status,
			})
			timestamp++
			continue
		}
		action := "call"
		if player.currentBet > currentMax {
			if stage != "preflop" && currentMax == 0 {
				action = "bet"
			} else {
				action = "raise"
			}
			currentMax = player.currentBet
		}
		history = append(history, TexasActionHistory{
			Stage: stage, Position: player.position, Action: action,
			Amount: player.currentBet, Timestamp: timestamp, Status: player.status,
		})
		timestamp++
	}
	return history
}

// inferStageCompletion adds the minimum missing calls, checks and folds needed
// when the next street proves that the previous betting round has completed.
func inferStageCompletion(previous, current *TexasResult) []TexasActionHistory {
	previousSeats := actionSeats(previous)
	currentSeats := actionSeats(current)
	if len(previousSeats) == 0 || len(currentSeats) == 0 {
		return nil
	}

	currentBySeat := make(map[int]actionSeat, len(currentSeats))
	for _, player := range currentSeats {
		currentBySeat[player.seat] = player
	}
	stage := normalizeStage(previous.TableInfo.Stage)
	sbSize, bbSize := parseBlindSize(previous.TableInfo.BlindSize)
	stageMax := initialStreetBet(stage, previousSeats, sbSize, bbSize)
	for _, player := range previousSeats {
		if player.currentBet > stageMax {
			stageMax = player.currentBet
		}
	}

	lastAmount := make(map[string]int)
	folded := make(map[string]bool)
	lastOrder := -1
	orderByPosition := make(map[string]int, len(previousSeats))
	for _, player := range previousSeats {
		orderByPosition[player.position] = player.order
	}
	for _, action := range previous.ObservedActionHistory {
		if normalizeStage(action.Stage) != stage {
			continue
		}
		if action.Amount > lastAmount[action.Position] {
			lastAmount[action.Position] = action.Amount
		}
		if action.Amount > stageMax {
			stageMax = action.Amount
		}
		if action.Action == "fold" {
			folded[action.Position] = true
		}
		if order, ok := orderByPosition[action.Position]; ok {
			lastOrder = order
		}
	}

	pending := make([]TexasActionHistory, 0)
	pendingOrder := make(map[string]int)
	for _, player := range previousSeats {
		next, ok := currentBySeat[player.seat]
		if !ok {
			continue
		}
		pendingOrder[player.position] = player.order
		if player.status == "all_in" {
			continue
		}
		if !isActiveStatus(next.status) {
			if isActiveStatus(player.status) && !folded[player.position] {
				pending = append(pending, TexasActionHistory{
					Stage: stage, Position: player.position, Action: "fold", Status: next.status,
				})
			}
			continue
		}
		if !isActiveStatus(player.status) {
			continue
		}

		accounted := lastAmount[player.position]
		if stage == "preflop" {
			if blind := forcedBlind(player.position, sbSize, bbSize); blind > accounted {
				accounted = blind
			}
		}
		if stageMax == 0 {
			pending = append(pending, TexasActionHistory{
				Stage: stage, Position: player.position, Action: "check", Status: player.status,
			})
		} else if accounted < stageMax {
			pending = append(pending, TexasActionHistory{
				Stage: stage, Position: player.position, Action: "call", Amount: stageMax, Status: player.status,
			})
		}
	}

	playerCount := len(previousSeats)
	sort.SliceStable(pending, func(i, j int) bool {
		iOrder := (pendingOrder[pending[i].Position] - lastOrder - 1 + playerCount) % playerCount
		jOrder := (pendingOrder[pending[j].Position] - lastOrder - 1 + playerCount) % playerCount
		return iOrder < jOrder
	})
	timestamp := nextTimestamp(previous.ObservedActionHistory)
	for i := range pending {
		pending[i].Timestamp = timestamp + i
	}
	return pending
}

// buildActionHistory 构建德州扑克的行动历史
//
// 参数:
//   - ctx: 上下文对象
//   - recResult: 识别结果数组，按照时间顺序排列（preflop → flop → turn → river）
//
// 返回:
//   - []TexasActionHistory: 行动历史数组，包含每个阶段的玩家行动
//
// 说明:
//
//	缓存已累计连续快照动作时，直接返回最新结果携带的历史；旧调用方没有累计历史时，
//	继续使用阶段末快照推断，保持兼容。
func buildActionHistory(ctx context.Context, recResult []*TexasResult) []TexasActionHistory {
	logrus.WithContext(ctx).Infof("[buildActionHistory] len(recResult):%v", len(recResult))
	if len(recResult) < 2 {
		if len(recResult) == 1 && recResult[0].ObservedActionHistory != nil {
			return append([]TexasActionHistory(nil), recResult[0].ObservedActionHistory...)
		}
		return make([]TexasActionHistory, 0)
	}
	latest := recResult[len(recResult)-1]
	if latest.ObservedActionHistory != nil {
		return append([]TexasActionHistory(nil), latest.ObservedActionHistory...)
	}

	history := make([]TexasActionHistory, 0)
	timestamp := int(0)

	// 只处理到第len(recResult)-2个结果，最后一个是当前阶段的数据
	processedResults := recResult[:len(recResult)-1]

	// 按stage分组，取每个stage的最后一个结果
	groupByStage := make(map[string]*TexasResult)
	for _, result := range processedResults {
		stage := result.TableInfo.Stage
		if stage == "" {
			stage = "preflop"
		}
		groupByStage[stage] = result // 总是覆盖，保留最后一个
	}

	// 按stage顺序处理：preflop → flop → turn → river
	stages := []string{"preflop", "flop", "turn", "river"}
	for _, stage := range stages {
		result, ok := groupByStage[stage]
		if !ok {
			continue
		}

		// 像buildPlayers那样，从单个结果中推断出该阶段的行动历史
		type seatInfo struct {
			seat       int
			isHero     bool
			status     string
			stack      int
			currentBet int
			heroCards  []string
			isHeroTurn bool
		}

		// 收集所有有玩家的座位
		allSeats := make([]seatInfo, 0)
		if isParticipatingStatus(result.HeroInfo.Status) {
			allSeats = append(allSeats, seatInfo{
				seat:       result.HeroInfo.Seat,
				isHero:     true,
				status:     result.HeroInfo.Status,
				stack:      result.HeroInfo.Stack,
				currentBet: result.HeroInfo.CurrentBet,
				heroCards:  result.HeroInfo.HeroCards,
				isHeroTurn: result.HeroInfo.IsHeroTurn,
			})
		}

		for _, villain := range result.VillainsInfo {
			if !isParticipatingStatus(villain.Status) {
				continue
			}
			allSeats = append(allSeats, seatInfo{
				seat:       villain.Seat,
				isHero:     false,
				status:     villain.Status,
				stack:      villain.Stack,
				currentBet: villain.CurrentBet,
			})
		}

		// 从按钮位开始，按顺时针顺序找到有玩家的座位。
		totalSeats := tableSeatCount(len(result.VillainsInfo))
		orderedSeats := make([]seatInfo, 0)

		for offset := 0; offset < totalSeats; offset++ {
			seat := ((result.TableInfo.ButtonSeat - 1 + offset) % totalSeats) + 1

			for _, si := range allSeats {
				if si.seat == seat {
					orderedSeats = append(orderedSeats, si)
					break
				}
			}
		}

		// 与 buildPlayers 保持一致：位置按实际参与玩家数，而非桌容量计算。
		positions := tablePositions(len(orderedSeats))

		// 先创建带位置信息的玩家列表
		type playerWithPos struct {
			seatInfo seatInfo
			pos      string
		}
		playerList := make([]playerWithPos, 0)

		for i, si := range orderedSeats {
			position := positions[i%len(positions)]
			var fullPos string
			if si.isHero {
				fullPos = position + "-Hero"
			} else {
				fullPos = position
			}
			playerList = append(playerList, playerWithPos{
				seatInfo: si,
				pos:      fullPos,
			})
		}

		// 确定行动开始位置
		var startIndex int
		if stage == "preflop" {
			// 有 straddle 时从其下一位开始；短桌没有 straddle 时从 BTN 开始。
			for i, p := range playerList {
				if p.pos == "straddle" || p.pos == "straddle-Hero" {
					startIndex = (i + 1) % len(playerList)
					break
				}
			}
		} else {
			// 其他阶段从SB开始
			for i, p := range playerList {
				if p.pos == "SB" || p.pos == "SB-Hero" {
					startIndex = i
					break
				}
			}
		}

		// 按行动顺序处理
		actionOrder := make([]int, 0)
		for i := 0; i < len(playerList); i++ {
			actionOrder = append(actionOrder, (startIndex+i)%len(playerList))
		}

		// 构建action history
		// 因为我们是从单个状态推断行动历史，amount直接表示该阶段的总下注
		prevBet := 0 // 前位的下注
		isFirst := true
		hasAmount := false
		for _, idx := range actionOrder {
			p := playerList[idx]
			si := p.seatInfo
			isActive := isActiveStatus(si.status)

			var action string
			var amount int

			// 判断action，逻辑和buildPlayers中的ActionTaken判断一致
			if !isActive {
				action = "fold"
				amount = 0
			} else if si.currentBet == 0 && !hasAmount {
				action = "check"
				amount = 0
			} else if si.currentBet == 0 && hasAmount {
				action = "call"
				amount = 0
			} else if isFirst || (prevBet == 0 && si.currentBet > 0) {
				action = "bet"
				amount = si.currentBet
				hasAmount = true
			} else if si.currentBet == prevBet && si.currentBet > 0 {
				action = "call"
				amount = si.currentBet
				hasAmount = true
			} else if si.currentBet >= 2*prevBet {
				action = "raise"
				amount = si.currentBet
				hasAmount = true
			} else {
				action = "call"
				amount = si.currentBet
				hasAmount = true
			}

			if stage == "preflop" && action == "bet" { // preflop下注时，实际是call
				action = "call"
			}
			// 添加到history
			history = append(history, TexasActionHistory{
				Stage:     stage,
				Position:  p.pos,
				Action:    action,
				Amount:    amount,
				Timestamp: timestamp,
				Status:    si.status,
			})
			timestamp++

			// 更新前位bet
			if si.currentBet > prevBet {
				prevBet = si.currentBet
			}
			isFirst = false
		}
	}

	return history
}
