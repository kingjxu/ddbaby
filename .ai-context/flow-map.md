---
version: "1.0"
last_updated: "2026-10-02"
---

# 德州扑克决策数据流

1. `TexasPokerDecisionHandler.Handle` 接收图片并调用 `service.RecognizePoker`。
2. 识别结果写入 `dal.SaveUserData`；同一 BTN、同一 stage 的重复识别覆盖最后一帧。
3. `dal.GetLastUserData` 从最近一次 BTN 变化之后取出当前手牌的全部阶段。
4. `model.Conv2TexasGtoDecisionReq` 调用 `buildPlayers` 与 `buildActionHistory`。
5. 位置映射遍历牌桌物理座位，只跳过 `empty` 和 `waiting`；弃牌玩家继续占据原位置。
6. 转换后的请求交给 `service.GtoDecision`，响应再映射为中文行动。

## 修改约束

- `buildPlayers` 与 `buildActionHistory` 必须使用相同的桌容量、空座过滤与位置表。
- 空座影响本手实际参与人数；弃牌不能触发位置重排。
- 缓存切手边界目前依赖 `ButtonSeat` 变化。
