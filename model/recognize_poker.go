package model

import (
	"strconv"
	"strings"
)

type RecognizeRequest struct {
	ImageBase64 string `json:"image_base64"`
	App         string `json:"app"`
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
}
type TexasGtoDecisionReq struct {
	GameType              string               `json:"game_type"`
	NumPlayers            int                  `json:"num_players"`
	CurrentPot            int                  `json:"current_pot"`
	SbSize                int                  `json:"sb_size"`
	BbSize                int                  `json:"bb_size"`
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

func Conv2TexasGtoDecisionReq(recResult []*TexasResult) *TexasGtoDecisionReq {
	if len(recResult) == 0 {
		return nil
	}

	currentResult := recResult[len(recResult)-1]

	sbSize, bbSize := parseBlindSize(currentResult.TableInfo.BlindSize)

	players, currentPlayerPos := buildPlayers(currentResult)

	actionHistory := buildActionHistory(recResult)

	return &TexasGtoDecisionReq{
		GameType:              "no_limit_holdem",
		NumPlayers:            len(players),
		CurrentPot:            currentResult.TableInfo.MainPot,
		SbSize:                sbSize,
		BbSize:                bbSize,
		CommunityCards:        parseCards(currentResult.TableInfo.CommunityCards),
		GameStage:             currentResult.TableInfo.Stage,
		Players:               players,
		CurrentPlayerPosition: currentPlayerPos,
		ActionHistory:         actionHistory,
	}
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

func buildPlayers(result *TexasResult) ([]TexasPlayer, string) {
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
	allSeats = append(allSeats, seatInfo{
		seat:       result.HeroInfo.Seat,
		isHero:     true,
		status:     result.HeroInfo.Status,
		stack:      result.HeroInfo.Stack,
		currentBet: result.HeroInfo.CurrentBet,
		heroCards:  result.HeroInfo.HeroCards,
		isHeroTurn: result.HeroInfo.IsHeroTurn,
	})

	for _, villain := range result.VillainsInfo {
		if villain.Status == "empty" || villain.Status == "waiting" {
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

	// 从按钮位开始，按顺时针顺序找到有玩家的座位
	const totalSeats = 8
	orderedSeats := make([]seatInfo, 0)

	for offset := 0; offset < totalSeats; offset++ {
		seat := result.TableInfo.ButtonSeat + offset
		if seat > totalSeats {
			seat = seat - totalSeats
		}

		for _, si := range allSeats {
			if si.seat == seat {
				orderedSeats = append(orderedSeats, si)
				break
			}
		}
	}

	positions := []string{"BTN", "SB", "BB", "UTG", "UTG+1", "MP", "MP+1", "CO"}
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
				IsActive:  si.status == "active" || si.status == "allin",
			}
			currentPlayerPos = position + "-Hero"
		} else {
			player = TexasPlayer{
				Position:    position,
				HoleCards:   []string{"X-X", "X-X"},
				Stack:       si.stack,
				Bet:         si.currentBet,
				IsActive:    si.status == "active" || si.status == "allin",
				ActionTaken: "",
			}
		}
		playerList = append(playerList, playerWithInfo{player, si, position})
	}

	// 确定行动开始位置
	var startIndex int
	if result.TableInfo.Stage == "preflop" {
		// preflop从BB+1开始
		for i, pi := range playerList {
			if pi.pos == "BB" {
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
	heroProcessed := false

	for _, idx := range actionOrder {
		pi := &playerList[idx]

		// 如果已经处理过hero，就停止
		if heroProcessed {
			break
		}

		// 检查是否是hero
		if pi.info.isHero {
			heroProcessed = true
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
		} else if pi.player.Bet >= 2*prevBet {
			pi.player.ActionTaken = "raise"
		} else {
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

// recResult 按照 preflop, flop, turn, river 顺序排
func buildActionHistory(recResult []*TexasResult) []TexasActionHistory {
	history := make([]TexasActionHistory, 0)
	timestamp := int(0)

	for _, result := range recResult {
		stage := result.TableInfo.Stage

		action := TexasActionHistory{
			Stage:     stage,
			Position:  "BTN-Hero",
			Action:    "call",
			Amount:    result.HeroInfo.CurrentBet,
			Timestamp: timestamp,
		}
		history = append(history, action)
		timestamp++
	}

	return history
}
