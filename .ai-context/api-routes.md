---
version: "1.4"
last_updated: "2026-10-17"
---

# API 路由

## `POST /lyxz/gto_decision`

客户端提交一张屏幕截图，服务端完成授权校验、牌桌识别、状态缓存和 GTO 决策。JSON 请求使用 `images`、`timestamp`、`uuid`、可选 `image_type`，并支持可选整数 `vpip`、`pfr`；老客户端缺少两个策略字段时按 `0/0` 处理。响应继续使用 `Result`、`HeroCard`、`CommunityCard`。

`vpip/pfr=0/0` 保持原 `/gto/decision` 上游链路。非默认策略只接受客户端设置中列出的六组固定组合，随后调用上游 `/gto/decision_v2`；上游 v2 的 VPIP 字段按其契约拼写为 `vvip`，ddbaby 将客户端 `vpip` 映射为上游 `vvip`。任意其他组合在授权和识别前拒绝，不允许外部请求构造任意策略参数。

### 可选时延追踪头

- `X-GTO-Trace-ID`：客户端生成的非身份 trace id，只允许 1–64 位字母、数字、`-_.:`。
- `X-GTO-Client-Stages`：当前帧在客户端的单调时钟阶段值，供服务端计算成帧、最新帧队列、JPEG 和发起 HTTP 前耗时。
- `X-GTO-Previous-Trace`：上一条已完成悬浮窗展示的追踪数据，随下一次正常决策请求回传，不创建并发遥测请求。
- 响应 `X-GTO-Trace-ID`：回显被服务端接受的 trace id。
- 响应 `X-GTO-Server-Timing`：服务端 bind、校验/授权、识别、请求准备、GTO 决策和总耗时。

所有追踪头都是可选的。没有追踪头的旧客户端保持原行为，也不会产生逐请求追踪日志。服务端对外部头做长度、字符、字段数、非负值、单调性和最大 60 秒窗口校验；无效追踪数据不进入统计，也不改变业务决策。

客户端完整展示追踪由下一次请求携带，因此最后一次结果在停止录制且不再发请求时可能只保留在本机 `TRACE_COMPLETE` 日志中。服务端阶段使用自己的时钟，客户端阶段使用 `elapsedRealtime`，不依赖设备与服务器墙上时钟同步。

1.0.27 客户端在 `X-GTO-Client-Stages` 和 `X-GTO-Previous-Trace` 的 v2 格式中附加随机 `decision_cycle_id`、Hero 控件首次检测时间和 cycle 起始/首次展示标记。服务端对 cycle id、字段数量、时间顺序、布尔值及 60 秒窗口做同等校验；同一 cycle 仅首次实际展示写入 `client_complete`，后续重复结果写入 `client_complete_repeat`。没有 cycle 的 v1 头继续按原格式解析。

### 免费 check 兜底

决策服务返回合法 `fold` 后，ddbaby 会比较 Hero 当前投入与本轮所有玩家的最大投入。Hero 无需补注时，服务端将最终动作规范化为 `check`，并通过原有 JSON `Result` 返回“过牌”；面对更高下注时仍返回“弃牌”。客户端无需解析额外响应头或实现动作兜底。

### All in 展示兜底

决策服务返回合法 `bet` 或 `raise` 后，ddbaby 使用同一份已校验请求计算 Hero 的全下目标总额 `hero.bet + hero.stack`。当决策目标总额达到该上限时，原有 JSON `Result` 直接返回 `All in`；未达到时继续返回原有“下注 N”或“加注 N”。响应协议不增加字段，客户端无需新增判断。

## 其他路由

本上下文文件目前只覆盖德州扑克决策热路径；其他路由以 Hertz 生成的 router 和 handler 源码为准。
