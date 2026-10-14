# action_taken 同街第二圈修复报告

日期：2026-10-14

## 问题与证据

2026-10-13 10:04 的 K9 flop 牌局中，Hero 首次行动为 check，随后 BTN bet 84、SB call 84、BB fold，再次轮到 Hero。历史请求却把 `players.action_taken` 生成为 `SB=bet`、`BB=fold`、`BTN=""`。

旧实现只看当前快照，从 SB 开始按座位和最终投入依次推断，且遍历到 Hero 立即停止。它默认当前请求是本街第一次轮到 Hero；同街第二圈中，SB 与 BTN 都投入 84 时无法判断谁先下注，因而把座位靠前的 SB 错当成 bettor，并漏掉 Hero 后面的 BTN。

原始证据来自 `/data00/home/xujin.xj/outputs/hero-decision-audit-20261013-0700/audit.json`：

- 10:04:20 flop 首次轮到 Hero：Hero、BTN、SB、BB 的 `current_bet` 均为 0，响应 check。
- 10:04:36 再次轮到 Hero：BTN=84、SB=84、BB=fold、Hero=0；旧请求为 `SB=bet`、`BTN=""`。
- 同桌位置为 BTN seat7、SB seat8、BB seat9、straddle-Hero seat1。

## 修复

1. 当 Hero 可行动帧进入紧随其后的 decided 帧，且 Hero 仍 active、投入未变化、此前没有未跟注金额时，累计历史补记 Hero check。翻前只允许 BB/straddle 的合法 option check，避免把面对下注后的 turn 消失误记为 check。
2. 同一街已经存在 Hero 动作且之后观察到对手动作时，`players.action_taken` 使用该 Hero 动作之后的累计历史。保留首圈的旧推断作为兼容兜底。
3. 不采用简单删除“遍历到 Hero 停止”的方案，避免首圈把尚未行动的 Hero 后位玩家误判为已经行动。

K9 回归后的关键结果为：

- action history 尾部：`Hero check -> BTN bet 84 -> SB call 84 -> BB fold`。
- players：`BTN.action_taken=bet`、`SB.action_taken=call`、`BB.action_taken=fold`。

## 验证

- `go test ./model ./dal`：通过，共 57 个顶层测试套件。
- `go test -race ./model ./dal`：通过。
- `git diff --check`：通过。
- 新增现场数据回归：按 K9 的 9 人桌座位、筹码和 84/84/fold 状态验证 history 与 players。
- 新增边界回归：合法免费过牌才补 Hero check；Hero 面对未跟注下注时不补造 check。
- 原有多轮 3bet/4bet、短码 all-in、fold、跨街和 straddle option check 用例继续通过。

本机 `go test ./...` 受仓库既有 `sonic/loader v0.1.1` 与本机 Go 1.24 不兼容影响，在未涉及本次改动的依赖编译阶段失败；model/dal 目标包及 race 回归均已通过。部署前需在服务器既有 Go 1.20 构建环境完成全量构建和隔离回放。

## 风险与回滚

修复只在“当前街已有 Hero 动作且 Hero 后又观察到动作”时覆盖旧 `action_taken`，首圈行为不变。若线上出现异常，可回滚本次代码提交并重建二进制；不涉及数据库、协议或配置变更。
