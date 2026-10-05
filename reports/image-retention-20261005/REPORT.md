# 生产截图保留策略

## 目标

定期删除 `/usr/local/webserver/images` 中不用于牌局证据的普通 JPEG 截图，持续保留
`*-hero.jpg` 和 `*-decided.jpg`。普通截图保留最近 5 天，便于故障回溯。

## 实现

- `scripts/cleanup_images.py` 仅遍历目标目录的直接子文件；跳过子目录、符号链接、
  非 JPEG、Hero 和 decided 文件。
- 默认只删除修改时间早于 7,200 分钟（5 天）的普通 `*.jpg`，支持 `--dry-run` 先统计。
- 使用 `/run/lock/ddbaby-image-cleanup.lock` 防止定时任务重入；单个文件删除失败时记录
  错误并以非零状态退出。
- ddbaby 的两条生产图片落盘入口与清理脚本共同使用跨进程排他锁
  `/run/lock/ddbaby-images.lock`。清理先扫描候选文件，再在持锁期间完成最终身份检查和
  删除；写入也在同一把锁内完成，因此检查通过后至 `unlink` 之间不能插入同名写入。
- 真正删除前仍会再次读取文件元数据；设备号、inode、大小或纳秒 mtime 任一变化，
  或文件已不再是过期普通文件时，跳过删除并计入 `changed_files`。
- cron 每天 03:17 执行，日志按周轮转，保留 4 份压缩归档。

## 首次上线前只读统计（当时采用 6 小时阈值）

- JPEG 总数：13,685。
- Hero：765；decided：1,690。
- 早于 6 小时的普通 JPEG：11,230，共 1,842,131,538 bytes。
- 根盘：40 GB，总使用约 30 GB，剩余 7.8 GB，使用率 80%。

## 验证与边界

- Python 单元测试 7/7 通过，覆盖普通过期图删除、近期普通图保留、Hero/decided
  保留、非 JPEG 保留、符号链接跳过、dry-run 不删除和非法保留时长拒绝；
  `py_compile` 通过。
- 竞态回归覆盖扫描后同名文件被替换、原文件 mtime 被更新，以及写入方在最终检查后、
  `unlink` 前尝试写入。最后一种场景验证写入方会等待清理释放共享锁，再写入并保留
  新文件。
- Go 回归验证图片写入方在共享锁被占用时不会落盘，释放后才能完成写入。
- 服务器 Go 1.22.2 隔离目录验证通过：`go test ./service -count=1`、
  `go test ./... -count=1`、`go test -race ./service -count=1` 和 `bash build.sh`。
- 修复后的候选脚本再次在生产图片目录执行 dry-run：匹配 11,230 个文件、
  1,842,131,538 bytes，删除 0，`changed_files=0`。
- 部署时先在生产目录执行 dry-run，核对数量和字节数后再首次清理。
- 本策略不清理 Hero/decided 证据图，也不清理服务日志、构建目录、备份或 swap。
- 共享锁协议覆盖当前仓库内全部生产图片写入入口；新增图片写入入口时必须复用
  `writeImageToFileWithLock`，不能绕过该锁直接写入生产图片目录。

## 生产部署与首次清理

- ddbaby 运行时已部署 `e557483` 对应构建，二进制 SHA256 为
  `4eb739d7cb76ba52241b396dec0b36a5bdca9cf3b84f1b1ae1419014c40f55b4`；进程
  PID=1098849、PPID=1，8888 监听和 SSH 重连后的 `/ping` 均正常。
- 清理脚本安装至 `/usr/local/sbin/ddbaby-cleanup-images`，cron 和 logrotate 配置已
  安装。服务器 `/var/log` 权限要求 logrotate 显式指定用户，规则已增加
  `su root adm`，经同目录临时文件校验后原子替换到生产；生产配置 SHA256 为
  `1f0e465457a41fc7135cec658b3a785c6c1644fe146960e342b4ce0fbf353c72`。
  生产路径 `logrotate -d` 无错误，受控强制轮转生成包含 dry-run 摘要的 `.1.gz`
  归档，并按 root:adm/0640 重新创建当前日志；cron 服务为 active。
- 首次实际清理结果：`matched_files=11230`、`matched_bytes=1842131538`、
  `deleted_files=11230`、`deleted_bytes=1842131538`、`changed_files=0`。紧接着的
  第二次运行匹配和删除均为 0。
- 清理前后 Hero 数量均为 765、decided 数量均为 1,690；清理前为全部 2,455 个
  保留文件生成 SHA256 清单，清理后逐项校验全部通过。JPEG 总数由 13,685 降为
  2,455。
- 根盘已用空间由 32,115,527,680 bytes 降为 30,249,852,928 bytes，可用空间由
  8,198,955,008 bytes 增至 10,064,629,760 bytes，使用率由 80% 降为 76%。
- 回滚与审计目录：`/home/ubuntu/ddbaby-backups/20261005-135714-before-e557483`。

## 2026-10-05 保留周期调整

- 按用户要求将普通截图保留周期由 6 小时调整为 5 天，执行频率由每小时调整为每天
  一次，固定在服务器时间 03:17 执行。
- Hero、decided 保留规则、共享写删锁、目录边界和日志轮转策略均不变。
- 调整候选通过 Python 7/7、`py_compile` 和 `git diff --check`；生产目录使用
  7,200 分钟参数 dry-run 匹配 0、删除 0、`changed_files=0`，服务 `/ping` 正常。
