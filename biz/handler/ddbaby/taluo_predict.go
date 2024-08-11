package ddbaby

import (
	"context"
	"errors"
	"github.com/kingjxu/ddbaby/biz/model/ddbaby"
	"github.com/kingjxu/ddbaby/service"
	"github.com/kingjxu/ddbaby/util"
	"github.com/sirupsen/logrus"
)

type TaLuoPredictHandler struct {
	req     *ddbaby.TaLuoPredictReq
	Explain string
}

func NewTaLuoPredictHandler(req *ddbaby.TaLuoPredictReq) *TaLuoPredictHandler {
	return &TaLuoPredictHandler{
		req: req,
	}
}

func (h *TaLuoPredictHandler) check() error {
	if h.req.GetQuery() == "" {
		return errors.New("query is empty")
	}
	return nil
}

func (h *TaLuoPredictHandler) Handle(ctx context.Context) (*ddbaby.TaLuoPredictResp, error) {
	logrus.WithContext(ctx).Infof("[TaLuoPredictHandler] req:%v", util.ToJSON(h.req))
	if err := h.check(); err != nil {
		logrus.WithContext(ctx).Errorf("[TaLuoPredictHandler] check err:%v", err)
		return h.newResp(ctx, -1, "param err"), nil
	}
	content, err := service.TaLuoPredict(ctx, h.req.GetQuery())
	if err != nil {
		logrus.WithContext(ctx).Errorf("[TaLuoPredictHandler] ta luo predict err:%v", err)
		return h.newResp(ctx, -2, "predict err"), nil
	}
	h.Explain = content
	return h.newResp(ctx, 0, ""), nil
}

func (h *TaLuoPredictHandler) newResp(ctx context.Context, code int32, msg string) *ddbaby.TaLuoPredictResp {
	resp := &ddbaby.TaLuoPredictResp{
		BaseResp: &ddbaby.BaseResp{
			StatusMessage: msg,
			StatusCode:    code,
		},
	}
	resp.Explain = &h.Explain

	return resp
}
