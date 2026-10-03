---
version: "1.1"
last_updated: "2026-10-03"
---

# 测试规则

- 项目声明 Go 1.20；本地验证使用 `GOTOOLCHAIN=go1.20.14 go test ./... -count=1`。
- 缓存与座位转换的快速回归使用 `go test -race ./dal ./model -count=1`。
- `biz/handler/texas_poker_decision_test.go` 依赖 MySQL、识别服务、决策服务和本地图片，属于集成测试；仅在 `DDBABY_INTEGRATION_TEST=1` 时运行。
- 座位相关改动必须覆盖：8/9 人桌、单/多空座、waiting、fold、按钮位为空、强制位置连续顺延，以及 `buildPlayers`、`actionSeats` 与 `buildActionHistory` 的位置一致性。
- `action_history` 改动必须覆盖连续 Hero 决策之间的完整行动圈：多名对手依次行动、3bet/4bet、等额 call、短码 all-in、fold、同街第二圈以及跨街补全。校验目标下注额和顺时针行动顺序，不得用 stack/pot 差额补造图上未识别出的下注。
