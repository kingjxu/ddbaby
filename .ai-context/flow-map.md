---
version: "1.15"
last_updated: "2026-10-28"
---

# 德州扑克决策数据流

## AAPoker事件链路

1. 桃园将规范化事件与当前登录账号`uuid`、VPIP/PFR一起串行上报；ddbaby先执行统一4 MiB、IDL字段绑定和来源检查，再对非空uuid复用decision的数据库激活校验；无频率限制和日志预算，激活失败在完整日志和状态更新前返回503。
2. 服务端只消费`TABLE + ServerToClient`状态，按`uuid + AAPoker uid + dataExtra`隔离牌桌；`ENTER_ROOM/SEAT_DOWN`用内部userId确认Hero座位，旁观状态不推断Hero。
3. `START_INFOR`建立新手，`RECV_ACTION`按`operationRound`幂等累计动作与目标下注额，`RECV_CARDS`推进街道并清零本街下注，`SHOW_SIDE_POTS`校正底池。物理座位从banker开始映射到与截图链路一致的位置名。
4. `operationID`轮到Hero时，只有两张Hero手牌、对应街道公共牌、active状态和剩余后手全部有效才生成`TexasGtoDecisionReq`。同一行动点只有一个请求在途；调用失败保留资格，成功后忽略重传。
5. 激活校验在完整日志、状态更新和缓存回放之前完成；Hero决策复用查到的激活信息，执行策略白名单、上游调用、响应合法性、all-in/free-check规范化、次数计费和`gto_decision_record`审计。带非空`Result`的响应由桃园转给现有悬浮窗。

## DPZX事件链路

1. 服务端只消费`com.dpzx.pro + TABLE + ServerToClient`事件，按`uuid + DPZX uid`维护当前桌状态，结果缓存键继续包含`roomID + HandNum + cmd432时间戳 + stage`，防止账号、房间和行动点串线。
2. cmd 430建立新手的房间、庄位、Hero座位、盲注和前注上下文；cmd 431以本手实际参与者及盲注快照启动状态机。`mySeat`必须对应本手参与座位，旁观状态不触发决策。位置以431的startState=1/2确定SB/BB，多人桌BTN为SB之前的本手参与者，单挑BTN兼SB；不能保留已离座bankIndex作为空庄位。SB缺失仅允许仍参与且顺序一致的bankIndex作为BTN，否则阻断决策。无抓盲不插入straddle。
3. cmd 492读取Hero两张私牌；cmd 432与随后同座位的cmd 433必须成对出现，后者用`score/betted/betState`累计后手、当街目标投入与fold/check/call/bet/raise/all-in。未知动作或缺失中间行动会使本手停止决策。cmd 403校正底池，cmd 490按3/4/5张公共牌推进flop/turn/river并清零当街投入。
4. cmd 432轮到Hero时，状态机把DPZX牌编号转换为统一牌面格式，按庄位扫描实际参与者并生成`TexasGtoDecisionReq`。同一行动点只允许一个请求在途；临时失败保留重试资格，成功响应缓存回放。
5. 前置uuid激活检查及后续VPIP/PFR策略、GTO调用、响应规范化、按次计费、`gto_decision_record`审计和桃园悬浮窗展示与AAPoker共用同一链路。

## 截图链路

1. `TexasPokerDecisionHandler.Handle` 默认以 `parse_all=false` 调用 `service.RecognizePoker`，先做轻量 Hero 轮次检测。每次识别到 Hero 行动帧后，仅将紧接着的一帧设为 `parse_all=true`；该后继帧完整识别为非 Hero 后恢复轻量检测。
2. Hero 行动帧以及它后面一次完整解析的牌局帧写入 `dal.SaveUserData`；同一 BTN、同一 stage 的快照在覆盖前比较状态，把确定的下注、加注、跟注或弃牌追加到内存动作历史。更后面的非 Hero 帧不做全桌 OCR，也不进入缓存。
3. `dal.GetLastUserData` 从最近一次 BTN 变化之后取出当前手牌的全部阶段。
4. `model.Conv2TexasGtoDecisionReq` 调用 `buildPlayers` 与 `buildActionHistory`；优先输出连续快照累计的动作，首次进入阶段或发生漏帧时按最终下注额生成最小合法历史。翻后 Hero 可行动且桌面无人下注时，Hero 之前仍 active 的玩家被记录为已经 check；Hero 可行动帧到 decided 帧保存行动前基线，Hero 动作及 decided 后新出现的 fold/下注逐帧暂存而不提交，避免只看最后一帧丢掉已完成 call 后消失的下注。后续同街快照在 Hero 自身投入、后手或状态发生可见变化时，从基线按时间顺序重放 Hero 动作及后续事件。单独出现对手投入或一次行动返回 Hero 仍保持暂存，因为 Hero 筹码也可能延迟显露；若连续两次完整 Hero 帧的街道、按钮、Hero 投入/后手/状态和全桌行动状态一致，并且这些 Hero 字段也与 pending 基线一致、相对基线存在可见的对手下注或状态变化，则确认已经完成一个行动圈，把上一圈 Hero 结算为 check 后恢复第二圈决策。Hero 后手相对基线下降说明仍有未解释投入，必须继续 pending。没有对手可见变化时也保持 pending。行动圈边界后才出现的合法 call/raise/fold 归入第二圈，不能倒灌成第一圈 bet。若直接跨街，则重放完整暂存序列，再由阶段完成逻辑补齐剩余的最小合法动作。跨街第一帧中新出现的 fold，若该玩家已经完成上一街行动，则归入新街；仍欠上一街响应的 fold 继续归入上一街。同街再次轮到 Hero 时，`players.action_taken` 只有在 Hero 上次动作已经确认后才从累计历史还原，不能从 HeroTurn 或 SB 起始的单帧筹码顺序猜造。
5. 位置映射从 `ButtonSeat` 开始遍历物理座位，只保留本手实际参与者：`empty` 和 `waiting` 不占位置，弃牌玩家仍占位置。若按钮或后续强制位置对应的物理座位无人，`BTN`、`SB`、`BB`、`straddle` 依次顺延给下一名实际参与者。
6. 转换前先检查最新快照没有未决 Hero 行动；`PendingHeroBefore` 或逐帧 pending 快照仍存在时直接停止本次决策，不能把“players 已变化、history 尚未结算”的混合时点请求发给策略服务。转换后再校验当前行动玩家存在、active 且剩余筹码大于0。客户端策略为默认 `0/0` 时交给原 `service.GtoDecision`；六组白名单 VPIP/PFR 之一则映射为上游 `vvip/pfr` 并走 v2。策略值不进入牌局缓存，也不改变识别、位置和行动历史。
7. 决策返回的 bet/raise 若超过 Hero 可用总额，先保留动作并把目标额收敛为 `hero.bet + hero.stack`（all-in），再校验动作与当前下注是否相容；其余非法结果返回空动作且不计次。
8. 合法决策为 `fold` 且 Hero 当前投入不低于本轮最大投入时，ddbaby 将最终动作规范化为 `check`；面对更高下注时不改写。客户端只消费原有 JSON 响应。
9. GTO返回可解析决策后，无论是否合法均复制实际请求和原始响应，通过独立 goroutine 写入 `gto_decision_record`；合法结果保存all-in/free-check规范化后的最终响应，被拒绝的结果final_response为JSON null且不计费、不下发。写库失败只记录日志，不阻塞或改写客户端响应。最终生成原有 `Result` 时，bet/raise 的目标总额若达到同一请求中的 `hero.bet + hero.stack`，展示为 `All in`；不新增客户端协议字段。

