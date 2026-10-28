---
version: "1.9"
last_updated: "2026-10-28"
---

# 德州扑克术语

- **物理座位 / seat**：识别结果中的固定桌面座位号，如 `HeroInfo.Seat`、`VillainsInfo[].Seat`。
- **BTN / ButtonSeat**：本手庄家按钮所在的物理座位，也是位置映射的起点。
- **位置 / position**：从 `ButtonSeat` 沿物理座位顺序扫描实际参与者后得到的 `BTN`、`SB`、`BB`、`straddle`、`UTG` 等牌局位置；按钮物理座位无人时，`BTN` 顺延给下一名实际参与者，后续位置同样连续顺延。
- **empty / waiting**：当前没有参与本手的座位，不进入 `players` 和 `action_history`。
- **fold**：已经参与本手但后来弃牌的玩家；仍保留其原位置，只将其标记为非活跃并记录弃牌行动。
- **stage**：`preflop`、`flop`、`turn`、`river`。
- **观察动作 / observed action**：由同一手牌、同一阶段连续快照中的状态或累计下注变化确定的动作；优先于单张阶段快照推断。
- **最小合法历史**：缺少中间快照时，根据强制盲注、行动顺序和最终累计下注构造的最少动作序列，可能少算实际发生的加注轮次。
- **VPIP/PFR 策略档位**：客户端选择的翻前总体频率组合；`0/0` 表示默认策略并保持旧决策链路。当前非默认值只允许产品下拉框中的固定组合，不代表 ddbaby 自行计算的玩家统计。
- **vvip**：上游 v2 接口为 VPIP 定义的字段拼写。ddbaby 对外请求仍接收 `vpip`，只在调用 v2 时映射为 `vvip`。
- **DPZX Hero**：由DPZX房间快照`mySeat`指向的实际入座玩家；旁观值（当前样本为9）不属于任何参与座位，不能触发决策。昵称只用于人工核对，服务端不会硬编码`raise就allin`或据昵称识别身份。
- **DPZX行动点**：数字cmd 432给出的下一行动座位。同一手中只有房间快照、盲注快照、Hero两张私牌和此前动作连续完整，并且cmd 432指向Hero时，才生成GTO请求。

- **DPZX位置**：以startState=1/2锚定SB/BB；多人桌BTN为SB之前的实际参与者，单挑BTN兼SB。SB缺失时须有仍参与且顺序一致的bankIndex，否则停止决策；不能直接把已离座bankIndex当成空庄位。其余参与者从BB后顺时针分配普通位置。与截图/AAPoker的位置顺延规则不同，无抓盲不插入straddle；不按下注金额判断抓盲。2026-10-28用户确认DPZX startState=3/4分别表示straddle/straddle+1，以431标记为准，即使房间IsStraddle=false也有效；强制投入取标记玩家的Value，GTO的Straddle取最后一档强制投入。标记必须从BB后连续按3、4排列，不允许重复、缺档、与BTN/SB重叠或非递增投入。

- DPZX事件cmd433动作仅按betState映射：2=call、3=check、4=raise、5=allin、6=fold、13=bet；不比较betted与全桌最大投入改写动作。AAPoker事件按自己的action码映射：2=call、3=raise、4=allin、5=check、6=fold，不能套DPZX枚举。金额仅维护筹码和目标总额，allin保留当时目标投入，check/fold历史金额为0；未知动作阻断本手决策。截图decision接口保留原快照推断逻辑。
