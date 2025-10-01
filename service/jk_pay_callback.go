package service

import (
	"context"
	"fmt"
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
	return nil
}
