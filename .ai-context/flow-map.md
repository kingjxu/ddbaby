---
version: "1.6"
last_updated: "2026-10-13"
---

# 德州扑克决策数据流

1. `TexasPokerDecisionHandler.Handle` 默认以 `parse_all=false` 调用 `service.RecognizePoker`，先做轻量 Hero 轮次检测。每次识别到 Hero 行动帧后，仅将紧接着的一帧设为 `parse_all=true`；该后继帧完整识别为非 Hero 后恢复轻量检测。
2. Hero 行动帧以及它后面一次完整解析的牌局帧写入 `dal.SaveUserData`；同一 BTN、同一 stage 的快照在覆盖前比较状态，把确定的下注、加注、跟注或弃牌追加到内存动作历史。更后面的非 Hero 帧不做全桌 OCR，也不进入缓存。
3. `dal.GetLastUserData` 从最近一次 BTN 变化之后取出当前手牌的全部阶段。
4. `model.Conv2TexasGtoDecisionReq` 调用 `buildPlayers` 与 `buildActionHistory`；优先输出连续快照累计的动作，首次进入阶段或发生漏帧时按最终下注额生成最小合法历史。
5. 位置映射从 `ButtonSeat` 开始遍历物理座位，只保留本手实际参与者：`empty` 和 `waiting` 不占位置，弃牌玩家仍占位置。若按钮或后续强制位置对应的物理座位无人，`BTN`、`SB`、`BB`、`straddle` 依次顺延给下一名实际参与者。
6. 转换后先校验当前行动玩家存在、active 且剩余筹码大于0。客户端策略为默认 `0/0` 时交给原 `service.GtoDecision`；六组白名单 VPIP/PFR 之一则映射为上游 `vvip/pfr` 并走 v2。策略值不进入牌局缓存，也不改变识别、位置和行动历史。
7. 决策返回的 bet/raise 若超过 Hero 可用总额，先保留动作并把目标额收敛为 `hero.bet + hero.stack`（all-in），再校验动作与当前下注是否相容；其余非法结果返回空动作且不计次。
8. 合法决策为 `fold` 且 Hero 当前投入不低于本轮最大投入时，ddbaby 将最终动作规范化为 `check`；面对更高下注时不改写。客户端只消费原有 JSON 响应。
9. 合法决策完成 all-in/free-check 规范化后，复制实际上游请求、原始上游响应和最终响应，通过独立 goroutine 写入 `gto_decision_record`；写库失败只记录日志，不阻塞或改写客户端响应。

## 修改约束

- `buildPlayers` 与 `buildActionHistory` 必须使用相同的桌容量、空座过滤与位置表。
- 空座影响本手实际参与人数；弃牌不能触发位置重排。
- 缓存切手边界目前依赖 `ButtonSeat` 变化。
- SB、BB、straddle 的初始投入属于强制盲注，不能直接转换为自愿 `call` 或 `raise`。
- `action_history.amount` 保留每次动作当时的目标下注额，不能统一覆盖为阶段最大值。
- `parse_all` 是一次性状态：Hero 行动帧把它置为 `true`，紧接着的一张有效非 Hero 完整识别结果保存后将它恢复为 `false`。这样可以补充 Hero 动作后的牌局变化，同时避免后续所有非 Hero 帧持续执行全桌 OCR。3bet/4bet 仍以 Hero 快照的最终下注额和行动顺序推断，并结合这一张后继帧或后续 Hero 快照累计可观察差分。
- `is_hero_turn=true` 但 Hero stack<=0 属于不可用于决策的识别结果；不能继续请求决策服务。
- 决策协议中的 `raise_size` 是目标总下注额。超过 Hero 可用总额时按 all-in 上限收敛；check/fold/call 必须为0；bet只允许在无人下注时；raise 要求牌桌已有下注且目标仍高于当前最大下注，因此 Hero 已匹配 BB/straddle 后的 option raise 也是合法动作。
- 免费 check 兜底必须在决策响应通过校验后执行，并从同一决策请求的玩家投入计算；不能根据客户端显示文本或截图推断。
- 缺少 `vpip/pfr` 与显式 `0/0` 等价，必须保持旧客户端和原决策接口兼容；非默认值必须成对命中固定白名单，不能只传一个值或透传任意百分比。
- 决策审计只记录通过请求/响应校验并准备返回客户端的合法决策。v2 记录必须包含实际线上的 `vvip/pfr`；原始响应与规范化后的最终响应分别保存。异步任务不能复用会随 HTTP 请求结束而取消的 context，也不能因写库失败影响主链路。
