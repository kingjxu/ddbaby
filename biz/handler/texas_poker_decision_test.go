package handler

import (
	"context"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/kingjxu/ddbaby/biz/model/ddbaby"
	"github.com/kingjxu/ddbaby/util"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

func TestTexasPokerDecisionHandlerWriteDecisionHeaders(t *testing.T) {
	tests := []struct {
		name       string
		handler    *TexasPokerDecisionHandler
		wantAction string
		wantCheck  string
	}{
		{name: "no decision", handler: &TexasPokerDecisionHandler{}},
		{name: "fold facing bet", handler: &TexasPokerDecisionHandler{HasDecision: true, Action: "fold"}, wantAction: "fold"},
		{name: "free fold", handler: &TexasPokerDecisionHandler{HasDecision: true, Action: "fold", CanCheck: true}, wantAction: "fold", wantCheck: "1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := app.NewContext(0)
			tt.handler.writeDecisionHeaders(ctx)
			if got := string(ctx.Response.Header.Peek(decisionActionHeader)); got != tt.wantAction {
				t.Fatalf("action header = %q, want %q", got, tt.wantAction)
			}
			if got := string(ctx.Response.Header.Peek(canCheckHeader)); got != tt.wantCheck {
				t.Fatalf("can-check header = %q, want %q", got, tt.wantCheck)
			}
		})
	}
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
