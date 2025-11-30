package _const

import (
	"github.com/kingjxu/ddbaby/biz/model/ddbaby"
	"github.com/kingjxu/ddbaby/util"
)

var jf = []*ddbaby.JkQoItem{
	{
		ID:       util.Ptr(int64(1)),
		Question: util.Ptr("你的性别"),
		Options:  []string{"男", "女"},
	},
	{
		ID:       util.Ptr(int64(2)),
		Question: util.Ptr("你的年龄"),
		Options:  []string{"小于35岁", "35-55岁", "55岁以上"},
	},
	{
		ID:       util.Ptr(int64(3)),
		Question: util.Ptr("你对自己当前体重的感受是"),
		Options:  []string{"满意，无减重需求", "略重，想微调 1-3kg", "明显超重，需减 4-8kg", "严重肥胖，需减 8kg 以上"},
	},
	{
		ID:       util.Ptr(int64(4)),
		Question: util.Ptr("你穿裤子时的感受更接近哪种"),
		Options:  []string{"合身不紧绷", "腰部略紧", "明显紧绷，久坐后不适", "完全穿不上合身尺码"},
	},
	{
		ID:       util.Ptr(int64(5)),
		Question: util.Ptr("过去 1 年你的体重变化趋势"),
		Options:  []string{"稳定（波动≤3kg）", "持续上升", "反复波动（升降超 5kg）", "持续下降"},
	},
	{
		ID:       util.Ptr(int64(6)),
		Question: util.Ptr("家族中是否有肥胖、糖尿病、高血压等代谢相关疾病史"),
		Options:  []string{"无", "1 位亲属有", "2 位及以上亲属有"},
	},
	{
		ID:       util.Ptr(int64(7)),
		Question: util.Ptr("你是否规律吃三餐"),
		Options:  []string{"规律（三餐定时定量）", "偶尔不吃早餐", "经常不吃某一餐", "三餐完全不规律"},
	},
	{
		ID:       util.Ptr(int64(8)),
		Question: util.Ptr("你每天摄入高糖饮料（奶茶、可乐等）或甜点的频率"),
		Options:  []string{"几乎不", "1-2 次 / 周", "3-5 次 / 周", "每天 1 次及以上"},
	},
	{
		ID:       util.Ptr(int64(9)),
		Question: util.Ptr("你对油炸、肥肉等高脂食物的偏好"),
		Options:  []string{"很少吃", "偶尔吃（1-2 次 / 周）", "经常吃（3-5 次 / 周）", "每天都吃"},
	},
	{
		ID:       util.Ptr(int64(10)),
		Question: util.Ptr("你的进食速度如何"),
		Options:  []string{"较慢（每口咀嚼 20 次以上，用餐≥20 分钟）", "中等（用餐 15-20 分钟）", "较快（用餐 10-15 分钟）", "很快（用餐＜10 分钟）"},
	},
	{
		ID:       util.Ptr(int64(11)),
		Question: util.Ptr("你是否有吃宵夜的习惯"),
		Options:  []string{"几乎不吃", "1-2 次 / 周", "3-5 次 / 周", "每天都吃"},
	},
	{
		ID:       util.Ptr(int64(12)),
		Question: util.Ptr("压力大或情绪低落时，你是否会大量进食"),
		Options:  []string{"几乎不会", "偶尔会", "经常会", "每次情绪波动都这样"},
	},
	{
		ID:       util.Ptr(int64(13)),
		Question: util.Ptr("你每天吃蔬菜水果的情况更接近哪种"),
		Options:  []string{"每餐都有足量蔬菜，每天吃 1-2 种水果", "每天吃蔬菜但量不多，偶尔吃水果", "很少吃蔬菜，几乎不吃水果", "几乎不吃蔬菜水果"},
	},
	{
		ID:       util.Ptr(int64(14)),
		Question: util.Ptr("你的工作 / 日常以哪种活动强度为主"),
		Options:  []string{"重度体力活动（如建筑、搬运）", "轻度体力活动（如导购、保洁）", "久坐办公（每天坐姿≥6 小时）", "几乎不活动（多躺坐）"},
	},
	{
		ID:       util.Ptr(int64(15)),
		Question: util.Ptr("每周进行中等强度运动（快走、慢跑等）的累计时间"),
		Options:  []string{">=300 分钟", "150-299 分钟", "1-149 分钟", "＜300 分钟"},
	},
	{
		ID:       util.Ptr(int64(16)),
		Question: util.Ptr("你是否有力量训练（举铁、俯卧撑等）习惯"),
		Options:  []string{"每周 3 次及以上", "每周 1-2 次", "偶尔尝试", "从不做"},
	},
	{
		ID:       util.Ptr(int64(17)),
		Question: util.Ptr("每天主动步行的步数大概是"),
		Options:  []string{">=8000 步", "5000-7999 步", "3000-4999 步", "＜3000 步"},
	},
	{
		ID:       util.Ptr(int64(18)),
		Question: util.Ptr("你是否因体型顾虑拒绝去健身房或公开场合运动"),
		Options:  []string{"完全不会", "偶尔会犹豫", "经常会回避", "完全不敢"},
	},
	{
		ID:       util.Ptr(int64(19)),
		Question: util.Ptr("运动后你是否会额外增加饮食量"),
		Options:  []string{"不会，正常饮食", "偶尔会多吃一点", "经常刻意多吃", "每次运动后都大量进食"},
	},
	{
		ID:       util.Ptr(int64(20)),
		Question: util.Ptr("你每天的睡眠时间大概是"),
		Options:  []string{">=7.5 小时", "6-7.4 小时", "4.5-5.9 小时", "＜4.5 小时"},
	},
	{
		ID:       util.Ptr(int64(21)),
		Question: util.Ptr("你是否有熬夜习惯（凌晨 12 点后入睡）"),
		Options:  []string{"几乎不", "1-2 次 / 周", "3-5 次 / 周", "每天都熬夜"},
	},
	{
		ID:       util.Ptr(int64(22)),
		Question: util.Ptr("你是否吸烟"),
		Options:  []string{"从不", "已戒烟", "偶尔吸（＜1 次 / 周）", "经常吸（≥1 次 / 周）"},
	},
	{
		ID:       util.Ptr(int64(23)),
		Question: util.Ptr("你是否饮酒"),
		Options:  []string{"从不", "偶尔饮（＜1 次 / 周）", "经常饮（≥1 次 / 周）", "每天都饮"},
	},
	{
		ID:       util.Ptr(int64(24)),
		Question: util.Ptr("你减重的主要动力是什么"),
		Options:  []string{"改善健康（如控制血糖、血脂）", "提升体能，让身体更轻松", "迎合审美或他人期待", "盲目跟风，无明确目标"},
	},
	{
		ID:       util.Ptr(int64(25)),
		Question: util.Ptr("过去一个月，你是否因体重问题感到焦虑、低落或自我否定"),
		Options:  []string{"几乎没有", "1-2 次", "3-5 次", "几乎每天"},
	},
	{
		ID:       util.Ptr(int64(26)),
		Question: util.Ptr("你是否会因一次破例吃高热量食物，就放弃整个减重计划"),
		Options:  []string{"完全不会", "偶尔会动摇", "经常会放弃", "每次破例都放弃"},
	},
	{
		ID:       util.Ptr(int64(27)),
		Question: util.Ptr("你对减重效果的期待是"),
		Options:  []string{"每月减 3-5% 体重，循序渐进", "每月减 5-10% 体重", "快速减重，不在乎方式", "没有明确期待"},
	},
	{
		ID:       util.Ptr(int64(28)),
		Question: util.Ptr("你是否被医生诊断过以下疾病？"),
		Options:  []string{"无任何相关疾病", "脂肪肝、血脂异常", "高血压、糖尿病前期", "2 型糖尿病、睡眠呼吸暂停综合征等"},
	},
	{
		ID:       util.Ptr(int64(29)),
		Question: util.Ptr("你是否有以下不适症状？"),
		Options:  []string{"无", "偶尔关节疼痛、活动后心慌", "经常打鼾严重、白天嗜睡", "频繁气短、皮肤褶皱处感染"},
	},
	{
		ID:       util.Ptr(int64(30)),
		Question: util.Ptr("你是否经常在睡前 2 小时内吃大量食物？（原第二十九题调整，无需计算热量）"),
		Options:  []string{"几乎不", "偶尔会（1-2 次 / 月）", "经常会（1-2 次 / 周）", "每天都这样"},
	},
	{
		ID:       util.Ptr(int64(31)),
		Question: util.Ptr("你是否能长期坚持调整饮食和运动习惯？"),
		Options:  []string{"完全可以，有明确规划", "大概率可以，需要偶尔调整", "很难坚持，容易半途而废", "完全无法坚持"},
	},
}
