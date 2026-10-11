# GTO 策略档位路由验证记录

## 变更

- 对外决策请求新增可选整数 `vpip`、`pfr`。缺失字段与显式 `0/0` 等价，继续调用原 `/gto/decision`。
- 非默认策略仅接受客户端提供的六组固定组合，并将客户端 `vpip` 映射为上游 v2 契约字段 `vvip`，调用 `/gto/decision_v2`。
- 不支持的单值或组合在识别和上游调用前拒绝，策略值不进入牌局缓存。
- 上游鉴权改为从 `GTO_API_KEY` 环境变量读取，源码不再保存密钥。

## 候选验证

- 业务提交：`bf82deb`。
- 生产同版本 Go 1.22.2 隔离目录 `/home/ubuntu/ddbaby-gto-v2-candidate-bf82deb`：
  - `go test ./... -count=1` 通过；
  - `go test -race ./biz/handler ./model ./dal -count=1` 通过；
  - `bash build.sh` 通过；
  - 候选二进制 `output/bin/hertz_service` SHA256 为 `bf45af779e9805f2383e23147d6715b34b51dc633ab162233d80ecc2b8aecebd`。
- 2026-10-11 18:28（+08:00）使用完整合法牌局请求做真实上游联调：
  - 非默认白名单档位 `vpip=42,pfr=20` 经候选 `GtoDecisionByProfile` 映射为 `vvip=42,pfr=20`，`POST /gto/decision_v2` 返回 HTTP 200，响应可解析且动作非空；
  - 默认 `0/0` 经同一候选继续调用 `POST /gto/decision`，返回 HTTP 200，响应可解析且动作非空。

## 发布前置与边界

- 当前只完成候选构建和隔离联调，没有替换生产二进制、重启进程或改变线上流量。
- 部署前必须把 `GTO_API_KEY` 以进程环境配置到生产启动链路，并在重启前核对环境变量名称存在；任何验证日志都不得回显值。
- 部署和客户端官网发布须在用户明确批准后执行。发布后还需回验默认档位、至少一个非默认档位、进程健康、运行时 SHA 和公网 APK。
