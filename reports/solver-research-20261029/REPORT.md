# 德州扑克决策 Solver 适配调研

日期：2026-10-29
范围：`/lyxz/gto_decision`、`/lyxz/gto_event` 最终共用的 GTO 决策出口
结论等级：选型调研，未改生产代码

## 结论

1. **没有可以直接替换现有 `/gto/decision` 的现成 solver。** 不是"没找到"，而是主流 solver 的产品前提与当前模式正面冲突：
   - solver 需要**每个玩家明确的 range**，当前请求里没有 range；
   - 生产级 solver 基本都是**单挑（heads-up）翻后**，而当前牌局是 2–9 人、经常 4–8 人进池；
   - 翻牌全树求解是**秒级到分钟级**，当前 HTTP 超时是 5 秒；
   - 当前牌局带 **straddle / ante / mushroom 池**等自定义规则，通用 solver 不建模。
2. **真正可落地的是"组件化分层"，不是一个 solver 整体替换。** 优先级最高的是翻前（占实际决策约 2/3），它可以用离线求解生成的查表在 1ms 内解决；翻后按人数和街道拆分：2 人用 HU 引擎，3 人用多人 CFR，4+ 用合并 range 或预解库。
3. **每层都能接多人，但质量必须分档说清楚**：2 人是 Nash；3 人是 CCE（无 Nash、无可利用度上界）；4+ 是近似。没有"多人 + 实时 + Nash"的现成方案，自建多人 CFR 是唯一能把 3 人做到实时求解的路。
4. **候选排序（按"能否真正接入当前模式"）：**
   - ① `ucsandman/postflop`（Rust，MIT）：在 MIT 许可下唯一可以放心自托管、嵌入商业服务的成熟求解引擎，但只支持 HU 翻后；
   - ② `TexasSolver`（AGPLv3，可向作者购买商业授权）：单挑翻后，转/河速度可用，翻牌分钟级；
   - ③ `MonkerSolver`（€499，桌面离线）：支持任意人数、任意街道，是多人预解库（L3）的唯一现成离线引擎，但没有 API，商用再分发需先确认许可；
   - ④ `pokerai.bet` / `Deepsolver API`：接口形态最接近，但两家都在条款中**明文禁止实时辅助真实牌局**，不能用于本产品线上链路（见 §7）；
   - ⑤ `PokerData`（预解数据包）：不是实时 solver，但**下载包明确允许商用**，8-max MTT 翻前包最接近当前 8 人带 ante 的牌局，是 L1 最现实的现成数据源（见 §4.9）。
5. **"买一个 solver API 接进来"这条路基本被堵死**：不是技术问题，是合同问题——商用 solver 服务普遍禁止 live play 实时辅助。可走的路是：自托管 MIT 引擎、购买 AGPL 商业授权后自托管、买可商用的预解数据包，或完全自研。
6. **最省钱的短期收益不是换 solver，而是修掉当前策略服务的确定性错误**（免费 fold、JJJ88 误判成两对、`EV(call) > EV(raise)` 仍 raise、顶对在多人底池 3 倍加注）。这些不是"不够 GTO"，是逻辑错误；换 solver 也可能因为状态映射错误继续存在。

## 1. 当前模式对 solver 的硬约束

### 1.1 入口与共享出口

两个接口最终都收敛到同一个函数，替换点只有一个：

- `POST /lyxz/gto_decision`：截图识别链路，`Conv2TexasGtoDecisionReq` → `service.GtoDecisionByProfile`
- `POST /lyxz/gto_event`：AAPoker / DPZX 事件链路，`ProcessPokerEvent` → `executeAAPokerDecision` → `service.GtoDecisionByProfile`

`service.GtoDecisionByProfile` 按 `vpip/pfr` 选择 `/gto/decision` 或 `/gto/decision_v2`（`service/texas_poker.go`）。这意味着新引擎只要实现同样的 `func(ctx, *TexasGtoDecisionReq, vpip, pfr) (*TexasGtoDecisionResp, error)`，就能同时覆盖两个入口，改动面很小。

### 1.2 请求契约（`model.TexasGtoDecisionReq`）

| 字段 | 现状 | solver 是否需要 |
| --- | --- | --- |
| `game_type` | 固定 `no_limit_holdem` | 需要 |
| `num_players` | 2–9（位置表含 `straddle`）；**含已弃牌玩家** | 需要；但 solver 必须用 `is_active` 统计在场人数 |
| `current_pot` / `sb_size` / `bb_size` | OCR/事件快照 | 需要 |
| `ante` / `straddle` / `mushroom` | AAPoker/DPZX 来自真实牌桌；截图链路是常量占位（`ante=bb`、`straddle=2bb`） | 需要，但通用 solver 不认 mushroom |
| `community_cards` / `game_stage` | preflop/flop/turn/river | 需要 |
| `players[]` | 位置、Hero 手牌、后手、本街投入、是否活跃、action_taken | 需要 |
| `current_player_position` | 当前行动者 | 需要 |
| `raise_range` | **始终为空** | 需要下注尺度抽象 |
| `action_history[]` | 街道、位置、动作、当时目标额、时间戳 | 需要，用于推断 range |
| `vvip` / `pfr` | 6 组固定档位 | 可作为 range 偏置/节点锁定 |

响应契约是 `action`（fold/check/call/bet/raise）+ `raise_size`（**目标总下注额**，不是增量）+ `action_probability` + `detail`。solver 输出的是混合频率（例如 fold 25% / call 11% / raise 64%），需要一层"频率 → 单动作 + 目标额"的确定性映射。

### 1.3 决策量级与街道分布

2026-10-13 至 10-14 的两个窗口共 1,735 次决策、903 个唯一状态；其中 409 个唯一状态的街道分布为：

| 街道 | 唯一状态 | 占比 | 实时求解可行性 |
| --- | ---: | ---: | --- |
| preflop | 259 | 63% | 不需要实时求解，查表即可 |
| flop | 82 | 20% | 全树求解分钟级，不可实时 |
| turn | 47 | 11% | 2 人秒级；3 人 CFR 1–3s；4+ 只能近似 |
| river | 21 | 5% | 2 人亚秒级；3 人 CFR <1.5s；4+ 只能近似 |

