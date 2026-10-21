---
version: "1.7"
last_updated: "2026-10-21"
---

# ddbaby 项目上下文

## Scope

当前覆盖德州扑克图片识别结果到 GTO 决策请求的链路、本地手牌缓存，以及AAPoker规范化事件首版接收入口。其他业务模块尚未整理。

## Reading Order

1. `architecture.md`
2. `business-glossary.md`
3. `flow-map.md`

## Task Router

| Task Type | Context Files to Load |
|---|---|
| 德州扑克识别、座位或行动历史 | `business-glossary.md`, `flow-map.md`, `architecture.md` |
| 缓存截取问题 | `flow-map.md`, `architecture.md` |
| Tests / CI | `testing-rules.md` |
| Build / Deploy / Env | `env-and-ops.md` |
| API / RPC changes | `api-routes.md`, `architecture.md` |
| Database / Schema changes | `data-model.md` |
| Bug / root cause unknown | `troubleshooting.md`, `flow-map.md` |

## Context Version

Context v1.7。在线截图链路默认使用轻量 Hero 门控；每个 Hero 行动帧只触发紧接着的一帧完整识别，随后恢复轻量检测。动作历史从这些有限快照推断并累计可观察差分；图上可见下注由 dezhou 直接识别，ddbaby 不从筹码或底池差额反推下注。可选时延追踪支持 `decision_cycle_id` 和首次展示口径，重复 Hero 结果单独记录；合法决策返回前异步写入 MySQL 审计表；旧 v1 客户端协议保持兼容。新增`POST /lyxz/gto_event`仅接收并确认AAPoker规范化事件，尚未接入状态重建或决策。

## Pending Files

- [x] `testing-rules.md`
- [ ] `env-and-ops.md`
- [x] `api-routes.md`
- [x] `data-model.md`
- [ ] `troubleshooting.md`
