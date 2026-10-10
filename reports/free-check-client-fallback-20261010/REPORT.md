# Free-fold 问题页与服务端兜底

## 变更

- `POST /lyxz/gto_decision` 在决策服务返回合法 `fold` 后，比较 Hero 当前投入与本轮最大投入；Hero 无需补注时将最终动作改为 `check`，原有 JSON `Result` 返回“过牌”。
- 客户端无需新增响应头或动作改写；此前 1.0.28 客户端方案已撤回，源码恢复为 1.0.27。
- 增加确定性问题页生成器。修正审计后，页面从冻结的138张绿桌样本中选择1个真实 free-fold 错误状态和2个 solver-sensitive 状态，内嵌同 trace 原图、完整请求、原始响应与识别结果。

## 安全与边界

- 兜底只在完整请求和原始决策响应均校验通过后执行。面对更高下注、缺少 Hero、非法请求、非法响应和非 `fold` 动作均不改写。
- 页面生成器只接受图片目录的直接 `.jpg` 子文件，解析后再次校验路径归属；动态文本使用 HTML 转义，图片使用 base64 data URL，避免路径穿越与 HTML 注入。
- 该兜底只修复无需补注时的确定性 dominated action，不证明其他动作 GTO 最优。Hero 首个行动且本轮无人下注自然满足该条件；翻前仍面对盲注或任意更高投入时不满足。

## 候选验证

- Go 单元测试覆盖翻前 BB/straddle option、翻后首个行动、翻前非盲位相等快照、面对下注、非 fold、缺少 Hero、nil 请求和 nil 响应。部署候选的全仓、race、构建与 SHA 在本轮部署后补录。
- J♣K♥ 原图的对手下注264曾被识别为26，不能作为 free-fold；已从决策问题页移除并在审计中改列为关键识别错误。修正页面包含3个状态和3张代表图，重复生成 SHA256 一致：`881ff80880ebcbe6f3467d3802c3f46a667aa38995f9c829337c72b62e812e04`，大小 `769358` bytes。
- 本机 Go 1.24 无法编译项目固定的旧 Sonic 依赖；已按项目既有方式使用生产同版本 Go 1.22.2 完成验证，没有为本机工具链改动生产依赖或代码。

## 历史部署与纠偏

- `a773067`/客户端 1.0.28 是已撤回的首版方案，曾通过事实响应头让客户端改写展示；本轮按产品口径改为 ddbaby 直接规范化最终动作。
- 运行时二进制 SHA256 为 `f79b591b59c118af7578c42da5e8d8bb13da79b860ce0f7b7b9b97ded3f1b8cf`；`texas_poker_decision.go` 和 `recognize_poker.go` 分别为 `277b4d958f4f723d4d83cf3541725c0eba31f607b0198ff96581cd8bd81ccd22`、`35e2e7acce26fbffebc80d8c9cba7fda95b6a47071774079700ea8087468ae6b`，与构建输入一致。
- 首版部署的运行时证据保留用于回滚审计，但不代表纠偏版本状态；纠偏版本部署结果在本轮完成后补录。
- 问题页地址保持 `https://www.ddbaby.site/tyzs/strategy-issues-20261010.html`；生产页面将在本轮部署时原子替换并补录公网大小与 SHA。

## 纠偏版本部署

- 服务端提交 `ae84dd1` 在生产同版本 Go 1.22.2 环境通过 `go test ./... -count=1`、`go test -race ./biz/handler ./model ./dal -count=1` 和 `bash build.sh`。候选及运行时二进制 SHA256 均为 `57803792bb99ab24c24a8f22f50d2ceada27128a337bc5506b6982881527fb0b`；运行时 `texas_poker_decision.go`、`recognize_poker.go` SHA256 分别为 `0df08577bc1ada32a36f56203a27e51caa7d0d4bbeca363239285921ed8b04a9`、`428549aaf584f23af13b9e33f7d0d3cb9bf07dbd5628b4d843abea47ca8dfea1`。
- 原服务资产备份到 `/home/ubuntu/ddbaby-backups/20261010-181328-before-ae84dd1`，候选经同目录临时文件校验后原子替换。部署后 PID `2887499`、PPID `1`，`/proc/2887499/exe` 与运行时文件 SHA 一致，监听 8888，本机 `/ping` 返回 `pong`。
- 无鉴权空请求仅用于验证路由与兼容边界：响应仍为原 JSON，且已撤回的 `X-GTO-Decision-Action`、`X-GTO-Can-Check` 均不存在；该 smoke 不冒充合法决策验证，合法归一化分支由单元测试覆盖。
- 修正问题页已原子替换。公网 HTTP 200、大小 `769358` bytes、SHA256 `881ff80880ebcbe6f3467d3802c3f46a667aa38995f9c829337c72b62e812e04`，与确定性构建产物一致。
- 官网 APK 已从 1.0.28 恢复为已验证的 1.0.27，公网 HTTP 200、大小 `6913436` bytes、SHA256 `f5b80700112852b096a65004c38941e53d461975851b23f706d5c37538afefc4`。替换前 1.0.28 APK 与旧问题页备份在 `/home/ubuntu/taoyuan-backups/20261010-181328-freecheck-correction`。
