---
version: "1.4"
last_updated: "2026-10-10"
status: "active"
---

# 环境与运维

## 生产服务器

- 服务目录：`/home/ubuntu/ddbaby`
- 进程：`output/bin/hertz_service`
- 监听端口：8888
- 本机健康检查：`GET http://127.0.0.1:8888/ping`，成功返回 `{"message":"pong"}`。
- 8888 未向公网开放，外部直连超时不代表服务异常；应在服务器本机检查。
- ddbaby 与 dezhou 部署在同一台服务器；识别请求固定走本机
  `http://127.0.0.1:5555/recognize`，避免公网回环产生约 200–300ms 常态开销及偶发秒级抖动。

## 2026-10-02 部署

- 业务基线：`fix/ddbaby-poker-regression-20261002`，提交 `a6d9d1e`。
- 运行时源码与本地提交逐文件 SHA256 一致；部署二进制 SHA256：`62fbdf8c592b139d06ee52e1fee042ef298de61a14a202dff02f7ee855d7f1d5`。
- 远端使用 Go 1.22.2 执行 `go test ./...` 和 `build.sh`，均通过。
- 进程以 root、PPID=1 独立运行；部署后 PID=38187，本机 `/ping` 验证通过。
- 回滚目录：`/home/ubuntu/ddbaby-backups/20261002-1649-before-a6d9d1e`，包含旧二进制和变更前运行时源码归档。

## 2026-10-02 parse_all 后继帧修复部署

- 业务基线：`fix/ddbaby-poker-regression-20261002`，提交 `d2f9e5f`；核心状态机恢复提交为 `1f56020`。
- 远端使用 Go 1.22.2 执行 `go test ./... -count=1` 和 `build.sh`，均通过；部署二进制 SHA256：`1ff25809586bfedcadf5188448f20b77b94198716fe86a16ac6d137f4c65eced`。
- 部署后进程 PID=72578、PPID=1，监听 8888，本机 `/ping` 返回 `{"message":"pong"}`；重新建立 SSH 连接后进程仍存活。
- 线上日志验证了 `Hero -> 下一张非 Hero 完整识别 -> 后续恢复轻量检测`：18:42:29 Hero 完整识别，18:42:32 后继帧记录 `needParseAll:true`，随后请求恢复约 24ms 的轻量结果；18:42:36/18:42:41 再次观察到相同状态迁移。
- 回滚目录：`/home/ubuntu/ddbaby-backups/20261002-184204-before-d2f9e5f`，包含上一版二进制和本次变更前源码。

## 2026-10-02 dezhou loopback 调用部署

- 业务提交：`35858c9`，ddbaby 调用同机 dezhou 时固定使用
  `http://127.0.0.1:5555/recognize`。
- 远端使用 Go 1.22.2 执行 `go test ./... -count=1` 和 `build.sh`，均通过；部署二进制 SHA256：`231ec3838c3bdf2a9161c945808550aaa30b106559bc37f78ad82f00ec289b0e`。
- 部署后进程 PID=119078、PPID=1，监听 8888；本机 `/ping` 返回 `{"message":"pong"}`，重新建立 SSH 连接后进程仍存活。
- 使用同一张 Hero 图片连续验证 3 次，ddbaby 外层/dezhou 内层耗时分别为
  `2573/2500ms`、`2320/2314ms`、`2275/2271ms`；热请求额外开销已从约 200–300ms 降至 3–6ms。
- 回滚目录：`/home/ubuntu/ddbaby-backups/20261002-211105-before-35858c9`。

## 2026-10-03 straddle 位置顺延部署

- 业务提交：`c6cc23f`。位置按本手实际参与人数分配；`empty`、`waiting`
  不占位置，`folded` 仍占位置。若按钮位或后续物理座位无人，`BTN`、`SB`、
  `BB`、`straddle` 顺时针依次顺延给下一名实际参与者。
