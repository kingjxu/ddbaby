---
version: "1.1"
last_updated: "2026-10-03"
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
