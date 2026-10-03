# 2026-10-03 新增 Hero 图线上回归

## 结论

- 服务器当天相对 `manifest v5` 新增 129 张 Hero 图，文件名和图片 SHA256 均唯一；时间范围为 18:22:40 至 19:06:11。
- 使用当前线上 dezhou 代码逐张顺序回放，129/129 返回成功。外层耗时均值 1.609 秒、P50 1.590 秒、P95 2.183 秒、最大 2.680 秒。
- 线上重放结果与图片首次进入生产时的识别日志逐字段归一化对比为 129/129 一致。阶段分布为 preflop 91、flop 17、turn 12、river 9；逐张程序校验未发现街道/公共牌数量、重复牌、盲注、按钮位、Hero 状态、座位集合或下注上限异常。
- 33 张四联审阅页覆盖全部 129 张原图。人工查看确认唯一异常边界是 `2026-10-03 18:53:33-hero.jpg`：Hero 的两张牌仍显示牌背，dezhou 正确返回空 `hero_cards`，但旧 ddbaby 仍调用 GTO，历史请求得到 HTTP 500 `list index out of range`，悬浮窗最终为空。
- 候选 ddbaby 在调用 GTO 前校验 Hero 手牌、公共牌数量、牌值格式和重复牌。上述牌背帧被本地拒绝，不再调用 GTO；其余 128 个请求调用当前 GTO 后全部通过 ddbaby 合法性校验，且没有触发筹码上限归一化。当前动作分布为 fold 72、raise 19、check 18、bet 12、call 7。

## 数据与运行版本

- 图片来源：生产服务器 `/usr/local/webserver/images`。
- 新增口径：文件日期为 2026-10-03、文件名后缀为 `-hero.jpg`，且图片 SHA 不在 `reports/hero-regression/manifest.json` v5 中。
- 本报告的逐图索引见 `manifest.json`，记录文件名、SHA、识别耗时、牌面、历史决策状态和候选回放结果；原图保留在受控回归工作目录，不提交仓库。
- dezhou 线上运行时：`app/recognizer.py` SHA256 `19b06782aa57f4855f3164e8647786e0ca8a68f0e3ccf1af11f8e281c53ba659`，`app/layouts/polerv2.py` SHA256 `9b957ebf90d8f93df631343a94a22de56ddefd2478eca4a115d5316fa7d64d23`，与九人桌优化部署版本一致。
- ddbaby 回归前线上版本为 `c6cc23f`，运行时二进制 SHA256 `ef581480f4fbb7c42e8703d6d514e723069a29e59a9939dc13a6b824c1dab4c2`。

## 验证方法

1. 远端盘点并下载当天 Hero 图，核对文件名与 SHA 唯一性，并排除 manifest v5 已有图片。
2. 通过生产 dezhou 本机 `http://127.0.0.1:5555/recognize`，按文件名顺序单并发回放 129 张图，避免压测并发改变延迟口径。
3. 将重放结果与 ddbaby 原始生产日志中的识别结果归一化对比；仅忽略时间戳、耗时、ddbaby 结构未携带的 `table_info.current_bet`，以及 JSON `null` 与 Go 数值零值的表示差异。
4. 对全部图片执行字段一致性和业务约束检查，并人工查看覆盖全部图片的 33 张四联审阅页。
5. 提取 129 个原始决策请求：历史结果为 128 个 HTTP 200 合法响应和 1 个 HTTP 500。使用候选 ddbaby 校验器和当前 GTO 顺序回放，得到 1 个请求在 GTO 前拒绝、128/128 当前响应合法。

## 修复与测试

- `ValidateTexasGtoDecisionRequest` 现在要求当前 Hero 恰有两张合法牌，并按 preflop/flop/turn/river 分别要求 0/3/4/5 张公共牌。
- Hero 手牌与公共牌统一校验牌值、花色和全局唯一性，避免 OCR 漏牌、非法牌或重复牌进入 GTO。
- 新增缺少 Hero 牌、只有一张牌、非法牌、Hero 重复牌、Hero/公共牌重复、未知街道和各街公共牌不足的回归测试。
- 本地 `go test ./model ./dal -count=1`、`go test -race ./model ./dal -count=1` 和 `git diff --check` 通过。
- 远端 Go 1.22.2 候选目录执行 `go test ./... -count=1`、`go test -race ./model ./dal -count=1` 和 `bash build.sh` 通过。

## 边界

- 129/129 与生产日志一致说明当前线上代码可稳定复现当时识别结果；它本身不是人工逐字段标注真值。程序约束检查与全图人工审阅未发现除牌背帧外的明显识别问题。
- 128 个当前 GTO 响应均符合下注合法性并被 ddbaby 接受；本轮没有独立证明每个动作在扑克策略上最优。
- 回放没有注入生产缓存，也没有重放客户端悬浮窗；修复验证覆盖请求生成后的 GTO 调用边界。