## 修改约束

- `buildPlayers` 与 `buildActionHistory` 必须使用相同的桌容量、空座过滤与位置表。
- 空座影响本手实际参与人数；弃牌不能触发位置重排。
- 缓存切手边界目前依赖 `ButtonSeat` 变化。
- SB、BB、straddle 的初始投入属于强制盲注，不能直接转换为自愿 `call` 或 `raise`。
- `action_history.amount` 保留每次动作当时的目标下注额，不能统一覆盖为阶段最大值。
- 零金额 check 也是已完成动作。跨街补全不得在没有新 bet/raise 的同一行动圈再次补记同一玩家的 check 或 fold；如果该玩家 check 后出现新 bet/raise，仍可在新行动圈补记其 call/fold/raise。
- decided 页面上的动作提示不属于识别输入，不能作为 Hero 已执行该动作的证据。Hero turn 消失只建立待确认状态；在 Hero 动作确认前，后续观察到的 fold/下注也必须与 Hero 动作一起暂存，最终由真实筹码/玩家状态变化或跨街完成事实按顺序解析。
- `parse_all` 是一次性状态：Hero 行动帧把它置为 `true`，紧接着的一张有效非 Hero 完整识别结果保存后将它恢复为 `false`。这样可以补充 Hero 动作后的牌局变化，同时避免后续所有非 Hero 帧持续执行全桌 OCR。3bet/4bet 仍以 Hero 快照的最终下注额和行动顺序推断，并结合这一张后继帧或后续 Hero 快照累计可观察差分。
- `is_hero_turn=true` 但 Hero stack<=0 属于不可用于决策的识别结果；不能继续请求决策服务。
- 决策协议中的 `raise_size` 是目标总下注额。超过 Hero 可用总额时按 all-in 上限收敛；check/fold/call 必须为0；bet只允许在无人下注时。raise 不要求 `players.bet` 中已有下注，因为翻前快照可能不包含强制盲注；目标仍必须高于当前最大下注且不超过 Hero 可用总额，Hero 已匹配 BB/straddle 后的 option raise 也属于合法动作。
- 免费 check 兜底必须在决策响应通过校验后执行，并从同一决策请求的玩家投入计算；不能根据客户端显示文本或截图推断。
- 缺少 `vpip/pfr` 与显式 `0/0` 等价，必须保持旧客户端和原决策接口兼容；非默认值必须成对命中固定白名单，不能只传一个值或透传任意百分比。
- 决策审计记录GTO实际返回的所有可解析决策，包括校验拒绝的非法结果；非法结果final_response保存JSON null，仍不下发或计费。v2 记录必须包含实际线上的 `vvip/pfr`；原始响应与规范化后的最终响应分别保存。异步任务不能复用会随 HTTP 请求结束而取消的 context，也不能因写库失败影响主链路。

- 非法/空GTO决策在event入口终止当前行动点：返回HTTP 200、BaseResp.StatusCode=0、StatusMessage=decision_rejected，不带Result或客户端动作，不计费。可解析的非法原始响应仍异步审计，final_response为JSON null。service缓存非nil但Result为空的终止标记，沿用30分钟TTL和1024条容量；相同行动重传不重复GTO、审计或计费。后续真实行动继续推进状态，可生成下一行动点的决策。网络/上游临时错误仍返回503并释放重试资格；激活检查仍先于缓存。两协议HTTP回归覆盖拒绝、重传、同手后续行动及正常决策。

- DPZX事件cmd433动作仅按betState映射：2=call、3=check、4=raise、5=allin、6=fold、13=bet；不比较betted与全桌最大投入改写动作。AAPoker事件按自己的action码映射：2=call、3=raise、4=allin、5=check、6=fold，不能套DPZX枚举。金额仅维护筹码和目标总额，allin保留当时目标投入，check/fold历史金额为0；未知动作阻断本手决策。截图decision接口保留原快照推断逻辑。
