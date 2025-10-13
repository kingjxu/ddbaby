package service

import (
	"context"
	"fmt"
	"github.com/kingjxu/ddbaby/dal/mysql/jk"
	"github.com/kingjxu/ddbaby/wx_new"
	"github.com/sirupsen/logrus"
)

func GetOrderInfo(ctx context.Context, orderId string) (*jk.JkOrder, error) {
	jkOrder, err := jk.GetOrderByOrderID(ctx, orderId)
	if err != nil {
		return nil, fmt.Errorf("GetOrderByOrderID err:%w", err)
	}
	if jkOrder.Status == 20 {
		return jkOrder, nil
	}
	if wx_new.IsOrderPaySuccess(ctx, orderId) {
		logrus.WithContext(ctx).Infof("order %v pay success,i:%v", jkOrder.OrderID, i)
		jkOrder.Status = 20
	}
	return jkOrder, nil
}
