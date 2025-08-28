package service

import (
	"context"
	"github.com/kingjxu/ddbaby/dal/mysql/jk"
)

func GetJkQoInfo(ctx context.Context, jkType string) ([]*jk.JkQo, error) {
	return jk.GetQuestionsByType(ctx, jkType)
}
