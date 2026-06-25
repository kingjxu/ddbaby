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
var userNeedParseAllMap = make(map[string]bool)

func SaveUserData(ctx context.Context, uid string, result *model.TexasResult) error {
	if len(userDataMap[uid]) > maxUserDataCount {
		userDataMap[uid] = userDataMap[uid][maxUserDataCount/2:]
	}
	userData := userDataMap[uid]
	if len(userData) == 0 {
		userDataMap[uid] = append(userData, result)
		return nil
	}
	if userData[len(userData)-1].TableInfo.Stage == result.TableInfo.Stage { // 同阶段，更新
		userData[len(userData)-1] = result
		return nil
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
	buttonSeat := userData[len(userData)-1].TableInfo.ButtonSeat
	preFlopIndex := len(userData) - 1
	for i := len(userData) - 1; i >= 0; i-- {
		if userData[i].TableInfo.ButtonSeat != buttonSeat {
			preFlopIndex = i + 1
			break
		}
	}
	return userData[preFlopIndex:], nil
}

func SetNeedParseAll(uid string, parseAll bool) {
	userNeedParseAllMap[uid] = parseAll
}
func GetNeedParseAll(uid string) bool {
	return userNeedParseAllMap[uid]
}
