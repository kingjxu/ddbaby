package dal

import (
	"context"
	"testing"

	"github.com/kingjxu/ddbaby/model"
	"github.com/stretchr/testify/require"
)

func texasResult(buttonSeat int, stage string, mainPot int) *model.TexasResult {
	result := &model.TexasResult{}
	result.TableInfo.ButtonSeat = buttonSeat
	result.TableInfo.Stage = stage
	result.TableInfo.MainPot = mainPot
	return result
}

func resetLocalMemory() {
	userDataMap = make(map[string][]*model.TexasResult)
	userNeedParseAllMap = make(map[string]bool)
}

func TestGetLastUserDataKeepsAllStagesWhenCacheOnlyContainsCurrentHand(t *testing.T) {
	resetLocalMemory()
	t.Cleanup(resetLocalMemory)

	ctx := context.Background()
	uid := "current-hand-only"
	require.NoError(t, SaveUserData(ctx, uid, texasResult(7, "preflop", 90)))
	require.NoError(t, SaveUserData(ctx, uid, texasResult(7, "flop", 185)))

	history, err := GetLastUserData(ctx, uid)
	require.NoError(t, err)
	require.Len(t, history, 2)
	require.Equal(t, "preflop", history[0].TableInfo.Stage)
	require.Equal(t, "flop", history[1].TableInfo.Stage)
}

func TestSaveUserDataDoesNotReplaceSameStageFromPreviousHand(t *testing.T) {
	resetLocalMemory()
	t.Cleanup(resetLocalMemory)

	ctx := context.Background()
	uid := "successive-preflops"
	require.NoError(t, SaveUserData(ctx, uid, texasResult(6, "preflop", 145)))
	require.NoError(t, SaveUserData(ctx, uid, texasResult(7, "preflop", 90)))

	require.Len(t, userDataMap[uid], 2)
	history, err := GetLastUserData(ctx, uid)
	require.NoError(t, err)
	require.Len(t, history, 1)
	require.Equal(t, 7, history[0].TableInfo.ButtonSeat)
	require.Equal(t, 90, history[0].TableInfo.MainPot)
}

func TestSaveUserDataReplacesSameStageWithinCurrentHand(t *testing.T) {
	resetLocalMemory()
	t.Cleanup(resetLocalMemory)

	ctx := context.Background()
	uid := "same-stage-update"
	require.NoError(t, SaveUserData(ctx, uid, texasResult(7, "preflop", 90)))
	require.NoError(t, SaveUserData(ctx, uid, texasResult(7, "preflop", 165)))

	history, err := GetLastUserData(ctx, uid)
	require.NoError(t, err)
	require.Len(t, history, 1)
	require.Equal(t, 165, history[0].TableInfo.MainPot)
}

func TestGetLastUserDataReturnsStagesAfterPreviousButton(t *testing.T) {
	resetLocalMemory()
	t.Cleanup(resetLocalMemory)

	ctx := context.Background()
	uid := "multiple-hands"
	for _, result := range []*model.TexasResult{
		texasResult(6, "preflop", 145),
		texasResult(6, "flop", 300),
		texasResult(7, "preflop", 90),
		texasResult(7, "flop", 185),
	} {
		require.NoError(t, SaveUserData(ctx, uid, result))
	}

	history, err := GetLastUserData(ctx, uid)
	require.NoError(t, err)
	require.Len(t, history, 2)
	require.Equal(t, "preflop", history[0].TableInfo.Stage)
	require.Equal(t, "flop", history[1].TableInfo.Stage)
	require.Equal(t, 7, history[0].TableInfo.ButtonSeat)
}
