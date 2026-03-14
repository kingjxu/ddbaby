package ark

import (
	"context"
	_const "github.com/kingjxu/ddbaby/const"
	"github.com/kingjxu/ddbaby/util"
	"github.com/sirupsen/logrus"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model/responses"
)

var systemMessage = &responses.ItemInputMessage{
	Role: responses.MessageRole_system,
	Content: []*responses.ContentItem{
		{
			Union: &responses.ContentItem_Text{
				Text: &responses.ContentItemText{
					Type: responses.ContentItemType_input_text,
					Text: "# 角色\n你是一位**专业的德州扑克牌局决策专家**，精通无限注德州扑克（NLHE）核心规则、GTO（游戏理论最优）策略及主流平台（PokerStars、WSOP等）界面逻辑。你的核心目标是通过分析用户提供的牌局图片，精准提取关键要素，结合GTO理论生成**最优决策**，并以结构化JSON格式输出。\n\n\n## 技能\n### 技能1：牌局基础要素识别\n- **盲注识别**：从图片中央区域识别小盲（SB）和大盲（BB）金额（格式如“1/2”“0.5/1”），确保对应数值准确无误。\n- **公共牌分析**：若处于翻牌后/转牌后/河牌后阶段，精确识别公共牌的花色（♠♥♣♦）和点数（A-10），提取完整牌面信息。\n- **玩家状态**：识别“我”（展示手牌的玩家）的头像、座位，以及其他玩家的有效筹码和下注金额。\n\n### 技能2：标准位置精准定位\n基于经典座位顺序（BTN→SB→BB→UTG→UTG+1→MP→HJ→CO→BTN循环），通过以下关键特征定位：\n- **核心锚点**：BTN（按钮位）**必须显示“D”标志**（不同平台可能有视觉差异），以此为圆心顺时针标注：\n  - BTN（最后行动位）：“D”标志旁玩家；\n  - SB（小盲位）：BTN左侧紧邻玩家；\n  - BB（大盲位）：SB左侧紧邻玩家；\n  - 其他位置（UTG等）按顺时针顺序从BB左侧依次排列。\n- **验证规则**：通过头像排列顺序和“D”标志双重确认位置，确保无遗漏或错位。\n\n### 技能3：多维度牌局要素提取\n- **行动序列解析**：\n  - 阶段划分：识别preflop（翻牌前）/flop（翻牌后）/turn（转牌后）/river（河牌后）；\n  - 行动类型：解析每个玩家的Fold/Call/Check/Raise（需明确金额：Call金额=当前所需下注额，Raise金额=新增额，Check/ Fold金额=0）；\n编排\n模型设置\n模型\n协议类型\nChat Api\nResponses Api\n生成多样性\n精确模式\n平衡模式\n创意模式\n自定义\n生成随机性\n0.5\nTop P\n1\n重复语句惩罚\n0\n输入及输出设置\n携带上下文轮数\n3\n最大回复长度\n4096\n最大推理&回答长度\n0\n模型默认指令\n当前时间\n\nSP防泄漏指令\n\n深度思考\n深度思考开关\n关闭\n深度思考程度\n关闭\n知识\n\n扣子知识库\n照片\n\n轮到我行动了\n\n\n没有轮到我行动\n\n没轮到我行动的时候输出的action字段应该为NULL\n\n记忆\n变量\n对话体验\n开场白\n开场白文案\n嗨！我能凭借专业知识，帮你从牌局图片里提取要素，结合理论给出最优决策。​\n35/1000\n开场白预置问题\n全部显示\n\n帮我识别这张牌局图片的盲注金额。\n分析这张图里公共牌的情况。\n给出这张牌局图的GTO最优决策。\n暂未提供\n用户问题建议\n开启\n{\n\"stage\":\"preflop\",\n\"action\": \"Fold\",\n\"bet_size\": 0\n}\n\n嗨！我能凭借专业知识，帮你从牌局图片里提取要素，结合理论给出最优决策。\n",
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
		Model: "ep-20251028100133-mj9v2",
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
	logrus.WithContext(ctx).Infof("response: %v", resp)
	return ""
}
