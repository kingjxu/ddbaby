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

	BaiduUploadUrl = "https://ocpc.baidu.com/ocpcapi/api/uploadConvertData"

	ProfessorUrl = "https://work.weixin.qq.com/ca/cawcde2a866722fe78"
)

const (
	CTypeSubmit          = 3
	CTypePurchaseService = 10
	CTypeAddWechat       = 79
)

const (
	JKWXPayNotifyUrl = "https://ddbaby.site/jk/pay_callback" // 支付回调地址
)

var JkType2Title = map[string]string{
	"gxt": "高血糖症状自测评估",
	"gxy": "高血压症状自测评估",
	"cw":  "肠胃症状自测评估",
}

var JkType2Pic = map[string]string{
	"gxt": "https://lf3-static.bytednsdoc.com/obj/eden-cn/qeeh7upqbe/gxt.png",
	"gxy": "高血压症状自测评估",
	"cw":  "https://lf3-static.bytednsdoc.com/obj/eden-cn/qeeh7upqbe/cw.png",
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
			Question: util.Ptr("是否经常感到的腹痛、腹泻、腹胀"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(4)),
			Question: util.Ptr("是否在吃油腻食物后容易感到腹胀或腹泻"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(5)),
			Question: util.Ptr("是否有便秘的问题"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(6)),
			Question: util.Ptr("是否经常有 “想上厕所却排不出来” 的排便不尽感"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(7)),
			Question: util.Ptr("是否时常觉得恶心，甚至伴有呕吐"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(8)),
			Question: util.Ptr("是否偶尔会因为空腹吃刺激性食物（如咖啡、浓茶）感到胃痛"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(9)),
			Question: util.Ptr("是否经常食欲不振，吃什么都没胃口"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(10)),
			Question: util.Ptr("是否经常在饭后半小时内出现反酸或烧心的感觉"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(11)),
			Question: util.Ptr("是否在排便时发现大便颜色偏黑（排除食物影响）"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(12)),
			Question: util.Ptr("是否吃完东西后感到胃痛、胃胀"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(13)),
			Question: util.Ptr("是否偶尔会出现 “胃里往上返气”（嗳气）的现象"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(14)),
			Question: util.Ptr("是否偶尔会出现 “肚子叫”（肠鸣）且声音较大、持续时间长"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(15)),
			Question: util.Ptr("是否有排便不规律的问题，或排便时有腹痛的感觉"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(16)),
			Question: util.Ptr("是否有大便带血的现象"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(17)),
			Question: util.Ptr("是否偶尔或经常出现大便稀溏、不成形的情况？"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(18)),
			Question: util.Ptr("是否经常觉得口臭，特别是早晨起床的时候"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(19)),
			Question: util.Ptr("是否经常吃辛辣等重口味食物"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(20)),
			Question: util.Ptr("是否经常因为吃多了或吃快了而感到胃部胀"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(21)),
			Question: util.Ptr("是否偶尔会因为喝水太急或喝冰水导致胃部痉挛"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(22)),
			Question: util.Ptr("是否经常在夜间被腹痛或腹胀弄醒"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(23)),
			Question: util.Ptr("是否经常因为肠胃不适导致精神状态差、容易疲劳"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(24)),
			Question: util.Ptr("是否偶尔会出现口腔里有 “酸味” 或 “苦味”"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(25)),
			Question: util.Ptr("是否在肠胃不适时还伴有轻微的头痛或头晕"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(26)),
			Question: util.Ptr("是否经常因为肠胃问题影响睡眠质量（如入睡难、易醒）"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(27)),
			Question: util.Ptr("是否经常在排便后仍觉得腹部有坠胀感"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(28)),
			Question: util.Ptr("是否在肠胃不适时还伴有轻微的头痛或头晕"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(29)),
			Question: util.Ptr("是否在吃生冷食物（如冰饮、生鱼片）后感到肠胃不适"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(30)),
			Question: util.Ptr("是否在外出就餐（尤其是陌生餐厅）后容易腹泻"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(31)),
			Question: util.Ptr("是否经常因为担心肠胃不适而刻意控制食量"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(32)),
			Question: util.Ptr("是否在肠胃不适时还伴有食欲不振和体重轻微下降"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(33)),
			Question: util.Ptr("是否经常在排便后仍觉得腹部有坠胀感"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(34)),
			Question: util.Ptr("是否在空腹时感到胃部隐隐作痛或不适"),
			Options:  []string{"没有", "有时", "经常"},
		},
		{
			ID:       util.Ptr(int64(35)),
			Question: util.Ptr("是否经常需要通过吃胃药、益生菌等缓解肠胃症状"),
			Options:  []string{"没有", "有时", "经常"},
		},
	},
}
