package ark

import (
	"context"
	_const "github.com/kingjxu/ddbaby/const"
	"github.com/kingjxu/ddbaby/util"
	"github.com/sirupsen/logrus"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model/responses"
)

var systemPrompt = `# 角色
你是一位**专业的德州扑克大师**，精通德州扑克核心规则、GTO（游戏理论最优）策略及主流平台（德扑之星、WePoker、AAPoker）界面逻辑。你的核心目标是通过分析用户提供的牌局图片，精准提取关键要素，结合GTO理论生成**最优决策**，并以结构化JSON格式输出，你的德扑风格可以是紧凶型。


## 技能
### 技能1：牌局基础要素识别
- **盲注识别**：从图片中央区域识别小盲（SB）和大盲（BB）金额（格式如“1/2”“0.5/1”），确保对应数值准确无误。
- **公共牌分析**：若处于翻牌后/转牌后/河牌后阶段，精确识别公共牌的花色（♠♥♣♦）和点数（A-10），提取完整牌面信息。
- **玩家状态**：识别“我”（展示手牌的玩家）的头像、座位，以及其他玩家的有效筹码和下注金额。

### 技能2：多维度牌局要素提取
- **行动序列解析**：
  - 阶段划分：识别preflop（翻牌前）/flop（翻牌后）/turn（转牌后）/river（河牌后）；
  - 行动类型：解析每个玩家的行动，Fold（弃牌）/Check（过牌）/Bet（下注）/Call（跟注）/Raise（加注）；
- **池底与状态**：计算有效池底金额（Pot Size），以及每个玩家的剩余筹码

### 技能3：GTO最优决策生成
基于GTO策略，结合我的手牌强度、公共牌面、位置、行动历史生成决策：
- **Fold**：弃牌，牌力弱、没潜力放弃当前局，或者觉得对手牌力很强，放弃减少损失；
- **Check**：过牌，不下注，把行动权传给下一位，只有当前前位玩家没人下注时才能过牌，适用于牌力不强、想免费看牌时用；
- **Bet**：下注，当前圈还没人下注，你第一个投筹码；
- **Call**：跟注，前面有人下注，你付出同样筹码继续玩；
- **Raise**：加注，前面有人下注，增加下注金额，适用于牌力比较强，加注必须是前位玩家下注金额的至少2倍以上
## 输出规范
仅返回严格格式的JSON，字段定义：
{
  "hole_cards": "A♣Q♥", // 我（用户）的手牌，若图片无手牌显示则为"NULL"
  "community_cards": "K♣8♥5♠", // 公共牌，preflop时为"NULL"，flop后按顺序列出
  "stage": "preflop|flop|turn|river", // 当前牌局阶段
  "action": "Fold|Check|Bet|Call|Raise|NULL", // 决策结果，无人行动时为"NULL"
  "bet_size": 0 // 仅当action为Bet/Raise时为正整数，其他为0，单位与盲注一致
}

## 注意
1. **行动顺序**：严格按照阶段规则执行，若图片显示“我”的手牌不在当前行动序列中（如已弃牌，或者还没轮到我行动），直接返回""。
2. **GTO原则**：优先选择最小化对手优势，最大化自身优势的策略，避免情绪化决策。
4. **行动的原则**：Raise**必须至少**是前位行动玩家的2倍下注金额，如果前位玩家下注比较小，可适当raise大些


## 输出限制
- 仅输出上述规范JSON，**禁止任何额外文字说明、分析过程**。
- 如果输入多张图，只需要为最后一张图做出决策，前面的图是玩家最近的历史行动，你可以用它来做辅助分析
- **若“我”未参与当前行动（如已弃牌/过牌后对手已行动），或者没有 无“我”的手牌标识，或者还没轮到我行动**，返回 json中的action字段为 "NULL"。
- 若图片不是德州牌局，返回 json中的action字段为 "NULL"，stage字段也为"NULL"`

var systemMessage = &responses.ItemInputMessage{
	Role: responses.MessageRole_system,
	Content: []*responses.ContentItem{
		{
			Union: &responses.ContentItem_Text{
				Text: &responses.ContentItemText{
					Type: responses.ContentItemType_input_text,
					Text: systemPrompt,
				},
			},
		},
	},
}

var client *arkruntime.Client

func init() {
	client = arkruntime.NewClientWithApiKey(
		//通过 os.Getenv 从环境变量中获取 ARK_API_KEY
		_const.ArkApiKey,
		arkruntime.WithBaseUrl("https://ark.cn-beijing.volces.com/api/v3"),
	)
}

func TexasPokerDecision(ctx context.Context, imageURL string) string {
	inputMessage := &responses.ItemInputMessage{
		Role: responses.MessageRole_user,
		Content: []*responses.ContentItem{
			{
				Union: &responses.ContentItem_Image{
					Image: &responses.ContentItemImage{
						Type:     responses.ContentItemType_input_image,
						ImageUrl: util.Ptr(imageURL),
					},
				},
			},
		},
	}

	resp, err := client.CreateResponses(ctx, &responses.ResponsesRequest{
		Temperature: util.Ptr(0.5),
		TopP:        util.Ptr(1.0),
		Thinking:    &responses.ResponsesThinking{Type: util.Ptr(responses.ThinkingType_auto)},
		Model:       "ep-20251028100133-mj9v2",
		Input: &responses.ResponsesInput{
			Union: &responses.ResponsesInput_ListValue{
				ListValue: &responses.InputItemList{ListValue: []*responses.InputItem{
					{
						Union: &responses.InputItem_InputMessage{
							InputMessage: systemMessage,
						},
					},
					{
						Union: &responses.InputItem_InputMessage{
							InputMessage: inputMessage,
						},
					},
				},
				},
			},
		}})
	if err != nil {
		logrus.WithContext(ctx).Errorf("client.CreateResponses failed: err=%v", err)
		return ""
	}
	logrus.WithContext(ctx).Infof("response: %v", util.ToJSON(resp))
	return resp.GetOutput()[0].GetOutputMessage().GetContent()[0].GetText().GetText()
}
