---
version: "1.4"
last_updated: "2026-10-27"
---

# 架构

## 入口与层级

- `main.go` 创建 Hertz 服务并初始化 MySQL。
- `biz/router/ddbaby/ddbaby.go` 注册 `POST /lyxz/gto_decision`。
- `biz/handler/texas_poker_decision.go` 负责校验、识别、缓存和决策编排。
- `service/texas_poker.go` 调用外部识别服务与 GTO 决策服务。
- `dal/local_mem.go` 按用户保存同一手牌各阶段的识别结果。
- `model/recognize_poker.go` 把识别结果转换为玩家、位置和行动历史。
- `router.go` 的 `POST /lyxz/gto_event` 与 `biz/handler/texas_poker_event.go` 接收桃园助手透传的扑克事件并按packageName分流；`aapoker_event_state.go`和`dpzx_event_state.go`分别解释各自协议，生成同一`TexasGtoDecisionReq`。`aapoker_event_decision.go`复用身份认证、激活校验、策略调用、计费和`gto_decision_record`审计写入。临时失败返回503供客户端重传；成功的完整响应按隔离后的会话、牌桌和行动点缓存30分钟、每种协议最多1024条，重复事件只回放而不重复副作用。

## 当前覆盖边界

本文只描述德州扑克决策热路径；支付、健康、塔罗等模块不在当前范围内。
