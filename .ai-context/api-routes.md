---
version: "1.20"
last_updated: "2026-10-27"
---

# API 路由

## `POST /lyxz/gto_event`

接口由`idl/ddbaby.thrift`中的`TexasPokerEvent`、`TexasPokerEventReq/Resp`维护，路由位于生成的`biz/router/ddbaby/ddbaby.go`，不在`router.go`重复注册。请求公共包络使用生成类型；动态字段`pid/uid/PacketId/data/seq/dataExtra`采用`PokerEventJSON`（IDL binary typedef），其HTTP JSON语义是原始JSON token，由`biz/model/ddbaby/poker_event_json.go`编解码，不是base64或转义字符串。生成路由通过`_texaspokereventMw`挂载`PrepareTexasPokerEvent`中间件，请求体由Hertz统一缓冲并限制为4 MiB，移除event专属512 KiB检查和手工流式读取，只接受`application/json`（允许charset参数）的单个JSON对象；绑定时屏蔽query并在绑定后恢复原URI，禁止查询参数补值或覆盖body字段。来源只读取顶层精确`packageName`键，拒绝任何大小写变体（包括与精确键并存的情况），路由使用该精确键的字符串值；入口函数再通过`c.BindAndValidate`解析生成请求，再由单参数构造的Handler检查packageName路由；固定字段类型不符返回固定400，不再best-effort忽略解码错误。响应使用生成类型，基础成功/错误只返回BaseResp；有Result时包含原有全部决策字段，HeroCard/CommunityCard保持数组或null。截图decision的旧IDL响应保持独立，不能复用于event。

接收桃园助手从 PokerEYE 分析模块透传的一条牌局事件。JSON请求体不超过4 MiB（与其他HTTP接口统一，超限由Hertz返回413），按IDL类型绑定，并携带已登记的字符串`packageName`；缺失、非字符串或未知包名返回固定400且不记录原始body。包名与路由类型集中维护在`service/poker_event.go`的同一张映射表中，新增应用必须显式选择已有状态机或“仅完整日志”路由，不能默认落入AAPoker。`timestamp/vpip/pfr`按整数、`cmd/direction/location/uuid`按字符串绑定，类型不符返回400；不额外校验枚举或业务字段内容。动态字段`pid/uid/PacketId/data/seq/dataExtra`仍保留原始JSON，包括大整数及变化中的DPZX结构，data不设480 KiB子上限。

白名单内事件通过校验后都会记录完整JSON；携带非空uuid的事件先复用decision的数据库激活码检查，检查失败不记录完整body、不更新状态、不回放缓存。AAPoker和DPZX由`service.ProcessPokerEvent`进入service内独立的协议状态机，不能把DPZX数字cmd映射到AAPoker的`REQ_GAME_*`分支；两者在生成统一`TexasGtoDecisionReq`后由handler直接调用共享决策函数复用GTO、计费、审计和结果回放链路。

当前Android 17770入口没有本机调用方认证；任意同机App可伪造事件，再由桃园补入当前登录用户的uuid和Bearer token，从而触发GTO、次数计费和审计。用户已明确接受该风险以简化本轮真机验证；不得将该入口表述为可信来源或已防护，后续需单独补齐。

服务端按 `uuid + AAPoker uid + dataExtra(牌桌)` 隔离有界、30分钟过期的状态，只有服务端下行事件参与重建。登录账号必须已经坐下并能由 `uid/playerIDArray/mySeatID` 一致定位 Hero；旁观者不触发决策。服务从 `START_INFOR` 开始收集本手参与者、位置、盲注/ante/straddle、筹码与手牌，并用 `RECV_ACTION/RECV_CARDS/SHOW_SIDE_POTS` 累计逐街行动、公共牌和底池。每条携带非空uuid的事件在最前面的check阶段、完整日志/状态机/缓存之前，按请求uuid查询数据库激活码，复用decision的有效期和次数额度规则；失败返回HTTP 503与active_code_unavailable，恢复后可重试。当 `operationID` 指向 Hero、手牌和公共牌完整且请求通过原 GTO 请求校验时，复用此次已校验的激活信息并调用对应 VPIP/PFR 策略。event不再读取Bearer token或调用VerifyTaoyuanIdentity；激活有效只能证明该uuid有额度，不能证明调用方拥有该uuid，伪造已激活uuid可冒用额度和污染审计。每个 `handNum + operationRound + stage` 在进程内只允许一个并发决策；激活、GTO或审计编排失败不消费重试资格并返回HTTP 503，Android必须保留当前事件重试。成功后以`uuid + AAPoker uid + table + hand/turn`为键缓存最终完整响应，命中在重新通过激活检查后只回放，不重复GTO、计费或审计。缓存由互斥锁保护，最多1024条、30分钟过期。合法结果沿用 `Result`，并返回 `DecisionAction/DecisionAmount/HeroMaxTarget` 供悬浮窗显示保护。

DPZX按`uuid + DPZX uid`维护最多512个、30分钟过期的当前会话，决策结果按`uuid + uid + roomID + HandNum + cmd432时间戳 + stage`隔离，最多1024条并保留30分钟。cmd 430/431建立本手庄位、Hero座位、参与者、盲注和前注；cmd 492提供Hero私牌；cmd 432/433成对记录行动提示及动作，cmd 403校正底池，cmd 490推进公共牌。缺少动作、未知betState、旁观座位、非法牌号或不完整牌面都会使该手停止决策。cmd 432指向Hero且统一请求校验通过后，执行与AAPoker相同的前置激活校验、策略、计费、审计和结果响应。DPZX位置独立于AAPoker抓盲位置表：431的startState=1/2分别锚定SB/BB；多人桌BTN取SB逆时针的上一名本手参与者，单挑BTN兼SB，不把已离座的bankIndex保留为“空庄位”。SB缺失时仅在bankIndex仍为本手参与者且盲位顺序一致时使用该庄位，否则停止决策。其余位置按BB之后顺时针分配，不插入straddle。重复/缺失BB、强制位置顺序矛盾、未知startState或IsStraddle=true使本手停止决策，等待协议确认，不从下注金额猜抓盲。DPZX当前实现按`value%13`从2至A及花色区间`红桃/方块/梅花/黑桃`转换。注意旧牌号转换存在已确认bug：用户第486手截图明确为6♥4♦，日志原牌31/3却被当前代码转成7♣5♥。余数5→6、余数3→4及花色组0→D、组2→H已获截图佐证；黑色花色仍待证据，不能把当前整张映射表当成已确认协议事实。

