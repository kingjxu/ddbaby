package ddbaby

import (
	"context"
	"errors"
	"github.com/kingjxu/ddbaby/biz/model/ddbaby"
	"github.com/sirupsen/logrus"
)

type DreamExplainHandler struct {
	req *ddbaby.DreamExplainReq
}

func NewDreamExplainHandler(req *ddbaby.DreamExplainReq) *DreamExplainHandler {
	return &DreamExplainHandler{
		req: req,
	}
}

func (h *DreamExplainHandler) check() error {
	if h.req.GetDream() == "" {
		return errors.New("dream is empty")
	}
	return nil
}

func (h *DreamExplainHandler) Handle(ctx context.Context) (*ddbaby.DreamExplainResp, error) {
	logrus.WithFields(logrus.Fields{"req": h.req})
	if err := h.check(); err != nil {
		logrus.WithFields(logrus.Fields{"err": err.Error()})
		return h.newResp(ctx, -1, "param err"), nil
	}

	return h.newResp(ctx, 0, ""), nil
}

func (h *DreamExplainHandler) newResp(ctx context.Context, code int32, msg string) *ddbaby.DreamExplainResp {
	return &ddbaby.DreamExplainResp{
		BaseResp: &ddbaby.BaseResp{
			StatusMessage: msg,
			StatusCode:    code,
		},
	}
}
