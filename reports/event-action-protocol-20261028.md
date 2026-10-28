# 事件动作协议修复（2026-10-28）

## 问题与修复

旧DPZX把4/13按金额变成bet/call/raise，把5全下也变成这三种动作；AAPoker把3按是否有下注改成bet/raise，把4全下按金额改成call/bet/raise，导致GTO无法看到真实动作。现在两个event状态机各自按协议动作码直接映射，金额仅更新筹码和目标投入。保留allin金额，玩家action_taken与action_history一致。未知AAPoker动作和DPZX一致阻断本手，禁止基于金额补造。截图decision接口不改。

| 协议 | call | check | raise | allin | fold | bet |
|---|---|---|---|---|---|---|
| DPZX betState | 2 | 3 | 4 | 5 | 6 | 13 |
| AAPoker action | 2 | 5 | 3 | 4 | 6 | 无独立已接入码 |

AAPoker action=3即使该街无人下注也保留raise，不为了匹配截图推断再改为bet。未知码继续阻断，不臆造其他协议动作。

## 原包与日志证据

DPZX原包SHA256：af9faa66314b9a1fb791e28a5fbc3a8dc79eeb4dd6502022509b99a44b9520bc。
包内assets/cache/res.dpzx.com/88f91ff1为英文资源，SHA256=8b64db9be8a00cfc0d2c07fa4886ed70d5042592703e04369836772049f0906c，明确包含betState2=Call、3=Check、4=Raise、5=All In、6=Fold、13=Betting。中文资源d6580eb0（SHA256=27abbefc94192690907f6c276748638794e2445a812df1cedd9a5a1479856536）对应跟分、过牌、加分、AllIn、弃牌、下分。仅静态读取ZIP资源，未执行APK。

09:31:13 CO的原始betState=5、score=0、betted=12780，旧GTO请求误为raise；09:31:28 Hero的5、score=0、betted=6423因小于12780误为call。09:29三家5且score=0分别为4429/4925/4975，均发生于Hero已弃牌之后，不应补入之前的GTO请求。

AAPoker沿用已接入协议码及原有action=4清空后手的明确语义，去掉金额二次分类；本次APK注入连接器只证实转发原始action，未从游戏原生枚举独立证明全部数字。需保留这一证据边界，不能把DPZX资源当成AAPoker枚举证据。

## 验证与边界

Go 1.20.14 全仓 `go test ./... -count=1`、`go test -race ./service ./biz/handler ./model . -count=1` 和 diff check 均通过。专项覆盖两协议动作与不同已有投入组合、allin目标金额及未知码阻断；现有事件去重、跨街、非法决策终止重试和decision回归继续执行。
真实上游对allin动作标签、SQL落库及Android展示仍需真实牌局验证。当前修复只改输入动作，不改GTO输出合法性、计费和审计；数据库故障下异步审计必达限制沿用用户已接受口径。
