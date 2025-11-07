package service

import (
	"context"
	"fmt"
	constdef "github.com/kingjxu/ddbaby/const"
	"github.com/kingjxu/ddbaby/dal/mysql/jk"
	"github.com/kingjxu/ddbaby/util"
	"github.com/kingjxu/ddbaby/wx_new"
	"github.com/sirupsen/logrus"
	"net/http"
)

func PayCallback(ctx context.Context, req *http.Request) error {
	trans, err := wx_new.NotifyCallback(ctx, req)
	if err != nil {
		return fmt.Errorf("NotifyCallback err: %v", err)
	}
	logrus.WithContext(ctx).Infof("trans:%+v", util.ToJSON(trans))
	err = jk.UpdateOrderPaySuccess(ctx, *trans.OutTradeNo, *trans.Payer.Openid, *trans.TransactionId)
	if err != nil {
		return fmt.Errorf("UpdateOrderPaySuccess err:%v", err)
	}

	orderInfo, err := jk.GetOrderByOrderID(ctx, *trans.OutTradeNo)
	if err != nil || orderInfo == nil {
		logrus.Errorf("GetOrderByOrderID err:%v,orderInfo:%v", err, util.ToJSON(orderInfo))
		return nil
	}
	Upload2Baidu(ctx, orderInfo, constdef.CTypePurchaseService)
	return nil
}
