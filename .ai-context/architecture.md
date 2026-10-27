---
version: "1.7"
last_updated: "2026-10-27"
---

# 架构

## 入口与层级

- `main.go` 创建 Hertz 服务并初始化 MySQL。
- `idl/ddbaby.thrift`统一维护`TexasPokerDecision`和`TexasPokerEvent`，生成的`biz/router/ddbaby/ddbaby.go`注册`POST /lyxz/gto_decision`与`POST /lyxz/gto_event`。`make generate`使用hz v0.9.1、thriftgo v0.4.2执行update（slim、use_type_alias=false）；`.hz`保存目录配置，禁止使用force new覆盖业务handler。动态事件JSON编解码和基础响应字段省略规则放在生成目录内的独立`poker_event_json.go`，重新生成不得覆盖该扩展文件。
- `biz/handler/texas_poker_decision.go` 负责校验、识别、缓存和决策编排。
- event入口与decision采用相同的`NewTexasPokerEventHandler → Handle → check/newResp`组织方式：HTTP层只读取/解析body、提取Bearer、调用Handler并写响应；Handler持有生成请求、原始body、token与接收时间，负责路由检查、完整日志、状态重建与决策编排，不依赖Hertz RequestContext。入口直接调用`c.BindAndValidate`解析IDL请求，不再保留`texasPokerEventAt`或手工best-effort解析；接收时间在入口记录。动态JSON字段仍由`PokerEventJSON`保留原始token，固定字段类型不符返回400。BaseResp.StatusCode为0时返回HTTP 200，非法入口固定400，决策临时失败固定503；业务校验/重试/回放语义保持不变。
- `service/texas_poker.go` 调用外部识别服务与 GTO 决策服务。
- `dal/local_mem.go` 按用户保存同一手牌各阶段的识别结果。
- `model/recognize_poker.go` 把识别结果转换为玩家、位置和行动历史。
- `biz/handler/texas_poker_event.go`通过IDL生成的请求/响应接收桃园助手透传的扑克事件并按packageName分流；`aapoker_event_state.go`和`dpzx_event_state.go`分别解释各自协议，生成同一`TexasGtoDecisionReq`。`aapoker_event_decision.go`复用身份认证、激活校验、策略调用、计费和`gto_decision_record`审计写入。临时失败返回503供客户端重传；成功的完整响应按隔离后的会话、牌桌和行动点缓存30分钟、每种协议最多1024条，重复事件只回放而不重复副作用。

## 当前覆盖边界

本文只描述德州扑克决策热路径；支付、健康、塔罗等模块不在当前范围内。
