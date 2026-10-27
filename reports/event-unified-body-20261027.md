# 删除手工body读取

- 删除`readTexasPokerEventBody`、event专属512 KiB限制、`WithStreamBody(true)`与其他路由配套的手工4 MiB读取中间件。
- 所有HTTP路由统一使用Hertz缓冲和4 MiB总请求体上限。event超过512 KiB可接收；超过4 MiB由框架返回413，不再由event返回`invalid_body_size`的400。
- event中间件直接使用已缓冲body，不再读取流或重设body/Content-Length。单JSON对象、精确包名/大小写歧义拒绝、query隔离保持不变。
- 删除`handler.now`从HTTP上下文透传的赋值与时间context key，使用Handler构造时取得的服务端时间参与状态及结果TTL。
- 原始body日志和Bearer身份校验保持原行为。桃园转发端仍保留512 KiB事件及480 KiB data限制，仅同步服务端协议文档。

验证全部通过：

- `GOTOOLCHAIN=go1.20.14 go test ./... -count=1`。
- `GOTOOLCHAIN=go1.20.14 go test -race . ./biz/handler ./service -count=1`。
- 两仓`git diff --check`。
- 使用Hertz真实HTTP解析链路和内存连接验证event及普通路由：定长与chunked请求在512 KiB+1和4 MiB返回200，在4 MiB+1返回413。
- 既有来源拒绝、原始JSON/大整数、AAPoker/DPZX决策、503重试和成功回放回归继续通过。

风险与边界：单请求可缓冲并记录的数据上限由512 KiB增至4 MiB，内存及日志写入峰值可能增大。该变化已向用户说明。未部署，尚无新的生产或真机端到端验证。
