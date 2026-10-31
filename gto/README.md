# gto-solver

一个独立的决策服务，**原样实现 ddbaby 现有上游的 `POST /gto/decision` 和 `POST /gto/decision_v2` 契约**。ddbaby 只需把 URL 指向它，`/lyxz/gto_decision` 和 `/lyxz/gto_event` 的客户端协议不变。

```
客户端
  → ddbaby /lyxz/gto_decision 或 /lyxz/gto_event
    → service.GtoDecisionByProfile
      → POST {GTO_DECISION_URL}          ← gto-solver
           ├─ L1 翻前表（JSON，启动时加载）
           ├─ L2 solver sidecar（可选，HU 翻后）
           └─ L4 启发式兜底（Monte Carlo equity + 底池赔率）
      → {action, raise_size, action_probability, detail}
```

## 契约

| 项 | 值 |
| --- | --- |
| 方法 | `POST` |
| 路径 | `/gto/decision`、`/gto/decision_v2` |
| 认证 | `X-API-Key`（配置了 key 才校验） |
| 请求体 | 与 `model.TexasGtoDecisionReq` 完全一致 |
| 响应体 | `{action, raise_size, action_probability, detail}` |
| `raise_size` | bet/raise 的**目标总投入**，不是增量 |
| 健康检查 | `GET /healthz` |

服务对每个响应执行最终的合法性收敛：面对下注不能 `check`、无人下注不能 `call`、`bet` 只在无人下注时出现、`raise` 目标必须大于当前最大注且不超过 Hero 的 `bet+stack`。因此即使 L1/L2 返回了越界金额，客户端拿到的仍然是合法动作。

## 快速开始

```bash
# 1) 构建（仓库根目录）
go build -o output/bin/gto-solver ./cmd/gto-solver

# 2) 准备翻前表（见 data/preflop/README.md）
mkdir -p data/preflop

# 3) 启动
./output/bin/gto-solver \
  -addr :9200 \
  -tables ./data/preflop \
  -api-key "$GTO_API_KEY"

# 4) 验证
curl -s http://127.0.0.1:9200/healthz
curl -s -H 'Content-Type: application/json' -H "X-API-Key: $GTO_API_KEY" \
  --data @request.json http://127.0.0.1:9200/gto/decision
```

`request.json` 就是 ddbaby 实际发给上游的请求体，可以用 `gto_decision_record.decision_request` 里的一条记录直接回放。

## 配置

| 参数 | 环境变量 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `-addr` | `GTO_SOLVER_ADDR` | `:9200` | 监听地址 |
| `-tables` | `GTO_PREFLOP_TABLES` | `./data/preflop` | 翻前表目录（`*.json`） |
| `-solver-url` | `GTO_SOLVER_SIDECAR_URL` | 空 | L2 solver base URL；空则禁用 |
| `-solver-timeout-ms` | `GTO_SOLVER_TIMEOUT_MS` | `1500` | L2 超时 |
| `-decision-timeout-ms` | `GTO_DECISION_TIMEOUT_MS` | `4000` | 单次决策总超时 |
| `-mc-samples` | `GTO_MC_SAMPLES` | `2000` | 启发式 Monte Carlo 样本数 |
| `-api-key` | `GTO_SOLVER_API_KEY` / `GTO_API_KEY` | 空 | 为空则不校验 |

## 路由规则

1. `game_stage == preflop`：查 L1 表；命中直接返回。
2. 非翻前、仅 2 人活跃、配置了 `-solver-url`：调用 L2 sidecar；失败/超时降级。
3. 其余（包括所有多人翻后）：L4 启发式。
4. 任何引擎返回非法动作：`finalize` 收敛为合法动作。

L1 查表顺序：

1. 精确动作线：`BTN|UTG:raise:2.5|MP:call:2.5`；
2. 金额四舍五入到 0.5bb 的动作线；
3. 局面 key：`BTN|RFI`、`BTN|VS_OPEN_UTG`、`BTN|VS_OPEN_CALL_UTG`、`BTN|VS_3BET_UTG`、`BTN|VS_LIMP`；
4. 表内 `"*"` 默认项（可选）；
5. 都未命中 → L4。

## L1 表

格式见 [`data/preflop/README.md`](../data/preflop/README.md)。要点：

- 只接受 JSON/SQLite/HTTP JSON 这类**机器可读**数据，不解析图片或 PDF 图表；
- `raise_to` 用 BB 为单位，运行期乘以 `bb_size` 后作为 `raise_size` 返回；
- 表按 `players` 精确匹配，`stack_bb`/`ante_bb`/`straddle_bb` 取最近档位（超出阈值则放弃）；
- `position_aliases` 可把 `LJ`/`HJ` 映射到服务使用的 `MP`/`MP+1`，Hero 的 `-Hero` 后缀会自动去掉。

数据来源（按可商用性）：

- **买**：PokerData 的 `preflop-table/v1` 导出（下载包允许商用），需要写一个一次性转换脚本；
- **自己生成**：用 MIT/Apache 许可的 solver（HU 可用 `cRITlamb/poker_solver`；多人可用 OpenSpiel 或自研 CFR）离线跑表；
- **不要**抓免费图表网站：多数禁止商业利用/抓取，且不是机器可读格式。

## L2 solver sidecar 协议

可选。服务只把“非翻前 + 2 人活跃”的请求转给它：

```
POST {solver-url}/solve
Content-Type: application/json

<TexasGtoDecisionReq>

→ 200 {"action":"raise","raise_size":1250,"action_probability":0.62,"detail":"..."}
```

sidecar 负责 range 构造、下注树和求解；主服务只负责路由、超时和降级。接 `postflop-solver` 时，建议在 sidecar 里完成 range 传播和 combo 查询，不要把 range 逻辑放进 Go 服务。

## 当前边界

- **L1 是查表，不是求解**：表没覆盖的 spot 会落到启发式；覆盖率和表的质量决定 L1 的上限。
- **L4 不是 GTO**：它是 range-blind 的 Monte Carlo equity + 底池赔率，用于兜底和验证，不应对外宣称 GTO。
- **L3 预解库尚未实现**：接口位置预留在路由里，后续可以按 `(stage, N, board cluster, pot type)` 接 SQLite/Redis。
- **多人翻后没有求解**：3 人以上走 L4；多人 CFR 是后续独立工程（见 `reports/solver-research-20261029/REPORT.md`）。
- **mushroom / straddle / ante**：表匹配只认 `ante_bb`/`straddle_bb`；mushroom 是自定义池，当前不参与 EV 计算，接入前需要产品侧先定义语义。
- **截图链路的 ante/straddle 是常量占位**：`model/recognize_poker.go` 里 `ante=bb`、`straddle=2bb`，不是真实牌桌值；接表前必须修，否则会匹配到错误的 ante 档。
- **vvip/pfr 目前只透传**：不参与表选择；若要按策略档位选表，可在表里加 `profile` 字段并扩展 `nearestPreflopTable`，不要直接改启发式阈值。

## 测试

```bash
go test ./gto/... -count=1
```

覆盖：表加载/校验、精确线与 situation 回退、最近筹码档、BB option、启发式合法性、7 张牌 evaluator、equity 单调性、HTTP 契约与鉴权。

`gto/testdata/preflop_8max_100bb.json` 只是**测试夹具**，手写的示例数据，不是 GTO 数据，不要用于生产。
