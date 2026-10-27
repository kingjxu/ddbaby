---
version: "1.11"
last_updated: "2026-10-28"
status: "active"
---

# 环境与运维

## 2026-10-17 Result All in 展示兜底部署

- 业务提交：`145d733`。仅在 ddbaby 生成原有 `Result` 时，将达到
  `hero.bet + hero.stack` 全下目标的合法 bet/raise 显示为 `All in`；未修改
  IDL、响应字段、Android 或 APK。
- 部署前已撤销错误的双端实现方向：ddbaby 恢复至上一版二进制
  `cebc09100a7c8184ae936572a8871766a699f453deb685cdeeb7ba4a96fc7053`，官网 APK
  恢复 1.0.33/code26，公网文件 SHA256 为
  `f942589b9c86b0c7dab5d8e9c8a72e32ad6651588b0e5e47b13165f02269ce43`。
- 远端 Go 1.22.2 执行全仓测试、race
  `./dal ./model ./biz/handler`、专项 Result 边界测试和构建均通过。部署二进制
  SHA256 为 `acf4ee79f53ab964b5b2781210ff7e791e890befc58abfe976132d72011554f1`；
  handler 源码 SHA256 为
  `aa5d89cf85bde5624e59e3947baeed75767df48599ad6accddc74bcdf582ec08`。
- 2026-10-17 01:09:37 CST 完成原子切换，PID `970065 -> 972663`，新进程
  PPID=1，磁盘和运行中二进制 SHA 一致，监听 8888；本机 `/ping` 返回
  `{"message":"pong"}`，启动日志延迟扫描无 panic、fatal 或 error。
- 回滚目录：`/home/ubuntu/ddbaby-backups/20261017-010928-before-145d733`，包含
  上一版二进制、运行时源码归档以及部署前后事实。

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
- 生产首次按 1 天规则 dry-run 匹配 20,624 张、3,368,503,831 bytes；正式清理、
  边界补清理和配置替换后的最终补清理共删除 20,754 张、3,390,399,087 bytes，
  `changed_files=0`，最终 dry-run 匹配 0。
- 清理前 Hero/decided 分别为 5,030/3,646 张；最终因实时流量增加至
  5,046/3,654 张，均未减少。根盘可用空间由 742,875,136 bytes 增至
  4,154,245,120 bytes，服务 `/ping` 保持正常。
- 生产脚本与 cron 已原子替换，SHA256 分别为
  `6d42be352292192a86d6dac8584a1e7635a2d7379b27e33f1f16b1327422f989` 和
  `18f55ad93045d5ea34c8d1a1e8905fa7cd7be7b17ccd843395e9a8bc3671dd26`；
  权限为 root:root/0755 和 root:root/0644，cron active，回滚目录为
  `/home/ubuntu/ddbaby-backups/20261013-083636-before-image-retention-1d`。

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

## 2026-10-13 GTO 决策审计入库部署

- 业务提交 `bcc0ee1`，生产已创建 `gto_decision_record`：`id` 自增主键，
  `uid`、`decision_time` 分别建索引，请求、原始响应、最终响应使用 JSON 列。
- 运行时二进制 SHA256 为
  `56e16bba37b6cbc29eb1059d21be0964aa723a7157eda8d37ced30b2ca9d0ad7`；
  部署后 PID `3901582`、PPID `1`，`/proc` 二进制一致且 `/ping` 正常。
- `GTO_API_KEY` 由 root-only `/etc/ddbaby/runtime.env` 提供，权限 0600；
  `/home/ubuntu/ddbaby/run.sh` 只负责加载环境并 exec 服务。密钥不得写入仓库、
  群聊、日志或进程命令行。手工重启必须使用该 launcher，避免环境丢失。
- default 与 `42/20` v2 使用同一张真实 preflop Hero 图回验，均 HTTP 200
  且返回 `raise`；入库分别验证缺省 wire 字段与 `vvip=42,pfr=20`，并确认
  raw/final 响应分列。验证用激活码及审计行已删除。
