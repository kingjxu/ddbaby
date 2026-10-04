---
version: "1.0"
last_updated: "2026-10-04"
---

# API 路由

## `POST /lyxz/gto_decision`

客户端提交一张屏幕截图，服务端完成授权校验、牌桌识别、状态缓存和 GTO 决策。JSON 请求继续使用 `images`、`timestamp`、`uuid` 和可选 `image_type`；响应继续使用 `Result`、`HeroCard`、`CommunityCard`，现有客户端协议不变。

### 可选时延追踪头

- `X-GTO-Trace-ID`：客户端生成的非身份 trace id，只允许 1–64 位字母、数字、`-_.:`。
- `X-GTO-Client-Stages`：当前帧在客户端的单调时钟阶段值，供服务端计算成帧、最新帧队列、JPEG 和发起 HTTP 前耗时。
- `X-GTO-Previous-Trace`：上一条已完成悬浮窗展示的追踪数据，随下一次正常决策请求回传，不创建并发遥测请求。
- 响应 `X-GTO-Trace-ID`：回显被服务端接受的 trace id。
- 响应 `X-GTO-Server-Timing`：服务端 bind、校验/授权、识别、请求准备、GTO 决策和总耗时。

所有追踪头都是可选的。没有追踪头的旧客户端保持原行为，也不会产生逐请求追踪日志。服务端对外部头做长度、字符、字段数、非负值、单调性和最大 60 秒窗口校验；无效追踪数据不进入统计，也不改变业务决策。

客户端完整展示追踪由下一次请求携带，因此最后一次结果在停止录制且不再发请求时可能只保留在本机 `TRACE_COMPLETE` 日志中。服务端阶段使用自己的时钟，客户端阶段使用 `elapsedRealtime`，不依赖设备与服务器墙上时钟同步。

## 其他路由

本上下文文件目前只覆盖德州扑克决策热路径；其他路由以 Hertz 生成的 router 和 handler 源码为准。