- 远端使用 Go 1.22.2 执行 `go test ./... -count=1` 和 `build.sh`，均通过；
  部署二进制 SHA256：`ef581480f4fbb7c42e8703d6d514e723069a29e59a9939dc13a6b824c1dab4c2`。
- 运行时 `model/recognize_poker.go` SHA256：
  `2f95a6f6fce07a552e0a4a548a78a6cd631f356649bd2f458c6ac9c15ab3cf44`，
  与提交构建输入一致。
- 部署后进程 PID=349622、PPID=1，监听 8888；本机 `/ping` 返回
  `{"message":"pong"}`，重新建立 SSH 连接后进程仍存活，启动日志无异常。
- 回滚目录：`/home/ubuntu/ddbaby-backups/20261003-111847-before-c6cc23f`。

## 2026-10-03 Hero open 历史补偿回滚

- 曾尝试从相邻快照的筹码和底池差额恢复漏读的 Hero 下注，
  但该方法会把 OCR 错误扩散到业务历史，不符合“图上可见下注必须
  由识别结果直接提供”的口径，已撤回。
- 线上 ddbaby 已回滚到 `c6cc23f` 二进制，SHA256：
  `ef581480f4fbb7c42e8703d6d514e723069a29e59a9939dc13a6b824c1dab4c2`；
  运行时 `model/recognize_poker.go` SHA256：
  `2f95a6f6fce07a552e0a4a548a78a6cd631f356649bd2f458c6ac9c15ab3cf44`。
- 回滚后进程 PID=499283、PPID=1，监听 8888，本机 `/ping` 返回
  `{"message":"pong"}`。
- 2026-10-02 18:54:54 原图的 Hero 下注 33 在当前 dezhou 线上版本中
  连续重放 3 次均直接识别为 `current_bet=33`；后续回归以原图
  ROI/OCR 识别结果为准，不在 ddbaby 通过筹码差额推算下注。

## 2026-10-03 Hero 牌面完整性拦截部署

- 业务提交：`92ddbef`。在请求 GTO 前校验当前 Hero 恰有两张合法明牌、
  各街公共牌数量正确，并拒绝非法牌和 Hero/公共牌重复。
- 远端 Go 1.22.2 执行 `go test ./... -count=1`、
  `go test -race ./model ./dal -count=1` 和 `bash build.sh` 均通过。
- 部署二进制 SHA256：
  `694b03f331bfc7a490b5ba7d0cf89ce149a6458655c8879dea07cb888e99c47d`；
  运行时 `model/recognize_poker.go` SHA256：
  `2d1547726a0025d778de08d31ee95b255290e726886c24fba3766a2e677ac823`，
  与提交构建输入一致。
- 部署后进程 PID=525120、PPID=1，监听 8888；重新建立 SSH 连接后
  本机 `/ping` 返回 `{"message":"pong"}`，启动日志无异常。
- 当天 129 张 Hero 图的候选回放中，18:53:33 牌背帧在 GTO 前以
  `current player BB-Hero has 0 hole cards, want 2` 拒绝；其余 128 个
  请求调用当前 GTO 后均通过响应合法性校验。线上未注入用户缓存，
  运行时行为以源码/二进制哈希和候选回放为证据。
- 回滚目录：`/home/ubuntu/ddbaby-backups/20261003-224307-before-92ddbef`，
  包含上一版二进制、运行时源码和校验信息。

## 2026-10-04 全链路时延追踪部署

- 业务提交：`1f4c38c`，安全边界修复提交：`97cd83d`；客户端配套提交：
  `a779a81`。旧客户端不发送追踪头时保持原行为。
- 远端 Go 1.22.2 执行 `go test ./... -count=1`、
  `go test -race ./biz/handler ./model ./dal -count=1` 和 `bash build.sh`
  均通过。
