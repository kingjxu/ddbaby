package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/coze-dev/coze-go"
	jsoniter "github.com/json-iterator/go"
	_const "github.com/kingjxu/ddbaby/const"
	"github.com/kingjxu/ddbaby/util"
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

type TexasPokerDecision struct {
	Action  string `json:"action"`
	BetSize int32  `json:"bet_size"`
}

func GetTexasPokerDecision(ctx context.Context, images []string) (string, int32, error) {
	req := GetCozeHttpRequestV3()
	botParam := &BotReqParamV3{
		BotId:  _const.TexasPokerDecisionBotID,
		UserId: fmt.Sprintf("%v", time.Now().Unix()),
		Stream: false,
	}
	var botContent []BotContent
	for _, image := range images {
		botContent = append(botContent, BotContent{
			Type:    "image",
			FileUrl: image,
		})
	}
	botParam.AdditionalMessages = []AdditionalMessage{
		{
			Role:        "user",
			Content:     util.ToJSON(botContent),
			ContentType: "object_string",
		},
	}

	body := util.ToJSON(botParam)
	logrus.WithContext(ctx).Infof("[GetTexasPokerDecision] botBody:%v", body)
	req.Body = io.NopCloser(strings.NewReader(body))
	botRespData, err := ProcessBotRespV3(ctx, req)
	if err != nil {
		logrus.WithContext(ctx).Errorf("[GetTexasPokerDecision] get coze response err:%v", err)
		return "", 0, err
	}
	decision := util.UnmarshalString[TexasPokerDecision](botRespData.Content)
	return decision.Action, decision.BetSize, nil
}

func GetTexasPokerDecisionV2(ctx context.Context, images []string) (string, int32, error) {
	authCli := coze.NewTokenAuth(_const.CozeTokenV3)

	cozeCli := coze.NewCozeAPI(authCli, coze.WithBaseURL("https://api.coze.cn"))
	var messageObject []*coze.MessageObjectString
	for _, image := range images {
		messageObject = append(messageObject, &coze.MessageObjectString{
			Type:    "image",
			FileURL: image,
		})
	}
	req := &coze.CreateChatsReq{
		BotID:  _const.TexasPokerDecisionBotID,
		UserID: fmt.Sprintf("%v", time.Now().Unix()),
		Stream: util.Ptr(false),
		Messages: []*coze.Message{
			coze.BuildUserQuestionObjects(messageObject, nil),
		},
	}

	resp, err := cozeCli.Chat.Stream(ctx, req)
	if err != nil {
		logrus.WithContext(ctx).Errorf("[GetTexasPokerDecision] cozeCli.Chat.Stream err:%v", err)
		return "", 0, err
	}
	defer resp.Close()
	for {
		event, err := resp.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			logrus.WithContext(ctx).Errorf("[GetTexasPokerDecision] cozeCli.Chat.Stream.Recv err:%v", err)
			return "", 0, err
		}
		logrus.WithContext(ctx).Infof("[GetTexasPokerDecision] msg:%v,event:%v", util.ToJSON(event.Message), util.ToJSON(event))
	}

	return "", 0, nil
}
