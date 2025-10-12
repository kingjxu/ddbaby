package service

import (
	"context"
	"fmt"
	"github.com/kingjxu/ddbaby/dal/mysql/jk"
	"github.com/kingjxu/ddbaby/wx_new"
	"github.com/sirupsen/logrus"
	"time"
)

func GetOrderInfo(ctx context.Context, orderId string) (*jk.JkOrder, error) {
	for i := 0; i < 2; i++ {
		jkOrder, err := jk.GetOrderByOrderID(ctx, orderId)
		if err != nil {
			return nil, fmt.Errorf("GetOrderByOrderID err:%w", err)
		}
		if jkOrder.Status == 20 {
			return jkOrder, nil
		}
		if wx_new.IsOrderPaySuccess(ctx, orderId) {
			logrus.WithContext(ctx).Infof("order %v pay success", jkOrder.OrderID)
			jkOrder.Status = 20
		}
		if i == 1 {
			return jkOrder, nil
		}

		if jkOrder.Status == 10 {
			time.Sleep(time.Second * 2)
			continue
		}
		return jkOrder, nil
	}
	jkOrder, err := jk.GetOrderByOrderID(ctx, orderId)
	if err != nil {
		return nil, fmt.Errorf("GetOrderByOrderID err:%w", err)
	}
	if jkOrder.Status == 20 {
		return jkOrder, nil
	}
	if wx_new.IsOrderPaySuccess(ctx, orderId) {
		logrus.WithContext(ctx).Infof("order %v pay success", jkOrder.OrderID)
		jkOrder.Status = 20
	}
	return jkOrder, nil
}
