package _const

import (
	"github.com/kingjxu/ddbaby/biz/model/ddbaby"
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
	CTypeClick           = 5
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
	"jf":  "减肥症状自测评估",
}

var JkType2Pic = map[string]string{
	"gxt": "https://lf3-static.bytednsdoc.com/obj/eden-cn/qeeh7upqbe/gxt.png",
	"cw":  "https://lf3-static.bytednsdoc.com/obj/eden-cn/qeeh7upqbe/cw.png",
	"jf":  "https://lf3-static.bytednsdoc.com/obj/eden-cn/qeeh7upqbe/jf.png",
}

var Seq2Amount = map[int32]int32{
	1: 1999,
	2: 999,
}

var Question2Options = map[string][]*ddbaby.JkQoItem{
	"gxt": gxt,
	"jf":  jf,
	"cw":  cw,
}
