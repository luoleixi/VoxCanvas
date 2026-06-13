# VoxCanvas 设计说明

本文档记录后端实现方案和后续扩展方向。接口请求、响应格式和 curl 示例见 [API.md](./API.md)；日志和云服务器排查见 [OPERATIONS.md](./OPERATIONS.md)。

## 文档分层

| 文档 | 职责 |
| --- | --- |
| [API.md](./API.md) | 面向前端和测试，描述接口契约、请求响应、错误格式 |
| [DESIGN.md](./DESIGN.md) | 面向后端开发，描述会话隔离、数据表、事务、撤销和未来设计 |
| [OPERATIONS.md](./OPERATIONS.md) | 面向部署和排查，描述日志位置、查看方式和运维注意事项 |

## 用户隔离与会话

当前项目没有登录功能，后端通过匿名 Cookie 实现用户隔离：

| Cookie | 说明 |
| --- | --- |
| `vox_client_id` | 匿名用户 ID，同一浏览器长期保持 |
| `vox_session_id` | 当前会话 ID，指向用户正在编辑的绘图会话 |

流程：

1. 用户首次打开页面时 Cookie 为空。
2. 前端调用 `POST /api/v1/session/start`，或首次调用绘图接口时由后端自动初始化。
3. 后端创建 `vox_client_id` 和 `vox_session_id`，并通过 `Set-Cookie` 写回浏览器。
4. 后续请求自动携带 Cookie，后端只操作当前匿名用户和当前会话。
5. 用户说“切换会话”时，当前版本创建一个新会话，并更新 `vox_session_id`。

## 会话标题与摘要

`sessions` 表保存 `title` 和 `summary`，用于历史会话候选展示和后续语音匹配。它们是稳定的会话识别信息，一经自动确定便不再被后续操作自动覆盖。

当前实现不额外调用大模型生成标题，而是从精炼后的绘图文本中截取：

| 字段 | 来源 |
| --- | --- |
| `title` | 精炼文本截断后的短标题 |
| `summary` | 精炼文本截断后的摘要 |

自动写入规则：

- 新会话创建时为空
- 第一次需求精炼完成后，如果 `title/summary` 为空，则写入
- 第一次生成图片成功后，如果 `title/summary` 仍为空，则写入
- 后续需求精炼不覆盖
- 后续图片生成不覆盖
- 撤销不覆盖
- 清空不清空
- 未来如果需要改名，应通过用户主动重命名能力覆盖

## 数据表

### `sessions`

记录匿名用户下的绘图会话。

| 字段 | 说明 |
| --- | --- |
| `id` | 会话 ID |
| `client_id` | 匿名用户 ID |
| `dev` | 当前会话精炼后的绘图文本 |
| `title` | 会话标题 |
| `summary` | 会话摘要 |
| `current_image_id` | 当前会话正在展示或恢复的生成图 ID |
| `undo_image_id` | 下一次撤销要返回的生成图 ID |
| `current_version_id` | 当前会话状态对应的版本节点 ID |
| `undo_version_id` | 下一次撤销要恢复的版本节点 ID |
| `created_at` | 创建时间 |
| `updated_at` | 最近更新时间 |

### `sentences`

记录用户每次语音文本。

| 字段 | 说明 |
| --- | --- |
| `id` | 文本 ID |
| `session_id` | 文本所属会话 |
| `previous_image_id` | 用户说出这句话前，当前会话上一张生成图 ID |
| `content` | 用户语音识别出的原始文本 |
| `type` | 当前固定为 `user_input` |
| `created_at` | 创建时间 |

### `images`

记录每次成功生成图片的结果。

| 字段 | 说明 |
| --- | --- |
| `id` | 图片 ID |
| `session_id` | 图片所属会话 |
| `prompt` | 生成该图片使用的提示词 |
| `base64_data` | 图片 base64 |
| `created_at` | 创建时间 |

### `session_events`

