package dal

import (
	"context"
	"github.com/kingjxu/ddbaby/model"
	"github.com/sirupsen/logrus"
)

const (
	maxUserDataCount = 2000
)

var userDataMap = make(map[string][]*model.TexasResult)

func SaveUserData(ctx context.Context, uid string, result *model.TexasResult) error {
	if len(userDataMap[uid]) > maxUserDataCount {
		userDataMap[uid] = userDataMap[uid][maxUserDataCount/2:]
	}
	userData := userDataMap[uid]
	if len(userData) == 0 {
		userDataMap[uid] = append(userData, result)
		return nil
	}
	if userData[len(userData)-1].TableInfo.Stage == result.TableInfo.Stage {
		userData[len(userData)-1] = result
	}
	userDataMap[uid] = append(userData, result)
	return nil
}

func GetLastUserData(ctx context.Context, uid string) ([]*model.TexasResult, error) {
	userData, ok := userDataMap[uid]
	if !ok {
		logrus.WithContext(ctx).Errorf("user data not exist, uid: %s", uid)
		return nil, nil
	}
	preFlopIndex := -1
	for i := len(userData) - 1; i >= 0; i-- {
		if userData[i].TableInfo.Stage == "preflop" {
			preFlopIndex = i
			break
		}
	}
	if preFlopIndex == -1 {
		logrus.WithContext(ctx).Errorf("user data not found preflop, uid: %s, len(userData): %v", uid, len(userData))
		return nil, nil
	}
	return userData[preFlopIndex:], nil
}
