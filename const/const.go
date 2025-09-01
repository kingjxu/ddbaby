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

var JkType2Title = map[string]string{
	"gxt": "高血糖症状自测评估",
	"gxy": "高血压症状自测评估",
	"cw":  "肠胃症状自测评估",
}

var Question2Options = map[string][]*ddbaby.JkQoItem{
	"gxt": {
		{

			Question: util.Ptr("你的年龄"),
			Options:  []string{"是", "否"},
		},
		{
			Question: util.Ptr("1. 你是否经常感到头晕、头痛、恶心、呕吐、腹泻等症状？"),
			Options:  []string{"是", "否"},
		},
	},
}
