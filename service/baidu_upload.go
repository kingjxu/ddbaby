package service

import (
	"bytes"
	"context"
	constdef "github.com/kingjxu/ddbaby/const"
	"github.com/kingjxu/ddbaby/dal/mysql/jk"
	"github.com/kingjxu/ddbaby/util"
	"github.com/sirupsen/logrus"
	"net/http"
)

const (
	baiduUploadToken = "MKCzXyRkVWpwDQmWdp1HpTdVnwzLJ3gb@ICs46ZiSjLJNuiHJIq77IdrI4DxCNoU8"
)

type BaiduUploadParam struct {
	Token           string                      `json:"token"`
	ConversionTypes []BaiduUploadConversionType `json:"conversionTypes"`
}

type BaiduUploadConversionType struct {
	LogidUrl string `json:"logidUrl"`
	NewType  int    `json:"newType"`
}

func Upload2Baidu(ctx context.Context, orderInfo *jk.JkOrder) {
	// 上传到百度
	logidUrl := "http://ddbaby.site/dist/#/test?qo_type=cw&need_pic=false&bd_vid=" + orderInfo.BdVid
	param := BaiduUploadParam{
		Token: baiduUploadToken,
		ConversionTypes: []BaiduUploadConversionType{
			{
				LogidUrl: logidUrl,
				NewType:  79,
			},
		},
	}

	_, err := http.Post(constdef.BaiduUploadUrl, "application/json", bytes.NewBuffer([]byte(util.ToJSON(param))))
	if err != nil {
		logrus.WithContext(ctx).Errorf("Post err:%v", err)
	}
}
