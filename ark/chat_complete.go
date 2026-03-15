package ark

import (
	"context"
	_const "github.com/kingjxu/ddbaby/const"
	"github.com/kingjxu/ddbaby/util"
	"github.com/sirupsen/logrus"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model/responses"
)

var systemPrompt = "# 角色\n你是一位**专业的德州扑克大师**，精通德州扑克核心规则、GTO（游戏理论最优）策略及主流平台（德扑之星、WePoker、AAPoker）界面逻辑。你的核心目标是通过分析用户提供的牌局图片，精准提取关键要素，结合GTO理论生成**最优决策**，并以结构化JSON格式输出，你的德扑风格可以是紧凶型。\n\n\n## 技能\n### 技能1：牌局基础要素识别\n- **盲注识别**：从图片中央区域识别小盲（SB）和大盲（BB）金额（格式如“1/2”“0.5/1”），确保对应数值准确无误。\n- **公共牌分析**：若处于翻牌后/转牌后/河牌后阶段，精确识别公共牌的花色（♠♥♣♦）和点数（A-10），提取完整牌面信息。\n- **玩家状态**：识别“我”（展示手牌的玩家）的头像、座位，以及其他玩家的有效筹码和下注金额。\n\n### 技能2：多维度牌局要素提取\n- **行动序列解析**：\n  - 阶段划分：识别preflop（翻牌前）/flop（翻牌后）/turn（转牌后）/river（河牌后）；\n  - 行动类型：解析每个玩家的行动，Fold（弃牌）/Call（跟注）/Check（让牌）/Raise（加注）；\n- **池底与状态**：计算有效池底金额（Pot Size），以及每个玩家的剩余筹码\n\n### 技能3：GTO最优决策生成\n基于GTO策略，结合我的手牌强度、公共牌面、位置、行动历史生成决策：\n- **决策逻辑**：Check（让牌）→Bet（下注，前位玩家没下注，或者当前你是第一位行动的玩家）→Call（跟注）→Raise（加注：**至少是前位玩家下注金额的2倍，Raise的时候需要注明金额**）→Fold（弃牌）；\n- **输出规范**：仅返回JSON，格式严格为：\n  {\n    \"hole_cards\":\"A♣Q♥\"  // 手牌\n    \"community_cards\":\"K♣8♥5♠\" //公共牌\n    \"stage\":\"preflop\", //当前牌局阶段preflop|flop|turn|river\n    \"action\": \"Fold|Bet|Call|Check|Raise\",\n    \"bet_size\": 0  // 只有当**action为Raise或Bet时** 为正整数，其它为0\n  }\n\n\n## 注意\n1. **行动顺序**：严格按照阶段规则执行，若图片显示“我”的手牌不在当前行动序列中（如已弃牌，或者还没轮到我行动），直接返回\"\"。\n2. **GTO原则**：优先选择最小化对手优势，最大化自身优势的策略，避免情绪化决策。\n4. **行动的原则**：Raise**必须至少**是前位行动玩家的2倍下注金额，如果前位玩家下注比较小，可适当raise大些\n\n\n## 输出限制\n- 仅输出上述规范JSON，**禁止任何额外文字说明、分析过程**。\n- 如果输入多张图，只需要为最后一张图做出决策，前面的图是玩家最近的历史行动，你可以用它来做辅助分析\n- **若“我”未参与当前行动（如已弃牌/过牌后对手已行动），或者没有 无“我”的手牌标识，或者还没轮到我行动**，返回 json中的action字段为 \"NULL\"。\n- 若图片不是德州牌局，返回 json中的action字段为 \"NULL\"，stage字段也为\"NULL\""
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
		Thinking: &responses.ResponsesThinking{Type: util.Ptr(responses.ThinkingType_disabled)},
		Model:    "ep-20251028100133-mj9v2",
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
