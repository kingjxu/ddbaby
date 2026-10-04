# ddbaby 全链路时延追踪

## 目标

为客户端 3 秒以上体感时延建立可关联的分段证据，区分客户端排队/编码/网络、服务端授权/识别/请求构造/GTO 决策及悬浮窗展示耗时。

## 协议与实现

- `POST /lyxz/gto_decision` 可选接收 `X-GTO-Trace-ID`、`X-GTO-Client-Stages` 和 `X-GTO-Previous-Trace`。
- 响应回显 trace id，并通过 `X-GTO-Server-Timing` 返回 bind、授权、识别、请求准备、GTO 决策和服务端总耗时。
- 旧客户端不发送追踪头时行为不变，也不增加逐请求追踪日志。
- 所有外部追踪头均限制长度、字符、字段数、非负值、单调顺序、覆盖帧上限和 60 秒时间窗口；非法数据只丢弃追踪，不参与业务决策。
- 客户端已完成展示记录随下一次正常决策请求回传，不新增服务端接口或额外请求。

## 验证

- 远端 Go 1.22.2：`go test ./... -count=1` 通过。
- 远端 Go 1.22.2：`go test -race ./biz/handler ./model ./dal -count=1` 通过。
- 远端 Go 1.22.2：`bash build.sh` 通过。
- 追踪解析专项覆盖合法数据、非法 trace id、乱序时间、字段缺失、超时窗口和额外字段拒绝。
- `git diff --check`：通过。

## 当前判断与边界

现有生产证据只能确认服务端从识别开始到决策完成的均值约为 1.36 秒。串行上传意味着 Hero 帧可能在上一张图仍请求中时产生，随后再经历一轮服务端处理；这个组合可以解释 3 秒以上体感，但尚未由完整真实 trace 证实。部署服务端并让 1.0.26 产生真实请求后，应以 `queue_backlog_ms`、`http_ms`、`server_recognize_ms`、`server_decision_ms` 和 `response_to_display_ms` 分段确认实际瓶颈。

## 部署

- 2026-10-04 将 `97cd83d` 对应二进制和两份运行时 handler 源码原子替换到生产目录；上一版资产备份在 `/home/ubuntu/ddbaby-backups/20261004-185316-before-97cd83d`。
- 运行时二进制 SHA256 为 `74346e9563c52e0ad9f44a88cda328216e5abe4bcbc67341a00a2a8486bbaeca`，与远端 Go 1.22.2 构建产物一致。
- 部署后 PID=821094、PPID=1，8888 监听和 `/ping` 健康检查通过；smoke 请求验证 trace id 回显、服务端分段响应头和客户端阶段日志均生效。
- smoke 请求未带真实鉴权和图片，仅验证协议入口，不作为识别、决策或真实端到端时延样本。
