package handler

import (
	"context"
	"github.com/kingjxu/ddbaby/biz/model/ddbaby"
	"github.com/kingjxu/ddbaby/util"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestTexasPokerDecisionHandler_Handle(t *testing.T) {
	t.Run("全轮次的调用", func(t *testing.T) {
		reqs := []*ddbaby.TexasPokerDecisionReq{
			{
				Images:    []string{"/Users/bytedance/go/src/ddbaby/conf/badcase/1.jpg"},
				ImageType: util.Ptr("file_url"),
				UUID:      util.Ptr("123456"),
				Timestamp: util.Ptr(time.Now().Unix() * 1000),
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
