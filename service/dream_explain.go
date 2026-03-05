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

var cozeCli coze.CozeAPI

func init() {
	authCli := coze.NewTokenAuth(_const.CozeTokenV3)
	cozeCli = coze.NewCozeAPI(authCli, coze.WithBaseURL("https://api.coze.cn"))
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

func GetTexasPokerDecisionV2(ctx context.Context, images []string, imageType string) (string, int32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	var messageObject []*coze.MessageObjectString
	for _, image := range images {
		objectString := &coze.MessageObjectString{
			Type: "image",
		}
		if imageType == _const.ImageTypeFileID {
			objectString.FileID = image
		} else {
			objectString.FileURL = image
		}
		messageObject = append(messageObject, objectString)
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
	content := ""
	for {
		event, err := resp.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			logrus.WithContext(ctx).Errorf("[GetTexasPokerDecision] cozeCli.Chat.Stream.Recv err:%v", err)
			return "", 0, err
		}
		if event.Event == coze.ChatEventConversationMessageDelta && event.Message.Role == coze.MessageRoleAssistant {
			content += event.Message.Content
		}
		if event.Event == coze.ChatEventConversationMessageCompleted {
			break
		}
		logrus.WithContext(ctx).Infof("[GetTexasPokerDecision] msg:%v,event:%v", util.ToJSON(event.Message), util.ToJSON(event))
	}
	logrus.WithContext(ctx).Infof("[GetTexasPokerDecision] finalcontent:%v", content)
	decision := util.UnmarshalString[TexasPokerDecision](content)
	return decision.Action, decision.BetSize, nil
}

func UploadImages(ctx context.Context, images []string) ([]string, error) {
	imageIDs := make([]string, 0)
	for _, image := range images {
		resp, err := cozeCli.Files.Upload(ctx, &coze.UploadFilesReq{
			File: coze.NewUploadFile(strings.NewReader(image), fmt.Sprintf("%v.jpg", time.Now().UnixNano())),
		})
		if err != nil {
			logrus.WithContext(ctx).Errorf("[UploadImage] cozeCli.Files.Upload err:%v", err)
			return nil, err
		}
		logrus.WithContext(ctx).Infof("[UploadImage] resp:%v", util.ToJSON(resp))
		imageIDs = append(imageIDs, resp.FileInfo.ID)
	}
	return imageIDs, nil
}
