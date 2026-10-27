---
version: "1.7"
last_updated: "2026-10-27"
---

# 测试规则

- 项目声明 Go 1.20；本地验证使用 `GOTOOLCHAIN=go1.20.14 go test ./... -count=1`。
- 扑克事件入口必须覆盖合法事件、非法JSON、缺少data、非法cmd/direction/location、超长身份字段以及packageName白名单；还必须在同一时间窗口连续提交超过原全局上限的合法payload，证明入口不按全局或来源频率拒绝，并且每条均记录完整JSON。路由回归必须证明AAPoker仍可进入决策链路，DPZX只完整记录且不创建AAPoker状态、不调用GTO/计费/审计，未知或缺失packageName在完整日志前返回固定400且响应不回显body。测试通过只证明服务端接收边界，不代表注入模块或Android跨进程链路已真机打通。
- AAPoker牌局状态回归必须从`ENTER_ROOM → START_INFOR → RECV_ACTION`回放到Hero行动点，核对Hero身份、按钮位位置、手牌编号、ante/盲注/straddle、底池、目标下注额和行动顺序；旁观者、缺牌、非法公共牌不得触发。相同`operationRound`重传不能重复累计，中间round缺失必须阻断本手后续决策；临时决策失败必须返回503并保留同一行动点的重试资格，成功响应丢失后重传必须回放相同完整结果且不能再次调用决策；并发同一行动点只能一个请求取得执行资格。结果缓存还需验证30分钟TTL、1024条容量上限和并发保护；`RECV_CARDS`与上一动作共享round时允许首次推进，但相同发牌事件只能处理一次。
- AAPoker事件调用GTO前必须校验Bearer token有效且认证账号email与桃园uuid一致，再使用uuid校验激活状态；认证失败不得回退信任请求字段，游戏uid也不得当作桃园身份。成功决策必须复用`gto_decision_record`记录实际请求、原始响应和规范化响应，且只有合法决策可以计次。
- 缓存与座位转换的快速回归使用 `go test -race ./dal ./model -count=1`。
- `biz/handler/texas_poker_decision_test.go` 依赖 MySQL、识别服务、决策服务和本地图片，属于集成测试；仅在 `DDBABY_INTEGRATION_TEST=1` 时运行。
- 座位相关改动必须覆盖：8/9 人桌、单/多空座、waiting、fold、按钮位为空、强制位置连续顺延，以及 `buildPlayers`、`actionSeats` 与 `buildActionHistory` 的位置一致性。
- `action_history` 改动必须覆盖连续 Hero 决策之间的完整行动圈：多名对手依次行动、3bet/4bet、等额 call、短码 all-in、fold、同街第二圈以及跨街补全。Hero check 后由后位玩家首次 bet 时，必须按顺时针座位补齐中间仍 active 且投入未变化玩家的 check；跨街第一帧才出现的 fold 要依据该玩家是否仍欠上一街响应决定归属街道。跨入 flop 时，如果翻前无人加注超过最大强制盲注，还必须记录 BB 或 straddle 最后的 option check。校验目标下注额和顺时针行动顺序，不得用 stack/pot 差额补造图上未识别出的下注。
- 翻后全员过牌的跨街补全必须覆盖 Hero 位于首位、中间和末位：Hero 行动帧应先补齐其之前 active 玩家已经发生的 check；decided 帧只标记 Hero 动作待确认，跨街完成后再补齐 Hero check，且每人只出现一次。若 check 后发生新 bet/raise，先前 check 不能阻止该玩家在新行动圈继续 call/fold/raise。
- `players.action_taken` 改动必须同时覆盖首圈兼容和同街第二圈：decided 帧不能仅因 Hero turn 消失就记录 check；Hero 动作未确认期间观察到的 fold/下注必须逐帧暂存，不能只保留最后一帧，否则已完成 call 后消失的中间下注会丢失。单独出现对手投入或一次行动返回 Hero 仍不能证明 Hero 已 check，因为 Hero 的筹码变化也可能延迟显露；只有连续两个完整 Hero 帧保持相同街道、按钮、Hero 投入/后手/状态和全桌行动状态，这些 Hero 字段也与 pending 基线一致，并且相对基线存在可见对手动作时，才可把上一圈结算为 check。Hero 后手相对基线下降、没有对手变化或两帧不稳定时必须继续 pending。行动圈边界后才出现的 Hero call/raise/fold 必须留在第二圈，不能覆盖所有 pending 快照并倒灌成第一圈 bet。若直接跨街，则由完整暂存序列和阶段完成事实生成一套最小合法动作；pending 跨街只能结算一次，新手切换必须丢弃 pending。下一次回到 Hero 时必须以确认后的 Hero 动作为边界，从累计历史还原对手的 bet/call/raise/fold，不能把等额筹码中座位靠前的 caller 误判为 bettor。Hero 面对未跟注下注时不得因 turn 标记消失而补造 check。
- pending 未解析时必须同时阻断请求构建和 handler 策略调用；不能仅依靠 JSON 请求校验，因为内部 pending 字段不会序列化。Hero 自身动作显露、跨街完成、新手切换，或满足“连续两个稳定 Hero 帧且存在对手可见动作”的同街回圈证据后，后续合法 Hero 帧才可恢复决策请求。
- 手牌边界不能只依赖 BTN 移动。连续完整识别出现 BTN 变化、街道回退，或两张完整 Hero 手牌变化时，必须清空上一手缓存和 `action_history`；Hero 手牌顺序交换不算新手牌。调用 GTO 前还要拒绝未来街道、街道回退或时间戳不递增的历史，避免跨手牌污染进入决策服务。
- 调用 GTO 前必须校验当前 Hero 恰有两张合法且互不重复的明牌，并按街校验公共牌数量（preflop 0、flop 3、turn 4、river 5）；Hero 与公共牌之间也不得重复。牌背、漏牌、非法牌或重复牌必须在 ddbaby 拒绝，不能传给决策服务。
- 决策响应校验必须覆盖翻前强制盲注未进入 `players.bet` 的 open raise：即使所有玩家当前投入为0，合法的正数 raise 目标也必须放行；0目标、未超过当前最大投入或超过 Hero 可用总额仍必须拒绝。
- `Result` 的 All in 展示必须按目标总额比较：阈值为 `hero.bet + hero.stack`。测试必须覆盖 Hero 已有投入时低于/等于阈值，以及当前投入为0时低于/等于阈值；不能把剩余后手直接当作目标总额阈值。
