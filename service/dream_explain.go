package service

import (
	"context"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	CozeToken         = "pat_0ZCXIT2NguHJWbGYvr5SsU85onSnjIZlfZkkjw3Kl21SI4IpSkLsJr5pXYq30pC2"
	DreamExplainBotID = "7391123436175343616"
)

type BotReqParam struct {
	ConversationId string `json:"conversation_id"`
	BotId          string `json:"bot_id"`
	User           string `json:"user"`
	Query          string `json:"query"`
	Stream         bool   `json:"stream"`
}
type BotResp struct {
	Messages []struct {
		Role        string      `json:"role"`
		Type        string      `json:"type"`
		Content     string      `json:"content"`
		ContentType string      `json:"content_type"`
		ExtraInfo   interface{} `json:"extra_info"`
	} `json:"messages"`
	ConversationId string `json:"conversation_id"`
	Code           int    `json:"code"`
	Msg            string `json:"msg"`
}

func GetDreamExplain(ctx context.Context, dream string) (string, error) {
	req := GetCozeHttpRequest()
	botParam := &BotReqParam{
		ConversationId: fmt.Sprintf("%v", time.Now().UnixNano()),
		BotId:          DreamExplainBotID,
		User:           fmt.Sprintf("%v", time.Now().Unix()),
		Query:          dream,
		Stream:         false,
	}
	body, _ := jsoniter.MarshalToString(botParam)
	req.Body = io.NopCloser(strings.NewReader(body))
	botResp, err := ProcessBotResp(ctx, req)
	if err != nil {
		logrus.WithContext(ctx).Errorf("[GetDreamExplain] get coze response err:%v", err)
		return "", err
	}
	return botResp.Messages[0].Content, nil
}
func GetCozeHttpRequest() *http.Request {
	req, _ := http.NewRequest("POST", "https://api.coze.cn/open_api/v2/chat", nil)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "*/*")
	req.Header.Add("Host", "api.coze.cn")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %v", CozeToken))
	return req
}

func ProcessBotResp(ctx context.Context, req *http.Request) (*BotResp, error) {
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logrus.WithContext(ctx).Errorf("[GetDreamExplain] get coze response err:%v", err)
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logrus.WithContext(ctx).Errorf("[ProcessBotResp] io.ReadAll err:%v", err)
		return nil, err
	}
	logrus.WithContext(ctx).Infof("[ProcessBotResp] respBody:%v", string(respBody))
	botResp := &BotResp{}
	err = jsoniter.Unmarshal(respBody, botResp)
	if err != nil {
		logrus.WithContext(ctx).Errorf("[ProcessBotResp] jsoniter.Unmarshal failed err:%v,botReps:%v", err, string(respBody))
		return nil, err
	}
	if botResp.Code != 0 {
		logrus.WithContext(ctx).Errorf("[ProcessBotResp] botResp.Code%v,msg:%v", botResp.Code, botResp.Msg)
		return nil, fmt.Errorf("code:%v,msg:%v", botResp.Code, botResp.Msg)
	}
	if len(botResp.Messages) == 0 {
		logrus.WithContext(ctx).Errorf("[ProcessBotResp] botResp.Messages is empty")
		return nil, fmt.Errorf("botResp.Messages is empty")
	}
	return botResp, nil
}
