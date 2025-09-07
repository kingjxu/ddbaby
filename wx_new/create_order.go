package wx_new

import (
	"context"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/h5"
	"time"
)

func CreateOrder(ctx context.Context) string {
	svc := h5.H5ApiService{Client: client}
	_, _, _ = svc.Prepay(ctx,
		h5.PrepayRequest{
			Appid:       core.String(appID), //灵运先知
			Mchid:       core.String(mchID),
			Description: core.String("Image形象店-深圳腾大-QQ公仔"),
			OutTradeNo:  core.String("1217752501201407033233368018"),
			TimeExpire:  core.Time(time.Now().Add(30 * time.Minute)),
			Attach:      core.String("自定义数据说明"),
		},
	)

	_, _, _ = svc.Prepay(ctx,
		h5.PrepayRequest{
			Appid:         core.String(appID), //灵运先知
			Mchid:         core.String(mchID),
			Description:   core.String("Image形象店-深圳腾大-QQ公仔"),
			OutTradeNo:    core.String("1217752501201407033233368018"),
			TimeExpire:    core.Time(time.Now()),
			Attach:        core.String("自定义数据说明"),
			NotifyUrl:     core.String("https://www.weixin.qq.com/wxpay/pay.php"),
			GoodsTag:      core.String("WXG"),
			LimitPay:      []string{"LimitPay_example"},
			SupportFapiao: core.Bool(false),
			Amount: &h5.Amount{
				Currency: core.String("CNY"),
				Total:    core.Int64(100),
			},
			Detail: &h5.Detail{
				CostPrice: core.Int64(608800),
				GoodsDetail: []h5.GoodsDetail{h5.GoodsDetail{
					GoodsName:        core.String("iPhoneX 256G"),
					MerchantGoodsId:  core.String("ABC"),
					Quantity:         core.Int64(1),
					UnitPrice:        core.Int64(828800),
					WechatpayGoodsId: core.String("1001"),
				}},
				InvoiceId: core.String("wx123"),
			},
			SceneInfo: &h5.SceneInfo{
				DeviceId: core.String("013467007045764"),
				H5Info: &h5.H5Info{
					AppName:     core.String("王者荣耀"),
					AppUrl:      core.String("https://pay.qq.com"),
					BundleId:    core.String("com.tencent.wzryiOS"),
					PackageName: core.String("com.tencent.tmgp.sgame"),
					Type:        core.String("iOS"),
				},
				PayerClientIp: core.String("14.23.150.211"),
				StoreInfo: &h5.StoreInfo{
					Address:  core.String("广东省深圳市南山区科技中一道10000号"),
					AreaCode: core.String("440305"),
					Id:       core.String("0001"),
					Name:     core.String("腾讯大厦分店"),
				},
			},
			SettleInfo: &h5.SettleInfo{
				ProfitSharing: core.Bool(false),
			},
		},
	)
	return "123"
}