- 回滚目录：`/home/ubuntu/ddbaby-backups/20261013-115907-before-bcc0ee1`。

## 2026-10-13 翻前 option check 补全部署

- 业务提交 `03b8a6b`。跨街补全会为未记录主动动作的最终强制盲注位补上
  preflop option check；有 straddle 时作用于 straddle，无 straddle 时作用于 BB。
- Go 1.22.2 下真实 J♦T♣ 回归、通用 straddle/BB option、既有跨街序列、
  model/dal race、全仓测试和构建均通过；CR 已确认 LGTM。
- 运行时二进制 SHA256 为
  `7cea4efc9ea4e695078cc6c15b12c497e4bc846560b639d0395eacf9a3641fc5`；
  部署后 PID `3924762`、PPID `1`，`/proc` 与磁盘二进制一致，端口 8888
  正常监听且 `/ping` 返回 `pong`。本轮未在线上执行全量回归或注入模拟牌局。
- 回滚目录：`/home/ubuntu/ddbaby-backups/20261013-133505-before-03b8a6b`。

## 2026-10-14 同街第二圈 action_taken 修复部署

- 业务提交 `362c927`。decided 帧会在满足合法免费过牌条件时记录 Hero check；
  当前街已有 Hero 动作且其后观察到对手动作时，`players.action_taken` 按累计
  history 恢复第二圈顺序，首圈继续使用既有快照推断。
- 服务器 Go 1.22.2 隔离目录完成全仓测试、model/dal race、K9 三个专项回归
  和构建，均通过。运行时二进制 SHA256 为
  `76135e36cb599bf332d9f6ad0b0e18640a760e59736a1fecef0267a7787ab3b5`；
  `model/recognize_poker.go` SHA256 为
  `b7ffe32d8072574d187985b5d819df67da75273d4f74d1351318b45afd72029b`。
- 部署后 PID `39301`、PPID `1`，`/proc` 与磁盘二进制一致，8888 正常监听，
  SSH 重连后 `/ping` 返回 `pong`，启动日志未发现 panic/fatal。
- 回滚目录：`/home/ubuntu/ddbaby-backups/20261014-101927-before-362c927`。

## 2026-10-16 逐街 action history 修复部署

- 业务提交链为 `439edb4`、`e9e3626`、`c5b0f2e`，均已通过 CR。最终行为包括：
  允许 `players.bet` 未记录强制盲注时的合法翻前 open raise；按实际翻后行动顺序补齐
  Hero 前方的零金额 check；跨街首帧 fold 根据上一街是否仍欠响应归入正确街道。
- 服务器使用 Go 1.22.2 对 `c5b0f2e` 干净 Git 归档执行全仓测试、`dal/model`
  race 测试和构建，均通过。运行时二进制 SHA256 为
  `a526f0a386781fcff598b2a2420f40ab43303ef6d9ede8872daadd765c288d07`；
  `model/recognize_poker.go` SHA256 为
  `f1287cad8567f3219b0d006c50c59113870d3292658464063be58b1db908673b`。
- 部署后 PID `751841`、PPID `1`，`/proc/751841/exe` 与磁盘二进制一致，8888
  正常监听，本机 `/ping` 返回 `pong`，启动后未发现 panic/fatal。
- 部署后使用同一候选源码重放 2026-10-15 21:00 后的 191 张识别结果：生成100个
  Hero请求、重复行动0；逐街审计只剩2手已确认的识别输入异常。99 open-raise 和最近
  进攻者/未响应玩家跨街 fold 聚焦测试通过；未额外调用策略服务。
- 回滚目录：`/home/ubuntu/ddbaby-backups/20261016-101549-before-c5b0f2e`。

## 2026-10-16 Hero 延迟显露行动修复部署

- 最终业务提交 `e2febba`，已完成 CR 并获得 LGTM。decided 只建立待确认的 Hero
  行动段；同街仅由 Hero 自身可见投入或状态变化解析，行动权再次回到 Hero 不作为
  check 证据。第二圈 call/raise 保留在第二圈，跨街按最短合法序列完成上一街。
