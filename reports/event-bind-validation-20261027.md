# Event 请求绑定简化验证

## 变更

- 删除仅为HTTP测试注入时间的 `texasPokerEventAt`，入口直接记录接收时间并调用 Handler。
- 删除手工 `parseTexasPokerEvent`，通过 `c.BindAndValidate` 绑定IDL生成请求。
- 保留512 KiB总body边界；流式body有界读取后交给绑定器，同一份body用于完整日志。
- 入口限定 `Content-Type: application/json`（允许charset）与单个JSON对象；绑定期间屏蔽query，随后恢复URI，任何字段均不得从query补值或覆盖。JSON客户端应发送 `Content-Type: application/json`。固定字段类型不符由绑定器拒绝，返回固定400，不再忽略类型错误；动态JSON字段继续保留原始token。
- 来源使用精确`packageName`键，拒绝`PackageName/packagename`等大小写变体及其与精确键并存的歧义；实际路由固定使用精确键值。
- packageName白名单、Bearer提取、状态机、GTO/计费/SQL及结果回放未修改。

## 验证

- `GOTOOLCHAIN=go1.20.14 go test ./... -count=1`：通过。
- `GOTOOLCHAIN=go1.20.14 go test -race . ./biz/handler ./service -count=1`：通过。
- `git diff --check`：通过。
- 覆盖大小写变体单独出现、未知/合法精确包名与变体按不同顺序并存、同值变体并存的HTTP 400及marker不落日志。覆盖text/plain+query、JSON缺失/未知/非字符串包名+query、非法/多JSON值/数组/null的HTTP 400及marker不落日志；合法body与冲突query绑定时仍只使用body。覆盖真实生成路由JSON绑定、流式请求、无效包名/JSON、固定字段类型错误不泄露原文、大整数/动态JSON、超过480 KiB但低于总body上限的数据、301条完整日志、AAPoker与DPZX决策、503重试及成功结果回放。

## 边界

本轮为本地自动化验证，尚未部署，未执行真实Android→生产GTO→SQL→悬浮窗端到端验证。未改变既有17770未认证及全量日志无频率上限的已接受风险。
