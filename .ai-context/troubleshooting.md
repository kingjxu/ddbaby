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

## 图上可见的 Hero 下注未进入历史

- 2026-10-02 18:54:54 原图明确显示 Hero 已下注 33，当时 dezhou
  返回 `current_bet=null`，因而 ddbaby 无法生成 `CO-Hero raise 33`。
- 这类问题必须修复 Hero 下注 ROI/OCR，并用原图验证
  `current_bet=33`。禁止在 ddbaby 中用筹码减少量或底池增长量反推下注；
  这两个字段也可能漏读，且无法证明单个玩家的具体动作。
- 当前 dezhou 线上版本已通过备选 Hero 下注 ROI 直接读取该 33；
  同一原图连续重放 3 次均返回 `current_bet=33`。
