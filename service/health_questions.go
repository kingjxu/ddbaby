package service

import (
	"context"
	"github.com/kingjxu/ddbaby/dal/mysql"
	"github.com/kingjxu/ddbaby/dal/mysql/health_evaluate"
)

func GetHealthQuestions(ctx context.Context, questionType string) ([]*health_evaluate.HealthEvaluationQuestions, error) {
	var questions []*health_evaluate.HealthEvaluationQuestions
	err := mysql.GetDB(ctx).Where("question_type = ?", questionType).Find(&questions).Error
	return questions, err
}
