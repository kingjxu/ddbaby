---
version: "1.2"
last_updated: "2026-10-05"
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
  普通 `*.jpg` 只保留最近 6 小时。
- 清理脚本为 `/usr/local/sbin/ddbaby-cleanup-images`，cron 每小时第 17 分钟执行；
  运行日志写入 `/var/log/ddbaby-image-cleanup.log` 并按周轮转。
- 清理与 ddbaby 图片落盘共用 `/run/lock/ddbaby-images.lock`；清理持锁完成最终身份校验
  和删除，所有生产图片写入必须通过 `writeImageToFileWithLock` 持同一把锁，避免删除
  检查通过后被同名新文件替换。
- 修改清理规则时必须先执行 `--dry-run`，不得扩大到子目录、符号链接、Hero 或
  decided 证据图。

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
  权限策略并通过 debug 校验。
- 回滚目录：`/home/ubuntu/ddbaby-backups/20261005-135714-before-e557483`，包含旧二进制、
  运行时源码、清理配置备份、候选/旧二进制校验值和清理前保留图片 SHA256 清单。
