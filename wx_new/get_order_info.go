package wx_new

import (
	"context"
	"github.com/kingjxu/ddbaby/util"
	"github.com/sirupsen/logrus"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/h5"
)

func IsOrderPaySuccess(ctx context.Context, outTradNo string) bool {
	svc := h5.H5ApiService{Client: client}
	req := h5.QueryOrderByOutTradeNoRequest{
		OutTradeNo: util.Ptr(outTradNo),
		Mchid:      util.Ptr(mchID),
	}
	trans, _, err := svc.QueryOrderByOutTradeNo(ctx, req)
	if err != nil || trans == nil {
		logrus.WithContext(ctx).Errorf("GetOrderInfo failed, req:%v,err:%+v", util.ToJSON(req), err)
		return false
	}
	logrus.WithContext(ctx).Infof("GetOrderInfo,req:%v, resp:%v", util.ToJSON(req), util.ToJSON(trans))
	return *trans.TradeState == "SUCCESS"
}
