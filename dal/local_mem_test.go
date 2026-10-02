package dal

import (
	"context"
	"encoding/json"
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

func pokerCacheResult(t *testing.T, seat4Bet, seat5Bet int) *model.TexasResult {
	t.Helper()
	villains := make([]map[string]interface{}, 0, 7)
	for seat := 1; seat <= 7; seat++ {
		bet := 0
		switch seat {
		case 1:
			bet = 1
		case 2:
			bet = 2
		case 3:
			bet = 4
		case 4:
			bet = seat4Bet
		case 5:
			bet = seat5Bet
		}
		villains = append(villains, map[string]interface{}{
			"seat": seat, "status": "active", "stack": 1000, "current_bet": bet,
		})
	}
	payload := map[string]interface{}{
		"table_info": map[string]interface{}{
			"button_seat": 8, "stage": "preflop", "blind_size": "1/2/4(2)", "main_pot": 100,
		},
		"hero_info":     map[string]interface{}{"seat": 8, "status": "active", "stack": 1000},
		"villains_info": villains,
	}
	data, err := json.Marshal(payload)
	require.NoError(t, err)
	var result model.TexasResult
	require.NoError(t, json.Unmarshal(data, &result))
	return &result
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

func TestSaveUserDataKeepsActionsWhileReplacingSameStageSnapshot(t *testing.T) {
	resetLocalMemory()
	t.Cleanup(resetLocalMemory)

	ctx := context.Background()
	uid := "same-stage-actions"
	require.NoError(t, SaveUserData(ctx, uid, pokerCacheResult(t, 0, 0)))
	require.NoError(t, SaveUserData(ctx, uid, pokerCacheResult(t, 12, 0)))
	require.NoError(t, SaveUserData(ctx, uid, pokerCacheResult(t, 12, 40)))

	history, err := GetLastUserData(ctx, uid)
	require.NoError(t, err)
	require.Len(t, history, 1, "same-stage snapshots should still be replaced")
	require.Len(t, history[0].ObservedActionHistory, 2)
	require.Equal(t, "UTG", history[0].ObservedActionHistory[0].Position)
	require.Equal(t, "raise", history[0].ObservedActionHistory[0].Action)
	require.Equal(t, 12, history[0].ObservedActionHistory[0].Amount)
	require.Equal(t, "UTG+1", history[0].ObservedActionHistory[1].Position)
	require.Equal(t, "raise", history[0].ObservedActionHistory[1].Action)
	require.Equal(t, 40, history[0].ObservedActionHistory[1].Amount)
}
