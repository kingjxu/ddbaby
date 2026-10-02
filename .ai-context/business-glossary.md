---
version: "1.0"
last_updated: "2026-10-02"
---

# 德州扑克术语

- **物理座位 / seat**：识别结果中的固定桌面座位号，如 `HeroInfo.Seat`、`VillainsInfo[].Seat`。
- **BTN / ButtonSeat**：本手庄家按钮所在的物理座位，也是位置映射的起点。
- **位置 / position**：从 BTN 沿物理座位顺序、跳过开局前空座后得到的 `BTN`、`SB`、`BB`、`straddle`、`UTG` 等牌局位置。
- **empty / waiting**：当前没有参与本手的座位，不进入 `players` 和 `action_history`。
- **fold**：已经参与本手但后来弃牌的玩家；仍保留其原位置，只将其标记为非活跃并记录弃牌行动。
- **stage**：`preflop`、`flop`、`turn`、`river`。