- 部署二进制 SHA256：
  `74346e9563c52e0ad9f44a88cda328216e5abe4bcbc67341a00a2a8486bbaeca`；
  运行时 `texas_poker_decision.go` 和 `latency_trace.go` SHA256 分别为
  `2679cf11f156d9a100b7af2009981e8eb4a0421f63ee3c58125fbe60d05967f7`、
  `ba11b21f9b09fff6744573a90ee55c4666bd50606a8c1720a492bbf2267409e7`，
  与构建输入一致。
- 部署后 PID=821094、PPID=1，监听 8888；本机 `/ping` 返回
  `{"message":"pong"}`。无鉴权 smoke 请求验证响应包含同一 trace id 和
  `X-GTO-Server-Timing`，客户端阶段日志按预期产生。
- 回滚目录：`/home/ubuntu/ddbaby-backups/20261004-185316-before-97cd83d`。
- 尚无 1.0.26 真机真实流量，不能据此确认 3 秒以上体感的具体归因；需要后续
  `client_complete` 日志完成分段判断。

## 生产截图保留策略

- `/usr/local/webserver/images` 中 `*-hero.jpg` 和 `*-decided.jpg` 长期保留，
  普通 `*.jpg` 保留最近 1 天。
- 清理脚本为 `/usr/local/sbin/ddbaby-cleanup-images`，cron 每天 03:17 执行；
  运行日志写入 `/var/log/ddbaby-image-cleanup.log` 并按周轮转。
- 清理与 ddbaby 图片落盘共用 `/run/lock/ddbaby-images.lock`；清理持锁完成最终身份校验
  和删除，所有生产图片写入必须通过 `writeImageToFileWithLock` 持同一把锁，避免删除
  检查通过后被同名新文件替换。
- 修改清理规则时必须先执行 `--dry-run`，不得扩大到子目录、符号链接、Hero 或
  decided 证据图。
- 2026-10-06 已部署 5 天保留调整（提交 `b5eb1ba`）：生产脚本 SHA256
  `1dd0ce9922b5edc84ebc63b43fa4a538eae22bb21b63875fb741cde3068f00b3`，
  cron SHA256 `9bbbd3aefb9dd616e4eaaedb59c42a181874b26a9273672a97e8f903f07b5476`。
  生产无参数 dry-run 显示 `retention_minutes=7200` 且匹配/删除/变化均为 0；cron
  表达式为 `17 3 * * *`、服务 active，ddbaby `/ping` 正常。替换前脚本和 cron
  已备份到既有回滚目录的 `cleanup-config` 子目录。

### 2026-10-13 普通截图保留期调整

- 普通 `*.jpg` 保留期由 5 天缩短为 1 天；`*-hero.jpg` 和
  `*-decided.jpg` 的长期保留规则不变，cron 仍每天 03:17 执行。
- 生产首次按 1 天规则 dry-run 匹配 20,624 张、3,368,503,831 bytes；正式清理及
  边界补清理合计删除 20,635 张、3,370,372,703 bytes，`changed_files=0`，最终
  dry-run 匹配 0。
- 清理前 Hero/decided 分别为 5,030/3,646 张；清理后 Hero 因实时流量增加至
  5,032 张，decided 保持 3,646 张。根盘可用空间由 742,875,136 bytes 增至
  4,153,307,136 bytes，服务 `/ping` 保持正常。

## 2026-10-06 固定 BTN 手牌边界部署

- 业务提交 `19a7163`，校验边界测试提交 `cc7987c`。连续完整识别通过 BTN 变化、街道回退或两张完整 Hero 手牌变化切分新手牌；GTO 前拒绝未来街道、街道回退和非递增时间戳。
- 远端 Go 1.22.2 执行全仓测试、model/dal race 和 `build.sh` 均通过。运行时二进制 SHA256 为 `b46382170457d134538f99f8dc27685cbb7998ed23d4f1b1c0c3b3b6b8356275`；`model/recognize_poker.go` 与 `dal/local_mem.go` SHA256 分别为 `e351331cb3135bb62f3ca458825e75c352d8df2aafe75e53a48489a0350a0a35`、`53f8ecfbc3566185e91278a4df850ebbfa0ed9b8a2b29eb03cec470eef91c24a`。
- 部署后 PID 1513243、PPID 1，`/proc/1513243/exe` 与部署文件 SHA 一致；SSH 重连后 `/ping` 返回 `pong`。
- 回滚目录 `/home/ubuntu/ddbaby-backups/20261006-1837-before-cc7987c`。47 个真实可决策请求的隔离重放 history stage regression=0、生产 GTO 47/47 响应合法、归一化 0；未向真实用户缓存注入历史牌局。