- pending 未解析时，模型转换和 handler 均拒绝生成策略请求；门禁位于策略调用、
  决策审计写入、成功响应和计费之前。handler counter 回归确认策略 stub 调用为 0，
  pending 解析或跨街清空后恢复正常请求。
- 候选归档 SHA256 为
  `32341bef0b754eb4b8811f0039baee2509229e3a6cfa0a0cd2fd42312d23b379`。
  服务器使用 Go 1.22.2 对干净归档执行全仓测试、`dal/model/biz/handler` race 测试
  和构建，均通过。运行时二进制 SHA256 为
  `cebc09100a7c8184ae936572a8871766a699f453deb685cdeeb7ba4a96fc7053`；
  `model/recognize_poker.go` 与 `biz/handler/texas_poker_decision.go` SHA256 分别为
  `e93f0757fd63e2abb9cdee70c5dfed761187b53e9c94c92e3d4aaca2d7946c2e`、
  `8df2bad53fb76381653c6196811ad24b287fe2fcd4124d78f24fec69abc1dc25`。
- 生产于 2026-10-16 22:23:57 +08:00 开始切换，22:23:59 完成；PID 从
  `751841` 切换为 `928025`，新进程 PPID 为 1。`/proc/928025/exe` 与磁盘二进制
  SHA256 一致，8888 正常监听，`/ping` 返回 HTTP 200 和 `pong`，启动日志未发现
  panic/fatal。
- 单 Worker 对 2,227 个连续链路帧重放：Hero 帧 1,726，pending 门禁抑制 48，
  可生成策略请求 1,678，原始识别输入校验失败 2，重复行动 0。23:04 序列为
  `SB check -> BB check -> Hero bet37 -> SB fold -> BB raise121`；06:46 序列包含
  `Hero check -> MP+1 bet88 -> SB call88 -> Hero call88`，均保持合法时序。重放未调用
  生产策略服务，也未写生产审计或计费数据。
- 替换前二进制 SHA256 为
  `a526f0a386781fcff598b2a2420f40ab43303ef6d9ede8872daadd765c288d07`；回滚目录为
  `/home/ubuntu/ddbaby-backups/20261016-222357-before-e2febba`。

## 2026-10-26 AAPoker event 接收入口部署

- 最终业务提交 `67520a7`，将生产运行线 `6b8c7d3` 与 `/lyxz/gto_event` 合入
  `master`；CR 已确认 LGTM。入口只确认接收规范化事件，不参与牌局状态重建、策略、
  计费或持久化，并受请求体边界、全局/来源限流和完整日志字节预算保护。
- 服务器使用 Go 1.22.2 对干净归档执行全仓测试和构建，均通过。候选归档 SHA256 为
  `9dc7143abb5d759f621bc69452fa8e97dc8546dbb378fb52887e1a21c072764d`；
  运行时二进制 SHA256 为
  `47141a506fd940384caab4e42d03bc977ee0af00b26ebaf75a35e4ce2bb28288`。
- 部署后 PID 从 `1291086` 切换为 `125807`，新进程 PPID 为1；`/proc/125807/exe`
  与磁盘二进制一致，8888正常监听，`/ping` 返回HTTP 200。合法合成事件返回200和
  `BaseResp.StatusCode=0`，非法事件在本机与公网均返回400；新日志未发现 panic/fatal。
- 为采集首批真实协议，完整事件日志临时开启至 `2026-10-26 20:20:14 +08:00`，到期
  后由进程自动恢复为元数据日志。部署 smoke 只使用不含用户身份的合成事件；截至
  `2026-10-26 18:20:48 +08:00` 尚未观察到新的真实 AAPoker 事件。
- 回滚目录：`/home/ubuntu/ddbaby-backups/20261026-182014-before-67520a7`；运行日志：
  `/home/ubuntu/ddbaby/output/deploy-67520a7-20261026-182014.log`。

## 2026-10-26 AAPoker event 决策链路部署

