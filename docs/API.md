# VoxCanvas API 规范

## 1. 概述

VoxCanvas 是一个纯语音控制的绘图工具。用户不能使用鼠标或键盘，前端只负责把语音识别成文本并提交给后端；用户身份、当前会话和多轮绘图上下文由后端通过 Cookie 自动维护。

- 默认服务地址：`http://localhost:6060`
- 接口前缀：`/api/v1`
- 数据格式：`application/json; charset=utf-8`
- 用户隔离方式：匿名 Cookie
- 当前会话方式：会话 Cookie

后端使用两个 Cookie：

| Cookie | 说明 |
| --- | --- |
| `vox_client_id` | 匿名用户标识，用于区分同一浏览器用户 |
| `vox_session_id` | 当前会话标识，用于定位当前正在编辑的绘图会话 |

前端请求必须携带 Cookie。跨域请求时需要设置：

```js
fetch(url, {
  credentials: "include"
});
```

## 2. 通用响应结构

所有业务接口统一返回以下 JSON 结构：

```json
{
  "code": 200,
  "msg": "success",
  "data": {}
}
```

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `code` | number | 是 | 业务状态码，成功为 `200` |
| `msg` | string | 是 | 响应说明，成功为 `success` |
| `data` | object/null | 是 | 响应数据；失败时可为 `null` |

## 3. 绘图响应数据

`POST /api/v1/draw/understand` 的 `data` 固定返回三个字段：

```json
{
  "op": "requirement",
  "text": "一只猫在月光下散步，画面氛围安静柔和",
  "image": ""
}
```

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `op` | string | 是 | 操作类型 |
| `text` | string | 是 | `requirement` 时返回精炼后的绘图需求；`undo` 时返回撤销到的上一张生成图提示词；其他情况固定为空字符串 |
| `image` | string | 是 | `generate_image` 时返回图片 base64；`undo` 时返回撤销到的上一张生成图 base64；其他情况固定为空字符串 |

`op` 枚举：

| op | 说明 |
| --- | --- |
| `requirement` | 用户输入被识别为绘图需求 |
| `generate_image` | 用户要求生成图片 |
| `undo` | 用户要求撤销 |
| `clear` | 用户要求清空当前会话 |
| `switch_session` | 用户要求切换会话；当前版本表示新建并切换到一个新会话 |
| `unknown` | 无法识别的语音文本 |

## 4. 创建或初始化会话

### `POST /api/v1/session/start`

创建一个新的绘图会话，并将该会话设置为当前会话。

如果请求中没有 `vox_client_id` Cookie，后端会自动创建匿名用户标识。如果请求中已有 `vox_client_id` Cookie，后端会在该匿名用户下创建新会话。

#### 请求

请求体可以为空 JSON：

```json
{}
```

#### 成功响应

HTTP 状态码：`200 OK`

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "session_id": "sess_20260613_235959123"
  }
}
```

响应头会设置或更新 Cookie：

```http
Set-Cookie: vox_client_id=client_xxx; HttpOnly; SameSite=Lax; Path=/; Max-Age=31536000
Set-Cookie: vox_session_id=sess_20260613_235959123; HttpOnly; SameSite=Lax; Path=/; Max-Age=31536000
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `data.session_id` | string | 新创建并切换到的会话 ID |

#### curl 示例

```bash
curl -i -X POST http://localhost:6060/api/v1/session/start \
  -H "Content-Type: application/json" \
  -d "{}"
```

## 5. 理解绘图语音文本

### `POST /api/v1/draw/understand`

前端将语音识别得到的文本发送给后端。前端不需要在 JSON 中传 `client_id` 或 `session_id`，后端通过 Cookie 判断匿名用户和当前会话。

#### 请求

HTTP Header：

```http
Content-Type: application/json
Cookie: vox_client_id=client_xxx; vox_session_id=sess_xxx
```

请求体：

```json
{
  "sentences": "画一只正在月光下散步的猫"
}
```

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `sentences` | string | 是 | 前端语音识别出的文本 |

如果请求中没有当前会话 Cookie，后端可以自动创建一个新会话并写入 `vox_session_id` Cookie。