GTO返回可解析决策即使用与截图接口相同的 `gto_decision_record`（包括非法结果）：保存桃园 `uuid`、Hero 手牌/位置/街道、实际 GTO 请求、原始响应和规范化最终响应；被合法性校验拒绝时final_response保存JSON null，无客户端动作，不计费。按次数激活码只在合法结果后计费。AAPoker 的游戏 uid 只用于同一上报流里的 Hero/座位匹配，不能替代桃园 uuid。`operationRound`出现缺口说明行动事件丢失，本手立即停止决策并等待下一次`START_INFOR`重建，不能用不完整路线猜测。未携带 uuid 的旧客户端事件仍可被接收和记录，但不建立决策状态、不调用 GTO、不计费。用户明确接受严格前置检查的边界：最后一次额度成功计费后若HTTP响应丢失，原事件重传仍因额度耗尽返回503，已缓存的付费结果也不能回放；不设回放例外。

协议分析阶段会为每一条通过总body、IDL绑定、packageName路由和非空uuid激活检查的事件记录完整JSON，不再由环境开关、截止时间、日志字节预算或请求频率降级或拒绝。请求量和完整日志写入量没有频率上限，完整日志会持续包含牌局内容和用户标识，可能快速占满日志管道或磁盘；必须限制日志访问和监控容量，并在协议分析结束后恢复为受控的短期诊断日志。

## `POST /lyxz/gto_decision`

客户端提交一张屏幕截图，服务端完成授权校验、牌桌识别、状态缓存和 GTO 决策。JSON 请求使用 `images`、`timestamp`、`uuid`、可选 `image_type`，并支持可选整数 `vpip`、`pfr`；老客户端缺少两个策略字段时按 `0/0` 处理。响应继续使用 `Result`、`HeroCard`、`CommunityCard`。

`vpip/pfr=0/0` 保持原 `/gto/decision` 上游链路。非默认策略只接受客户端设置中列出的六组固定组合，随后调用上游 `/gto/decision_v2`；上游 v2 的 VPIP 字段按其契约拼写为 `vvip`，ddbaby 将客户端 `vpip` 映射为上游 `vvip`。任意其他组合在授权和识别前拒绝，不允许外部请求构造任意策略参数。

### 可选时延追踪头

- `X-GTO-Trace-ID`：客户端生成的非身份 trace id，只允许 1–64 位字母、数字、`-_.:`。
- `X-GTO-Client-Stages`：当前帧在客户端的单调时钟阶段值，供服务端计算成帧、最新帧队列、JPEG 和发起 HTTP 前耗时。
- `X-GTO-Previous-Trace`：上一条已完成悬浮窗展示的追踪数据，随下一次正常决策请求回传，不创建并发遥测请求。
- 响应 `X-GTO-Trace-ID`：回显被服务端接受的 trace id。
- 响应 `X-GTO-Server-Timing`：服务端 bind、校验/授权、识别、请求准备、GTO 决策和总耗时。

所有追踪头都是可选的。没有追踪头的旧客户端保持原行为，也不会产生逐请求追踪日志。服务端对外部头做长度、字符、字段数、非负值、单调性和最大 60 秒窗口校验；无效追踪数据不进入统计，也不改变业务决策。

客户端完整展示追踪由下一次请求携带，因此最后一次结果在停止录制且不再发请求时可能只保留在本机 `TRACE_COMPLETE` 日志中。服务端阶段使用自己的时钟，客户端阶段使用 `elapsedRealtime`，不依赖设备与服务器墙上时钟同步。

1.0.27 客户端在 `X-GTO-Client-Stages` 和 `X-GTO-Previous-Trace` 的 v2 格式中附加随机 `decision_cycle_id`、Hero 控件首次检测时间和 cycle 起始/首次展示标记。服务端对 cycle id、字段数量、时间顺序、布尔值及 60 秒窗口做同等校验；同一 cycle 仅首次实际展示写入 `client_complete`，后续重复结果写入 `client_complete_repeat`。没有 cycle 的 v1 头继续按原格式解析。

### 免费 check 兜底

决策服务返回合法 `fold` 后，ddbaby 会比较 Hero 当前投入与本轮所有玩家的最大投入。Hero 无需补注时，服务端将最终动作规范化为 `check`，并通过原有 JSON `Result` 返回“过牌”；面对更高下注时仍返回“弃牌”。客户端无需解析额外响应头或实现动作兜底。

### All in 展示兜底

决策服务返回合法 `bet` 或 `raise` 后，ddbaby 使用同一份已校验请求计算 Hero 的全下目标总额 `hero.bet + hero.stack`。当决策目标总额达到该上限时，原有 JSON `Result` 直接返回 `All in`；未达到时继续返回原有“下注 N”或“加注 N”。响应协议不增加字段，客户端无需新增判断。

## 其他路由

本上下文文件目前只覆盖德州扑克决策热路径；其他路由以 Hertz 生成的 router 和 handler 源码为准。