- 最终业务提交 `dbf6d8f`，在事件接收链路上增加牌局状态重建、Hero 行动识别、GTO
  请求、结果回放、计费与决策审计。CR 已在用户明确接受本机注入风险的边界下确认
  LGTM；临时失败返回 503，成功结果按隔离的 session+turn 键缓存 30 分钟，容量上限
  1024，重复请求只回放完整结果，不重复调用 GTO、计费或写审计。
- 候选干净归档 SHA256 为
  `15c3884fa700486639128815fb81746b2bd768f8dd284dfa4cbdd2d253de5b44`。
  服务器使用 Go 1.22.2 执行全仓测试、`biz/handler` 与 `service` race 测试和构建，
  均通过。运行时二进制大小为 47,583,887 bytes，SHA256 为
  `16ceddf4c32ad61eedaf51be47aea47dc9b89e1e05f56b282d1514803ed99753`。
- 部署后 PID 从 `125807` 切换为 `193957`，新进程 PPID 为 1；磁盘二进制与
  `/proc/193957/exe` SHA256 一致，8888 正常监听，`/ping` 返回 `pong`。本机合法
  合成 `/lyxz/gto_event` 返回 HTTP 200 和 `BaseResp.StatusCode=0`，非法 JSON 返回
  HTTP 400 和 `invalid_event`；公网 80 端口的合法合成请求返回 HTTP 200。启动日志
  未发现 panic/fatal。
- 17770 仍是未认证的本机 TCP 入口，任意同机 App 均可注入事件；桃园会附加当前登录
  用户的 uuid 与 Bearer token，因此伪造事件可借当前身份触发 GTO、计费和审计。该风险
  由用户明确接受以简化本轮真机验证，不能表述为入口已受保护。
- 本轮线上 smoke 只使用合成事件，尚未完成真实 AAPoker 真机事件、桃园悬浮窗结果及
  审计页面记录的端到端验证。
- 回滚目录：`/home/ubuntu/ddbaby-backups/20261026-224028-before-dbf6d8f`；运行日志：
  `/home/ubuntu/ddbaby/output/deploy-dbf6d8f-20261026-224028.log`。

## 2026-10-27 packageName 分流与持续完整事件日志部署

- 最终业务提交为 `7386fdd`，其中 `8a2cb84` 按用户明确要求移除完整事件日志的运行期开关、到期时间和日志字节预算，`7386fdd` 增加显式 packageName 分流。AAPoker `com.plusaa.amula/com.ty.aapk` 继续进入既有状态重建、GTO、计费、审计和结果回放；DPZX `com.dpzx.pro` 在协议分析阶段只记录完整事件并返回基础成功；未知或缺失包名在记录完整 body 前返回400。CR 已确认 LGTM。
- 干净归档 SHA256 为 `d3e9872981c5e3cb7dbe6c4c965f555f224cb143fcf69b00697748c9b5252179`。服务器使用 Go 1.22.2 执行 `go test ./... -count=1`、`go test -race ./biz/handler -count=1` 和构建，均通过。运行二进制大小为 47,588,272 bytes，SHA256 为 `d34698c99b4b6865d4994d908e5ce0b1ccfae8c9ed4425c2f130c04df974aa9b`。
- 2026-10-27 08:18:58 +08:00 完成切换，PID 从 `234556` 切换为 `334419`，新进程 PPID 为1；磁盘二进制与 `/proc/334419/exe` SHA256一致，8888正常监听，本机 `/ping` 返回`pong`。旧版使用的 `DDBABY_GTO_EVENT_FULL_LOG` 与 `DDBABY_GTO_EVENT_FULL_LOG_UNTIL` 已从运行环境移除，完整事件日志不再自动过期。
- 本机合成smoke中，AAPoker和DPZX均返回HTTP 200与`BaseResp.StatusCode=0`；两者完整marker各写入日志一次。未知和缺失packageName均返回固定HTTP 400 `invalid_event`，对应原始marker未写入日志。公网 `/lyxz/gto_event` 的DPZX合成事件返回HTTP 200。新日志未发现panic/fatal。
- 持续完整日志可能快速增长，并长期留存uid、房间与牌局内容；该风险由用户在协议分析阶段明确接受。请求仍受512 KiB请求体上限以及全局300次/分钟、单来源120次/分钟限流约束，但这些限制不构成日志容量保证。
- 本轮仅完成服务端合成smoke；真实DPZX牌局采集与还原、双APK真机转发以及后续DPZX GTO策略尚未验证或实现。17770仍未认证，外层packageName只是路由优先级，任意同机App仍可伪造事件并借当前桃园身份触发AAPoker GTO、计费和审计。
- 回滚目录：`/home/ubuntu/ddbaby-backups/20261027-081858-before-7386fdd`；运行日志：`/home/ubuntu/ddbaby/output/deploy-7386fdd-20261027-081858.log`。

