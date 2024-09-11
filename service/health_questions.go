package service

import (
	"context"
	"github.com/kingjxu/ddbaby/dal/mysql/health_evaluate"
)

func GetHealthQuestions(ctx context.Context, questionType string) ([]*health_evaluate.HealthEvaluationQuestions, error) {
	return health_evaluate.GetQuestionsByType(ctx, questionType)
}