记录会话内关键事件，作为审计、历史回放、后续带参数撤销和图生图来源追踪的基础。

| 字段 | 说明 |
| --- | --- |
| `id` | 事件 ID |
| `session_id` | 事件所属会话 |
| `event_type` | 事件类型 |
| `sentence_id` | 关联的用户语音文本 ID |
| `image_id` | 本次事件产生或恢复的图片 ID |
| `previous_image_id` | 事件发生前的上一张生成图 ID |
| `sentence` | 触发事件的用户原始语音文本 |
| `dev` | 事件完成后的会话文本状态 |
| `before_dev` | 事件发生前的会话文本状态 |
| `before_image_id` | 事件发生前的上一张生成图 ID |
| `created_at` | 创建时间 |

事件类型：

| event_type | 说明 |
| --- | --- |
| `sentence` | 用户输入一句话 |
| `requirement_refined` | 需求精炼完成 |
| `image_generated` | 图片生成完成 |
| `undo` | 执行撤销 |
| `clear` | 清空当前会话 |
| `switch_session` | 切换到新会话 |

### `session_versions`

记录会话状态版本树。生成图片和清空都会创建版本节点，节点之间通过 `parent_id` 串成树。

| 字段 | 说明 |
| --- | --- |
| `id` | 版本节点 ID |
| `session_id` | 版本所属会话 |
| `parent_id` | 父版本节点 ID；为空表示根版本 |
| `event_type` | 产生该版本的事件类型，例如 `image_generated`、`clear` |
| `image_id` | 当前版本对应的图片 ID；清空版本为空 |
| `dev` | 当前版本对应的文本状态；清空版本为空 |
| `created_at` | 创建时间 |

## 事务边界

数据库写入采用同步事务封装，不再使用异步写入队列。后端会先完成意图识别、需求精炼、图片生成等外部模型调用，再把相关数据库变更放入同一个 SQLite 事务中提交，避免长时间持有数据库锁。

| 操作 | 同一事务内写入 |
| --- | --- |
| 用户输入 | 写入 `sentences`，写入 `session_events(sentence)` |
| 需求精炼 | 更新 `sessions.dev`，仅当 `title/summary` 为空时写入会话标题摘要，写入 `session_events(requirement_refined)` |
| 图片生成成功 | 写入 `images`，创建 `session_versions(image_generated)` 节点，仅当 `title/summary` 为空时写入会话标题摘要，更新当前版本和撤销目标，写入 `session_events(image_generated)`，清空 `sessions.dev` |
| 撤销 | 查询 `undo_version_id` 对应版本，恢复 `sessions.dev/current_image_id/current_version_id`，前移撤销目标，写入 `session_events(undo)` |
| 清空 | 创建 `session_versions(clear)` 节点，清空 `sessions.dev/current_image_id`，将撤销目标指向清空前版本，写入 `session_events(clear)` |
| 切换新会话 | 创建或更新新 `sessions`，写入 `session_events(switch_session)` |

如果事务内任一写入失败，本次业务状态和事件日志都会一起回滚，避免出现状态与日志不一致。

## 版本树与连续撤销

当前版本先不做带参数撤销。用户每次说“撤销”，都沿当前会话版本树向父节点回退。

实现摘要：

1. 每次成功生成图片后，后端将图片写入 `images`。
2. 同一事务内创建 `session_versions(image_generated)`，其 `parent_id` 指向生成前的 `current_version_id`。
3. 生成后，`sessions.current_version_id` 和 `sessions.undo_version_id` 都指向本次生成版本。
4. 用户触发 `undo` 时，后端读取 `sessions.undo_version_id`。
5. 如果目标版本是图片版本，后端根据 `image_id` 查询 `images.prompt` 和 `images.base64_data`，返回图片和文本。
6. 如果目标版本是清空版本，后端恢复为空画布，返回空 `text/image`。
7. 恢复完成后，`sessions.current_version_id` 更新为目标版本，`sessions.undo_version_id` 前移到目标版本的 `parent_id`。
8. 如果没有父版本，再次撤销返回空 `text/image`。
9. 撤销后如果再次生成图片，新图片版本会以当前版本为父节点，形成新的分支；旧分支不会被删除。
10. 每次撤销都会写入 `session_events(undo)`。