## 2026-10-06 Hero 决策机会首次结果追踪部署

- 业务提交 `6ea22c3`，配套客户端提交 `58737df`。trace v2 增加随机 `decision_cycle_id`、Hero 控件首次检测时间和首次展示标记；v1 客户端保持兼容。
- 服务器 Go 1.22.2 全仓测试、handler/model/dal race 和构建通过。运行时二进制 SHA256 为 `21024ffa93e42d44b2fc4ad1252956feebd99ad5c16309f2d40d11f389b38eb9`；运行时 `biz/handler/latency_trace.go` SHA256 为 `69eab9f64b56b882b1d48d157b1b463e5d4ae5225c8904fb4aa54703775db04e`，与构建输入一致。
- 部署后 PID `1554013`、PPID `1`，`/proc/1554013/exe` 与部署文件 SHA 一致；SSH 重连后 `/ping` 返回 `pong`。v2 smoke 验证首次展示进入 `client_complete`、重复展示进入 `client_complete_repeat`。
- 回滚目录 `/home/ubuntu/ddbaby-backups/20261006-211121-before-6ea22c3`。smoke 未使用真实鉴权或图片；需等待 1.0.27 真机流量后再分析 `hero_turn_detected -> first_display`。

## 2026-10-10 免费 check 首版部署（已确认需纠偏）

- 业务提交 `a773067`，配套客户端提交 `792de53`。合法决策响应增加原始动作和 can-check 事实头，响应 JSON 与旧客户端行为保持兼容。
- 服务器 Go 1.22.2 全仓测试、handler/model/dal race 和构建通过。运行时二进制 SHA256 为 `f79b591b59c118af7578c42da5e8d8bb13da79b860ce0f7b7b9b97ded3f1b8cf`；运行时 handler/model 源码 SHA256 为 `277b4d958f4f723d4d83cf3541725c0eba31f607b0198ff96581cd8bd81ccd22`、`35e2e7acce26fbffebc80d8c9cba7fda95b6a47071774079700ea8087468ae6b`，均与候选一致。
- 部署后 PID `2880168`、PPID `1`，`/proc` 二进制匹配，端口 8888 和 `/ping` 正常；非法/无决策 smoke 不输出决策事实头。回滚目录为 `/home/ubuntu/ddbaby-backups/20261010-174934-before-a773067`。
- 绿桌问题页发布为 `https://www.ddbaby.site/tyzs/strategy-issues-20261010.html`，公网大小 `1031766` bytes、SHA256 `aa490cc94cacaea8312e1936fedb1276123783e3efea8107be8307582f7e4b3f`。
- 补充原图复核发现 J♣K♥ 对手下注264被识别为26，首版审计误判为 free-fold；事实头/客户端方案不再作为最终设计。纠偏版本改为 ddbaby 服务端直接处理，并限制翻前仅 BB/straddle option，部署证据待完成后补录。

## 2026-10-10 免费 check 纠偏版本部署

