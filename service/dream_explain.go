package service

import (
	"context"
	"fmt"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	jsoniter "github.com/json-iterator/go"
	_const "github.com/kingjxu/ddbaby/const"
	"github.com/sirupsen/logrus"
	"io"
	"strings"
	"time"
)

func GetDreamExplain(ctx context.Context, dream string) (string, error) {
	req := GetCozeHttpRequest()
	botParam := &BotReqParam{
		ConversationId: fmt.Sprintf("%v", time.Now().UnixNano()),
		BotId:          _const.DreamExplainBotID,
		User:           fmt.Sprintf("%v", time.Now().Unix()),
		Query:          dream,
		Stream:         false,
	}
	body, _ := jsoniter.MarshalToString(botParam)
	req.Body = io.NopCloser(strings.NewReader(body))
	botResp, err := ProcessBotResp(ctx, req)
	if err != nil {
		hlog.CtxInfof(ctx, "[GetDreamExplain] get coze response err:%v", err)
		logrus.WithContext(ctx).Errorf("[GetDreamExplain] get coze response err:%v", err)
		return "", err
	}
	return botResp.Messages[0].Content, nil
}
