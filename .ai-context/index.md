---
version: "1.0"
last_updated: "2026-10-02"
---

# ddbaby 项目上下文

## Scope

当前只覆盖德州扑克图片识别结果到 GTO 决策请求的链路，以及本地手牌缓存。其他业务模块尚未整理。

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

Context v1.0，生成于 2026-10-02。

## Pending Files

- [x] `testing-rules.md`
- [ ] `env-and-ops.md`
- [ ] `api-routes.md`
- [ ] `data-model.md`
- [ ] `troubleshooting.md`
