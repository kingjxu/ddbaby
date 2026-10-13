# GTO 决策审计入库

## 目标

`POST /lyxz/gto_decision` 返回合法决策时，将 UID、决策时间、Hero 手牌、位置、stage、完整决策请求和最终决策响应保存到 MySQL。写库不能增加外部接口的数据库等待时间。

## 实现

- 新表 `gto_decision_record` 使用自增 `id` 主键，分别为 `uid`、`decision_time` 建普通索引；手牌、请求和响应使用 MySQL JSON 列。
- 默认策略记录原 `/gto/decision` 请求；非默认策略记录实际 v2 wire 字段 `vvip/pfr`。
- `decision_response` 保存 GTO 原始响应；`final_response` 保存合法性校验、all-in 上限修正和免费 fold→check 修正后的响应，因此既能复盘上游，也能对应客户端最终动作。
- handler 在主链路中只复制小型请求/响应对象并启动 goroutine；JSON 序列化和参数化 GORM `Create` 都在 goroutine 内完成。goroutine 使用独立3秒 context，写库失败只写错误日志。
- DDL 由 `dal/mysql/texas_poker/sql/20261013_create_gto_decision_record.sql` 管理；应用没有启用 `AutoMigrate`。

## 验证

- `GOTOOLCHAIN=go1.20.14 go test ./... -count=1`：通过。
- `GOTOOLCHAIN=go1.20.14 go test -race ./biz/handler ./dal ./model ./dal/mysql/texas_poker -count=1`：通过。
- 2026-10-13 已在生产 MySQL 应用版本化 DDL；复核确认自增主键、`uid`/`decision_time` 两个独立索引以及 JSON 列与脚本一致。
- 生产部署二进制 SHA256 为
  `56e16bba37b6cbc29eb1059d21be0964aa723a7157eda8d37ced30b2ca9d0ad7`；
  部署后 PID `3901582`、PPID `1`，`/proc/3901582/exe` 与候选一致，
  `/ping` 返回 `{"message":"pong"}`。回滚目录为
  `/home/ubuntu/ddbaby-backups/20261013-115907-before-bcc0ee1`。
- 首次部署 smoke 发现手工重启未继承 `GTO_API_KEY`，因此未将该次空决策当作通过。
  已将密钥迁移到 root-only 的 `/etc/ddbaby/runtime.env`（0600），通过
  `/home/ubuntu/ddbaby/run.sh` 注入环境后重启；密钥未写入仓库、命令行或报告。
- 使用同一张真实 preflop Hero 图分别回验 default 与 v2：两次 HTTP 均为 200，
  Hero 手牌均为 `K-D,9-D`、位置 `BTN-Hero`、stage `preflop`，响应动作均为
  `raise`。default 入库请求没有 `vvip/pfr` 字段；v2 入库值为 `42/20`；
  两条记录的 `decision_response.action` 和 `final_response.action` 均为 `raise`。
- 异步写入在接口返回后可查询到，证明生产表、连接和 goroutine 写入路径可用。
  回验使用的临时激活码及两条审计记录随后删除，未残留测试数据。
