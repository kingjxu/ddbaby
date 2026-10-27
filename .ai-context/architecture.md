---
version: "1.15"
last_updated: "2026-10-27"
---

# 架构

## 入口与层级

- `main.go` 创建Hertz服务并初始化MySQL；所有路由统一由Hertz缓冲请求体并执行4 MiB上限，不开启WithStreamBody，也不再手工限流式读取。
- `idl/ddbaby.thrift`统一维护`TexasPokerDecision`和`TexasPokerEvent`，生成的`biz/router/ddbaby/ddbaby.go`注册`POST /lyxz/gto_decision`与`POST /lyxz/gto_event`。`make generate`使用hz v0.9.1、thriftgo v0.4.2执行update（slim、use_type_alias=false）；`.hz`保存目录配置，禁止使用force new覆盖业务handler。动态事件JSON编解码和基础响应字段省略规则放在生成目录内的独立`poker_event_json.go`，重新生成不得覆盖该扩展文件。
- `biz/handler/texas_poker_decision.go` 负责校验、识别、缓存和决策编排。
- event入口与decision采用相同的`BindAndValidate → NewTexasPokerEventHandler(req) → Handle → 响应`组织方式，HTTP入口和业务Handler保留在`texas_poker_event.go`。HTTP入口只绑定生成请求、补入原始body并映射响应；业务`Handle`在最前面的`check(ctx)`中完成来源校验及非空uuid数据库激活校验，随后调用service重建牌局并执行决策或结果回放，各失败分支提前返回，不依赖Hertz RequestContext。传输约束放在`texas_poker_event_http.go`的`PrepareTexasPokerEvent`路由中间件，由生成路由的`_texaspokereventMw`挂载：单个application/json对象、精确packageName/大小写歧义检查、query隔离及URI恢复；该中间件直接使用Hertz已缓冲的body，不再读取或重设请求体。Handler构造器与decision一样只接收生成请求；状态和结果TTL的服务端时间由service在处理/完成时获取，Handler不再持有now。动态JSON保留原始token及大整数，完整日志在来源路由及非空uuid激活校验通过之后；旧客户端无uuid只采集日志。event已移除Bearer认证，激活检查不构成uuid归属认证。BaseResp.StatusCode为0返回HTTP200，非法入口固定400，决策临时失败固定503；这些业务语义不因代码风格调整而改成一律200。
- `service/texas_poker.go` 调用外部识别服务与 GTO 决策服务。
- `dal/local_mem.go` 按用户保存同一手牌各阶段的识别结果。
- `model/recognize_poker.go` 把识别结果转换为玩家、位置和行动历史。
- `biz/handler/texas_poker_event.go`通过IDL生成的请求/响应接收桃园助手透传的扑克事件并按packageName分流；`service/poker_event.go`持有统一包名路由及两个状态容器，`service/aapoker_event_state.go`和`service/dpzx_event_state.go`分别解释各自协议，通过`ProcessPokerEvent`返回统一Hero行动点，`CompletePokerEventDecision`负责成功缓存或失败释放重试资格。互斥锁、TTL与回放缓存均由service管理，不暴露给handler。Handler直接调用`executePokerDecision`，不保留供单测替换的全局函数变量。`poker_event_decision.go`复用Handle已校验的decision Handler及其激活信息进行策略调用、计费和`gto_decision_record`审计写入。GTO返回可解析决策即异步记录：非法结果保留原始响应，final_response为JSON null，不下发、不计费；合法结果保留规范化最终响应。临时失败返回503供客户端重传；成功的完整响应按隔离后的会话、牌桌和行动点缓存30分钟、每种协议最多1024条；重复事件须先通过check中的激活检查，再回放而不重复副作用。额度耗尽即拒绝，包含最后一次计费成功后丢失响应的重传，用户已接受该边界。

## 当前覆盖边界

本文只描述德州扑克决策热路径；支付、健康、塔罗等模块不在当前范围内。

- 非法/空GTO决策在event入口终止当前行动点：返回HTTP 200、BaseResp.StatusCode=0、StatusMessage=decision_rejected，不带Result或客户端动作，不计费。可解析的非法原始响应仍异步审计，final_response为JSON null。service缓存非nil但Result为空的终止标记，沿用30分钟TTL和1024条容量；相同行动重传不重复GTO、审计或计费。后续真实行动继续推进状态，可生成下一行动点的决策。网络/上游临时错误仍返回503并释放重试资格；激活检查仍先于缓存。两协议HTTP回归覆盖拒绝、重传、同手后续行动及正常决策。