- 业务提交 `ae84dd1`。生产同版本 Go 1.22.2 全仓测试、handler/model/dal race 与构建通过；运行时二进制 SHA256 为 `57803792bb99ab24c24a8f22f50d2ceada27128a337bc5506b6982881527fb0b`，handler/model 源码 SHA256 分别为 `0df08577bc1ada32a36f56203a27e51caa7d0d4bbeca363239285921ed8b04a9`、`428549aaf584f23af13b9e33f7d0d3cb9bf07dbd5628b4d843abea47ca8dfea1`。
- 部署后 PID `2887499`、PPID `1`，`/proc/2887499/exe` 与运行时二进制 SHA 一致，监听 8888，本机 `/ping` 返回 `pong`；无鉴权 smoke 保持原 JSON，且不再输出首版两个决策事实头。
- 服务端回滚目录为 `/home/ubuntu/ddbaby-backups/20261010-181328-before-ae84dd1`。官网恢复已验证的 1.0.27 APK，公网 SHA256 为 `f5b80700112852b096a65004c38941e53d461975851b23f706d5c37538afefc4`；修正问题页公网 SHA256 为 `881ff80880ebcbe6f3467d3802c3f46a667aa38995f9c829337c72b62e812e04`，Web 资产回滚目录为 `/home/ubuntu/taoyuan-backups/20261010-181328-freecheck-correction`。

## 2026-10-10 Hero 全量复核问题页发布

- 审计业务提交为 `bd159b6`，识别业务提交为 dezhou `c50ad37`，客户端业务提交为 taoyuan-assist `847ff00`，均经 CR LGTM；只记录问题页发布事实的ddbaby提交为 `baff58b`。本轮不修改、不重启ddbaby业务进程，`baff58b` 不代表运行时代码部署。dezhou 与 taoyuan-assist 的部署事实记录提交分别为 `5ced119`、`5266dc1`。
- 公网问题页 `https://www.ddbaby.site/tyzs/strategy-issues-20261010.html` 返回HTTP 200、2,263,318 bytes，SHA256 `cc167cd416c6b472f265b192f24334634eb3764ff8a7125efce7480e6100711f`，与候选一致。
- 页面收录5张下注截断识别错误原图、A♦J♠免费check却fold、9♠9♣面对加注仍高概率fold，以及2个待solver复核状态；各分类保持独立。Web资产回滚目录为 `/home/ubuntu/taoyuan-backups/20261010-210905-before-1.0.29`。

## 2026-10-05 生产截图保留部署

- 业务提交：`e557483`。服务器 Go 1.22.2 隔离目录通过 service、全仓、service race
  测试及 `build.sh`；运行时二进制 SHA256 为
  `4eb739d7cb76ba52241b396dec0b36a5bdca9cf3b84f1b1ae1419014c40f55b4`。
- 部署后进程 PID=1098849、PPID=1，监听 8888；SSH 重连后 `/ping` 返回
  `{"message":"pong"}`，运行中 `/proc/1098849/exe` 与部署文件 SHA 一致。
- 首次清理前 dry-run 匹配 11,230 张普通过期图、1,842,131,538 bytes；实际删除数量
  和字节数完全一致，`changed_files=0`。二次执行匹配/删除均为 0。
- 清理前的 765 张 Hero 与 1,690 张 decided 共 2,455 个文件已生成逐文件 SHA256
  清单，清理后全部校验通过；JPEG 总数由 13,685 降为 2,455。
- 根盘使用率由 80% 降为 76%，可用空间由 8,198,955,008 bytes 增至
  10,064,629,760 bytes。
- cron 服务为 active；logrotate 规则显式使用 `su root adm`，适配服务器 `/var/log`
  权限策略并已原子替换到生产。生产配置 SHA256 为
  `1f0e465457a41fc7135cec658b3a785c6c1644fe146960e342b4ce0fbf353c72`；debug
  校验无错误，受控强制轮转生成 `.1.gz` 归档并重新创建 root:adm/0640 的当前日志。
- 回滚目录：`/home/ubuntu/ddbaby-backups/20261005-135714-before-e557483`，包含旧二进制、
  运行时源码、清理配置备份、替换前 logrotate 规则、候选/旧二进制校验值和清理前
  保留图片 SHA256 清单。