这组数字决定了投入顺序：**先把翻前做对，收益最大、成本最低**；翻后只对单挑转/河值得实时求解。

注意：上表没有拆分"单挑 vs 多人"。L2 的真实覆盖率取决于多人底池占比，建议从 `gto_decision_record` 统计一次（MySQL 8 可用 `JSON_TABLE` 展开 `$.players[*]`，统计 `is_active=true` 的数量，再按 `stage × active_count` 分组）；这个数字会直接决定 P2/P3 的优先级。

### 1.4 solver 视角下缺失的输入

通用 solver 的输入不是"牌局状态"，而是"**状态 + 每个玩家的 range + 下注树**"。当前请求缺三样：

1. **range**：只能从 `action_history` + 翻前基准 range 推断；
2. **下注树**：`raise_range` 为空，需要定义合法的 bet/raise 尺度集合；
3. **rake / 抽水**：多人底池的跟注阈值对此敏感。

另外，`mushroom` 是自定义池，通用 solver 没有对应概念；`ante`/`straddle` 在截图链路是常量占位，直接喂给 solver 会算错底池。接入前需要产品侧先确认：mushroom 池是不是死钱、是否计入跟注赔率、是否可被赢取——确认后才能决定把它加进 `pot` 还是忽略。

## 2. 选型判据

按当前模式，一个方案必须同时满足：

1. **接受完整状态**（位置、后手、本街投入、底池、行动历史），而不是只接受"board + 两条 range"；
2. **覆盖多人**（至少 3 人，最好到 8–9 人）；
3. **翻前 + 翻后**全街道；
4. **单次决策可接受**：2 人 p95 < 1s，3 人 p95 < 3s，硬上限 5s；
5. **可商用、可网络服务**（许可证 + ToS）；
6. **可维护**：能版本化、能回放、能灰度。

按这 6 条，绝大多数 solver 在第 2、4 条就会被淘汰。

## 3. 候选矩阵

> "实时任意状态"指：给一个此前没见过的完整牌局快照，在秒级返回动作。

| 方案 | 形态 | 人数 | 翻前 | 翻后 | 实时任意状态 | 接口 | 许可 / 条款 | 适配度 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `ucsandman/postflop` | 开源 Rust 引擎 | 2 | ✗ | ✓ | 转/河可（turn 0.2s 级；flop 370ms/iter，分钟级） | CLI / JSON / WASM | MIT，可商用嵌入 | ★★★☆ |
| `TexasSolver` | 开源 C++/CUDA | 2 | ✗ | ✓ | flop 172s@6 线程；转河快 | CLI + 策略 JSON | AGPLv3；**网络服务需商业授权** | ★★☆ |
| `TexasSolverGPU` | 商用/个人免费 | 2 | ✗ | ✓ | GPU 快 | GUI（Windows） | 个人免费，商用需谈 | ★★ |
| `MonkerSolver` | 商用桌面 | 任意人数 | ✓ | ✓ | 离线求解，非实时 | GUI，无 API | €499，桌面许可 | ★★★（离线造库） |
| `PokerData`（数据包） | 商用数据 | 6-max / 8-max MTT / PLO 9-max | ✓（预解） | ✓（HU 翻后包） | 非实时求解，下载/查询 | 下载包 / API / MCP | **下载包允许商用（含自建软件内使用）** | ★★★（L1 + HU-L3 数据源） |
| `PioSOLVER` | 商用桌面 | 2 | ✗（Edge 版含 HU 翻前） | ✓ | 离线求解（flop 分钟级、turn/river 秒级） | GUI / CLI / 脚本 | 商业许可 €450 / €800 | ★☆ |
| `GTO+` | 商用桌面 | 2 | ✗ | ✓ | 离线求解 | GUI | $75 一次性 | ★☆ |
| `Deepsolver API` | 云 API | 2 | ✗ | ✓ | 2–8s，异步队列 | HTTP | **明文禁止 live play 实时辅助** | ✗ |
| `pokerai.bet` | 云 API | 6-max | ✓（预解） | ✓ | 预解 ms；实时 solver flop ~70s、转河更快 | HTTP / SDK / MCP | **明文禁止真实牌局实时辅助** | ✗（技术最接近，条款不可用） |
| `GTO Wizard` | 云策略库 | 6-max 为主 | ✓ | ✓ | 无 solver API | 无公开 API | 订阅 | ✗ |
| `PokerSnowie` | 云 AI | 覆盖范围需确认 | ✓ | ✓ | 场景评估 | 私有 API（需洽谈） | 个人使用为主 | ★★ |
| `OpenSpiel` / `ReBeL` / `DeepStack` | 研究框架 | 2 | 部分 | 部分 | 否 | 代码 | 开源 | ✗（研究） |
| 自研 CFR | 自建 | 任意 | ✓ | ✓ | 需大量 GPU + 工程 | 自建 | 自有 | ★☆（成本最高） |

> 评分口径：适配度按"**能否用于线上实时决策链路**"打分，不是按求解质量打分。求解质量最高的 MonkerSolver 因为只能离线、无 API，反而拿不到最高分。

另外排查过、但**不解决"任意状态实时求解"**的名字（避免重复调研）：

- `Slumbot`：单挑限注（HULHE）bot 对局 API（`new_hand`/`act`），不是无上限、也不是 solver；
- `BluffNet` / `OpenPoker` / `DuelAmp` / `Poker Bots All Stars`：AI vs AI 对局/竞技平台，提供的是"打牌接口"，不提供策略查询；
- `GTOBase`（Simple Postflop/Simple Preflop 云算力）：学习工具 + 预解库，无公开 API、偏个人使用；
- `SolvePoker`：Monker 预解库 + 学习/训练，无公开求解 API；
- `carnuare/pokerSolver`：Heroku 时代的演示 API，已不可用；
- `poker-calculations`（npm）：手牌评估/权益/`decideAction`，是启发式库，不是 GTO solver，可以作为 L4 的实现参考。

## 4. 候选逐项说明

### 4.1 `ucsandman/postflop`（最推荐的可嵌入引擎）

