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
	Stage   string `json:"stage"`
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
	ctx, cancel := context.WithTimeout(ctx, 100*time.Second)
	defer cancel()
	logrus.WithContext(ctx).Infof("[GetTexasPokerDecisionV2] messageObject:%v", util.ToJSON(messageObject))
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
		logrus.WithContext(ctx).Infof("[GetTexasPokerDecisionV2] messageObject:%v, finalcontent:%v", util.ToJSON(messageObject), content)
	}
	logrus.WithContext(ctx).Infof("[GetTexasPokerDecisionV2] messageObject:%v, finalcontent:%v", util.ToJSON(messageObject), content)
	decision := util.UnmarshalString[TexasPokerDecision](content)
	if !util.Contains(_const.TexasPokerStageAll, strings.ToLower(decision.Stage)) {
		logrus.WithContext(ctx).Errorf("[GetTexasPokerDecision] unknown stage:%v", decision.Stage)
		return "", 0, nil
	}
	if !util.Contains(_const.TexasPokerActionAll, strings.ToLower(decision.Action)) {
		logrus.WithContext(ctx).Errorf("[GetTexasPokerDecision] unknown action:%v", decision.Action)
		return "", 0, nil
	}
	return decision.Action, decision.BetSize, nil
}

// UploadImages 上传到coze，返回的是fileID
func UploadImages(ctx context.Context, images []string) ([]string, error) {
	imageIDs := make([]string, 0)
	for index, image := range images {
		logrus.WithContext(ctx).Infof("[UploadImages] len(orginImage):%v", len(image))
		imageData, err := util.Base64Decode(image, false)
		if err != nil {
			logrus.WithContext(ctx).Errorf("[UploadImage] Base64Decode image err:%v", err)
			return nil, err
		}
		imageType := util.DetectImageType(imageData)
		if imageType == util.ImageTypeUnknown {
			logrus.WithContext(ctx).Errorf("[UploadImage] unknown image type:%v", imageType)
			return nil, fmt.Errorf("unknown image type:%v", imageType)
		}
		logrus.WithContext(ctx).Infof("[UploadImage] upload image index:%v,type:%v,len(realImage):%v", index, imageType, len(imageData))
		resp, err := cozeCli.Files.Upload(ctx, &coze.UploadFilesReq{
			File: coze.NewUploadFile(strings.NewReader(string(imageData)), fmt.Sprintf("%v.jpg", time.Now().UnixNano())),
		})
		if err != nil {
			logrus.WithContext(ctx).Errorf("[UploadImage] cozeCli.Files.Upload err:%v", err)
			return nil, err
		}
		imageUrls, err := UploadImagesV2(ctx, []string{string(imageData)})
		if err != nil {
			logrus.WithContext(ctx).Errorf("[UploadImage] UploadImagesV2 err:%v", err)
		}
		if len(imageUrls) > 0 {
			logrus.WithContext(ctx).Infof("[UploadImage] upload fileID:%v,images_url:%v", resp.FileInfo.ID, imageUrls[0])
		}
		logrus.WithContext(ctx).Infof("[UploadImage] len(realImageData):%v,resp:%v", len(imageData), util.ToJSON(resp))
		_ = util.WriteImageToFile(imageData, fmt.Sprintf("/usr/local/webserver/images/%v.jpg", resp.FileInfo.ID))
		imageIDs = append(imageIDs, resp.FileInfo.ID)
	}
	return imageIDs, nil
}

// UploadImagesV2 返回图片的URL
func UploadImagesV2(ctx context.Context, images []string) ([]string, error) {
	imageUrls := make([]string, 0)
	for index, image := range images {
		logrus.WithContext(ctx).Infof("[UploadImages] len(orginImage):%v", len(image))
		imageData, err := util.Base64Decode(image, false)
		if err != nil {
			logrus.WithContext(ctx).Errorf("[UploadImage] Base64Decode image err:%v", err)
			return nil, err
		}
		imageType := util.DetectImageType(imageData)
		if imageType == util.ImageTypeUnknown {
			logrus.WithContext(ctx).Errorf("[UploadImage] unknown image type:%v", imageType)
			return nil, fmt.Errorf("unknown image type:%v", imageType)
		}
		logrus.WithContext(ctx).Infof("[UploadImage] upload image index:%v,type:%v,len(realImage):%v", index, imageType, len(imageData))
		fileName := fmt.Sprintf("/usr/local/webserver/images/%v.jpg", time.Now().UnixNano())
		_ = util.WriteImageToFile(imageData, fileName)
		imageUrl, err := util.UploadImage(fileName)
		if err != nil {
			logrus.WithContext(ctx).Errorf("[UploadImage] UploadImage err:%v", err)
			return nil, err
		}
		imageUrls = append(imageUrls, imageUrl)
	}
	return imageUrls, nil
}
