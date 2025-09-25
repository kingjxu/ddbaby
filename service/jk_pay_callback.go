package service

import (
	"context"
	"github.com/kingjxu/ddbaby/util"
	"github.com/kingjxu/ddbaby/wx_new"
	"github.com/sirupsen/logrus"
	"net/http"
)

func PayCallback(ctx context.Context, req *http.Request) error {
	trans, err := wx_new.NotifyCallback(ctx, req)
	if err != nil {
		return err
	}
	logrus.WithContext(ctx).Infof("trans:%v", util.Marshal(trans))
	return nil
}
