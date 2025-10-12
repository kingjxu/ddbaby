package _const

import (
	"github.com/kingjxu/ddbaby/biz/model/ddbaby"
	"github.com/kingjxu/ddbaby/util"
)

const (
	CozeToken         = "pat_0ZCXIT2NguHJWbGYvr5SsU85onSnjIZlfZkkjw3Kl21SI4IpSkLsJr5pXYq30pC2"
	DreamExplainBotID = "7391123436175343616"
	NameFortuneBotID  = "7392945320009400347"
	PickNameBotID     = "7392946548130955275"
	TaLuoPredictBotID = "7401724843219419187"
)

const (
	JKWXPayNotifyUrl = "https://ddbaby.site/jk/pay_callback" // 支付回调地址
)

var JkType2Title = map[string]string{
	"gxt": "高血糖症状自测评估",
	"gxy": "高血压症状自测评估",
	"cw":  "肠胃症状自测评估",
}

var Seq2Amount = map[int32]int32{
	1: 2,
	2: 1,
}

var Question2Options = map[string][]*ddbaby.JkQoItem{
	"gxt": {
		{
			ID:       util.Ptr(int64(1)),
			Question: util.Ptr("你的性别"),
			Options:  []string{"男", "女"},
		},
		{
			ID:       util.Ptr(int64(2)),
			Question: util.Ptr("你的年龄"),
			Options:  []string{"小于45岁", "45-65岁", "65岁以上"},
		},
		{
			ID:       util.Ptr(int64(3)),
			Question: util.Ptr("是否经常感到的口渴想喝水"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(4)),
			Question: util.Ptr("是否有尿量增加的的症状，有时候频繁至每小时一次"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(5)),
			Question: util.Ptr("是否尿尿有泡沫"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(6)),
			Question: util.Ptr("是否有体重缓慢减轻的症状，且无明显的诱因"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(7)),
			Question: util.Ptr("是否经常出汗，哪怕天气不热"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(8)),
			Question: util.Ptr("是否时常觉得恶心，甚至伴有呕吐"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(9)),
			Question: util.Ptr("是否经常感到疲劳乏力，没有精神"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(10)),
			Question: util.Ptr("是否感觉视力下降，看东西越来越模糊"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(11)),
			Question: util.Ptr("是否有头晕的症状，特别是早上起床的时候"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(12)),
			Question: util.Ptr("是否饥饿感频繁出现，食量显著增加"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(13)),
			Question: util.Ptr("是否感到皮肤干燥发痒"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(14)),
			Question: util.Ptr("是否感到胸闷、胸痛，有时心率过快，特别是活动后"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(15)),
			Question: util.Ptr("是否当前正在进行降糖治疗"),
			Options:  []string{"没有", "有时", "经常"},
		},
	},

	"cw": {
		{
			ID:       util.Ptr(int64(1)),
			Question: util.Ptr("你的性别"),
			Options:  []string{"男", "女"},
		},
		{
			ID:       util.Ptr(int64(2)),
			Question: util.Ptr("你的年龄"),
			Options:  []string{"小于45岁", "45-65岁", "65岁以上"},
		},
		{
			ID:       util.Ptr(int64(3)),
			Question: util.Ptr("是否经常感到腹痛、腹泻、腹胀"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(4)),
			Question: util.Ptr("是否有便秘的问题"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(5)),
			Question: util.Ptr("是否时常觉得恶心，甚至伴有呕吐"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(6)),
			Question: util.Ptr("是否经常出现反酸的现象"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(7)),
			Question: util.Ptr("是否经常食欲不振，吃什么都没胃口"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(8)),
			Question: util.Ptr("是否有烧心的感觉"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(9)),
			Question: util.Ptr("是否吃完东西后感到胃痛、胃胀"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(10)),
			Question: util.Ptr("是否有肠鸣的现象"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(11)),
			Question: util.Ptr("是否有排便不规律的问题，或排便时有腹痛的感觉"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(12)),
			Question: util.Ptr("是否有大便带血的现象"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(13)),
			Question: util.Ptr("是否经常觉得口臭，特别是早晨起床的时候"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(14)),
			Question: util.Ptr("是否经常吃辛辣等重口味食物"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(15)),
			Question: util.Ptr("是否正在接受肠胃治疗方案"),
			Options:  []string{"没有", "有时", "经常"},
		},
	},
}
