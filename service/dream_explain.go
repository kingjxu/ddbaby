package service

import (
	"context"
	"fmt"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/coze-dev/coze-go"
	jsoniter "github.com/json-iterator/go"
	_const "github.com/kingjxu/ddbaby/const"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strings"
	"time"
)

var cozeCli coze.CozeAPI

func init() {
	authCli := coze.NewTokenAuth(_const.CozeTokenV3)
	httpClient := &http.Client{
		Timeout: time.Second * 10,
	}
	cozeCli = coze.NewCozeAPI(authCli, coze.WithBaseURL("https://api.coze.cn"), coze.WithHttpClient(httpClient))
}
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
