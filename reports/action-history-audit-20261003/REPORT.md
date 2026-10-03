# action_history 多轮行动审计

日期：2026-10-03

## 结论

本轮未发现当前实现对 `Hero → A → B → C → Hero` 行动圈、3bet/4bet 或同街多圈处理的新缺陷。新增回归覆盖单圈多次加注、短码 all-in、fold、Hero 再行动后的第二圈，以及切换到下一街时补齐上一街的最小合法动作。

本轮只增加测试与验证记录，没有修改生产逻辑，因此不需要重新部署服务。

## 实现口径

- 连续两个 Hero 决策之间，按牌局顺时针顺序，每个对手最多行动一次；快照中的累计目标下注额足以区分更高目标的 raise、等于当前最高额的 call、短于当前最高额的 all-in call，以及状态变化产生的 fold。
- 同一街再次回到 Hero 后，后续快照继续在既有 `ObservedActionHistory` 后追加，保留第一圈和第二圈的动作。
- 进入下一街时，`inferStageCompletion` 只补齐上一街仍未记录但由下一街状态证明已经发生的最小 call、check 或 fold。
- 精确下注额必须来自识别结果中的显式 `current_bet`。若中间截图缺失或 OCR 漏掉图上下注，系统只能输出现有证据支持的最小合法历史，禁止用 stack/pot 差额推算下注。

## 新增回归

1. `Hero raise 12 → SB raise 40 → BB call 40 → straddle raise 100 → Hero`，校验 3bet/4bet 顺序及 GTO 请求透传。
2. `Hero raise 12 → SB raise 40 → BB all-in call 25 → straddle raise 100`，校验短码 all-in 插入加注圈的顺序。
3. 第一圈结束后 `Hero call 100 → SB raise 200 → BB fold → straddle call 200 → Hero`，校验同街第二圈累计历史。
4. `Hero call 100` 后进入 flop，校验上一街按顺序补齐 `SB call 100 → BB fold`，且不会重复已有动作。

专项命令：

```bash
go test ./model -run 'TestPrepareActionHistory(ReconstructsHeroABCThreeBetFourBetOrbit|OrdersFoldAndShortAllInWithinRaiseOrbit|KeepsMultipleHeroOrbitsOnSameStreet|CompletesMultiRaiseOrbitOnNextStreet)' -count=1 -v
```

结果：4/4 通过。

完整验证：

- 本地 `go test ./model ./dal -count=1`：通过。
- 本地 `go test -race ./model ./dal -count=1`：通过。
- 服务器临时目录 Go 1.22.2 `go test ./... -count=1`：通过。
- 服务器临时目录 Go 1.22.2 `go test -race ./model ./dal -count=1`：通过。
- `git diff --check`：通过。

本机默认 Go 版本无法编译旧版 sonic，按项目文档切换 Go 1.20.14 时又因工具链缓存只读无法下载，因此全仓验证在服务器临时目录完成；这不是代码或测试失败。

## 线上日志审计

审计边界从当前生产版本 `c6cc23f` 的部署时间 `2026-10-03T11:18:47+08:00` 开始，避免把旧版本日志混入结论。

- 请求数：129
- 带 `action_history` 的请求：117
- 单街包含至少两次 raise 的历史：10
- 时间戳不连续、raise/bet 目标不递增、call 超过当前最高额、check/fold 金额非零或未知动作：0

该检查证明当前日志中的动作历史结构合法，不等同于逐张原图人工核对 OCR 语义。OCR 错值或漏值仍需用对应原图和识别结果单独回归。
