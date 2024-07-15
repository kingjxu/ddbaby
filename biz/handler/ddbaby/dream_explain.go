package ddbaby

import (
	"context"
	"errors"
	"github.com/kingjxu/ddbaby/biz/model/ddbaby"
	"github.com/kingjxu/ddbaby/service"
	"github.com/sirupsen/logrus"
)

type DreamExplainHandler struct {
	req     *ddbaby.DreamExplainReq
	Explain string
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
	logrus.WithContext(ctx).Infof("[DreamExplainHandler] req:%v", h.req)
	if err := h.check(); err != nil {
		logrus.WithContext(ctx).Errorf("[DreamExplainHandler] check err:%v", err)
		return h.newResp(ctx, -1, "param err"), nil
	}
	content, err := service.GetDreamExplain(ctx, h.req.GetDream())
	if err != nil {
		logrus.WithContext(ctx).Errorf("[DreamExplainHandler] get dream explain err:%v", err)
		return h.newResp(ctx, -1, "get dream explain err"), nil
	}
	h.Explain = content
	return h.newResp(ctx, 0, ""), nil
}

func (h *DreamExplainHandler) newResp(ctx context.Context, code int32, msg string) *ddbaby.DreamExplainResp {
	resp := &ddbaby.DreamExplainResp{
		BaseResp: &ddbaby.BaseResp{
			StatusMessage: msg,
			StatusCode:    code,
		},
	}
	resp.Explain = &h.Explain

	return resp
}
