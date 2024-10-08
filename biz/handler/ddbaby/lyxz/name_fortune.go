package lyxz

import (
	"context"
	"errors"
	"github.com/kingjxu/ddbaby/biz/model/ddbaby"
	"github.com/kingjxu/ddbaby/service"
	"github.com/kingjxu/ddbaby/util"
	"github.com/sirupsen/logrus"
)

type NameFortuneHandler struct {
	req     *ddbaby.NameFortuneReq
	Explain string
}

func NewNameFortuneHandler(req *ddbaby.NameFortuneReq) *NameFortuneHandler {
	return &NameFortuneHandler{
		req: req,
	}
}

func (h *NameFortuneHandler) check() error {
	if h.req.GetName() == "" {
		return errors.New("dream is empty")
	}
	return nil
}

func (h *NameFortuneHandler) Handle(ctx context.Context) (*ddbaby.DreamExplainResp, error) {
	logrus.WithContext(ctx).Infof("[NameFortuneHandler] req:%v", util.ToJSON(h.req))
	if err := h.check(); err != nil {
		logrus.WithContext(ctx).Errorf("[NameFortuneHandler] check err:%v", err)
		return h.newResp(ctx, -1, "param err"), nil
	}
	content, err := service.GetNameFortune(ctx, h.req.GetName())
	if err != nil {
		logrus.WithContext(ctx).Errorf("[NameFortuneHandler] get name fortune err:%v", err)
		return h.newResp(ctx, -2, "get name fortune err"), nil
	}
	h.Explain = content
	return h.newResp(ctx, 0, ""), nil
}

func (h *NameFortuneHandler) newResp(ctx context.Context, code int32, msg string) *ddbaby.DreamExplainResp {
	resp := &ddbaby.DreamExplainResp{
		BaseResp: &ddbaby.BaseResp{
			StatusMessage: msg,
			StatusCode:    code,
		},
	}
	resp.Explain = &h.Explain

	return resp
}
