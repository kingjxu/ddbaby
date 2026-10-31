package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/coze-dev/coze-go"
	_const "github.com/kingjxu/ddbaby/const"
	"github.com/kingjxu/ddbaby/model"
	"github.com/kingjxu/ddbaby/util"
	"github.com/sirupsen/logrus"
	"image"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type TexasPokerDecision struct {
	HoleCards      string `json:"hole_cards"`
	CommunityCards string `json:"community_cards"`
	IsMyTurn       bool   `json:"is_my_turn"`
	Stage          string `json:"stage"`
	Action         string `json:"action"`
	BetSize        int32  `json:"bet_size"`
}

const recognizePokerURL = "http://127.0.0.1:5555/recognize"

const gtoAPIKeyEnv = "GTO_API_KEY"

// The decision endpoints stay overridable so ddbaby can point at the local
// gto-solver service without changing the client contract. The defaults keep
// the previous upstream behaviour.
var (
	// vvip is the field spelling defined by the upstream v2 API contract.
	gtoDecisionURL   = envOrDefault("GTO_DECISION_URL", "http://114.132.228.98/gto/decision")
	gtoDecisionV2URL = envOrDefault("GTO_DECISION_V2_URL", "http://114.132.228.98/gto/decision_v2")
)

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
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

func GetTexasPokerDecisionV2(ctx context.Context, images []string, imageType string) (*TexasPokerDecision, error) {
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
		BotID:  _const.TexasPokerDecisionBotIDV2,
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
		return nil, err
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
			return nil, err
		}
		if event.Event == coze.ChatEventConversationMessageDelta && event.Message.Role == coze.MessageRoleAssistant {
			content += event.Message.Content
		}
		if event.Event == coze.ChatEventConversationMessageCompleted {
			content += event.Message.Content
			break
		}
	}
	logrus.WithContext(ctx).Infof("[GetTexasPokerDecisionV2] messageObject:%v, finalcontent:%v", util.ToJSON(messageObject), content)
	decision := util.UnmarshalString[*TexasPokerDecision](content)
	return decision, nil
}

// UploadImages 上传到coze，返回的是fileID
func UploadImages(ctx context.Context, images []string) ([]string, error) {
	imageIDs := make([]string, 0)
	for index, image := range images {
		imageData, err := util.Base64Decode(image, false)
		if err != nil {
			logrus.WithContext(ctx).Errorf("[UploadImage] Base64Decode image err:%v", err)
			return nil, err
		}
		logrus.WithContext(ctx).Infof("[UploadImage] upload image index:%v,len(realImage):%v", index, len(imageData))
		compressImageData, err := util.ResizeJpegBytes(imageData, 720, 1080)
		if err == nil {
			logrus.WithContext(ctx).Infof("[UploadImage] ResizeJpegBytes orgLen:%v,decodeLen:%v,compressLen:%v", len(image), len(imageData), len(compressImageData))
			imageData = compressImageData
		} else {
			logrus.WithContext(ctx).Errorf("[UploadImage] ResizeJpegBytes err:%v, but has no effect", err)
		}
		resp, err := cozeCli.Files.Upload(ctx, &coze.UploadFilesReq{
			File: coze.NewUploadFile(strings.NewReader(string(imageData)), fmt.Sprintf("%v.jpg", time.Now().UnixNano())),
		})
		if err != nil {
			logrus.WithContext(ctx).Errorf("[UploadImage] cozeCli.Files.Upload err:%v", err)
			return nil, err
		}
		logrus.WithContext(ctx).Infof("[UploadImage] upload to coze,resp:%v", util.ToJSON(resp))
		imageUrl, err := UploadImagesV2(ctx, string(imageData))
		if err != nil {
			logrus.WithContext(ctx).Errorf("[UploadImage] UploadImagesV2 err:%v", err)
		}

		if imageUrl != "" {
			logrus.WithContext(ctx).Infof("[UploadImage] upload fileID:%v,images_url:%v", resp.FileInfo.ID, imageUrl)
		}
		imageIDs = append(imageIDs, resp.FileInfo.ID)
	}
	return imageIDs, nil
}

