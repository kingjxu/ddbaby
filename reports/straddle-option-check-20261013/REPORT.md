# 翻前 straddle option check 补全

## 问题

2026-10-13 00:47:18 的 J♦T♣ 手牌进入 flop 时，请求顶层仍有
`straddle=4`，`players` 也包含 active 的 straddle，但 `action_history`
遗漏了翻前 straddle 的最后一次 `check`。

原始识别无误：盲注为 `1/2/4(2)`，Hero 在 BB 从 2 跟到 4，straddle
仍以 4 筹码进入 flop。遗漏发生在跨街历史补全：旧逻辑只在
`accounted < stageMax` 时补 call，或 `stageMax == 0` 时补 check；straddle
满足 `accounted == stageMax == 4`，因此没有生成动作。

## 修复

跨街补全时，如果当前是 preflop、某位置的强制盲注等于本街最大投入，
且该位置没有已经记录的主动动作，则补一条 amount=0 的 `check`。在有
straddle 的牌局中该位置是 straddle；没有 straddle 时对应 BB option。

## 验证

- 以 00:47:18 真实 J♦T♣ 的座位、状态和投入构造回归，确认 flop 请求包含
  `preflop / straddle / check / amount=0`。
- 同时覆盖通用 straddle option，以及既有跨街最小合法动作补全。
- Go 1.22.2：目标测试、`go test -race ./model ./dal -count=1`、
  `go test ./... -count=1` 和 `build.sh` 均通过。

## 部署

- 业务提交 `03b8a6b` 已推送到远端 `master`。
- 生产二进制 SHA256 为
  `7cea4efc9ea4e695078cc6c15b12c497e4bc846560b639d0395eacf9a3641fc5`；
  部署后 PID `3924762`、PPID `1`，运行中 `/proc/3924762/exe` 与磁盘文件
  SHA256 一致，端口 8888 正常监听，`/ping` 返回 `{"message":"pong"}`。
- 本次只部署已完成本地回归和 CR 的候选，没有在线上执行全量回归或注入模拟牌局。
  启动后的日志没有新增业务错误。
- 回滚目录：`/home/ubuntu/ddbaby-backups/20261013-133505-before-03b8a6b`。
