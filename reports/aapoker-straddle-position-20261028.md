# AAPoker按本手抓盲标记分配位置

## 变更

旧实现4–9人桌总是插入straddle，普通桌因此丢失一个普通位置，且只取straddlePop最大值不能发现抓盲位置不合法。现在依据本手明确标记选择普通/抓盲位置表，验证后才生成GTO请求；不修改截图decision、DPZX、动作枚举、计费和审计。

AAPoker事件位置由本手参与者、bankerID和straddlePop确定：没有正数抓盲标记时不插入straddle；单个标记必须属于本手且为BB之后的下一参与者，其金额高于BB。按BTN开始跨过空座排序，单挑BTN兼SB、另一人为BB。位置写入玩家并供行动历史复用，弃牌不重排；下一手重新解析，不沿用前手抓盲。多级、非连续、负金额、不属于本手或盲位上的抓盲标记，以及缺失本手BTN/不支持的人数均阻断决策，不能取最大金额猜位置。isCurStraddle数值含义尚未确认，不把它当布尔开关覆盖straddlePop。前注仍沿用已观察样本bigChip口径，本轮未确认独立字段，保留此限制。

## 证据与范围

真实日志deploy-67520a7-20261026-182014.log中的START_INFOR带straddlePop和isCurStraddle。例如bigChip=4、straddlePop单座位8、anteCountArray=[4,6,8,12,4,0,4,4]。straddlePop明确指出座位，不能因已有位置表始终带抓盲就忽略零/缺失标记。

本次只修已明确的无抓盲和单级抓盲位置。AAPoker多级抓盲协议、非标准按钮/盲位、独立ante字段和isCurStraddle各个数值的语义未独立确认；不根据DPZX协议或下注差额创造映射。无标记按普通桌构造，不擅自猜有抓盲。

## 验证

专项覆盖8人普通/抓盲、空座与绕桌、Hero抓盲、单挑、2–9人普通桌、4–9人抓盲、跨手撤销抓盲、fold后玩家和history位置一致，以及多标记、跳位、盲位、非参与者、负数和非合法抓盲额阻断。

本地Go1.20.14全仓测试、service/handler/model/main race及diff check均通过。真实上游、SQL与Android仍未进行端到端验证；单测仅证明本地位置和请求构造。

## 2026-10-28 AAPoker按本手抓盲分配位置部署

- 业务`507cb7d`，CR审查`6f73e01..507cb7d`明确LGTM。仅修AAPoker普通/单级抓盲位置与明确异常标记阻断，不改变前注、DPZX、截图decision或动作映射。
- 本地Go1.20.14全仓及service/handler/model/main race通过；服务器Go1.22.2离线全仓、相同race和构建通过，测试输出由SSH返回、未另存专用测试日志。源码`/home/ubuntu/ddbaby-releases/507cb7d-20261028`，归档SHA256`d2afc63e45d028d18e83f39d886c9a9af61c58e25c2ab3fb056b4f403866afd2`。
- 20:52:53 CST切换`718330 → 875285`，PPID=1，沿用原环境、启动参数和工作目录。构建及运行二进制SHA256均为`2bdb58910a1f2b67b98af633b595aba0080843c8500982e32c6a4ef028adb715`，47,642,481字节；JSON pong通过。
- 合成smoke：两协议无uuid采集200、未知/缺失来源400、非JSON+query400、未激活uuid503；拒绝marker不落完整日志，当前日志未见panic。证据`/home/ubuntu/ddbaby-507cb7d-smoke.json`；合成请求不调用GTO或计费，不代表真实AAPoker上游/SQL/Android已验通。
- 日志`/home/ubuntu/ddbaby/output/deploy-507cb7d-20261028-205253.log`。备份`/home/ubuntu/ddbaby-backups/20261028-205253-before-507cb7d`保留旧e7b22d3二进制、私有环境和启动参数；`sudo python3 <该目录>/rollback.py`核对预期PID875285、路径与SHA后恢复并检查JSON pong。目录700、环境600、脚本700。
- 重启清空内存牌局，需完整新手重建。前注独立字段、isCurStraddle、多级抓盲语义及真实上游/真机链路仍未确认；不将本地回归或合成smoke表述为真实端到端验证。