- **能力**：Rust 实现，Discounted CFR + 实测可利用度；支持 flop/turn/river 的 HU 翻后求解；输入是 board、OOP/IP range、有效筹码、底池、每条街的 bet/raise 尺度；输出 `solution.json`，可按节点/组合查询。
- **性能**：README 实测 turn 小树 0.2s 收敛到 0.12% pot；flop 830k 节点 370ms/iter@24 线程（按 600 iter 约 3–4 分钟）；river 几十毫秒级。
- **限制**：明确不支持 preflop；明确不做 multiway（作者说明多人 CFR 收敛到 coarse correlated equilibrium，没有可利用度保证）。
- **许可**：MIT，可改、可商用、可网络服务。
- **接入方式**：作为独立 sidecar（Rust binary 或 HTTP 服务）部署，ddbaby 通过本地 HTTP 调用；或把引擎编译成 C ABI 由 Go 调。每次请求带超时和降级。

### 4.2 `TexasSolver` / `TexasSolverGPU`

- **能力**：C++ 求解器，单挑翻后，支持 Texas 和 short deck；有策略 JSON dump；GPU 版速度更快。
- **限制**：同样是 2 人翻后；翻牌全树仍是分钟级；无翻前。
- **许可**：AGPLv3。作者在 README 明确：**通过互联网提供服务需要商业授权**。如果要把 solver 作为在线决策服务，必须单独购买商业许可，否则有法律风险。
- **结论**：适合内部离线跑数据/做基准，不适合直接嵌入在线链路（除非买授权）。

### 4.3 `MonkerSolver`（多人 + 翻前，离线造库）

- **能力**：官方描述"Solve Omaha and Hold'em from any street with any number of players"，是少数真正支持多人、且支持翻前的 solver；有 abstraction 和可定制下注树；桌面 GUI（Windows/macOS，≥8GB RAM）。
- **限制**：CPU 求解，9 人翻前全树可能非常慢；没有 API，只能人工/脚本批量跑；产物需要自己导出成库。
- **价格**：完整版 €499（turn/river 有免费版）。
- **用法建议**：**离线**生成翻前查表和多人翻后策略库，不在线上直接调用。技术上它正好补上翻前这块最大的缺口，但商用/再分发的许可需要先确认（见 §4.8）。

### 4.4 `Deepsolver API`

- **能力**：HTTP API，提交 game tree + ranges + board + 参数，异步排队，单次计算 2–8 秒（GPU），结果可返回 EV/equity/action frequencies/per-node 策略矩阵，支持批量排队 1 万+。
- **限制**：**当前只支持 HU 翻后，明确"Multiway trees are not yet available"**；异步、按 job 计费。
- **条款**：官方博客写明"provided it is not used to offer real-time assistance during live play"——与本产品的实时悬浮窗形态直接冲突。
- **结论**：不可用于线上实时链路；最多作为离线校准/回归的参考。

### 4.5 `pokerai.bet`（接口形态最接近的第三方 API）

- **能力**：NLHE HTTP API，翻前和翻牌走**预解库**（官方称 5.8M+ 预解、66K 同构翻牌，毫秒级），自定义 flop/turn/river 走**实时 solver 池**（官方 OpenAPI 描述 flop ~70s，turn/river 更快）；返回的是各动作的混合频率，不直接给推荐动作；预解的有效筹码固定 100BB；实时求解需要显式传 `oop_range`/`ip_range`。
- **限制**：**6-max**（当前牌局是 2–9 人、含 straddle）；预解只覆盖标准位置/动作线；实时 solver 仍需自己提供 range；turn/river 也是**排队求解 + 轮询**（不是同步亚秒返回），翻牌约 70s，都远超 5s 超时。
- **条款（决定性）**：其 Terms of Service 写明 "**Real-time assistance at real-money poker tables is prohibited. You must not use the Services, directly or indirectly, to advise or automate decisions during live real-money play.**"（生效 2026-10-19）；官方 llms.txt 也自我定位为 "not ... autonomous real-money bot, or live-table coach"。这正好覆盖本产品的悬浮窗实时建议场景。
- **适配判断**：技术形态最接近，但**条款不允许用于线上实时辅助**；只能作为离线对照基准/学习工具，不能作为生产依赖。

### 4.6 `GTO Wizard` / `PokerSnowie`

- **GTO Wizard**：强在预解库和 UI，但公开的 Researcher API 只允许"打牌/看结果"，明确不提供 solver 能力；没有可用的任意状态查询 API。
- **PokerSnowie**：神经网络型 AI，官方提供"Scenario Evaluation / 策略建议"，API 需联系商务（个人使用为主），具体覆盖人数/牌局类型需向官方确认；可作为"不要求严格 GTO、只要合理建议"的候选，但同样有 ToS 和覆盖范围问题。

### 4.7 研究框架（OpenSpiel / ReBeL / DeepStack / PokerRL）

- 适合做算法验证和离线训练，不适合直接接生产：要么只支持有限注/小游戏，要么需要训练网络和 GPU 集群，要么已停止维护。DeepStack 曾做到实时 HU 决策，但依赖旧版 Lua/Torch 和预训练网络，工程上不建议复活。

### 4.8 一个被忽略的前提：翻前策略数据从哪来

所有 HU 翻后 solver 都不做翻前，而翻前占当前决策的约 63%。所以"接一个 solver"之前，必须先回答翻前数据从哪来：

| 来源 | 质量 | 成本 | 商用风险 |
| --- | --- | --- | --- |
| MonkerSolver / HRC 离线生成 | 高（多人 + 翻前） | €499 起 + 大量 CPU 时间；9-max 全树可能极慢 | 许可通常是个人使用，批量生成后商用/再分发需确认 |
| PokerData 预解包（见 §4.9） | 高（自有 solver，card-exact） | NLHE MTT 8-max 全部 $250；6-max 全部 $1,050 | **下载包明确允许商用**；API 商用需 Enterprise 且禁止镜像 |
| 采购现成翻前库（GTOSims / 极策等） | 高 | 按包购买 | 多为个人学习授权，商用/实时使用大概率被禁 |
| 自研翻前 CFR（169 抽象 + 固定翻后抽象） | 中高 | 1–3 人月 | 自有，可把 straddle/ante/mushroom 建成 dead money |
| 启发式 + range/equity 校准公开 chart | 中 | 1–2 周 | 无 |

