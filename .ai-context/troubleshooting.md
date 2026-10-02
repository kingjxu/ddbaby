---
version: "1.0"
last_updated: "2026-10-02"
status: "active"
---

# 故障排查

## 识别成功但决策金额非法

- 先检查决策请求中 `current_player_position` 对应玩家的 `stack`、`bet` 和全桌最大 `bet`。
- `is_hero_turn=true` 且 stack=0 通常表示识别漏读，不应调用决策服务。2026-10-02 的1176x2400截图曾触发该问题。
- `bet/raise` 的 `raise_size` 是目标总下注额，必须大于0且不超过 `hero.bet + hero.stack`；面对已有下注不能返回 bet，面对下注时不能 check。
- handler 对上述异常返回空动作并跳过计次。排查时同时保留识别响应、完整决策请求和原始决策响应，不能只看最终中文动作。
