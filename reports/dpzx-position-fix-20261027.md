# DPZX 位置映射修复

生产日志21:46:42的room28823985/HandNum506：bankIndex=1但seat1未参与；431明确seat2 startState=1为SB、seat3 startState=2为BB、Hero为seat5。原实现套用AAPoker带straddle位置表并从下一参与者当BTN，错误生成seat2 BTN、seat3 SB、seat4 BB和seat5 straddle-Hero。

初版ebeba00把bankIndex已离座理解为空庄位，用户明确纠正BTN后才是SB，本轮取消此假设。DPZX独立映射用startState=1/2锚定SB/BB，多人桌BTN取SB逆时针上一名实际参与者，单挑BTN兼SB；SB缺失时只有仍参与且顺序一致的bankIndex可作BTN，否则停止决策。请求玩家排序也从恢复后的真实BTN开始。该样本现在为BTN seat0、SB seat2、BB seat3、UTG seat4、UTG+1-Hero seat5、MP seat6、CO seat7。正常无抓盲桌也不再插入straddle位置。没有修改AAPoker或截图协议的位置构造。

不按投入金额推测强制位置。重复/缺失BB、强制位置顺序矛盾、未知startState或IsStraddle=true均阻断本手决策；抓盲仍等待更多协议数据。金额、后手、牌面及行动连续性规则保持不变。

验证覆盖生产bankIndex已离座样本、正常8人桌、SB缺失、单挑；构造与实际盲注不相符的金额，证明依赖标识而非金额。既有真实Hero输入及HTTP响应测试更新为正确位置，断言请求和历史不含虚假straddle。全仓及main/handler/service race结果见本轮验证日志。

仅修复映射；没有将GTO非法check转换为call，也没有改动503重试机制。位置错误可能影响策略，但不能单凭日志断言是上游错误check的唯一原因。尚未部署或重新发送真实策略请求。

用户“所有GTO决策都写DB”的审计改动作为后续独立变更继续处理，避免与本次优先位置修复混在一起。

用户提供21:48 Hero K♦8♦、Hero为BB且“nb的神”为BTN的界面对照。已收到座位记录对应Hero seat5、“nb的神”seat3；加入BTN3/SB4/BB5构造回归，不宣称已取得该手原始报文。服务器21:48的29条事件全是21:46:42原cmd432重传，GTO里仍为8♣7♦，并未收到K♦8♦。按现有转换表检索的19/24组合未命中；但用户又指出21:45 Hero为64，而日志21:45:29原牌31/3被当前代码转为7♣5♥，点数表本身待界面对照，不能据这次牌号检索单独断言K♦8♦不存在。21:48只收到旧cmd432重传这一事实不依赖牌号转换。

牌号转换的后续核对与本次BTN纠正分开处理；已询问用户21:45这手是否为6♣4♥，未确认前不擅自改整副牌编码。

用户提供截图om_x100b613f115154a0cf957d3635f4196，图中第486手、底池550、Hero后手4975与日志对应：BTN为www0312/seat0、SB seat1投入50、BB seat2投入100；8人普通位置表下Hero seat5为MP。截图明确手牌6♥4♦，对应原牌31/3，证明旧点数和花色映射错误；不等同于另一手Hero为BB的K♦8♦。目前仅确认余数5→6、余数3→4、花色组0→D、组2→H，尚待黑色花色编码证据，不在本次位置补丁中猜全副牌映射。