对 ddbaby 的现实建议：先用最后一种把翻前错误清零，同时评估 PokerData 的 8-max MTT 包（最接近 8 人带 ante 的牌局）；不要把"买了 MonkerSolver"默认当成可商用方案，先读许可。

### 4.9 商用数据源：PokerData（新发现，适合 L1）

- **是什么**：HFT Studio 运营的 GTO 预解数据商店，卖"pack"（预解文件）和 ranges API/MCP。其 MTT 包声称 card-exact（对 MonkerSolver 的 bucketing 做过对比），官网有 methodology 页；6-max 现金包未明确说明是否 card-exact。
- **覆盖**（2026-10 官网）：
  - NLHE 6-max 现金：8 个翻前包（20/30/40/50/70/100/150/200bb）+ 22 个翻后包（SRP/3bet/4bet，100bb，49 个 board，含 turn/river）；
  - **NLHE MTT 8-max：10 个翻前包（10–300bb，BB ante，chip EV），全部 $250**；
  - PLO 6-max / PLO5 / PLO 9-max；NLHE 9-max 现金和 9-max 翻前目前没有。
- **条款（关键）**：下载包 "may be used for any purpose, including commercial use ... in software you build and sell"；API 的 Starter/Pro 仅个人使用，商用要 Enterprise（$250/月），且**禁止把 API 数据镜像/批量落库**（可缓存 1 个月）。
- **适配判断**：
  - 翻前：8-max MTT 包（BB ante）最接近当前 8 人带 ante 的牌局；straddle、mushroom、9 人桌需要自己补规则/插值；
  - 翻后：只有 HU 单挑底池的包，多人翻后仍不覆盖；
  - 购买 pack 后可以在自己的服务里使用，这是目前看到的**唯一"数据可商用 + 无实时辅助禁令"**的现成来源，值得作为 L1 的首选试验。

### 4.10 L1 的免费方案（以及为什么大多不能直接上线）

"免费"要拆成三件事：**数据免费、代码免费、商用免费**。三者同时满足的 L1 方案，目前没有。

| 来源 | 覆盖 | 免费形式 | 能商用吗 | 说明 |
| --- | --- | --- | --- | --- |
| ThinkGTO 免费图表 | MTT 40bb 8-max、live 200bb 9-max、6-max 200bb，**只有 RFI** | 网页 / PDF | ❌ 条款禁止"商业利用、再分发、抓取" | 覆盖 8/9 人，最适合起步验证 |
| PokerData 免费图表 | 6-max cash 100bb，RFI + 3bet/defend 共 20 张 | 网页 / 邮件 PDF | ⚠️ 免费图表条款需单独确认；API 免费档仅个人 | 质量高，但只有 6-max |
| Preflop Wizard / GTO Wizard 免费图 | 6-max / MTT | App / 网页 | ❌ 个人学习 | 只能参考 |
| `cRITlamb/poker_solver` | HU 翻前 + 翻后，DCFR（Python + Rust） | MIT 代码 | ✅ | 自己跑，HU only |
| `hellomate2/gto-poker-overlay` | HU blueprint + 预解翻前图 + 实时搜索 | MIT 代码 | ✅ | 近似，含 PokerNow overlay（那部分不要用） |
| OpenSpiel | 多人通用博弈框架（universal_poker） | Apache 2.0 代码 | ✅ | 需要自己做抽象和工程，1–3 人月 |
| 自研翻前 CFR | 任意人数 | 自有 | ✅ | 许可免费，时间成本最高 |

结论：

- **只做原型/验证**：ThinkGTO 的 8-max/9-max 免费图 + PokerData 的 6-max 免费图足够把 L1 查表和回归跑通，但不能直接上线（授权）。
- **要上线且零授权成本**：HU 用 `cRITlamb/poker_solver`（MIT）自己生成；多人用 OpenSpiel 或自研 CFR 生成。工作量 1–3 人月，但数据和代码都归自己。
- **要最快上线且可商用**：PokerData 8-max MTT 包 $250 仍是最便宜的合规方案——它不免费，但比自研 1–3 人月便宜得多。
- **不要**把免费图表抓下来直接塞进生产：ThinkGTO 明确禁止抓取和商业利用，PokerData 免费图表也有单独的条款，风险不值得省这 $250。

## 5. 推荐方案：分层决策引擎

### 5.1 总体架构

先给结论：**四层都可以做成多人，但"多人"的实现方式和理论保证不同。** 现成引擎里没有"多人 + 实时 + Nash"的组合；要做到每层都接多人，必须接受"3 人以上是近似（CCE / 合并 range），不是 Nash"。

| 层 | 2 人 | 3 人 | 4–6 人 | 7–9 人 | 多人实现方式 | 理论 | 延迟 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| L1 PreflopChart | ✅ | ✅ | ✅ | ✅ | N 人翻前表（人数是表的 key） | ε-Nash（含抽象） | <1ms |
| L2 PostflopSolver | ✅ HU 引擎 | ✅ 3 人 CFR | ⚠️ 合并 range 近似 | ⚠️ 合并 range 近似 | 2 人：HU 求解；3 人：多人 CFR；4+：合并 range | 2 人 Nash；3 人 CCE；4+ 启发式 | 0.2–3s |
| L3 PrecomputedLib | ✅ | ✅ 3 人预解 | ⚠️ 4 人需造库 | ⚠️ 5+ 成本极高 | Monker 离线造库（支持任意人数） | CCE 近似 | <5ms |
| L4 Heuristic | ✅ | ✅ | ✅ | ✅ | equity + 底池赔率 + MDF | 近似 | <10ms |

