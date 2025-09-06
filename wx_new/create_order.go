package wx_new

import (
	"context"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/h5"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"
	"log"
	"time"
)

var (
	mchID                      string = "1689220293"                               // 木一橙科技商户号(早期)
	mchCertificateSerialNumber string = "1E0D7FDB591F09356CD1FAC75EBC398C03D11A9F" // 商户证书序列号
	mchAPIv3Key                string = "abcdefghijklmnopqrstuvwxyz123456"         // 商户APIv3密钥 自己设置的32位key
)

func CreateOrder() string {

	// 使用 utils 提供的函数从本地文件中加载商户私钥，商户私钥会用来生成请求的签名
	mchPrivateKey, err := utils.LoadPrivateKeyWithPath("/path/to/merchant/apiclient_key.pem")
	if err != nil {
		log.Print("load merchant private key error")
	}

	ctx := context.Background()
	// 使用商户私钥等初始化 client，并使它具有自动定时获取微信支付平台证书的能力
	opts := []core.ClientOption{
		option.WithWechatPayAutoAuthCipher(mchID, mchCertificateSerialNumber, mchPrivateKey, mchAPIv3Key),
	}
	client, err := core.NewClient(ctx, opts...)
	if err != nil {
		log.Printf("new wechat pay client err:%s", err)
	}

	svc := h5.H5ApiService{Client: client}
	_, _, _ = svc.Prepay(ctx,
		h5.PrepayRequest{
			Appid:       core.String("wx8888888888888888"),
			Mchid:       core.String("1900000109"),
			Description: core.String("Image形象店-深圳腾大-QQ公仔"),
			OutTradeNo:  core.String("1217752501201407033233368018"),
			TimeExpire:  core.Time(time.Now()),
			Attach:      core.String("自定义数据说明"),
		},
	)

	return "123"
}
