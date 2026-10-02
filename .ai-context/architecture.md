---
version: "1.0"
last_updated: "2026-10-02"
---

# 架构

## 入口与层级

- `main.go` 创建 Hertz 服务并初始化 MySQL。
- `biz/router/ddbaby/ddbaby.go` 注册 `POST /lyxz/gto_decision`。
- `biz/handler/texas_poker_decision.go` 负责校验、识别、缓存和决策编排。
- `service/texas_poker.go` 调用外部识别服务与 GTO 决策服务。
- `dal/local_mem.go` 按用户保存同一手牌各阶段的识别结果。
- `model/recognize_poker.go` 把识别结果转换为玩家、位置和行动历史。

## 当前覆盖边界

本文只描述德州扑克决策热路径；支付、健康、塔罗等模块不在当前范围内。
