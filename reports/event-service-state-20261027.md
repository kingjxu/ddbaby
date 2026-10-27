# 状态服务迁移与测试钩子清理

## 变更

- 删除正式代码中的`var executeAAPokerDecisionForEvent = executeAAPokerDecision`，handler直接调用决策函数。
- AAPoker/DPZX状态机实现及其状态单测迁到`service`包。服务持有两套状态容器、统一来源路由、锁、会话TTL、in-flight资格及完整结果缓存。
- `ProcessPokerEvent`生成Hero行动点或缓存结果；`CompletePokerEventDecision`保存成功结果或释放失败行动点。会话/行动点键和协议路由保留在service内部，handler不接触容器和锁。
- handler继续负责HTTP请求、完整日志、身份/激活/GTO/计费/审计编排及响应。Handler不再保存用于状态TTL的时间字段，service在处理和完成时取服务端当前时间。
- 无新增接口注入层、可替换业务函数或测试专用生产分支；没有依赖版本变化。

## 验证

以下全部通过：

- `GOTOOLCHAIN=go1.20.14 go test ./... -count=1`。
- `GOTOOLCHAIN=go1.20.14 go test -race . ./biz/handler ./service -count=1`。
- handler/service回归曾连续运行两轮，均通过；测试会话使用独立牌桌或账号，避免单例缓存跨轮影响。
- `git diff --check`。

状态机原有缺动作、非法私牌/公共牌、跨街连续性、并发、TTL和有界缓存用例随实现迁移。HTTP失败回归经过真实决策函数，在测试文件内通过标准HTTP transport模拟认证503，验证Bearer传递和重试资格释放。HTTP成功回放通过service完成接口准备结果，验证缓存响应与不再触发认证。该回归没有实际运行认证成功后的GTO/SQL/计费成功链路，也不将它表述为完整生产端到端验证。

未部署；生产及真机验证仍待后续执行。