// UploadImagesV2 返回图片的URL
func UploadImagesV2(ctx context.Context, imageData string) (string, error) {
	fileName := fmt.Sprintf("/usr/local/webserver/images/%v.jpg", time.Now().UnixNano())
	if err := writeImageToFileWithLock([]byte(imageData), fileName); err != nil {
		logrus.WithContext(ctx).Errorf("[UploadImagesV2] write image err:%v", err)
		return "", err
	}
	imageUrl, err := util.UploadImage(fileName)
	if err != nil {
		logrus.WithContext(ctx).Errorf("[UploadImagesV2] UploadImage err:%v", err)
		return "", err
	}
	return imageUrl, nil
}
func WriteImageToLocalFile(ctx context.Context, base64Data string, tag string) {
	startTime := time.Now()
	imageData, err := util.Base64Decode(base64Data, false)
	if err != nil {
		logrus.WithContext(ctx).Errorf("[WriteImageToLocalFile] Base64Decode image err:%v", err)
		return
	}
	conf, _, _ := image.DecodeConfig(bytes.NewReader(imageData))
	fileName := fmt.Sprintf("/usr/local/webserver/images/%v.jpg", time.Now().Format(time.DateTime)+tag)
	if err := writeImageToFileWithLock(imageData, fileName); err != nil {
		logrus.WithContext(ctx).Errorf("[WriteImageToLocalFile] write image err:%v", err)
		return
	}
	logrus.WithContext(ctx).Infof("[WriteImageToLocalFile] height:%v,width:%v, cost %vms", conf.Height, conf.Width, time.Since(startTime).Milliseconds())
}

func RecognizePokerByFilePath(ctx context.Context, filePath string) (*model.TexasResult, error) {
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		logrus.WithContext(ctx).Errorf("[RecognizePokerByFilePath] os.ReadFile err:%v, filePath:%v", err, filePath)
		return nil, err
	}

	imageBase64 := base64.StdEncoding.EncodeToString(fileData)
	return RecognizePoker(ctx, imageBase64, true)
}

func RecognizePoker(ctx context.Context, imageBase64 string, parseAll bool) (*model.TexasResult, error) {
	start := time.Now()
	// 1. 构造请求体
	reqData := model.RecognizeRequest{
		ImageBase64: imageBase64,
		App:         "polerv2",
		ParseAll:    parseAll,
	}

	reqBytes, err := json.Marshal(reqData)
	if err != nil {
		fmt.Printf("构造 JSON 请求体失败: %v\n", err)
		os.Exit(1)
	}

	// 2. 发送 HTTP POST 请求
	resp, err := http.Post(recognizePokerURL, "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("[RecognizePoker] http Post err:%v", err)
	}
	defer resp.Body.Close()
	logrus.WithContext(ctx).Infof("[RecognizePoker] request to recognize poker, cost %vms", time.Since(start).Milliseconds())
	// 5. 读取并打印返回结果
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("[RecognizePoker] io.ReadAll err:%v", err)
	}
	logrus.WithContext(ctx).Infof("[RecognizePoker] response status code:%v,data:%s", resp.StatusCode, string(respBody))
	result := util.UnmarshalString[*model.TexasResult](string(respBody))
	return result, nil
}

func GtoDecision(ctx context.Context, req *model.TexasGtoDecisionReq) (*model.TexasGtoDecisionResp, error) {
	return doGtoDecision(ctx, req, gtoDecisionURL)
}

func GtoDecisionByProfile(
	ctx context.Context,
	req *model.TexasGtoDecisionReq,
	vpip int32,
	pfr int32,
) (*model.TexasGtoDecisionResp, error) {
	if vpip == 0 && pfr == 0 {
		return GtoDecision(ctx, req)
	}
	if req == nil {
		return nil, errors.New("[GtoDecision] request is nil")
	}
	v2Req := *req
	v2Req.VVIP = vpip
	v2Req.PFR = pfr
	return doGtoDecision(ctx, &v2Req, gtoDecisionV2URL)
}

func doGtoDecision(
	ctx context.Context,
	req *model.TexasGtoDecisionReq,
	url string,
) (*model.TexasGtoDecisionResp, error) {
	reqBytes := util.ToJSON(req)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer([]byte(reqBytes)))
	if err != nil {
		logrus.WithContext(ctx).Errorf("[GtoDecision] http.NewRequest err:%v", err)
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	apiKey := strings.TrimSpace(os.Getenv(gtoAPIKeyEnv))
	if apiKey == "" {
		return nil, fmt.Errorf("[GtoDecision] %s is not configured", gtoAPIKeyEnv)
	}
	httpReq.Header.Set("X-API-Key", apiKey)

	logrus.WithContext(ctx).Infof("[GtoDecision] request:%s", reqBytes)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		logrus.WithContext(ctx).Errorf("[GtoDecision] http client.Do err:%v", err)
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logrus.WithContext(ctx).Errorf("[GtoDecision] io.ReadAll err:%v", err)
		return nil, err
	}
	logrus.WithContext(ctx).Infof("[GtoDecision] response status code:%v,data:%s", resp.StatusCode, string(respBody))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[GtoDecision] response status code:%v,data:%s", resp.StatusCode, string(respBody))
	}
	result := util.UnmarshalString[*model.TexasGtoDecisionResp](string(respBody))
	return result, nil
}