## 2026-10-27 分析期入口精简部署

- 最终业务提交为 `d0c7578`，其中 `f545c9e` 移除 `/lyxz/gto_event` 的全局与来源
  限流、HTTP 429 和聚合丢弃统计；`d0c7578` 将入口校验精简为：请求体非空且不超过
  512 KiB、内容是单一 JSON 对象、`packageName` 是路由表中的字符串。其余事件字段
  按 best-effort 解码并完整记录。集中路由表当前将 `com.plusaa.amula`、
  `com.ty.aapk` 分派到 AAPoker 状态机，将 `com.dpzx.pro` 分派到只记录日志的
  DPZX 路径；未知、缺失或非字符串包名仍在完整 body 日志前返回 HTTP 400。CR 已对
  两个业务提交确认 LGTM。
- 干净归档 SHA256 为
  `c022e91478cd2db08554bd1c591266358e4773ffb0e935b31916f351f57dccd1`。服务器使用
  Go 1.22.2 执行 `go test ./... -count=1`、
  `go test -race ./biz/handler -count=1` 和构建，均通过。运行二进制大小为
  47,563,592 bytes，SHA256 为
  `331e6ff812265d48412384e8c70115c3e394b08f5da540165007abee9ecc6d64`。
- 2026-10-27 08:45:03 +08:00 完成切换，PID 从 `334419` 切换为 `341888`，新进程
  PPID 为 1；磁盘二进制与 `/proc/341888/exe` 指向同一运行文件，8888 正常监听，
  本机 `/ping` 返回 `pong`，启动日志未发现 panic/fatal。
- 合成 smoke 中，字段类型异常且缺少常规数据字段的 DPZX 事件返回 HTTP 200 并完整
  记录，AAPoker 基础事件返回 HTTP 200；未知、缺失、非字符串 `packageName` 均返回
  HTTP 400，三个拒绝请求的原始 marker 均未写入日志。连续 305 条 DPZX 事件全部返回
  HTTP 200 且日志逐条记录，确认当前无请求频率上限；公网 DPZX 合成事件返回 HTTP 200。
- 当前入口会对每条白名单来源事件持续写完整 body，且没有请求频率、来源配额、日志
  字节预算或自动截止时间。高频大请求可能耗尽日志管道或磁盘；这是用户为协议分析阶段
  明确接受的风险。512 KiB 单请求上限只限制单包大小，不限制累计写入量。
- 本轮仍只完成服务端合成 smoke；真实 DPZX 牌局采集与还原、双 APK 真机转发和 DPZX
  GTO 策略尚未验证或实现。17770 仍未认证，外层 `packageName` 只表示路由优先级，
  不构成调用方身份认证。
- 回滚目录：`/home/ubuntu/ddbaby-backups/20261027-084503-before-d0c7578`；运行日志：
  `/home/ubuntu/ddbaby/output/deploy-d0c7578-20261027-084503.log`。

## 2026-10-27 DPZX 事件决策链路部署

