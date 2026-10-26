---
version: "1.2"
last_updated: "2026-10-21"
---

# 架构

## 入口与层级

- `main.go` 创建 Hertz 服务并初始化 MySQL。
- `biz/router/ddbaby/ddbaby.go` 注册 `POST /lyxz/gto_decision`。
- `biz/handler/texas_poker_decision.go` 负责校验、识别、缓存和决策编排。
- `service/texas_poker.go` 调用外部识别服务与 GTO 决策服务。
- `dal/local_mem.go` 按用户保存同一手牌各阶段的识别结果。
- `model/recognize_poker.go` 把识别结果转换为玩家、位置和行动历史。
- `router.go` 的 `POST /lyxz/gto_event` 与 `biz/handler/texas_poker_event.go` 接收桃园助手透传的AAPoker规范化事件；`aapoker_event_state.go` 按桃园uuid、AAPoker uid和牌桌维护有界牌局状态并在Hero轮次生成同一GTO请求模型，`aapoker_event_decision.go`复用激活校验、策略调用、计费和`gto_decision_record`审计写入。

## 当前覆盖边界

本文只描述德州扑克决策热路径；支付、健康、塔罗等模块不在当前范围内。
