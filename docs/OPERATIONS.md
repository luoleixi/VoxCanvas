# VoxCanvas 运维日志

## 日志输出

后端启动时会把标准日志同时写入：

- systemd journal
- 日志文件：`/var/log/voxcanvas/voxcanvas-backend.log`

本地开发默认写入：

- `backend/logs/voxcanvas-backend.log`

可以通过环境变量覆盖：

```bash
LOG_DIR=/var/log/voxcanvas
LOG_FILE=voxcanvas-backend.log
```

## 云服务器查看日志

查看实时 systemd 日志：

```bash
journalctl -u voxcanvas-backend -f
```

查看最近 200 行：

```bash
journalctl -u voxcanvas-backend -n 200 --no-pager
```

查看日志文件：

```bash
tail -f /var/log/voxcanvas/voxcanvas-backend.log
```

按关键链路过滤：

```bash
grep "\[DRAW\]" /var/log/voxcanvas/voxcanvas-backend.log
grep "\[LLM\]" /var/log/voxcanvas/voxcanvas-backend.log
grep "\[IMAGE\]" /var/log/voxcanvas/voxcanvas-backend.log
grep "\[DB\]" /var/log/voxcanvas/voxcanvas-backend.log
grep "\[SESSION\]" /var/log/voxcanvas/voxcanvas-backend.log
```

## 日志标签

| 标签 | 说明 |
| --- | --- |
| `[LOGGER]` | 日志系统初始化 |
| `[CONFIG]` | 后端配置，API Key 会脱敏 |
| `[SESSION]` | 会话创建 |
| `[DRAW]` | 语音理解、意图识别、精炼、生成、撤销、清空、切换会话 |
| `[LLM]` | 文本大模型请求与响应 |
| `[IMAGE]` | 图片生成模型请求与响应 |
| `[DB]` | SQLite 事务写入与事件日志写入 |

## 排查建议

1. 先看服务是否存活：

```bash
systemctl status voxcanvas-backend --no-pager
```

2. 再看最近错误：

```bash
journalctl -u voxcanvas-backend -n 200 --no-pager
```

3. 跟踪一次完整语音请求：

```bash
tail -f /var/log/voxcanvas/voxcanvas-backend.log
```

重点观察这些字段：

- `client_id`
- `session_id`
- `op`
- `duration_ms`
- `image_id`
- `previous_image_id`

## 注意事项

- 不要在日志中打印完整 API Key。
- 不要在业务日志中打印完整图片 base64。
- 当前 `[IMAGE] response` 只截断响应体前 500 字符。