```text
客户端
  │  截图 / AAPoker / DPZX 事件
  ▼
ddbaby（不变）── 校验、状态重建、合法性/支配保护、审计、计费
  │
  ▼
DecisionRouter（新增，替换 service.GtoDecisionByProfile 内部实现）
  ├─ L1 PreflopChart     翻前查表（任意人数，N 是表的 key）               <1ms
  ├─ L2 PostflopSolver   2 人 HU 求解 / 3 人多人 CFR / 4+ 合并 range      0.2–3s
  ├─ L3 PrecomputedLib   多人预解库（离线造库，按 N/位置/牌面匹配）       <5ms
  └─ L4 Heuristic        任意人数兜底（equity + 赔率 + MDF）              <10ms
```

路由规则（每一层都能接多人，但按"人数 × 街道 × 时间预算"选最优路径）：

- `stage == preflop` → L1（任意人数）；
- `stage in (turn, river)` 且 `活跃玩家 == 2` → L2 的 HU 路径；
- `stage in (turn, river)` 且 `活跃玩家 == 3` → L2 的 3 人 CFR 路径（超预算降级 L3 → L4）；
- `stage == flop` → L3（2/3 人预解库；4+ 直接 L4）；
- `活跃玩家 >= 4`（任意街道）→ L3 命中则用，否则 L4 的合并 range 路径；
- 任何一层超时、未命中或返回非法动作 → 回退 L4，并记录降级原因。

> 注：多人不是"能不能跑 solver"的问题，而是博弈论问题——3 人以上 CFR 收敛到 coarse correlated equilibrium，不是 Nash，没有可利用度上界。MonkerSolver 能在离线给出多人近似策略，但那是"可用的近似"，不是"多人 GTO"。

### 5.1.1 多人怎么解：三条实现路径

要让每一层都接多人，实际上只有三种手段，区别在"实时性 / 质量 / 成本"：

| 路径 | 做法 | 适用 | 质量 | 成本 |
| --- | --- | --- | --- | --- |
| 多人 CFR（CCE） | public-tree vector CFR + 牌力分桶 + 动作抽象；每层一个时间预算 | 2–3 人，转/河 | 近似 CCE，无可利用度上界 | 自研引擎，1–3 人月 |
| 合并 range 近似 | 把 N-1 个对手按 reach 加权合并成一条 range，解 HU，再按人数修正底池赔率/弃牌率 | 4+ 人，或 3 人超时 | 近似，质量随人数下降 | 复用 HU 引擎，1–2 周 |
| 离线预解库 | MonkerSolver（支持任意人数）离线跑常见树，导出查表 | 3–4 人常见 spot | CCE 近似，覆盖有限 | 离线算力 + 存储，按 spot 数量线性增长 |

三个必须一起做的工程前提：

1. **N 条 range**：每个对手一条 range，不能只算 Hero 的；
2. **边池**：3 人以上 all-in 必须处理主池/边池，否则 EV 和跟注赔率都会算错；
3. **时间预算**：多人 CFR 必须可中断，超时返回当前最优动作并标记 `approximate=true`。


### 5.1.2 原始 CFR 能不能直接用？

结论：**CFR 作为算法族是唯一实用的路，但"教科书版原始 CFR"直接上 NLHE 生产不靠谱。** 靠谱的是"CFR 家族 + 抽象 + 向量化 + 子博弈重解"，不是朴素实现。

为什么原始 CFR 不行：

1. **状态空间**：NLHE 完整树是指数级，vanilla CFR 要遍历整棵树、给每个 infoset 存 regret；内存和时间都放不下。
2. **收敛速度**：vanilla CFR 的平均遗憾是 O(1/√T)，要拿到 ε-Nash 大约需要 T ≈ 1/ε²；实时预算内根本迭代不到。CFR+ / DCFR 在实践中快 1–2 个数量级。
3. **抽象不可避免**：不压缩牌力（分桶）和动作（下注尺度），树建不出来；压缩之后就已经不是"原始 CFR"了。抽象质量直接决定策略质量，这一层的工程量往往比 CFR 本身大。
4. **多人没有 Nash**：3 人以上 CFR 收敛到 CCE，原始 CFR 也改变不了；4+ 实时求解不现实。
5. **工程细节全是坑**：card removal、side pot、range 传播、chance 权重、平均策略加权，任何一处错都会出现"能跑但决策不对"，而且很难察觉。

按场景选算法：

| 场景 | 用什么 | 判断 |
| --- | --- | --- |
| 学习/验证（Kuhn、Leduc、river toy） | 原始 CFR | ✅ 很好，适合入门和写单测 |
| HU 翻后生产 | vector DCFR/CFR+（`postflop-solver` 已实现） | ✅ 直接用，不要重写 |
| 3 人 river 生产 | public-tree vector DCFR + 牌力分桶 | ⚠️ 可自研，目标 1–2 周出原型 |
| 3 人 turn 生产 | 同上 + 更强抽象 + 时间预算 | ⚠️ 难，先验证 river 再上 |
| HU flop / 多人 flop | 预解库 / 合并 range | ❌ 实时 CFR 不做 |
| 翻前 N 人 | 离线抽象 CFR（Monker） | ⚠️ 离线可以，线上只查表 |

如果决定自研，最小可行技术栈是：Rust/C++ + 7 张牌 evaluator + public tree + 每个节点存 range 向量 + DCFR + equity 分桶 + raise cap + 可中断时间预算 + rayon 并行；先用 HU river 对 `postflop-solver` 做一致性验证，再做 3 人 river，最后 turn。**不要**一上来写"遍历整棵树的 CFR"，那是必然失败的路径。

### 5.2 L1 翻前查表（投入产出比最高）

- **数据形态**：`(人数, 位置, 前面动作序列, 有效筹码档, ante/straddle 档)` → 169 手牌 → `{fold, call, raise_to}` 频率。
- **生成方式**：见 §4.8/§4.9。最现实的起点是买 PokerData 的 8-max MTT 翻前包（可商用、$250），再补 straddle/9-max 规则；质量上限最高的是 MonkerSolver 离线跑，但许可和 9-max 全树耗时都要先确认；自研 CFR 只作为长期选项。
- **线上**：纯内存 map + 最近邻档位插值，延迟可忽略；`vpip/pfr` 档位用于对 range 做偏置（例如松凶档位放宽 call/raise 阈值），而不是改求解结果。
- **收益**：覆盖约 63% 的唯一状态、70% 左右的决策次数，且翻前错误是用户最容易感知的（例如 99 翻前 fold、免费 check 却 fold）。