前端不需要传撤销参数，仍只发送：

```json
{
  "sentences": "撤销"
}
```

## 清空

当前清空会话会：

- 清空内存中的当前精炼文本
- 清空 `sessions.dev`
- 清空 `sessions.current_image_id`
- 创建 `session_versions(clear)` 节点，父节点指向清空前的当前版本
- 将 `sessions.current_version_id` 指向清空版本
- 将 `sessions.undo_version_id` 指向清空前的父版本
- 写入 `session_events(clear)`

清空不会删除 `images` 历史数据，也不会删除旧版本节点。这样用户清空后再次说“撤销”，后端可以根据 `undo_version_id` 找回清空前的版本，并恢复：

- 返回清空前图片的 `base64_data`
- 返回清空前图片的 `prompt`
- 将该 `prompt` 写回 `sessions.dev`
- 将 `sessions.current_image_id` 恢复为该图片 ID
- 将 `sessions.current_version_id` 恢复为清空前版本
- 将 `sessions.undo_version_id` 前移到更早的父版本

无数据库模式下，内存 `GeneratedStore` 会保留历史生成结果。清空只会把当前显示游标置空，不删除历史栈，因此后续撤销仍能恢复清空前结果。

## 切回历史会话

当前版本的 `switch_session` 只创建并切换到新会话。后续支持切回历史会话时，建议仍保持前端只发送语音文本：

```json
{
  "sentences": "打开海边小屋那张"
}
```

后端扩展流程：

1. 根据 `vox_client_id` 查询当前匿名用户的历史会话。
2. 根据语音文本识别切换目标，例如“上一个会话”“最近一个会话”“海边小屋”。
3. 在该用户自己的会话列表中匹配 `title`、`summary` 和更新时间。
4. 如果匹配唯一会话，更新 `vox_session_id` Cookie。
5. 如果匹配多个或无法匹配，保持当前会话不变，并返回 `unknown` 或进入候选确认流程。

## 带参数撤销

后续可以把撤销扩展为带参数的语音指令，让用户通过自然语言决定撤销目标。

建议的内部目标：

| undo_target | 说明 |
| --- | --- |
| `last_requirement` | 撤销到上一次需求精炼后的文本 |
| `last_image` | 撤销到上一次成功生成的图片 |
| `nth_requirement` | 撤销到第 N 次需求精炼 |
| `nth_image` | 撤销到第 N 次成功生成的图片 |
| `previous_step` | 按事件时间线撤销一步 |
| `unknown` | 无法识别撤销目标 |

建议后端只让 LLM 负责解析 `op=undo`、`undo_target` 和可选序号 `undo_index`。真正的数据查询、权限隔离、状态恢复仍由后端完成。

## 图生图

图生图建议排在事件日志和撤销能力之后。原因是图生图需要明确：

- 当前图是哪张
- 上一张图是哪张
- 新图基于哪张图生成
- 清空、撤销、切换会话后当前 source image 如何确定

长期建议在 `images` 中增加 `source_image_id`，并通过 `session_events` 记录图生图来源关系。

## 后续优先级

| 优先级 | 能力 | 说明 |
| --- | --- | --- |
| P0 | 事件日志 | 已具备基础事件表，继续作为撤销、回放、图生图来源追踪基础 |
| P1 | 连续撤销 | 当前已支持按版本树连续撤销 |
| P2 | 清空恢复 | 基于事件日志恢复清空前状态 |
| P3 | 切回历史会话 | 基于 `title/summary` 和更新时间匹配目标会话 |
| P4 | 带参数撤销 | 支持撤销到指定需求或指定图片 |
| P5 | 图生图 | 使用当前图或历史图作为 source image 生成新图 |
