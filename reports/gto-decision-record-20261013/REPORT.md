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
- default 与 v2 的真实写入回验在代码部署后补录。
