package wx_new

import (
	"context"
	"github.com/kingjxu/ddbaby/util"
	"github.com/sirupsen/logrus"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/h5"
)

func IsOrderPaySuccess(ctx context.Context, transactionID string) bool {
	svc := h5.H5ApiService{Client: client}
	trans, _, err := svc.QueryOrderById(ctx, h5.QueryOrderByIdRequest{
		TransactionId: util.Ptr(transactionID),
		Mchid:         util.Ptr(mchID),
	})
	if err != nil || trans == nil {
		logrus.WithContext(ctx).Errorf("GetOrderInfo failed, err:%+v", err)
		return false
	}
	logrus.WithContext(ctx).Infof("GetOrderInfo resp:%v", util.ToJSON(trans))
	return *trans.TradeState == "SUCCESS"
}
