package handler

import (
	"context"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server/binding"
	"github.com/kingjxu/ddbaby/biz/model/ddbaby"
	"github.com/kingjxu/ddbaby/util"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

func TestTexasPokerDecisionRequestBindsLowercaseProfileFields(t *testing.T) {
	body := `{"images":["image"],"timestamp":1,"uuid":"account","vpip":42,"pfr":20}`
	var requestContext app.RequestContext
	requestContext.Request.Header.SetMethod("POST")
	requestContext.Request.Header.SetContentTypeBytes([]byte("application/json"))
	requestContext.Request.Header.SetContentLength(len(body))
	requestContext.Request.SetBodyString(body)

	var req ddbaby.TexasPokerDecisionReq
	assert.NoError(t, binding.DefaultBinder().BindAndValidate(
		&requestContext.Request, &req, nil))
	assert.Equal(t, int32(42), req.GetVPIP())
	assert.Equal(t, int32(20), req.GetPFR())
}

func TestTexasPokerDecisionHandler_Handle(t *testing.T) {
	if os.Getenv("DDBABY_INTEGRATION_TEST") != "1" {
		t.Skip("requires MySQL, recognition and decision services; set DDBABY_INTEGRATION_TEST=1 to run")
	}

	t.Run("全轮次的调用", func(t *testing.T) {
		reqs := []*ddbaby.TexasPokerDecisionReq{
			/*		{
						Images:    []string{"/Users/bytedance/go/src/ddbaby/conf/badcase/20.jpg"},
						ImageType: util.Ptr("file_url"),
						UUID:      util.Ptr("123456"),
						Timestamp: util.Ptr(time.Now().Unix() * 1000),
					},
					{
						Images:    []string{"/Users/bytedance/go/src/ddbaby/conf/badcase/22.jpg"},
						ImageType: util.Ptr("file_url"),
						UUID:      util.Ptr("123456"),
						Timestamp: util.Ptr(time.Now().Unix()*1000 + 5000),
					},*/
			{
				Images:    []string{"/Users/bytedance/go/src/ddbaby/conf/badcase/19.jpg"},
				ImageType: util.Ptr("file_url"),
				UUID:      util.Ptr("123456"),
				Timestamp: util.Ptr(time.Now().Unix()*1000 + 5000),
			},
		}
		for i, req := range reqs {
			handler := NewTexasPokerDecisionHandler(req)
			_, err := handler.Handle(context.Background())
			t.Logf("---------------- %d ----------------\n", i+1)
			t.Logf("-----------------%d ----------------\n", i+1)
			assert.NoError(t, err)
		}
	})

}

func TestTexasPokerDecisionHandlerRejectsUnsupportedDecisionProfile(t *testing.T) {
	now := time.Now().UnixMilli()
	tests := []struct {
		name    string
		vpip    int32
		pfr     int32
		wantErr bool
	}{
		{name: "legacy default"},
		{name: "42/20", vpip: 42, pfr: 20},
		{name: "33/15", vpip: 33, pfr: 15},
		{name: "54/11", vpip: 54, pfr: 11},
		{name: "38/14", vpip: 38, pfr: 14},
		{name: "40/10", vpip: 40, pfr: 10},
		{name: "55/25", vpip: 55, pfr: 25},
		{name: "only vpip", vpip: 55, wantErr: true},
		{name: "only pfr", pfr: 25, wantErr: true},
		{name: "unknown pair", vpip: 99, pfr: 99, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &ddbaby.TexasPokerDecisionReq{
				Images: []string{"image"}, Timestamp: util.Ptr(now),
				UUID: util.Ptr("account"), VPIP: util.Ptr(tt.vpip), PFR: util.Ptr(tt.pfr),
			}
			err := NewTexasPokerDecisionHandler(req).check()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
