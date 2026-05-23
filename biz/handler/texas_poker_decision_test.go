package handler

import (
	"context"
	"github.com/kingjxu/ddbaby/util"
	"testing"
	"time"

	"github.com/kingjxu/ddbaby/biz/model/ddbaby"
	"github.com/stretchr/testify/assert"
)

func TestTexasPokerDecisionHandler_Handle(t *testing.T) {

	t.Run("一次正常的调用", func(t *testing.T) {
		reqs := []*ddbaby.TexasPokerDecisionReq{
			{
				Images:    []string{"/Users/bytedance/go/src/ddbaby/conf/poker_images/111.jpg"},
				ImageType: util.Ptr("file_url"),
				UserID:    util.Ptr("123456"),
				ImageTime: util.Ptr(time.Now().Unix()),
			},
			{
				Images:    []string{"/Users/bytedance/go/src/ddbaby/conf/poker_images/222.jpg"},
				ImageType: util.Ptr("file_url"),
				UserID:    util.Ptr("123456"),
				ImageTime: util.Ptr(time.Now().Unix()),
			},
		}
		for _, req := range reqs {
			handler := NewTexasPokerDecisionHandler(req)
			_, err := handler.Handle(context.Background())
			t.Log("--------------------------------\n")
			t.Log("--------------------------------\n")
			assert.NoError(t, err)
		}
	})

}
