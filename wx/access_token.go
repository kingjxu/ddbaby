package wx

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"io/ioutil"
	"net/http"
	"time"
)

const WXPAY_ACCESS_TOKEN_URL = "https://api.weixin.qq.com/sns/oauth2/access_token?"
const WXPAY_APPID = "wxbe68382a9463a04e"
const WXPAY_APPKEY = "9318fdd4f32574796e92f15a180574b2"
const WX_TOKEN_URL = "https://api.weixin.qq.com/cgi-bin/token"
const WX_TICKET_URL = "https://api.weixin.qq.com/cgi-bin/ticket/getticket"

type AccessToken struct {
	AccessToken  string `json:"access_token"`
	ExpireIn     int    `json:"expire_in"`
	RefreshToken string `json:"refresh_token"`
	Openid       string `json:"openid"`
	Scope        string `json:"scope"`
}

func GetAccessToken(ctx context.Context, code string) (*AccessToken, error) {
	getContent := fmt.Sprintf("%vappid=%v&secret=%v&code=%v&grant_type=authorization_code", WXPAY_ACCESS_TOKEN_URL, WXPAY_APPID, WXPAY_APPKEY, code)
	rsp, err := http.Get(getContent)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}
	hlog.CtxInfof(ctx, "http rsp:%v", string(body))
	accessToken := new(AccessToken)
	err = json.Unmarshal(body, accessToken)
	if err != nil {
		return nil, err
	}
	return accessToken, nil
}

type ShareAccessToken struct {
	AccessToken string `json:"access_token"`
	ExpireIn    int    `json:"expire_in"`
}
type WxTicket struct {
	Errcode     int    `json:"errcode"`
	Errmsg      string `json:"errmsg"`
	Ticket      string `json:"ticket"`
	ExpiresIn   int    `json:"expires_in"`
	ExpireStamp int64
}

var ticket WxTicket

func GetTicket(ctx context.Context) string {
	hlog.CtxInfof(ctx, "[GetTicket] ticket:%v", ticket)
	if ticket.Ticket != "" && ticket.ExpireStamp > time.Now().Unix()-5 {
		return ticket.Ticket
	}
	tokenContent := fmt.Sprintf("%v?grant_type=client_credential&appid=%v&secret=%v", WX_TOKEN_URL, WXPAY_APPID, WXPAY_APPKEY)
	rsp, err := http.Get(tokenContent)
	if err != nil {
		hlog.CtxErrorf(ctx, "http.Get failed,err:%v", err)
		return ""
	}

	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		hlog.CtxErrorf(ctx, "ioutil.ReadAll failed,err:%v", err)
		return ""
	}
	hlog.CtxInfof(ctx, "get token http rsp:%v", string(body))
	token := new(ShareAccessToken)
	err = json.Unmarshal(body, token)
	if err != nil {
		hlog.CtxErrorf(ctx, "json.Unmarshal failed,err:%v", err)
		return ""
	}

	ticketContent := fmt.Sprintf("%v?access_token=%v&type=jsapi", WX_TICKET_URL, token.AccessToken)
	rsp, err = http.Get(ticketContent)
	if err != nil {
		hlog.CtxErrorf(ctx, "http.Get failed,err:%v", err)
		return ""
	}
	body, err = ioutil.ReadAll(rsp.Body)
	if err != nil {
		hlog.CtxErrorf(ctx, "ioutil.ReadAll failed,err:%v", err)
		return ""
	}
	hlog.CtxInfof(ctx, "get ticket rspBody:%v", string(body))
	err = json.Unmarshal(body, &ticket)
	if err != nil {
		hlog.CtxErrorf(ctx, "json.Unmarshal failed,err:%v\n", err)
		return ""
	}
	ticket.ExpireStamp = time.Now().Unix() + int64(ticket.ExpiresIn)
	return ticket.Ticket

}
