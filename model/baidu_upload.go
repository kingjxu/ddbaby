package model

type BaiduUploadParam struct {
	Token           string                      `json:"token"`
	ConversionTypes []BaiduUploadConversionType `json:"conversionTypes"`
}

type BaiduUploadConversionType struct {
	LogidUrl string `json:"logidUrl"`
	NewType  int    `json:"newType"`
}
