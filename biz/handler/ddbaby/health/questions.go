package health

import (
	"context"
	"errors"
	"github.com/apache/thrift/lib/go/thrift"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	jsoniter "github.com/json-iterator/go"
	"github.com/kingjxu/ddbaby/biz/model/ddbaby"
	"github.com/kingjxu/ddbaby/service"
	"github.com/kingjxu/ddbaby/util"
	"github.com/sirupsen/logrus"
)

type HealthQuestionsHandler struct {
	req       *ddbaby.HealthEvaluateQuestionsReq
	questions []*ddbaby.HealthEvaluateQuestionItem
}

func NewHealthQuestionsHandler(req *ddbaby.HealthEvaluateQuestionsReq) *HealthQuestionsHandler {
	return &HealthQuestionsHandler{
		req: req,
	}
}

func (h *HealthQuestionsHandler) check() error {
	if h.req.GetQuestionType() == "" {
		return errors.New("category_name is empty")
	}
	return nil
}

func (h *HealthQuestionsHandler) Handle(ctx context.Context) (*ddbaby.HealthEvaluateQuestionsResp, error) {
	hlog.CtxInfof(ctx, "[HealthQuestionsHandler] req:%v", util.ToJSON(h.req))
	logrus.WithContext(ctx).Infof("[HealthQuestionsHandler] req:%v", util.ToJSON(h.req))
	if err := h.check(); err != nil {
		logrus.WithContext(ctx).Errorf("[HealthQuestionsHandler] check err:%v", err)
		hlog.CtxErrorf(ctx, "[HealthQuestionsHandler] check err:%v", err)
		return h.newResp(ctx, -1, "param err"), nil
	}
	questions, err := service.GetHealthQuestions(ctx, h.req.GetQuestionType())
	if err != nil {
		logrus.WithContext(ctx).Errorf("[HealthQuestionsHandler] get questions err:%v", err)
		hlog.CtxErrorf(ctx, "[HealthQuestionsHandler] get questions err:%v", err)
		return h.newResp(ctx, -2, "get questions err"), nil
	}
	for _, q := range questions {
		var options []string
		jsoniter.UnmarshalFromString(q.Options, &options)
		h.questions = append(h.questions, &ddbaby.HealthEvaluateQuestionItem{
			QuestionID: thrift.Int64Ptr(int64(q.ID)),
			Content:    thrift.StringPtr(q.Content),
			Options:    options,
		})
	}
	return h.newResp(ctx, 0, ""), nil
}

func (h *HealthQuestionsHandler) newResp(ctx context.Context, code int32, msg string) *ddbaby.HealthEvaluateQuestionsResp {
	resp := &ddbaby.HealthEvaluateQuestionsResp{
		BaseResp: &ddbaby.BaseResp{
			StatusMessage: msg,
			StatusCode:    code,
		},
	}
	resp.Questions = h.questions
	hlog.CtxInfof(ctx, "[HealthQuestionsHandler] resp:%v", util.ToJSON(resp))
	return resp
}
