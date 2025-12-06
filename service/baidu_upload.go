package service

import (
	"bytes"
	"context"
	"fmt"
	constdef "github.com/kingjxu/ddbaby/const"
	"github.com/kingjxu/ddbaby/dal/mysql/jk"
	"github.com/kingjxu/ddbaby/util"
	"github.com/sirupsen/logrus"
	"net/http"
)

const (
	baiduUploadToken   = "MKCzXyRkVWpwDQmWdp1HpTdVnwzLJ3gb@ICs46ZiSjLJNuiHJIq77IdrI4DxCNoU8"
	baiduUploadTokenV2 = "u8Z19dRU7IEinN6IuA3T3UEEmj0RScAV@UGdU1JtuMz9y2Qqrdm24CWW9wfW8jME7"
)

type BaiduUploadParam struct {
	Token           string                      `json:"token"`
	ConversionTypes []BaiduUploadConversionType `json:"conversionTypes"`
}

type BaiduUploadConversionType struct {
	LogidUrl string `json:"logidUrl"`
	NewType  int    `json:"newType"`
}

var bdVidCTypeMap map[string]bool

func init() {
	bdVidCTypeMap = make(map[string]bool)
}
func buildBdVidCTypeKey(bdVid string, cType int) string {
	return fmt.Sprintf("%v_%v", bdVid, cType)
}

func Upload2Baidu(ctx context.Context, orderInfo *jk.JkOrder, cType int) {
	if cType != constdef.CTypePurchaseService && cType != constdef.CTypeAddWechat {
		logrus.WithContext(ctx).Infof("not purchase or add wechat  bd_vid:%v,cType:%v", orderInfo.BdVid, cType)
		return
	}
	if orderInfo == nil || orderInfo.BdVid == "" {
		return
	}
	if bdVidCTypeMap[buildBdVidCTypeKey(orderInfo.BdVid, cType)] {
		logrus.WithContext(ctx).Infof("bd_vid exist, bd_vid:%v,cType:%v", orderInfo.BdVid, cType)
		return
	}
	bdVidCTypeMap[buildBdVidCTypeKey(orderInfo.BdVid, cType)] = true
	token := baiduUploadToken
	if orderInfo.Version == "v2" {
		token = baiduUploadTokenV2
	}
	// 上传到百度
	logidUrl := "http://ddbaby.site/dist/#/test?qo_type=cw&need_pic=false&bd_vid=" + orderInfo.BdVid
	param := BaiduUploadParam{
		Token: token,
		ConversionTypes: []BaiduUploadConversionType{
			{
				LogidUrl: logidUrl,
				NewType:  cType,
			},
		},
	}
	logrus.WithContext(ctx).Infof("Upload2Baidu param:%v", util.ToJSON(param))
	_, err := http.Post(constdef.BaiduUploadUrl, "application/json", bytes.NewBuffer([]byte(util.ToJSON(param))))
	if err != nil {
		logrus.WithContext(ctx).Errorf("Post err:%v", err)
	}
}
