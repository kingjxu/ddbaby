# 翻前表目录（L1）

`gto-solver` 启动时加载本目录下所有 `*.json`，每个文件是一张 `preflop-table/v1` 表。目录为空或不存在时，服务只跑启发式兜底。

> 只接受机器可读数据（JSON/SQLite/HTTP JSON）。**不解析图片、PDF 或网页图表**：它们既不可靠，也通常没有商用授权。

## 文件格式

```json
{
  "format": "preflop-table/v1",
  "players": 8,
  "stack_bb": 100,
  "ante_bb": 1,
  "straddle_bb": 2,
  "positions": ["UTG", "UTG+1", "MP", "MP+1", "CO", "BTN", "SB", "BB"],
  "position_aliases": {"LJ": "MP", "HJ": "MP+1"},
  "source": "PokerData NLHE MTT 8-max 100bb, exported 2026-10",
  "nodes": {
    "BTN|": {
      "AA": {"raise": 1.0, "raise_to": 2.5},
      "KQs": {"raise": 0.8, "call": 0.2, "raise_to": 2.5},
      "72o": {"fold": 1.0}
    },
    "BTN|UTG:raise:2.5": {
      "AA": {"raise": 1.0, "raise_to": 8.0},
      "72o": {"fold": 1.0}
    },
    "BTN|VS_OPEN_CALL_UTG": {
      "AA": {"raise": 1.0, "raise_to": 10.0},
      "72o": {"fold": 1.0}
    }
  }
}
```

### 字段

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `format` | 否 | 固定 `preflop-table/v1` |
| `players` | 是 | 牌桌人数（2–9），与请求的 `num_players` 精确匹配 |
| `stack_bb` | 是 | 有效筹码，单位 BB |
| `ante_bb` | 否 | 前注，单位 BB |
| `straddle_bb` | 否 | straddle，单位 BB |
| `positions` | 否 | 位置名，建议使用服务统一命名 |
| `position_aliases` | 否 | 例如 `{"LJ":"MP"}`；Hero 的 `-Hero` 后缀会自动去掉 |
| `nodes` | 是 | `"HERO|ACTION_LINE"` → 169 手牌 → 混合频率 |
| `source` | 否 | 数据来源，便于审计 |

### 位置命名

服务内部使用 ddbaby 的位置名：`UTG`、`UTG+1`、`MP`、`MP+1`、`CO`、`BTN`、`SB`、`BB`、`straddle`。

客户端的位置可能带 `-Hero` 后缀（如 `BTN-Hero`），服务会自动去掉。数据里如果使用 `LJ`/`HJ`/`UTG+2`，请通过 `position_aliases` 显式映射，不要依赖隐式猜测。

### 动作线（`ACTION_LINE`）

```text
HERO|POS:action[:amountBB]|POS:action[:amountBB]|...
```

- `HERO` 是当前行动者位置；
- 只包含**翻前**动作，按时间顺序；
- `amount` 用 BB，`raise`/`bet`/`call` 带金额，`fold`/`check` 不带；
- 金额默认按 0.1bb 归一化；查表失败时会再用 0.5bb 归一化重试；
- 没有前置动作就是 `HERO|`（RFI）。

示例：

| 局面 | key |
| --- | --- |
| BTN 无人入池 | `BTN|` |
| BTN 面对 UTG 加注到 2.5bb | `BTN|UTG:raise:2.5` |
| BTN 面对 UTG 加注、MP 跟注 | `BTN|UTG:raise:2.5|MP:call:2.5` |
| BB option（无人加注） | `BB|` |

### 局面 key（situation）

精确动作线未命中时，服务会尝试局面 key；适合只导出“RFI / 对 open / 对 3bet”几类节点的数据包：

| 局面 | key |
| --- | --- |
| 无人入池 | `BTN|RFI` |
| 有人 limp | `BTN|VS_LIMP` |
| 面对一个 open | `BTN|VS_OPEN_UTG` |
| open 后被跟注 | `BTN|VS_OPEN_CALL_UTG` |
| 面对 3bet | `BTN|VS_3BET_UTG` |
| 面对 4bet+ | `BTN|VS_4BET_UTG` |

匹配顺序：精确动作线 → 0.5bb 动作线 → 局面 key → 表内 `"*"` 默认项 → 启发式。

### 频率与金额

```json
"AA": {"raise": 1.0, "raise_to": 2.5}
```

- `fold`/`call`/`check`/`raise`/`bet` 是频率，建议和为 1；
- `raise_to`/`bet_to` 是**目标总投入**，单位 BB；
- 如果只写 `raise_to` 没写 `raise`，服务会把剩余概率质量当作 raise 频率；
- 运行期 `raise_size = round(raise_to * bb_size)`，再由服务收敛到合法区间；
- 频率只用于选动作（取最大频率），不会每次随机，保证同一状态返回稳定结果。

### 表的选择

- `players` 必须精确匹配；
- `stack_bb`/`ante_bb`/`straddle_bb` 在阈值内取最近档位（筹码 ±40bb、ante ±3bb、straddle ±5bb），超出则放弃并走启发式；
- 同一目录可以放多张表（不同人数/筹码/ante），服务启动时全部加载。

## 数据从哪来

| 来源 | 可用性 | 授权 |
| --- | --- | --- |
| PokerData 预解包 / JSON API | 需要一次性转换成上面的格式 | 下载包允许商用；API 商用需 Enterprise |
| 自研 solver（MIT/Apache 引擎）离线生成 | 直接导出 JSON | 代码许可允许 |
| 免费图表网站 | 不可用：图片/PDF 无法可靠解析 | 多数禁止商业利用/抓取 |

## 校验

```bash
go test ./gto/... -count=1
```

`gto/testdata/preflop_8max_100bb.json` 是测试夹具，手写示例，不是 GTO 数据。
