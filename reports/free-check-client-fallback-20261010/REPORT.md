# Free-fold 问题页与客户端兜底

## 变更

- `POST /lyxz/gto_decision` 对通过请求与响应校验的决策增加两个事实响应头：原始 GTO 动作，以及 Hero 当前是否无需补注即可 check。
- 响应 JSON 和服务端决策日志保持原样，旧客户端行为不变；客户端 1.0.28 仅在 `fold + can-check` 两项事实同时成立时把展示层“弃牌”改成“过牌”。
- 增加确定性问题页生成器。页面从冻结的 138 张绿桌审计中固定选择 2 个 free-fold 错误状态和 2 个 solver-sensitive 状态，内嵌同 trace 原图、完整请求、原始响应与识别结果。

## 安全与边界

- 服务端只在完整请求和决策响应均校验通过后输出事实头；错误、未授权、识别不完整和非法决策均不输出。
- 页面生成器只接受图片目录的直接 `.jpg` 子文件，解析后再次校验路径归属；动态文本使用 HTML 转义，图片使用 base64 data URL，避免路径穿越与 HTML 注入。
- 客户端不依据中文展示文本单独改写 fold，也不在 Hero 面对下注时改写。该兜底修复确定性 dominated action，不证明其他动作 GTO 最优。

## 候选验证

- Go 单元测试覆盖 matched max、无人下注、面对下注、缺少 Hero、nil 请求，以及响应头仅对合法决策输出的边界。服务器 Go 1.22.2 执行 `go test ./... -count=1`、`go test -race ./biz/handler ./model ./dal -count=1` 和 `bash build.sh` 全部通过；候选二进制 SHA256 为 `f79b591b59c118af7578c42da5e8d8bb13da79b860ce0f7b7b9b97ded3f1b8cf`。
- 问题页包含 4 个状态和 4 张代表图，重复生成 SHA256 一致：`aa490cc94cacaea8312e1936fedb1276123783e3efea8107be8307582f7e4b3f`，大小 `1031766` bytes。
- 本机 Go 1.24 无法编译项目固定的旧 Sonic 依赖；已按项目既有方式使用生产同版本 Go 1.22.2 完成验证，没有为本机工具链改动生产依赖或代码。

## 部署

- 业务提交 `a773067`。旧二进制和两份运行时源码备份到 `/home/ubuntu/ddbaby-backups/20261010-174934-before-a773067`；候选经同目录临时文件校验后原子替换。
- 运行时二进制 SHA256 为 `f79b591b59c118af7578c42da5e8d8bb13da79b860ce0f7b7b9b97ded3f1b8cf`；`texas_poker_decision.go` 和 `recognize_poker.go` 分别为 `277b4d958f4f723d4d83cf3541725c0eba31f607b0198ff96581cd8bd81ccd22`、`35e2e7acce26fbffebc80d8c9cba7fda95b6a47071774079700ea8087468ae6b`，与构建输入一致。
- 部署后 PID `2880168`、PPID `1`，`/proc` 二进制 SHA 与部署文件一致，端口 8888 和 `/ping` 正常。无鉴权空请求保留原空 JSON、trace/timing 头，且不错误输出 decision/can-check 事实头；合法分支由单元测试覆盖，未用真实用户凭证做生产 smoke。
- 问题页发布到 `https://www.ddbaby.site/tyzs/strategy-issues-20261010.html`。公网 HTTP 200、`1031766` bytes、SHA256 `aa490cc94cacaea8312e1936fedb1276123783e3efea8107be8307582f7e4b3f`，与候选逐字节一致。
