package wx_new

import (
	"context"
	"fmt"
	"github.com/kingjxu/ddbaby/util"
	"github.com/sirupsen/logrus"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"net/http"
)

func NotifyCallback(ctx context.Context, request *http.Request) (*payments.Transaction, error) {
	transaction := new(payments.Transaction)
	notifyReq, err := handler.ParseNotifyRequest(context.Background(), request, transaction)
	// 如果验签未通过，或者解密失败
	if err != nil {
		return nil, fmt.Errorf("parse notify request fail,err:%+v", err)
	}
	logrus.WithContext(ctx).Infof("notifyReq:%+v", util.MarshalString(notifyReq))
	// 处理通知内容
	return transaction, nil
}
