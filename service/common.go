package service

import (
	"context"
	"fmt"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	jsoniter "github.com/json-iterator/go"
	_const "github.com/kingjxu/ddbaby/const"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
)

type BotContent struct {
	Type    string `json:"type"`
	FileUrl string `json:"file_url"`
	FileID  string `json:"file_id"`
	Text    string `json:"text"`
}
type AdditionalMessage struct {
	Role        string `json:"role"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
}
type BotReqParamV3 struct {
	BotId              string              `json:"bot_id"`
	User               string              `json:"user"`
	Stream             bool                `json:"stream"`
	AdditionalMessages []AdditionalMessage `json:"additional_messages"`
}
type BotRespV3Data struct {
	BotId       string `json:"bot_id"`
	Role        string `json:"role"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
}
type BotRespV3 struct {
	Data *BotRespV3Data `json:"data"`
	Code int            `json:"code"`
	Msg  string         `json:"msg"`
}

func GetCozeHttpRequestV3() *http.Request {
	req, _ := http.NewRequest("POST", "https://api.coze.cn/v3/chat", nil)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "*/*")
	req.Header.Add("Host", "api.coze.cn")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %v", _const.CozeTokenV3))
	return req
}
func ProcessBotRespV3(ctx context.Context, req *http.Request) (*BotRespV3Data, error) {
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		hlog.CtxInfof(ctx, "[ProcessBotRespV3] get coze response err:%v", err)
		logrus.WithContext(ctx).Errorf("[ProcessBotRespV3] get coze response err:%v", err)
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		hlog.CtxInfof(ctx, "[ProcessBotRespV3] io.ReadAll err:%v", err)
		logrus.WithContext(ctx).Errorf("[ProcessBotRespV3] io.ReadAll err:%v", err)
		return nil, err
	}
	hlog.CtxInfof(ctx, "[ProcessBotRespV3] respBody:%v", string(respBody))
	logrus.WithContext(ctx).Infof("[ProcessBotRespV3] respBody:%v", string(respBody))
	botResp := &BotRespV3{}
	err = jsoniter.Unmarshal(respBody, botResp)
	if err != nil {
		hlog.CtxInfof(ctx, "[ProcessBotRespV3] jsoniter.Unmarshal failed err:%v,botReps:%v", err, string(respBody))
		logrus.WithContext(ctx).Errorf("[ProcessBotRespV3] jsoniter.Unmarshal failed err:%v,botReps:%v", err, string(respBody))
		return nil, err
	}
	if botResp.Code != 0 {
		hlog.CtxInfof(ctx, "[ProcessBotRespV3] botResp.Code%v,msg:%v", botResp.Code, botResp.Msg)
		logrus.WithContext(ctx).Errorf("[ProcessBotRespV3] botResp.Code%v,msg:%v", botResp.Code, botResp.Msg)
		return nil, fmt.Errorf("code:%v,msg:%v", botResp.Code, botResp.Msg)
	}
	return botResp.Data, nil
}

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

func GetCozeHttpRequest() *http.Request {
	req, _ := http.NewRequest("POST", "https://api.coze.cn/open_api/v2/chat", nil)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "*/*")
	req.Header.Add("Host", "api.coze.cn")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %v", _const.CozeToken))
	return req
}

func ProcessBotResp(ctx context.Context, req *http.Request) (*BotResp, error) {
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		hlog.CtxInfof(ctx, "[GetDreamExplain] get coze response err:%v", err)
		logrus.WithContext(ctx).Errorf("[GetDreamExplain] get coze response err:%v", err)
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		hlog.CtxInfof(ctx, "[ProcessBotResp] io.ReadAll err:%v", err)
		logrus.WithContext(ctx).Errorf("[ProcessBotResp] io.ReadAll err:%v", err)
		return nil, err
	}
	hlog.CtxInfof(ctx, "[ProcessBotResp] respBody:%v", string(respBody))
	logrus.WithContext(ctx).Infof("[ProcessBotResp] respBody:%v", string(respBody))
	botResp := &BotResp{}
	err = jsoniter.Unmarshal(respBody, botResp)
	if err != nil {
		hlog.CtxInfof(ctx, "[ProcessBotResp] jsoniter.Unmarshal failed err:%v,botReps:%v", err, string(respBody))
		logrus.WithContext(ctx).Errorf("[ProcessBotResp] jsoniter.Unmarshal failed err:%v,botReps:%v", err, string(respBody))
		return nil, err
	}
	if botResp.Code != 0 {
		hlog.CtxInfof(ctx, "[ProcessBotResp] botResp.Code%v,msg:%v", botResp.Code, botResp.Msg)
		logrus.WithContext(ctx).Errorf("[ProcessBotResp] botResp.Code%v,msg:%v", botResp.Code, botResp.Msg)
		return nil, fmt.Errorf("code:%v,msg:%v", botResp.Code, botResp.Msg)
	}
	if len(botResp.Messages) == 0 {
		hlog.CtxInfof(ctx, "[ProcessBotResp] botResp.Messages is empty")
		logrus.WithContext(ctx).Errorf("[ProcessBotResp] botResp.Messages is empty")
		return nil, fmt.Errorf("botResp.Messages is empty")
	}
	return botResp, nil
}
