# DPZX 位置映射修复

生产日志21:46:42的room28823985/HandNum506：bankIndex=1但seat1未参与；431明确seat2 startState=1为SB、seat3 startState=2为BB、Hero为seat5。原实现套用AAPoker带straddle位置表并从下一参与者当BTN，错误生成seat2 BTN、seat3 SB、seat4 BB和seat5 straddle-Hero。

DPZX改为独立映射：用bankIndex锚定BTN、startState=1/2锚定SB/BB，保留空庄位/空小盲缺位，不把强制位置平移；其余位置从BB后顺时针分配。该样本现在为SB seat2、BB seat3、UTG seat4、UTG+1-Hero seat5、MP seat6、MP+1 seat7、CO seat0。正常无抓盲桌也不再插入straddle位置。没有修改AAPoker或截图协议的位置构造。

不按投入金额推测强制位置。重复/缺失BB、强制位置顺序矛盾、未知startState或IsStraddle=true均阻断本手决策；抓盲仍等待更多协议数据。金额、后手、牌面及行动连续性规则保持不变。

验证覆盖生产空庄位样本、正常8人桌、空小盲、单挑；构造与实际盲注不相符的金额，证明依赖标识而非金额。既有真实Hero输入及HTTP响应测试更新为正确位置，断言请求和历史不含虚假straddle。全仓及main/handler/service race结果见本轮验证日志。

仅修复映射；没有将GTO非法check转换为call，也没有改动503重试机制。位置错误可能影响策略，但不能单凭日志断言是上游错误check的唯一原因。尚未部署或重新发送真实策略请求。

用户“所有GTO决策都写DB”的审计改动作为后续独立变更继续处理，避免与本次优先位置修复混在一起。