## 6. 成功响应示例

### 绘图需求

用户说：

```text
画一只正在月光下散步的猫
```

响应：

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "op": "requirement",
    "text": "一只猫在月光下散步，画面氛围安静柔和",
    "image": ""
  }
}
```

### 生成图片

用户说：

```text
生成图片
```

响应：

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "op": "generate_image",
    "text": "",
    "image": "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJ..."
  }
}
```

### 撤销

用户说：

```text
撤销
```

响应：

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "op": "undo",
    "text": "一只猫在月光下散步，画面氛围安静柔和",
    "image": "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJ..."
  }
}
```

当前版本的撤销语义是：直接撤销到当前会话上一次成功生成的图片及其生成文本。后端会把该文本恢复为当前会话的 `dev`，用户可以继续基于这张图的文本进行语音修改。

如果当前会话还没有成功生成过图片，响应仍为：

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "op": "undo",
    "text": "",
    "image": ""
  }
}
```

### 清空当前会话

用户说：

```text
清空画布
```

响应：

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "op": "clear",
    "text": "",
    "image": ""
  }
}
```

### 切换会话

当前版本中，“切换会话”表示创建一个新会话并切换到该新会话。用户说：

```text
切换会话
```

后端行为：

1. 读取或创建 `vox_client_id`。
2. 创建新的 `session_id`。
3. 将新会话绑定到当前匿名用户。
4. 更新 `vox_session_id` Cookie。
5. 后续绘图请求自动进入新会话。

响应：

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "op": "switch_session",
    "text": "",
    "image": ""
  }
}
```

### 无法识别

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "op": "unknown",
    "text": "",
    "image": ""
  }
}
```

### 后续扩展：切回历史会话

当前版本只支持切换到新会话。后续如果需要支持“切回历史会话”，建议仍然保持前端只发送语音文本：

```json
{
  "sentences": "切回上一个会话"
}
```

或：

```json
{
  "sentences": "打开海边小屋那张"
}
```

后端扩展行为：

1. 根据 `vox_client_id` 查询当前匿名用户的历史会话。
2. 根据语音文本识别切换目标，例如“上一个会话”“最近一个会话”“海边小屋”。
3. 在该用户自己的会话列表中匹配目标会话。
4. 如果匹配唯一会话，更新 `vox_session_id` Cookie。
5. 如果匹配多个或无法匹配，保持当前会话不变，并返回 `op=unknown` 或后续新增的候选确认流程。

为了支持语义匹配，建议会话表后续保存摘要字段：

| 字段 | 说明 |
| --- | --- |
| `title` | 会话标题，可由第一次需求或最终提示词生成 |
| `preview` | 会话摘要，用于语音匹配和前端展示 |
| `updated_at` | 最近更新时间，用于“上一个”“最近一个”等语音指令 |

示例会话摘要：

```json
{
  "session_id": "sess_20260613_235959123",
  "title": "海边小屋",
  "preview": "夕阳下的海边小屋，天空有几只海鸥",
  "updated_at": "2026-06-13T23:59:59+08:00"
}
```

在不引入鼠标键盘的前提下，后续可以支持这些语音：

| 语音 | 行为 |
| --- | --- |
| “切回上一个会话” | 切换到当前会话之前的最近会话 |
| “打开最近那个会话” | 切换到最近更新时间最高的其他会话 |
| “打开海边小屋那张” | 根据标题或摘要匹配历史会话 |
| “新建会话” | 创建并切换到新会话 |

## 7. 多轮需求与会话隔离

同一个浏览器通过 `vox_client_id` 识别匿名用户。每个匿名用户可以拥有多个绘图会话，但当前请求只操作 `vox_session_id` 指向的当前会话。

多轮流程：

1. 前端调用 `POST /api/v1/session/start`，后端设置匿名用户和当前会话 Cookie。
2. 用户说绘图需求，前端调用 `POST /api/v1/draw/understand`。
3. 后端只在当前 `vox_session_id` 对应的会话中累积和精炼 `dev`。
4. 用户继续补充需求，后端将当前会话旧 `dev` 与新文本一起精炼。
5. 用户说“生成图片”，后端只使用当前会话的 `dev` 生成图片。
6. 用户说“切换会话”，后端创建新会话并更新 `vox_session_id`。
7. 后续语音请求进入新会话，不影响旧会话内容。

## 8. 数据记录与后续日志表设计

### 当前数据记录

`sessions` 表记录匿名用户下的会话，并保存当前精炼后的 `dev`。

| 字段 | 说明 |
| --- | --- |
| `id` | 会话 ID |
| `client_id` | 匿名用户标识 |
| `dev` | 当前会话精炼后的绘图需求文本 |
| `created_at` | 创建时间 |
| `updated_at` | 最近更新时间 |

`sentences` 表记录用户每次语音文本，并通过 `session_id` 绑定会话。

| 字段 | 说明 |
| --- | --- |
| `session_id` | 当前语音文本所属会话 |
| `previous_image_id` | 用户说出这句话时，该会话上一张成功生成图片的 ID；没有上一张图时为空 |
| `content` | 用户语音识别出的原始文本 |
| `type` | 当前固定为 `user_input` |
| `created_at` | 创建时间 |

`images` 表记录每次成功生成图片的结果。

| 字段 | 说明 |
| --- | --- |
| `session_id` | 图片所属会话 |
| `prompt` | 生成该图片使用的提示词 |
| `base64_data` | 图片 base64 |
| `created_at` | 创建时间 |

`previous_image_id` 用于把每句话和“说出这句话之前的上一张生成图”关联起来，方便后续实现更精细的撤销、历史查看和回放。

### 后续扩展：session_events 日志表

后续如果需要完整操作回放、多步撤销、恢复历史会话，建议新增事件日志表：

```sql
CREATE TABLE session_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    sentence TEXT,
    image_id INTEGER,
    previous_image_id INTEGER,
    dev TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

