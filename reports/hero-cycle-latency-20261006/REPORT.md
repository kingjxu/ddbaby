# Hero 决策机会首次结果时延追踪

## 变更

- `X-GTO-Client-Stages` 和 `X-GTO-Previous-Trace` 新增 v2 解析，接收随机 `decision_cycle_id`、Hero 控件首次检测时间以及 cycle 起始/首次展示标记。
- 同一 cycle 的首次实际展示保留为 `client_complete`；后续重复 Hero 展示记录为 `client_complete_repeat`，避免把重复帧当成独立用户等待。
- v1 头继续按原协议解析。无效 v2 cycle id、布尔值、字段数量、时间顺序或超过 60 秒窗口的输入只告警，不改变业务响应。

## 验证

- 服务器 Go 1.22.2：`go test ./biz/handler -count=1`、`go test ./... -count=1`、`go test -race ./biz/handler ./model ./dal -count=1` 和 `bash build.sh` 通过。
- 单测覆盖合法 v1/v2、cycle id 白名单、Hero 检测时间顺序、0/1 标记、缺字段以及服务端 timing 上限。
- `git diff --check` 通过。

## 口径

主指标为同一 `decision_cycle_id` 的 `hero_turn_detected -> first_display`。重复 Hero、decided 和其他玩家阶段不计入该指标。cycle id 是随机关联标识，不含用户身份；服务端仍不信任客户端传入的身份字段。

## 部署

待代码复审通过后补充二进制备份、原子替换、健康检查和协议 smoke 记录。
