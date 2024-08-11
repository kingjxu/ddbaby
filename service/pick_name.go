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

type PickNameParam struct {
	FamilyName string
	Gender     string
	NameLen    int
	Remark     *string
}

func PickName(ctx context.Context, param PickNameParam) (string, error) {
	req := GetCozeHttpRequest()
	query := buildQuery(param)
	logrus.WithContext(ctx).Infof("[PickName] query:%v", query)
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
		hlog.CtxInfof(ctx, "[PickName] get coze response err:%v", err)
		logrus.WithContext(ctx).Errorf("[PickName] get coze response err:%v", err)
		return "", err
	}
	return botResp.Messages[0].Content, nil
}

func buildQuery(param PickNameParam) string {
	query := "姓:" + param.FamilyName + ";"
	if param.Gender != "" {
		query += "性别:" + param.Gender + ";"
	}
	if param.NameLen > 0 {
		query += fmt.Sprintf("名字长度:%v;", param.NameLen)
	} else {
		query += fmt.Sprintf("名字长度:%v个字或者%v个字;", len([]rune(param.FamilyName))+1, len([]rune(param.FamilyName))+2)
	}

	if param.Remark != nil {
		query += fmt.Sprintf("备注:%v;", *param.Remark)
	}
	return query

}