建议事件类型：

| event_type | 说明 |
| --- | --- |
| `sentence` | 用户输入了一句话 |
| `requirement_refined` | 需求精炼完成 |
| `image_generated` | 图片生成完成 |
| `undo` | 执行撤销 |
| `clear` | 清空当前会话 |
| `switch_session` | 切换到新会话 |

该表作为未来设计，当前版本先通过 `sentences.previous_image_id` 和内存中的最近生成结果支持基础撤销。

### 后续扩展：带参数撤销

当前版本中，用户说“撤销”时，默认撤销到当前会话上一次成功生成的图片及其生成文本。后续可以把撤销扩展为带参数的语音指令，让用户通过自然语言决定撤销目标。

当前撤销不是完整版本栈，只保存当前会话最近一次生成结果。未来需要升级为基于 `session_events` 的版本控制：连续多次撤销应按历史版本逐步回退；撤销后如果再次生成新图，应以新生成结果作为新的当前版本，并丢弃或分叉撤销后的未来版本。

建议继续保持前端请求不变：

```json
{
  "sentences": "撤销到第二次生成的图"
}
```

后端内部可以让 LLM 在意图识别阶段解析撤销参数，但最终返回给前端仍保持 `op/text/image` 三字段。

内部撤销目标建议：

| undo_target | 说明 |
| --- | --- |
| `last_requirement` | 撤销到上一次需求精炼后的文本 |
| `last_image` | 撤销到上一次成功生成的图片 |
| `nth_requirement` | 撤销到第 N 次需求精炼 |
| `nth_image` | 撤销到第 N 次成功生成的图片 |
| `previous_step` | 按事件时间线撤销一步 |
| `unknown` | 无法识别撤销目标 |

语音示例：

| 语音 | undo_target | 说明 |
| --- | --- | --- |
| “撤销到上一次需求” | `last_requirement` | 返回上一版精炼文本，`image` 为空 |
| “回到上一张图” | `last_image` | 返回上一张图的 prompt 和 base64 |
| “撤销到第二次生成的图” | `nth_image` | 返回第 2 次生成图的 prompt 和 base64 |
| “回到第三版需求” | `nth_requirement` | 返回第 3 次需求精炼文本，`image` 为空 |
| “撤销一步” | `previous_step` | 根据 `session_events` 时间线回退一步 |

