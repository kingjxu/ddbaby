package model

import (
	"context"
	"encoding/json"
	"testing"
	"github.com/stretchr/testify/assert"
)

// TestBuildActionHistory 测试跨stage的情况（应该返回空）
func TestBuildActionHistory_CrossStage(t *testing.T) {
	input := `[{"received_at":"2026-05-23T08:22:42.726917Z","parsed_at":"2026-05-23T08:22:44.896693Z","elapsed_ms":2170,"app":"poler","table_info":{"stage":"preflop","community_cards":[],"main_pot":33,"button_seat":8,"blind_size":"1/2(2)"},"hero_info":{"seat":1,"status":"active","stack":438,"current_bet":1,"is_hero_turn":true,"hero_cards":["8 红桃","10 红桃"]},"villains_info":[{"seat":2,"status":"waiting","stack":0,"current_bet":0},{"seat":3,"status":"active","stack":313,"current_bet":2},{"seat":4,"status":"active","stack":560,"current_bet":4},{"seat":5,"status":"folded","stack":710,"current_bet":0},{"seat":6,"status":"active","stack":529,"current_bet":4},{"seat":7,"status":"active","stack":809,"current_bet":4},{"seat":8,"status":"active","stack":189,"current_bet":4}]},{"received_at":"2026-05-23T08:22:45.133023Z","parsed_at":"2026-05-23T08:22:49.440735Z","elapsed_ms":4308,"app":"poler","table_info":{"stage":"flop","community_cards":["4 黑桃","Q 红桃","2 黑桃"],"main_pot":38,"button_seat":8,"blind_size":"1/2(2)"},"hero_info":{"seat":1,"status":"active","stack":435,"current_bet":0,"is_hero_turn":true,"hero_cards":["8 红桃","10 红桃"]},"villains_info":[{"seat":2,"status":"waiting","stack":0,"current_bet":0},{"seat":3,"status":"active","stack":311,"current_bet":0},{"seat":4,"status":"active","stack":560,"current_bet":0},{"seat":5,"status":"folded","stack":710,"current_bet":0},{"seat":6,"status":"active","stack":529,"current_bet":0},{"seat":7,"status":"active","stack":809,"current_bet":0},{"seat":8,"status":"active","stack":189,"current_bet":0}]}]`

	var recResult []*TexasResult
	err := json.Unmarshal([]byte(input), &recResult)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(recResult))

	history := buildActionHistory(context.Background(), recResult)
	t.Logf("Generated history: %+v", history)
	// 跨stage的情况应该返回空
	assert.Empty(t, history)
}

// TestBuildActionHistory_SameStage 测试同一stage内有多个状态的情况
func TestBuildActionHistory_SameStage(t *testing.T) {
	input := `[{"received_at":"2026-05-23T08:22:42.726917Z","parsed_at":"2026-05-23T08:22:44.896693Z","elapsed_ms":2170,"app":"poler","table_info":{"stage":"preflop","community_cards":[],"main_pot":3,"button_seat":8,"blind_size":"1/2(2)"},"hero_info":{"seat":1,"status":"active","stack":440,"current_bet":0,"is_hero_turn":true,"hero_cards":["8 红桃","10 红桃"]},"villains_info":[{"seat":2,"status":"waiting","stack":0,"current_bet":0},{"seat":3,"status":"active","stack":315,"current_bet":0},{"seat":4,"status":"active","stack":560,"current_bet":0},{"seat":5,"status":"active","stack":710,"current_bet":0},{"seat":6,"status":"active","stack":529,"current_bet":0},{"seat":7,"status":"active","stack":809,"current_bet":0},{"seat":8,"status":"active","stack":191,"current_bet":0}]},{"received_at":"2026-05-23T08:22:43.726917Z","parsed_at":"2026-05-23T08:22:45.896693Z","elapsed_ms":2170,"app":"poler","table_info":{"stage":"preflop","community_cards":[],"main_pot":3,"button_seat":8,"blind_size":"1/2(2)"},"hero_info":{"seat":1,"status":"active","stack":439,"current_bet":1,"is_hero_turn":true,"hero_cards":["8 红桃","10 红桃"]},"villains_info":[{"seat":2,"status":"waiting","stack":0,"current_bet":0},{"seat":3,"status":"active","stack":313,"current_bet":2},{"seat":4,"status":"active","stack":560,"current_bet":0},{"seat":5,"status":"active","stack":710,"current_bet":0},{"seat":6,"status":"active","stack":529,"current_bet":0},{"seat":7,"status":"active","stack":809,"current_bet":0},{"seat":8,"status":"active","stack":189,"current_bet":0}]}]`

	var recResult []*TexasResult
	err := json.Unmarshal([]byte(input), &recResult)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(recResult))

	history := buildActionHistory(context.Background(), recResult)
	t.Logf("Generated history: %+v", history)
	// 同一stage内应该能检测到action
	assert.NotEmpty(t, history)
}
