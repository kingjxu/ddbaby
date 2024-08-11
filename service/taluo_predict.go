package service

import (
	"context"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	_const "github.com/kingjxu/ddbaby/const"
	"github.com/sirupsen/logrus"
	"io"
	"strings"
	"time"
)

func TaLuoPredict(ctx context.Context, query string) (string, error) {
	req := GetCozeHttpRequest()
	logrus.WithContext(ctx).Infof("[TaLuoPredict] query:%v", query)
	botParam := &BotReqParam{
		ConversationId: fmt.Sprintf("%v", time.Now().UnixNano()),
		BotId:          _const.PickNameBotID,
		User:           fmt.Sprintf("%v", time.Now().Unix()),
		Query:          query,
		Stream:         false,
	}
	body, _ := jsoniter.MarshalToString(botParam)
	req.Body = io.NopCloser(strings.NewReader(body))
	botResp, err := ProcessBotResp(ctx, req)
	if err != nil {
		logrus.WithContext(ctx).Errorf("[TaLuoPredict] get coze response err:%v", err)
		return "", err
	}
	return botResp.Messages[0].Content, nil
}