### 5.3 L2 转/河实时求解（按人数分三条路径）

- **2 人**：直接用 `postflop-solver`/HU 引擎；river 通常 <0.5s，turn 0.2–2s，硬超时 1.5s + 降级。
- **3 人**：自研多人 CFR（public-tree vector CFR + 牌力分桶 + 动作抽象），只对转/河启用；river 建议预算 0.5–1.5s，turn 1.5–3s；超时返回当前平均策略并标记 `approximate=true`。翻牌不做（树太大）。
- **4 人及以上**：不跑多人 CFR，改走"合并 range"：把 N-1 个对手按 reach 概率加权合并成一条 range，按 HU 求解，再按人数修正底池赔率（required equity 用 N 人底池算）、弃牌率和隐含赔率；本质是近似，质量随人数下降。
- **输入构造**：从 L1 的翻前 range 出发，用 `action_history` 逐街收窄（range propagation），得到**每个对手一条 range**；底池/有效筹码用 `players[].bet + stack` 重算，不直接信 OCR 的 `current_pot`。
- **下注树**：用固定尺度集合（例如 turn 50/75/100%，river 50/75/100% + all-in），由 `raise_range` 承载；多人时每增加一个对手，树的分支会指数增长，必须限制 raise cap（例如每条街最多 1 次加注）。
- **输出映射**：solver 给频率 → 按频率采样或取最大 EV 动作 → 把"目标总投入"换算成 `raise_size`（raise = call_amount + (pot + call_amount) × sizing）。
- **边池**：3 人以上 all-in 必须实现主池/边池分配，否则 EV 和跟注赔率算错。
- **注意**：2 人路径是 Nash；3 人是 CCE；4+ 是启发式近似。响应里要能区分（`detail` 或新字段），不要统一标成 "GTO"。

### 5.4 L3 多人预解库（离线造库）

- **2 人**：现成的 PokerData HU 翻后包可以直接用；也可以自己用 `postflop-solver` 批量跑。
- **3 人**：市面没有现成包，必须用支持多人的 solver（MonkerSolver，任意人数）离线造：`(N=3, 位置组合, pot_type(SRP/3BET/4BET/LIMP), 牌面簇, 筹码档, 动作线)` → 每个节点的策略矩阵；线上按 key 最近邻匹配，未命中走 L2/L4。
- **4 人**：只在少数高频配置造库（例如 4 人 limp pot + 常见牌面簇），成本是"人数 × 位置组合 × 牌面 × 筹码档"的组合爆炸；不建议全量。
- **5+ 人**：不建议造库，直接走 L4 的近似（或用合并 range）。
- **成本量级**：3 人预解库按 100 个牌面簇 × 10 个位置组合 × 4 个筹码档 ≈ 4,000 个 solve；Monker 是 CPU 求解，按单 solve 分钟到小时级估算，需要数天到数周的离线算力，存储也会到 GB–TB 级。先做 1–2 个 pot_type 验证命中率和收益，再扩。
- **不要**承诺多人 GTO：多人 CFR 收敛到 CCE 而非 Nash，没有可利用度上界，这是理论事实，不是实现问题。

### 5.5 Range 引擎（多人版，关键前置，别跳过）

多人接入后，range 引擎不再是"两条 range"，而是**每个对手一条 range + 边池**：

1. **翻前基准**：位置 + 人数 → 开池/跟注/3bet range（来自 L1 表）；人数越多，开池和跟注范围越紧。
2. **逐街收窄**：`action_history` 的每个动作按"该动作对应的 range 子集"过滤；每个对手独立维护，不能合并成一条（除非走 4+ 的合并 range 近似，此时才显式合并并记录）。
3. **reach 概率**：每个对手到达当前节点的概率不同，合并 range 时要按 reach 加权，不能简单平均。
4. **边池**：3 人以上 all-in 时按投入额拆分主池/边池，计算每个对手的可赢得金额；这是多人 EV 正确性的前提。
5. **对手建模**：`vpip/pfr` 六档 → 对每个对手的基准 range 做偏移（松档加宽、紧档收窄），并记录偏移量。
6. **不确定性**：range 推断是近似的，输出层应保留 `action_probability` 和"置信度/降级原因"，方便审计和 A/B。

没有 range 引擎，solver 只能收到"几条随便填的 range"，结果不会比现在的启发式更好。

### 5.6 输出稳定性与可观测性

- solver 是混合策略，同一状态多次请求可能给不同动作；线上应**按状态缓存**（同 `state_sha256` 返回同一结果），避免用户看到抖动；
- 在 `gto_decision_record` 增加 `strategy_version` / `engine` / `fallback_reason`（或写进响应 `detail`），否则无法定位"策略变了但不知道哪版"的问题（当前报告已经遇到"响应没有策略版本"的困境）；
- 保留 `raw_response` + `final_response` 的现有审计方式。

## 6. 落地路线

| 阶段 | 内容 | 工作量 | 验收 |
| --- | --- | --- | --- |
| P0 | 抽出 `DecisionProvider` 接口 + 配置化上游 URL + 记录 `strategy_version` | 1–2 天 | 现有行为不变，双入口回归通过 |
| P1 | **L1 多人翻前表（2–9 人）+ L4 多人启发式** | 1–2 周 | 所有人数、所有翻前状态都有输出；free-fold / 99 fold 类错误为 0 |
| P2 | **L2 2 人 HU**（河牌→转牌，postflop-solver sidecar + range 传播） | 1–2 周 | HU 河牌 p95 < 1s；超时降级正常 |
| P3 | **边池 + N 条 range**（多人正确性的前提） | 1–2 周 | 3 人 all-in 分池、每人 range 独立收窄通过单测 |
| P4 | **L3 3 人预解库**（Monker 离线，先 1–2 个 pot_type） | 2–4 周 | 3 人常见 spot 命中率、延迟、与 L4 A/B |
| P5 | **L2 3 人 CFR**（先 river，再 turn，带时间预算） | 3–6 周 | 3 人 river p95 < 1.5s；`approximate=true` 标记 |
| P6 | **4+ 合并 range 近似**（复用 HU 引擎 + 人数修正） | 2–4 周 | 4+ 不再走纯启发式，动作合理性回归 |
| P7 | 5+ 预解库扩展（可选，成本高） | 持续 | 命中率、收益 > 成本 |

