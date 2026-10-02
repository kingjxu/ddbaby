---
version: "1.0"
last_updated: "2026-10-02"
status: "active"
---

# 环境与运维

## 生产服务器

- 服务目录：`/home/ubuntu/ddbaby`
- 进程：`output/bin/hertz_service`
- 监听端口：8888
- 本机健康检查：`GET http://127.0.0.1:8888/ping`，成功返回 `{"message":"pong"}`。
- 8888 未向公网开放，外部直连超时不代表服务异常；应在服务器本机检查。

## 2026-10-02 部署

- 业务基线：`fix/ddbaby-poker-regression-20261002`，提交 `a6d9d1e`。
- 运行时源码与本地提交逐文件 SHA256 一致；部署二进制 SHA256：`62fbdf8c592b139d06ee52e1fee042ef298de61a14a202dff02f7ee855d7f1d5`。
- 远端使用 Go 1.22.2 执行 `go test ./...` 和 `build.sh`，均通过。
- 进程以 root、PPID=1 独立运行；部署后 PID=38187，本机 `/ping` 验证通过。
- 回滚目录：`/home/ubuntu/ddbaby-backups/20261002-1649-before-a6d9d1e`，包含旧二进制和变更前运行时源码归档。
