package lyxz

import (
	"context"
	"errors"
	"github.com/kingjxu/ddbaby/biz/model/ddbaby"
	"github.com/kingjxu/ddbaby/service"
	"github.com/kingjxu/ddbaby/util"
	"github.com/sirupsen/logrus"
)

type PickNameHandler struct {
	req     *ddbaby.PickNameReq
	Explain string
}

func NewPickNameHandler(req *ddbaby.PickNameReq) *PickNameHandler {
	return &PickNameHandler{
		req: req,
	}
}

func (h *PickNameHandler) check() error {
	if h.req.GetFamilyName() == "" {
		return errors.New("family name is empty")
	}
	return nil
}

func (h *PickNameHandler) Handle(ctx context.Context) (*ddbaby.DreamExplainResp, error) {
	logrus.WithContext(ctx).Infof("[NameFortuneHandler] req:%v", util.ToJSON(h.req))
	if err := h.check(); err != nil {
		logrus.WithContext(ctx).Errorf("[NameFortuneHandler] check err:%v", err)
		return h.newResp(ctx, -1, "param err"), nil
	}
	content, err := service.PickName(ctx, service.PickNameParam{
		FamilyName: h.req.GetFamilyName(),
		Gender:     h.req.GetGender(),
		NameLen:    int(h.req.GetNameLen()),
		Remark:     h.req.Remark,
	})
	if err != nil {
		logrus.WithContext(ctx).Errorf("[NameFortuneHandler] get name fortune err:%v", err)
		return h.newResp(ctx, -2, "get name fortune err"), nil
	}
	h.Explain = content
	return h.newResp(ctx, 0, ""), nil
}

func (h *PickNameHandler) newResp(ctx context.Context, code int32, msg string) *ddbaby.DreamExplainResp {
	resp := &ddbaby.DreamExplainResp{
		BaseResp: &ddbaby.BaseResp{
			StatusMessage: msg,
			StatusCode:    code,
		},
	}
	resp.Explain = &h.Explain

	return resp
}