后端执行建议：

1. LLM 只负责解析 `op=undo`、`undo_target` 和可选的序号 `undo_index`。
2. 后端根据当前 `vox_session_id` 查询 `session_events`。
3. 如果目标是需求精炼，查找 `requirement_refined` 事件并恢复 `dev`。
4. 如果目标是生成图，查找 `image_generated` 事件，再根据 `image_id` 查询 `images.base64_data`，同时恢复 `dev`。
5. 如果无法匹配目标，返回 `op=undo`，且 `text`、`image` 均为空。

返回示例：撤销到文本版本

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "op": "undo",
    "text": "一只猫坐在月光下，背景是安静的森林",
    "image": ""
  }
}
```

返回示例：撤销到生成图版本

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "op": "undo",
    "text": "一只猫坐在月光下，背景是安静的森林",
    "image": "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJ..."
  }
}
```

### 后续扩展优先级

建议后续能力按以下顺序推进：

| 优先级 | 能力 | 说明 |
| --- | --- | --- |
| P0 | `session_events` 事件日志 | 作为多步撤销、清空恢复、历史回放、图生图来源追踪的基础 |
| P1 | 会话标题与摘要 | 在 `sessions` 中增加 `title`、`preview`，支持“打开海边小屋那张”等语音匹配 |
| P2 | 完整撤销能力 | 支持撤销到上一次需求、上一次图片、第 N 次需求、第 N 次图片，以及清空后恢复 |
| P3 | 切回历史会话 | 基于会话摘要和事件历史，支持切回上一个会话、最近会话或指定主题会话 |
| P4 | 图生图 | 使用当前会话最近生成图或指定历史图作为 source image，根据语音需求生成新图 |

图生图建议排在事件日志和撤销能力之后。原因是图生图需要明确“当前图是哪张”“上一张图是哪张”“新图基于哪张图生成”，这些关系最好由 `session_events`、`images.source_image_id` 和版本状态共同支撑。

如果产品目标是尽快展示 AI 绘图能力，可以先做最小版图生图：默认使用当前会话最近一次生成图作为 source image。但长期仍建议补齐事件日志和图片来源关系，避免后续在撤销、清空、切换会话后难以判断图生图的来源。

## 9. curl 示例

使用 Cookie 文件保存匿名用户和当前会话：

```bash
curl -i -c cookies.txt -b cookies.txt \
  -X POST http://localhost:6060/api/v1/session/start \
  -H "Content-Type: application/json" \
  -d "{}"
```

提交绘图需求：

```bash
curl -i -c cookies.txt -b cookies.txt \
  -X POST http://localhost:6060/api/v1/draw/understand \
  -H "Content-Type: application/json" \
  -d "{\"sentences\":\"画一只正在月光下散步的猫\"}"
```

生成图片：

```bash
curl -i -c cookies.txt -b cookies.txt \
  -X POST http://localhost:6060/api/v1/draw/understand \
  -H "Content-Type: application/json" \
  -d "{\"sentences\":\"生成图片\"}"
```

切换到新会话：

```bash
curl -i -c cookies.txt -b cookies.txt \
  -X POST http://localhost:6060/api/v1/draw/understand \
  -H "Content-Type: application/json" \
  -d "{\"sentences\":\"切换会话\"}"
```

## 10. 错误响应

### 请求体错误

HTTP 状态码：`400 Bad Request`

```json
{
  "code": 400,
  "msg": "invalid request body",
  "data": null
}
```

常见原因：

- 请求体不是合法 JSON。
- `Content-Type` 与请求体格式不匹配。

### 服务端错误

HTTP 状态码：`500 Internal Server Error`

```json
{
  "code": 500,
  "msg": "error message",
  "data": null
}
```

常见原因：

- 云端 LLM 调用失败。
- 生成图模型调用失败。
- SQLite 写入失败。
- 服务端内部处理异常。

## 11. 健康检查

### `GET /health`

用于检查后端服务是否可用。

成功响应示例：

```json
{
  "status": "ok"
}
```
