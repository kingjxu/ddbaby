---
version: "1.3"
last_updated: "2026-10-17"
status: "active"
---

# 故障排查

## 固定 BTN 时 action_history 串到下一手

- 不能把 `button_seat` 不变视为同一手牌；部分牌桌会连续多手显示同一 BTN。
- 先检查相邻完整识别是否出现街道回退或两张完整 Hero 手牌变化。任一信号成立时，新快照应从空历史开始，缓存查询也只能返回该边界之后的数据。
- GTO 请求中的 `action_history` 必须按 preflop、flop、turn、river 单调推进，且时间戳严格递增。出现 river 后跟 preflop、历史包含未来街道或时间戳倒退时，ddbaby 应在调用决策服务前拒绝请求。
- Hero 手牌 OCR 在同一手内发生变化时，当前实现选择保守切断历史，避免将不确定数据继续串入策略请求；排查时应同步核对 dezhou 原始识别。

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

## 同一条街第二次轮到 Hero 但端上无提示

- 先按同一个 `decision_cycle_id` 对齐识别、handler 和客户端日志。识别成功但 handler 返回 `hero action is still pending` 且 `Result` 为空时，端上没有可展示内容，问题在服务端动作历史门禁。
- Hero 上一圈 check 后投入仍为0，不能只靠投入变化解除 pending。若连续两个完整 Hero 帧的街道、按钮、Hero 投入/后手/状态及各座位投入/状态完全一致，这些 Hero 字段也与上一圈基线一致，并且已有对手下注或状态变化，服务端应把上一圈 Hero 结算为 check 并恢复决策。Hero 后手下降但投入仍未显露时继续 pending。
- 只有一个返回 Hero 帧、两帧桌面状态不一致或没有对手可见变化时继续保持 pending，避免把 OCR 抖动或 Hero 筹码延迟显露误判成 check。
