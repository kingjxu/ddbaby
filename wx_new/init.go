package wx_new

import (
	"context"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"
)

var (
	appID                      string = "wxfa03f168f6c928c7"                       //灵运先知
	mchID                      string = "1689220293"                               // 木一橙科技商户号(早期)
	mchCertificateSerialNumber string = "1E0D7FDB591F09356CD1FAC75EBC398C03D11A9F" // 商户证书序列号
	mchAPIv3Key                string = "abcdefghijklmnopqrstuvwxyz123456"         // 商户APIv3密钥 自己设置的32位key
	client                     *core.Client
)

func init() {
	var err error
	// 使用 utils 提供的函数从本地文件中加载商户私钥，商户私钥会用来生成请求的签名
	mchPrivateKey, err := utils.LoadPrivateKeyWithPath("/path/to/merchant/apiclient_key.pem")
	if err != nil {
		hlog.Fatal("load merchant private key error")
		panic(err)
	}
	ctx := context.Background()
	// 使用商户私钥等初始化 client，并使它具有自动定时获取微信支付平台证书的能力
	opts := []core.ClientOption{
		option.WithWechatPayAutoAuthCipher(mchID, mchCertificateSerialNumber, mchPrivateKey, mchAPIv3Key),
	}
	client, err = core.NewClient(ctx, opts...)
	if err != nil {
		hlog.Fatalf("new wechat pay client err:%s", err)
		panic(err)
	}
}
