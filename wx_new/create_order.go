package wx_new

import (
	"context"
	"github.com/kingjxu/ddbaby/util"
	"github.com/sirupsen/logrus"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/h5"
	"time"
)

type CreateOrderParam struct {
	OutTradeNo string
	Amount     int
	NotifyURL  string
	Title      string
	ClientIP   string
}

func Prepay(ctx context.Context, param CreateOrderParam) (string, error) {
	svc := h5.H5ApiService{Client: client}

	payReq := h5.PrepayRequest{
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
		SceneInfo: &h5.SceneInfo{
			PayerClientIp: util.Ptr(param.ClientIP),
			H5Info: &h5.H5Info{
				Type: util.Ptr("Android"),
			},
		},
	}
	resp, _, err := svc.Prepay(ctx, payReq)
	logrus.WithContext(ctx).Infof("[CreateOrder] svc.Prepay  req:%v", util.MarshalString())
	if err != nil {
		logrus.WithContext(ctx).Errorf("[CreateOrder] svc.Prepay  err:%v", err)
		return "", err
	}
	jumpUrl := ""
	if resp.H5Url != nil {
		jumpUrl = *resp.H5Url
	}
	return jumpUrl, nil
}
