---
version: "1.0"
last_updated: "2026-10-13"
---

# 数据模型

## `user_active_code`

德州扑克激活码及用户绑定表。`id` 为自增主键，`user_id` 关联客户端 UUID；`code_type` 区分按次数和按时间过期，`invoke_cnt/total_cnt` 与 `expire_at` 保存对应额度。决策 handler 在识别前查询，合法按次决策成功后更新调用次数。

## `gto_decision_record`

合法 GTO 决策返回客户端前写入的审计表，建表脚本为 `dal/mysql/texas_poker/sql/20261013_create_gto_decision_record.sql`。

- `id`：`BIGINT UNSIGNED` 自增主键。
- `uid`：客户端已通过激活码校验的 UUID，独立索引 `idx_gto_decision_uid`。
- `decision_time`：决策完成时间，毫秒精度，独立索引 `idx_gto_decision_time`。
- `hero_cards`：决策请求中 Hero 的两张手牌，JSON。
- `position`：`current_player_position`。
- `stage`：当前 `preflop/flop/turn/river`。
- `decision_request`：实际发送给 GTO 的完整请求 JSON；v2 包含映射后的 `vvip/pfr`。
- `decision_response`：GTO 服务返回的原始响应 JSON。
- `final_response`：通过合法性校验并完成 all-in/free-check 规范化后的响应 JSON，与客户端最终动作一致。

写入通过独立 goroutine 执行并使用3秒后台超时；请求上下文结束不会取消写入，失败只记录日志，不改变决策响应。当前仓库不执行 `AutoMigrate`，部署前必须先应用版本化 DDL。
