package service

import (
	"context"
	"fmt"
	"github.com/kingjxu/ddbaby/dal/mysql/jk"
)

func GetOrderInfo(ctx context.Context, orderId string) (*jk.JkOrder, error) {
	jkOrder, err := jk.GetOrderByOrderID(ctx, orderId)
	if err != nil {
		return nil, fmt.Errorf("GetOrderByOrderID err:%w", err)
	}
	return jkOrder, nil
}