- 最终服务端范围为 `daf577e..3e8ac25`，桃园协议文档范围为
  `3a00d20..3d165e4`，均经 CR 确认 LGTM。`com.dpzx.pro` 现在进入与
  AAPoker 隔离的状态机：根据430/431/492/432/433/403/490/435重建牌局，在
  Hero 行动点生成统一GTO请求，并复用现有Bearer身份校验、激活、GTO、按次计费、
  `gto_decision_record`审计、503重试及30分钟/1024条结果回放。旁观、动作缺失、
  未知动作、跳街、公共牌前缀变化/重复以及Hero私牌异常均使本手停止决策。
- 最近生产旧日志的离线重放覆盖约13分钟、14手牌和25个Hero行动点，包含preflop、
  flop、turn、river；生成请求均通过统一请求校验。该结果证明样本能够被当前协议
  模型还原，不等同于部署后的真实生产GTO、SQL或Android悬浮窗验证。
- 干净归档大小为10,391,854 bytes，SHA256为
  `59b83d9b34f9fd72fc58535c9d7411cbebaa49040f97de100b034e80a67c1080`。服务器使用
  Go 1.22.2 对归档执行`go test ./... -count=1`、
  `go test -race ./biz/handler ./service -count=1`和构建，均通过。候选/运行二进制
  大小为47,609,881 bytes，SHA256为
  `1cd596f6c38050732b8919031131b2d17083e2f1ae540a01c5438ae35ef9c628`。
- 2026-10-27 13:25:58 +08:00完成切换，PID从`341888`变为`410577`，新进程
  PPID为1；磁盘和`/proc/410577/exe` SHA256一致，8888正常监听，本机`/ping`
  返回`pong`，启动日志无panic/fatal。无身份、无状态的DPZX合成事件返回HTTP 200
  与基础成功且marker只完整记录一次；未知packageName返回固定HTTP 400且marker未
  落日志。该smoke不会进入GTO、计费或审计。
- 部署后观察窗口内尚无新的真实DPZX事件，因此真实DPZX牌局经生产GTO返回、SQL
  审计落表及桃园真机悬浮窗展示仍待下一手实流验证；不能用本次合成smoke或离线
  重放替代该端到端结论。既有17770未认证本机注入风险、无频率上限的持续完整日志
  风险以及外层packageName仅作路由优先级的边界保持不变。
- 回滚目录：`/home/ubuntu/ddbaby-backups/20261027-132558-before-3e8ac25`；运行日志：
  `/home/ubuntu/ddbaby/output/deploy-3e8ac25-20261027-132558.log`。

## 2026-10-27 event 前置激活检查与重构部署

- 用户明确要求部署；业务版本为远程master已审查通过的`164f395`，替换此前运行的
  `3e8ac25`。同时包含IDL生成路由、Handler/service分层、Hertz统一4 MiB缓冲、
  共享命名和流程注释。非空uuid在最前面的check阶段查询数据库激活有效期与次数，
  不再校验Bearer；额度耗尽即503，包括已付费结果的重传，按用户确认不设回放例外。
- 干净git归档`/home/ubuntu/ddbaby-164f395.tar.gz`为10,413,896 bytes，SHA256
  `3e1261bff0adf1e0012995fb707208da6648fad899f2b65b41c6e539a369ed47`，本机与服务器一致。
  在独立目录`/home/ubuntu/ddbaby-releases/164f395-20261027`使用Go 1.22.2、
  `GOTOOLCHAIN=local GOPROXY=off`运行`go test ./... -count=1`、
  `go test -race ./biz/handler ./service . -count=1`及构建，全部通过。
  候选/运行二进制为47,638,962 bytes，SHA256
  `b94781700fdf0d32710442c019e724713096457ff6f80d1fcf3c70d7c7286645`。
- 首次21:38:19切换时，部署探针误将`/ping`的JSON响应与纯文本比较，造成误回滚；
  新旧服务日志均已正常监听，没有应用启动故障。自动回滚从旧PID`410577`恢复到
  `529982`，修正探针按`message=pong`判断后，于21:39:10成功切换到`530326`。
  最终进程PPID为1，8888正常监听，磁盘和`/proc/530326/exe` SHA256一致，
  本机`/ping`返回`{"message":"pong"}`，观察窗口无panic/fatal。
