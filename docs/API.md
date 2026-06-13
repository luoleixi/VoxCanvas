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
| `text` | string | 是 | 仅当 `op=requirement` 时返回精炼后的绘图需求；其他情况固定为空字符串 |
| `image` | string | 是 | 仅当 `op=generate_image` 时返回图片 base64；其他情况固定为空字符串 |

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

## 8. curl 示例

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

## 9. 错误响应

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

## 10. 健康检查

### `GET /health`

用于检查后端服务是否可用。

成功响应示例：

```json
{
  "status": "ok"
}
```
