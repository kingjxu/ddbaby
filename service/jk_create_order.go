package service

import (
	"context"
	"fmt"
	"github.com/kingjxu/ddbaby/biz/model/ddbaby"
	_const "github.com/kingjxu/ddbaby/const"
	"github.com/kingjxu/ddbaby/dal/mysql/jk"
	"github.com/kingjxu/ddbaby/model"
	"github.com/kingjxu/ddbaby/util"
	"github.com/kingjxu/ddbaby/wx_new"
	"github.com/sirupsen/logrus"
	"time"
)

func CreateOrder(ctx context.Context, param *ddbaby.JkCreateOrderReq, commonParam *model.CommonParam) (string, string, error) {
	qaItems := util.ToJSON(param.GetQaItems())
	if param.GetOrderID() != "" {
		order, err := jk.GetOrderByOrderID(ctx, param.GetOrderID())
		if err != nil {
			return "", "", fmt.Errorf("GetOrderByOrderID failed,%v", err)
		}
		qaItems = order.QaItems
	}
	orderID := fmt.Sprintf("E%v", time.Now().UnixMicro())
	amount := _const.Seq2Amount[param.GetSeq()]
	// 微信下单
	err := jk.CreateOrder(ctx, &jk.JkOrder{
		JkType:  param.GetQoType(),
		UserID:  param.GetUserID(),
		OrderID: orderID,
		Amount:  int(amount),
		Status:  10,
		Seq:     int(param.GetSeq()),
		QaItems: qaItems,
	})
	if err != nil {
		logrus.WithContext(ctx).Errorf("[CreateOrder] jk.CreateOrder err:%v", err)
		return "", "", err
	}

	if amount == 0 {
		return "", "", nil
	}
	h5Url, err := wx_new.Prepay(ctx, wx_new.CreateOrderParam{
		OutTradeNo:  orderID,
		Title:       _const.JkType2Title[param.GetQoType()],
		Amount:      int(amount),
		NotifyURL:   _const.JKWXPayNotifyUrl,
		CommonParam: commonParam,
	})
	if err != nil {
		logrus.WithContext(ctx).Errorf("[CreateOrder] wx_new.Prepay  err:%v", err)
		return "", "", err
	}
	return h5Url, orderID, nil
}