- 合成smoke：无uuid的AAPoker/DPZX分别HTTP200并完整记录；未知及缺失来源HTTP400；
  构造的无激活uuid返回HTTP503及`active_code_unavailable`。三个拒绝请求的原始marker
  不落完整日志且响应不回显body；text/plain加query包名也返回400。公网实际业务路径
  `/lyxz/gto_event`的无uuid DPZX smoke返回200且marker只记录一次。公网`/ping`未由nginx
  转发（404），健康以本机ping及公网业务路径为证。上述smoke没有产生GTO调用、计费或审计。
  服务器结果保存在`/home/ubuntu/ddbaby-164f395-smoke.json`；测试输出为
  `/home/ubuntu/ddbaby-164f395-test.log`和`/home/ubuntu/ddbaby-164f395-race.log`。
- 最终回滚目录`/home/ubuntu/ddbaby-backups/20261027-213910-before-164f395`保存旧二进制
  （SHA256 `1cd596f6c38050732b8919031131b2d17083e2f1ae540a01c5438ae35ef9c628`）、
  原运行环境及启动信息。环境快照root所有且0600，不在文档回显内容。
  可执行`sudo python3 /home/ubuntu/ddbaby-backups/20261027-213910-before-164f395/rollback.py`
  恢复旧二进制及原环境；脚本会核对当前部署PID，进程变化时需先人工核对。
  最终运行日志`/home/ubuntu/ddbaby/output/deploy-164f395-20261027-213910.log`。
- 本轮未验证真实生产GTO→SQL→真机悬浮窗链路。移除token后不能证明uuid归属、可能冒用
  额度与污染审计；每条非空uuid事件查库，耗尽后客户端持续重试，最后一次付费响应可能丢失。
  17770未认证、明文传输、无频率上限的完整日志增长风险继续保留。DPZX straddle仍按用户
  决定等待更多样本，不根据下注金额推断，本轮未调整其映射。

## 2026-10-27 DPZX位置、牌号及非法决策终止部署

- 用户要求立即部署。业务提交`a405080`，CR已对增量`9a11340..a405080`给出LGTM；包含此前已通过审查的位置纠正、牌号转换和非法GTO结果审计。
- 本地Go1.20.14全仓与service/handler/main race通过；服务器Go1.22.2离线全仓、同范围race和构建通过。源码归档SHA256：`50613dee0dd51df2e3c204ffe31eadecf26d971bc95e635c70fab4c3739fbe1b`，服务器源码`/home/ubuntu/ddbaby-releases/a405080-20261027`，测试输出`/tmp/a405080-full.log`与`/tmp/a405080-race.log`。
- 2026-10-27 23:42:21 CST切换，PID`530326 → 561408`、PPID=1；保留原启动参数、环境及工作目录。运行二进制与磁盘SHA256均为`ad19bb8d343895305487371a61dfe4c32ee166225c3bfb611c72a5257a5fb4d9`，47,633,476字节。
- 本机`/ping`返回JSON pong。合成smoke：两种来源无uuid事件200；缺失/未知来源400，非JSON+query400；未激活uuid503；拒绝事件marker未落完整日志。记录`/home/ubuntu/ddbaby-a405080-smoke.json`。这些请求不触发GTO或计费。
- 日志`/home/ubuntu/ddbaby/output/deploy-a405080-20261027-234221.log`。初次核验仅有2条合成采集事件、0次GTO请求、0次决策重试和0次panic，尚无部署后的真实牌局证明GTO→SQL→Android提示恢复。重启清空内存牌局，需收到新一手完整事件重建。
- 回滚目录`/home/ubuntu/ddbaby-backups/20261027-234221-before-a405080`保存旧164f395二进制、私有环境、启动信息及deployment.json；`sudo python3 <该目录>/rollback.py`恢复旧版本并检查JSON pong，脚本核对预期PID561408；进程变化须先核对。目录700、环境文件600、脚本700，不输出敏感环境值。
- 非法/空决策返回200 decision_rejected且无Result，终止标记防止重复GTO；不阻止后续行动。临时故障仍503。激活前置及额度耗尽拒绝回放的已接受边界不变。数据库故障或进程退出时异步审计仍不保证必达；完整牌号协议和真机链路证据边界详见牌号报告。

