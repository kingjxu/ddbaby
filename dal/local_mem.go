package dal

import (
	"context"
	"sync"

	"github.com/kingjxu/ddbaby/model"
	"github.com/sirupsen/logrus"
)

const (
	maxUserDataCount = 2000
)

var userDataMap = make(map[string][]*model.TexasResult)
var userDataMu sync.Mutex
var userNeedParseAllMap = make(map[string]bool)
var userNeedParseAllMu sync.Mutex

func SaveUserData(ctx context.Context, uid string, result *model.TexasResult) error {
	userDataMu.Lock()
	defer userDataMu.Unlock()

	if len(userDataMap[uid]) > maxUserDataCount {
		userDataMap[uid] = userDataMap[uid][maxUserDataCount/2:]
	}
	userData := userDataMap[uid]
	if len(userData) == 0 {
		model.PrepareActionHistory(nil, result)
		userDataMap[uid] = append(userData, result)
		return nil
	}
	lastResult := userData[len(userData)-1]
	model.PrepareActionHistory(lastResult, result)
	if !model.IsNewHand(lastResult, result) &&
		lastResult.TableInfo.Stage == result.TableInfo.Stage { // 同一手的同阶段，更新
		userData[len(userData)-1] = result
		return nil
	}
	userDataMap[uid] = append(userData, result)
	return nil
}

func GetLastUserData(ctx context.Context, uid string) ([]*model.TexasResult, error) {
	userDataMu.Lock()
	defer userDataMu.Unlock()

	userData, ok := userDataMap[uid]
	if !ok {
		logrus.WithContext(ctx).Errorf("user data not exist, uid: %s", uid)
		return nil, nil
	}
	preFlopIndex := 0
	for i := len(userData) - 1; i >= 0; i-- {
		if i > 0 && model.IsNewHand(userData[i-1], userData[i]) {
			preFlopIndex = i
			break
		}
	}
	return userData[preFlopIndex:], nil
}

func SetNeedParseAll(uid string, parseAll bool) {
	userNeedParseAllMu.Lock()
	defer userNeedParseAllMu.Unlock()
	userNeedParseAllMap[uid] = parseAll
}

func GetNeedParseAll(uid string) bool {
	userNeedParseAllMu.Lock()
	defer userNeedParseAllMu.Unlock()
	return userNeedParseAllMap[uid]
}
