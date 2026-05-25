package model

import (
	"context"
	"github.com/sirupsen/logrus"
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
	Status    string
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

func Conv2TexasGtoDecisionReq(ctx context.Context, recResult []*TexasResult) *TexasGtoDecisionReq {
	if len(recResult) == 0 {
		return nil
	}

	currentResult := recResult[len(recResult)-1]

	sbSize, bbSize := parseBlindSize(currentResult.TableInfo.BlindSize)

	players, currentPlayerPos := buildPlayers(currentResult)

	actionHistory := buildActionHistory(ctx, recResult)
	actionHistory = reviseActionHistory(actionHistory)

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
	for _, idx := range actionOrder {
		pi := &playerList[idx]

		// 检查是否是hero
		if pi.info.isHero {
			continue
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

func reviseActionHistory(history []TexasActionHistory) []TexasActionHistory {
	return nil
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
//	该函数从第0个到第len(recResult)-2个结果中解析数据，最后一个结果是当前阶段的数据，不用解析。
//	对于每个stage，函数会取该stage的最后一个结果，像buildPlayers那样从中推断出该阶段的行动历史。
func buildActionHistory(ctx context.Context, recResult []*TexasResult) []TexasActionHistory {
	logrus.WithContext(ctx).Infof("[buildActionHistory] len(recResult):%v", len(recResult))
	if len(recResult) < 2 {
		return make([]TexasActionHistory, 0)
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
			// preflop从BB+1开始
			for i, p := range playerList {
				if p.pos == "BB" || p.pos == "BB-Hero" {
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

		for _, idx := range actionOrder {
			p := playerList[idx]
			si := p.seatInfo
			isActive := si.status == "active" || si.status == "allin"

			var action string
			var amount int

			// 判断action，逻辑和buildPlayers中的ActionTaken判断一致
			if !isActive {
				action = "fold"
				amount = 0
			} else if si.currentBet == 0 {
				action = "check"
				amount = 0
			} else if isFirst || (prevBet == 0 && si.currentBet > 0) {
				action = "bet"
				amount = si.currentBet
			} else if si.currentBet == prevBet && si.currentBet > 0 {
				action = "call"
				amount = si.currentBet
			} else if si.currentBet >= 2*prevBet {
				action = "raise"
				amount = si.currentBet
			} else {
				action = "call"
				amount = si.currentBet
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
