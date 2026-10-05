# 生产截图保留策略

## 目标

定期删除 `/usr/local/webserver/images` 中不用于牌局证据的普通 JPEG 截图，持续保留
`*-hero.jpg` 和 `*-decided.jpg`。普通截图保留最近 6 小时，便于短期故障排查。

## 实现

- `scripts/cleanup_images.py` 仅遍历目标目录的直接子文件；跳过子目录、符号链接、
  非 JPEG、Hero 和 decided 文件。
- 默认只删除修改时间早于 360 分钟的普通 `*.jpg`，支持 `--dry-run` 先统计。
- 使用 `/run/lock/ddbaby-image-cleanup.lock` 防止定时任务重入；单个文件删除失败时记录
  错误并以非零状态退出。
- ddbaby 的两条生产图片落盘入口与清理脚本共同使用跨进程排他锁
  `/run/lock/ddbaby-images.lock`。清理先扫描候选文件，再在持锁期间完成最终身份检查和
  删除；写入也在同一把锁内完成，因此检查通过后至 `unlink` 之间不能插入同名写入。
- 真正删除前仍会再次读取文件元数据；设备号、inode、大小或纳秒 mtime 任一变化，
  或文件已不再是过期普通文件时，跳过删除并计入 `changed_files`。
- cron 每小时第 17 分钟执行，日志按周轮转，保留 4 份压缩归档。

## 上线前只读统计

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
