---
version: "1.2"
last_updated: "2026-10-15"
---

# 测试规则

- 项目声明 Go 1.20；本地验证使用 `GOTOOLCHAIN=go1.20.14 go test ./... -count=1`。
- 缓存与座位转换的快速回归使用 `go test -race ./dal ./model -count=1`。
- `biz/handler/texas_poker_decision_test.go` 依赖 MySQL、识别服务、决策服务和本地图片，属于集成测试；仅在 `DDBABY_INTEGRATION_TEST=1` 时运行。
- 座位相关改动必须覆盖：8/9 人桌、单/多空座、waiting、fold、按钮位为空、强制位置连续顺延，以及 `buildPlayers`、`actionSeats` 与 `buildActionHistory` 的位置一致性。
- `action_history` 改动必须覆盖连续 Hero 决策之间的完整行动圈：多名对手依次行动、3bet/4bet、等额 call、短码 all-in、fold、同街第二圈以及跨街补全。跨入 flop 时，如果翻前无人加注超过最大强制盲注，还必须记录 BB 或 straddle 最后的 option check。校验目标下注额和顺时针行动顺序，不得用 stack/pot 差额补造图上未识别出的下注。
- 翻后全员过牌的跨街补全必须覆盖 Hero 位于首位、中间和末位：Hero 行动帧应先补齐其之前 active 玩家已经发生的 check，decided 帧再记录 Hero check；跨街时每人只出现一次。若 check 后发生新 bet/raise，先前 check 不能阻止该玩家在新行动圈继续 call/fold/raise。
- `players.action_taken` 改动必须同时覆盖首圈兼容和同街第二圈：decided 帧应在合法免费过牌时记录 Hero check；下一次回到 Hero 时必须以该 Hero 动作为边界，从累计历史还原对手的 bet/call/raise/fold，不能把等额筹码中座位靠前的 caller 误判为 bettor。Hero 面对未跟注下注时不得因 turn 标记消失而补造 check。
- 手牌边界不能只依赖 BTN 移动。连续完整识别出现 BTN 变化、街道回退，或两张完整 Hero 手牌变化时，必须清空上一手缓存和 `action_history`；Hero 手牌顺序交换不算新手牌。调用 GTO 前还要拒绝未来街道、街道回退或时间戳不递增的历史，避免跨手牌污染进入决策服务。
- 调用 GTO 前必须校验当前 Hero 恰有两张合法且互不重复的明牌，并按街校验公共牌数量（preflop 0、flop 3、turn 4、river 5）；Hero 与公共牌之间也不得重复。牌背、漏牌、非法牌或重复牌必须在 ddbaby 拒绝，不能传给决策服务。
- 决策响应校验必须覆盖翻前强制盲注未进入 `players.bet` 的 open raise：即使所有玩家当前投入为0，合法的正数 raise 目标也必须放行；0目标、未超过当前最大投入或超过 Hero 可用总额仍必须拒绝。