建议先用现有 `reports/` 里的 bad case 做**回归集**（免费 fold、JJJ88、Q8、K9、K5、99），每阶段跑一遍；没有回归集就换引擎，无法证明变好。

## 7. 合规与风险

这里有三层约束，经常被混为一谈：

1. **服务条款（合同）**：Deepsolver 写明 "not used to offer real-time assistance during live play"；pokerai.bet 写明 "Real-time assistance at real-money poker tables is prohibited... You must not use the Services, directly or indirectly, to advise or automate decisions during live real-money play."。MonkerSolver / PioSOLVER / GTO+ 是桌面学习工具，EULA 也普遍限定个人学习用途。**结论：目前看到的实时 solver 服务都不允许这种用法。** PokerData 这类"预解数据包"的条款只约束 IP（下载包允许商用、API 商用需 Enterprise），没有用途禁令，但它不提供实时求解；是否用于实时辅助仍需自行判断。
2. **版权许可（著作权）**：`postflop-solver` 是 MIT，可自托管、可商用、可网络服务；`TexasSolver` 是 AGPLv3，**通过网络提供服务需要商业授权**；用商用 solver 生成的策略数据再分发，也要看其 EULA 是否允许。
3. **牌局平台规则**：即使全部自建，用第三方牌桌数据驱动实时建议仍受牌局平台规则约束。这不属于本仓库的技术范围，但是产品层面的实质风险。

其余风险：

- **数据与隐私**：请求里含 Hero 手牌、牌桌状态、uuid；接入第三方 API 等于把牌局数据发出去。
- **多人理论**：不要对外宣称"多人 GTO"；多人只能做到近似、可利用度未知。
- **自定义规则**：mushroom 池、straddle、ante 需要显式映射；截图链路的 `ante=bb`、`straddle=2bb` 是常量占位，接 solver 前必须先修。

## 8. 建议的下一步

1. **P0 + P1：让所有人数先有可用输出。** 翻前多人表和 L4 多人启发式一起上——这两层天然支持 2–9 人，成本最低。翻前数据优先评估 **PokerData 8-max MTT 包（$250）**（目前唯一看到的"下载包可商用 + 覆盖 8 人带 ante"的现成数据），不够的用启发式补。
2. **P3：边池 + N 条 range。** 这是多人正确性的前提，先于任何多人 solver；没有它，3 人以上的 EV 和跟注赔率都是错的。
3. **3 人先走 L3 预解库，不急着自研 CFR。** 用 MonkerSolver 离线跑 1–2 个高频 pot_type（例如 3 人 limp pot、3 人 SRP），验证命中率和用户收益；L3 的成本远低于 L2 的多人 CFR。
4. **3 人实时求解（L2 多人 CFR）作为第二步。** 先 river（树小、可验证），再 turn；用固定 raise cap + 牌力分桶控制规模，硬超时 1.5–3s，输出标记 `approximate=true`。
5. **4+ 人先用合并 range 近似**（复用 HU 引擎 + 人数修正），不要一上来就追求 4 人 Nash——不存在，也跑不动。
6. **HU 河牌 PoC 仍然值得做**：用 `postflop-solver` 对 5–10 个真实河牌状态跑一遍，测延迟和结果合理性；它是验证 range 引擎和输出映射是否正确的低成本方式。
7. **把 `pokerai.bet` / `Deepsolver` 只当离线对照基准**：不要接入线上链路——两家的条款都明确禁止实时辅助。
8. **不要**直接把 TexasSolver 嵌进在线服务（AGPL），除非向作者购买商业授权；`postflop-solver` 是 MIT，是当前唯一可以干净自托管的成熟引擎。

## 附 A：两种代码接入方式（都保持 `/gto/decision` 契约）

现有链路只有一个决策出口（`service/texas_poker.go` 的 `GtoDecisionByProfile`）。无论用哪种方式，最终都必须返回同一个 `model.TexasGtoDecisionResp`，客户端协议不变。

### 方式一：进程内 Provider（Go）

```go
// service/decision_provider.go
type DecisionProvider interface {
    Name() string
    Decide(ctx context.Context, req *model.TexasGtoDecisionReq) (*model.TexasGtoDecisionResp, error)
}

type DecisionRouter struct {
    preflop  DecisionProvider // L1：查表
    postflop DecisionProvider // L2：solver sidecar
    library  DecisionProvider // L3：预解库
    fallback DecisionProvider // L4 / legacy upstream
}
```

- 配置：`GTO_DECISION_ENGINE=legacy|hybrid`、`GTO_SOLVER_URL`、`GTO_SOLVER_TIMEOUT_MS`；
- 观测：`gto_decision_record` 增加 `engine` / `strategy_version` / `fallback_reason`；
- 现有 `NormalizeTexasGtoDecisionToStack`、`ValidateTexasGtoDecisionResponse`、`NormalizeTexasGtoFreeFoldToCheck` 保留在 handler。

### 方式二：独立 `gto-solver` 服务（推荐，真正 drop-in）

新服务直接实现现有上游的两个契约：

- `POST /gto/decision`
- `POST /gto/decision_v2`（`vvip`/`pfr`）
- 请求体就是 `model.TexasGtoDecisionReq`，响应体就是 `{action, raise_size, action_probability, detail}`；
- 认证继续用 `X-API-Key`；
- ddbaby 只改一件事：把 `gtoDecisionURL` 从常量改成配置，指向新服务；**`/lyxz/gto_decision` 和 `/lyxz/gto_event` 完全不动**。

服务内部再按 L1→L2→L3→L4 路由。语言不限（Python/Rust/Go 都行），solver 可以用 Rust sidecar；同机部署时多一跳 HTTP 通常 <1ms，换来独立发布/回滚/灰度的能力。

