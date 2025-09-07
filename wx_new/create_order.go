package wx_new

import (
	"context"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/h5"
	"time"
)

type CreateOrderParam struct {
	OutTradeNo string
	Amount     int
	NotifyURL  string
	Title      string
}

func CreateOrder(ctx context.Context, param CreateOrderParam) string {
	svc := h5.H5ApiService{Client: client}

	_, _, _ = svc.Prepay(ctx,
		h5.PrepayRequest{
			Appid:       core.String(appID), //灵运先知
			Mchid:       core.String(mchID),
			Description: core.String(param.Title),
			OutTradeNo:  core.String(param.OutTradeNo),
			TimeExpire:  core.Time(time.Now().Add(time.Hour * 1)),
			NotifyUrl:   core.String(param.NotifyURL),
			Amount: &h5.Amount{
				Currency: core.String("CNY"),
				Total:    core.Int64(int64(param.Amount)),
			},
		},
	)
	return "123"
}
