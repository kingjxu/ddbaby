package handler

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kingjxu/ddbaby/biz/model/ddbaby"
	"github.com/kingjxu/ddbaby/util"
	"github.com/stretchr/testify/assert"
	"os"
)

func TestLockTexasPokerUserSerializesSameUUID(t *testing.T) {
	const workers = 16
	start := make(chan struct{})
	var active int32
	var maxActive int32
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			unlock := lockTexasPokerUser("same-user")
			current := atomic.AddInt32(&active, 1)
			for {
				maximum := atomic.LoadInt32(&maxActive)
				if current <= maximum || atomic.CompareAndSwapInt32(&maxActive, maximum, current) {
					break
				}
			}
			time.Sleep(time.Millisecond)
			atomic.AddInt32(&active, -1)
			unlock()
		}()
	}

	close(start)
	wg.Wait()
	assert.Equal(t, int32(1), maxActive)
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