### L1 表的数据契约（code-first，不是图）

L1 在代码里就是一张表，加载进内存或 SQLite，运行期零手工步骤：

```json
{
  "format": "preflop-table/v1",
  "players": 8,
  "stack_bb": 100,
  "ante_bb": 1,
  "straddle_bb": 2,
  "positions": ["UTG", "UTG+1", "LJ", "HJ", "CO", "BTN", "SB", "BB"],
  "nodes": {
    "BTN|":                      {"AA": {"fold": 0, "call": 0, "raise_to": 2.5}},
    "BTN|UTG_raise_2.5|":        {"AA": {"fold": 0, "call": 0.2, "raise_to": 8.0}},
    "BTN|UTG_raise_2.5|CO_call|":{"AA": {"fold": 0, "call": 0.1, "raise_to": 9.0}}
  }
}
```

- `action_line` 用规范化字符串（位置 + 动作 + 金额，金额统一用 bb）；
- 查表 key：`(人数, 筹码档, ante/straddle 档, 位置, action_line, 169 手牌)`；
- 未命中：最近档位插值 → 仍不命中则降级 L4；
- 混合策略：按频率采样或取最大 EV；`raise_to` 乘 `bb_size` 后作为 `raise_size`（目标总额）返回；
- 数据来源只接受 **JSON/CSV/SQLite/HTTP JSON**；图片/PDF 图表不是运行时输入（无法可靠解析，也没有商用授权）。

## 附 B：请求字段 → solver 输入映射

| ddbaby 字段 | solver 侧 | 说明 |
| --- | --- | --- |
| `community_cards`（`A-H`） | `board`（`Ah`） | 需要格式转换 |
| `current_pot` | `starting_pot` | 建议用 `players[].bet` 重算，不完全信 OCR |
| Hero / 对手 `bet + stack` | `effective_stack` | HU 取 min，多人需先合并 |
| `players[].bet` | to-call / pot 计算 | 决定 check/call/raise 合法性 |
| `action_history` | range 传播（不是 solver 入参） | 用于把翻前 range 收窄到当前街 |
| Hero `hole_cards` | 求解后用 `show --combo` 查 | **solver 先解 range，再查具体组合** |
| `raise_range` | `sizings` | 目前为空，需要新建尺度集合 |
| `vpip/pfr` | range 偏置 / 节点锁定 | 不是 solver 标准入参 |

关键认知：solver 不直接吃"Hero 两张牌"，它先解两条 range，再查 Hero 组合对应的策略。所以 **range 引擎不是可选项**。

## 附 C：第一个可验证实验（河牌，1–2 天）

1. 从 `gto_decision_record` 取一个 `stage=river` 且只有两人的请求；
2. 用公开 chart 或简化规则手工填 OOP/IP range；
3. 按 `board / oop_range / ip_range / effective_stack / starting_pot / sizings` 写 TOML；
4. `solver solve --config spot.toml --out solution.json`，再 `solver show --line ... --combo ...`；
5. 记录：求解耗时、exploitability、Hero 动作频率、与现有响应的差异；
6. 判定：p95 < 1s 且结果合理 → 值得做 P2；否则只保留翻前查表 + 启发式。

## 附 D：3 人多人 CFR 引擎最小设计（如果决定自研）

1. **公共树**：按街道和下注序列建 public tree；每个节点存每个玩家的 range（对 1326 组合的权重向量），而不是单手牌。
2. **动作抽象**：fold / check / call + 2 个 raise size；每条街 raise cap = 1；all-in 单独分支。
3. **牌力分桶**：每街按 equity rollout 分 50–200 桶，桶内共享 regret，控制状态数。
4. **算法**：DCFR（α=1.5、β=0、γ=2）或 CFR+，vector 形式；3 人可用 public chance sampling 降方差。
5. **终局**：摊牌 + 边池分配；fold 终局按剩余玩家分池；这是多人 EV 正确性的核心。
6. **时间预算**：每 N 次迭代检查 elapsed，超时立即返回当前平均策略，并标记 `approximate=true`。
7. **验证**：先用 3 人 Kuhn / 简化 toy 验证收敛；再用真实 river spot 与 Monker 离线结果对比动作频率，而不是只看 EV。
8. **性能目标**：3 人 river 1,000–5,000 次迭代 < 1.5s；turn < 3s；flop 不做（树太大）。

## 附：参考

- `ucsandman/postflop`（MIT，HU 翻后，turn/river 性能）：https://github.com/ucsandman/postflop
- `bupticybee/TexasSolver`（AGPLv3，网络服务需商业授权）：https://github.com/bupticybee/TexasSolver
- `TexasSolverGPU`（Windows，个人免费）：https://bupticybee.github.io/texassolver_gpu_page
- `MonkerSolver`（任意人数、任意街道，€499）：https://www.monkerware.com/solver.html
- `Deepsolver API`（HU 翻后、2–8s、禁止实时辅助）：https://deepsolver.com/api
- Deepsolver API 说明（multiway 不可用、条款原文）：https://content-blog.deepsolver.com/build-and-scale-nlh-solving-with-the-deepsolver-api/
- `pokerai.bet`（6-max，预解 + 实时 solver）：https://pokerai.bet/ ，OpenAPI：https://pokerai.bet/openapi.yaml ，条款：https://pokerai.bet/terms
- `PokerData`（预解数据包，下载包可商用）：https://pokerdata.io/api ，条款：https://pokerdata.io/terms ，方法说明：https://pokerdata.io/methodology
- `GTOSims`（Simple Preflop/Postflop 预解包，个人学习向）：https://gtosims.com
- 极策GTO（中文翻前策略字典，含 9-max ChipEV）：https://www.geecegto.com
- GTO Wizard Researcher API（不提供 solver 能力）：https://github.com/gtowizard-ai/researcher-api-client
- PokerSnowie（API 需洽谈）：https://www.pokersnowie.com/pricing
- GTO+（HU 翻后、$75）：https://pokerjudge.com/blog/poker-products/gto-plus/
- 多人底池通常不被 solver 支持（行业说明）：https://riverodds.app/poker-solver/