## 2026-10-28 DPZX多级straddle部署

- 用户确认startState=3/4为straddle+1/+2后继续修复漏决策，提交`3d4f0e2`，整体范围`02838a4..3d4f0e2`经CR复审LGTM。开局按明确标记而非下注金额判断位置；房间IsStraddle=false不覆盖逐手标记。
- 本地Go1.20.14全仓与service/handler/model/main race通过；服务器Go1.22.2离线全仓、相同race和构建通过。服务器源码`/home/ubuntu/ddbaby-releases/3d4f0e2-20261028`，归档SHA256`07e75f20103498afe35725fae322f0e9367f04982bef2a9e12484f3df465db6c`；测试日志`/tmp/3d4f0e2-full.log`、`/tmp/3d4f0e2-race.log`。
- 00:13:28 CST切换`561408 → 571151`，PPID=1，保留原环境、参数、工作目录。运行与磁盘二进制SHA256均为`34351184c94be99beba31209de70b6800d81bd12e2cea37577d310717a2b7e3a`，47,636,596字节。JSON pong通过。
- 合成smoke验证两来源无uuid采集200、未知/缺失来源及非JSON+query400、未激活uuid503，拒绝marker不落完整日志；证据`/home/ubuntu/ddbaby-3d4f0e2-smoke.json`，日志`/home/ubuntu/ddbaby/output/deploy-3d4f0e2-20261028-001328.log`。初次扫描仅2条合成事件，无真实新牌局；上游新增位置标签、Straddle字段语义及真机提示仍未验证。
- 回滚目录`/home/ubuntu/ddbaby-backups/20261028-001328-before-3d4f0e2`含旧a405080二进制、私有环境、启动参数及deployment.json；`sudo python3 <该目录>/rollback.py`核对PID571151后恢复并检查JSON pong。目录700，环境600，脚本700。
- 第450手AK及519手J6日志回归仅证明候选请求构造；上游与真机证据边界见`reports/dpzx-straddle-20261028.md`。激活前置、非法决策终止重试与异步审计故障边界不变。

## 2026-10-28 DPZX straddle称谓纠正部署

- 用户更正3/4应分别为straddle、straddle+1，业务`78257ad`经CR LGTM，于00:34:28 CST上线。强制投入与顺序规则未变；当前不生成straddle+2标签。
- 本地Go1.20.14及服务器Go1.22.2全仓、service/handler/model/main race和服务器构建通过。源码`/home/ubuntu/ddbaby-releases/78257ad-20261028`；归档SHA256`0632dccffbe05fc3c0a5a57efbc5b9d072ecbd2ab04fb86c7d9a0324e09aae64`，测试日志`/tmp/78257ad-full.log`与`/tmp/78257ad-race.log`。
- PID`571151 → 577335`，PPID=1，沿用原环境/启动参数；运行与磁盘SHA256均为`4cbc1f5dbf00bbe8766371151d5fe63ebc9fa549e2d290be6dba3b8f9eff1f40`，47,636,108字节。JSON pong及既有合成smoke通过：两来源无uuid200、未知/缺失来源和非JSON/query400、未激活uuid503、拒绝marker不落完整日志；证据`/home/ubuntu/ddbaby-78257ad-smoke.json`。
- 日志`/home/ubuntu/ddbaby/output/deploy-78257ad-20261028-003428.log`；回滚`/home/ubuntu/ddbaby-backups/20261028-003428-before-78257ad/rollback.py`核对PID577335后恢复旧3d4f0e2二进制和私有原环境。目录700、环境600、脚本700。上游straddle+1标签及最高档Straddle字段语义、真实GTO→SQL→端上提示仍待验证，合成smoke不代表该链路已通过。
