# Event 前置激活校验

## 变更

按用户要求，event 移除 VerifyTaoyuanIdentity、Bearer 提取及 token 传递。非空 uuid 的每个事件在完整日志、状态机更新及结果回放前，复用 decision 的 verifyActiveCode，查询 user_active_code 最新记录并检查过期时间和按次额度。决策复用本次查询得到的 Handler/激活信息，不重复查库；合法结果后计费及审计逻辑保持原样。

激活缺失、到期、次数耗尽或数据库错误返回固定 HTTP 503 / active_code_unavailable，不回显数据库错误或请求 body；沿用 Android 当前事件重试机制。持续无有效激活时会持续重试，需恢复有效激活后继续。无 uuid 的旧客户端保留只记录日志、不建立状态及决策的兼容行为。

verifyActiveCode 不再输出完整激活码记录，避免前置检查后每条事件重复记录敏感字段。未修改 decision 的激活规则、数据库表或客户端实现。桃园协议文档和服务端 .ai-context 已同步。

## 验证

- go1.20.14 go test ./... -count=1：通过。
- go1.20.14 go test -race ./biz/handler ./service . -count=1：通过。
- 最终补充用例后重跑 handler 普通测试和 race：通过。
- 两仓 git diff --check：通过。
- AAPoker/DPZX 激活缺失、过期、次数耗尽、数据库错误：503，未输出完整 body，未进入后续 GTO。
- 有效时间码、仍有额度的次数码：无需 token 通过；测试核对按请求 uuid 使用参数化 user_id 查询。
- 原 HTTP 重试回归改为 GTO HTTP 边界失败，503 后可以重试；service 准备成功结果后可回放，回放不再调用 GTO，但仍重新检查激活。AAPoker 额外验证激活过期不能回放旧结果。

测试用独立 GORM 数据库的查询回调提供激活记录，用 HTTP transport 返回 GTO 失败；没有在正式代码增加测试替换函数。测试不连接生产数据库，不发送真实策略请求，不计费、不写真实审计，不代表生产 GTO→SQL→真机端到端验证。未部署。

## 边界

按请求 uuid 查询激活不构成调用方认证；移除 token 校验后，获知有效 uuid 的调用方可冒用额度并污染审计。该风险已告知用户，不能描述为已验证 uuid 归属。每条非空 uuid 事件（含缓存回放）都会查库，会增加数据库查询量；没有新增激活缓存。既有 loopback 未认证、明文传输和持续完整日志风险仍在。
