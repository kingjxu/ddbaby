package _const

import (
	"github.com/kingjxu/ddbaby/biz/model/ddbaby"
	"github.com/kingjxu/ddbaby/util"
)

var gxt = []*ddbaby.JkQoItem{
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
}
