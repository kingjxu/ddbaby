---
version: "1.1"
last_updated: "2026-10-03"
status: "active"
---

# 故障排查

## 识别成功但决策金额非法

- 先检查决策请求中 `current_player_position` 对应玩家的 `stack`、`bet` 和全桌最大 `bet`。
- `is_hero_turn=true` 且 stack=0 通常表示识别漏读，不应调用决策服务。2026-10-02 的1176x2400截图曾触发该问题。
- `bet/raise` 的 `raise_size` 是目标总下注额；超过 `hero.bet + hero.stack` 时保留策略动作并收敛为该 all-in 上限。面对已有下注不能返回 bet，面对下注时不能 check。
- handler 对超过后手的金额先做 all-in 收敛，再执行其他合法性校验；其余异常返回空动作并跳过计次。排查时同时保留识别响应、完整决策请求和原始决策响应，不能只看最终中文动作。

## 连续快照中的 Hero 下注丢失

- 2026-10-02 18:55:07 的历史牌局中，Hero 初始 stack=194、bet=0；open 33 后 stack=161、底池 25→58，但当时识别结果的 `current_bet=null` 被解码为0，导致差分历史漏掉 `CO-Hero raise 33`。
- 先核对前后快照的物理座位、stage、stack、current_bet 和 main_pot。同街且底池增长可以完整解释筹码减少时，ddbaby 会恢复缺失的累计下注；否则不做几何猜测。
- 当前 dezhou 重放该组图已能在 open 后与 18:55:07 Hero 帧直接识别 bet=33；ddbaby 兜底用于历史版本和未来偶发 OCR 漏值。
