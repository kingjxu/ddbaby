---
version: "1.0"
last_updated: "2026-10-02"
---

# 测试规则

- 项目声明 Go 1.20；本地验证使用 `GOTOOLCHAIN=go1.20.14 go test ./... -count=1`。
- 缓存与座位转换的快速回归使用 `go test -race ./dal ./model -count=1`。
- `biz/handler/texas_poker_decision_test.go` 依赖 MySQL、识别服务、决策服务和本地图片，属于集成测试；仅在 `DDBABY_INTEGRATION_TEST=1` 时运行。
- 座位相关改动必须覆盖：8/9 人桌、单/多空座、waiting、fold，以及 `buildPlayers` 与 `buildActionHistory` 的位置一致性。
