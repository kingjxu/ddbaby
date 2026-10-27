# Event 处理风格对齐 decision

## 最终组织

- `TexasPokerEvent`与decision一样按`BindAndValidate → NewHandler(req) → Handle → 响应`展开，入口仍保留在生成器预期的`texas_poker_event.go`。
- `NewTexasPokerEventHandler`只接收IDL请求；原始body、Bearer和接收时间由HTTP入口补入，业务Handler不持有Hertz上下文。
- 512 KiB有界读取、单个JSON对象、精确包名/大小写歧义校验、query隔离放入`texas_poker_event_http.go`的路由中间件，由生成路由的`_texaspokereventMw`挂载。
- 业务`Handle`按请求校验、牌局重建、策略/结果回放顺序执行；无候选和状态错误提前返回，避免多层嵌套。
- 保留400固定错误、503可重试和200成功响应，不为了风格统一而改变协议。Bearer、完整日志、独立状态机、GTO/计费/审计和回放链路保持原行为。

## 验证

以下检查全部通过：

- `GOTOOLCHAIN=go1.20.14 go test ./... -count=1`。
- `GOTOOLCHAIN=go1.20.14 go test -race . ./biz/handler ./service -count=1`。
- `git diff --check`。
- 既有真实HTTP路由回归覆盖非JSON、query补值、精确包名与大小写歧义拒绝；handler回归通过同一中间件链调用，继续覆盖流式body、完整日志及AAPoker/DPZX重试/回放。
- 新增中间件回归核对成功及绑定错误路径的原始body、URI恢复和接收时间，防止迁移过程中丢失HTTP上下文。

本轮仅本地验证，尚未部署；无新的生产或真机端到端结论。
