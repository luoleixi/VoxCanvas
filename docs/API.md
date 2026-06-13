# VoxCanvas API 规范

## 1. 概述

VoxCanvas 后端提供纯语音绘图能力。前端只需要把语音识别结果作为文本提交给后端；匿名用户标识、当前会话、多轮绘图上下文由后端通过 Cookie 和数据库维护。

- 默认服务地址：`http://localhost:6060`
- 接口前缀：`/api/v1`
- 数据格式：`application/json; charset=utf-8`
- 用户隔离方式：匿名 Cookie `vox_client_id`
- 当前会话方式：会话 Cookie `vox_session_id`

更详细的会话隔离、数据表、连续撤销和未来能力设计见 [DESIGN.md](./DESIGN.md)。

## 2. Cookie

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

## 3. 通用响应

所有业务接口统一返回：

```json
{
  "code": 200,
  "msg": "success",
  "data": {}
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `code` | number | 业务状态码，成功为 `200` |
| `msg` | string | 响应说明，成功为 `success` |
| `data` | object/array/null | 响应数据；失败时可为 `null` |

## 4. 会话接口

### `POST /api/v1/session/start`

创建新绘图会话，并把该会话设置为当前会话。

如果请求中没有 `vox_client_id`，后端会自动创建匿名用户标识。如果请求中已有 `vox_client_id`，后端会在该匿名用户下创建新会话。

请求体可以为空 JSON：

```json
{}
```

成功响应：

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "session_id": "sess_20260613_235959_abcd1234"
  }
}
```

响应头会设置或更新：

```http
Set-Cookie: vox_client_id=client_xxx; HttpOnly; SameSite=Lax; Path=/; Max-Age=31536000
Set-Cookie: vox_session_id=sess_xxx; HttpOnly; SameSite=Lax; Path=/; Max-Age=31536000
```

### `GET /api/v1/session/list`

查询当前匿名用户 `vox_client_id` 下的历史会话摘要，默认返回最近 20 条。

请求：

```http
GET /api/v1/session/list?limit=20
Cookie: vox_client_id=client_xxx
```

响应：

```json
{
  "code": 200,
  "msg": "success",
  "data": [
    {
      "session_id": "sess_20260613_235959_abcd1234",
      "title": "海边小屋",
      "summary": "夕阳下的海边小屋，天空有几只海鸥",
      "dev": "夕阳下的海边小屋，天空有几只海鸥",
      "updated_at": "2026-06-13 23:59:59"
    }
  ]
}
```

| 字段 | 说明 |
| --- | --- |
| `session_id` | 会话 ID |
| `title` | 会话标题，由当前精炼后的绘图文本自动截取生成 |
| `summary` | 会话摘要，由当前精炼后的绘图文本自动截取生成 |
| `dev` | 当前会话尚未出图的精炼文本，出图成功或清空后可能为空 |
| `updated_at` | 最近更新时间 |

## 5. 绘图理解接口

### `POST /api/v1/draw/understand`

前端将语音识别得到的文本发送给后端。前端不需要在 JSON 中传 `client_id` 或 `session_id`，后端通过 Cookie 判断匿名用户和当前会话。

请求：

```http
POST /api/v1/draw/understand
Content-Type: application/json
Cookie: vox_client_id=client_xxx; vox_session_id=sess_xxx
```

```json
{
  "sentences": "画一只正在月光下散步的猫"
}
```

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `sentences` | string | 是 | 前端语音识别出的文本 |

如果请求中没有当前会话 Cookie，后端会自动创建一个新会话并写入 `vox_session_id`。

## 6. 绘图响应数据

`POST /api/v1/draw/understand` 的 `data` 固定返回：

```json
{
  "op": "requirement",
  "text": "一只猫在月光下散步，画面氛围安静柔和",
  "image": ""
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `op` | string | 操作类型 |
| `text` | string | `requirement` 时返回精炼后的绘图需求；`undo` 时返回撤销到的生成图提示词；其他情况为空字符串 |
| `image` | string | `generate_image` 时返回图片 base64；`undo` 时返回撤销到的生成图 base64；其他情况为空字符串 |

`op` 枚举：

| op | 说明 |
| --- | --- |
| `requirement` | 用户输入被识别为绘图需求 |
| `generate_image` | 用户要求生成图片 |
| `undo` | 用户要求撤销 |
| `clear` | 用户要求清空当前会话 |
| `switch_session` | 用户要求切换会话；当前版本表示新建并切换到一个新会话 |
| `unknown` | 无法识别的语音文本 |

## 7. 响应示例

### 绘图需求

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

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "op": "generate_image",
    "text": "",
    "image": "iVBORw0KGgoAAAANSUhEUgAAA..."
  }
}
```

### 撤销

当前版本中，用户只需要说“撤销”。后端会沿当前会话的生成图片历史连续回退，并返回撤销到的图片和生成文本。实现细节见 [DESIGN.md#连续撤销](./DESIGN.md#连续撤销)。

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "op": "undo",
    "text": "一只猫在月光下散步，画面氛围安静柔和",
    "image": "iVBORw0KGgoAAAANSUhEUgAAA..."
  }
}
```

如果没有可撤销图片：

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

清空会移除当前画布展示和当前精炼文本。清空后再次说“撤销”，后端会尝试恢复清空前的上一张生成图及其文本。实现细节见 [DESIGN.md#清空](./DESIGN.md#清空)。

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

当前版本中，“切换会话”表示创建一个新会话并切换过去。后端会更新 `vox_session_id` Cookie。

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

撤销：

```bash
curl -i -c cookies.txt -b cookies.txt \
  -X POST http://localhost:6060/api/v1/draw/understand \
  -H "Content-Type: application/json" \
  -d "{\"sentences\":\"撤销\"}"
```

查询历史会话摘要：

```bash
curl -i -c cookies.txt -b cookies.txt \
  http://localhost:6060/api/v1/session/list?limit=20
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

- 请求体不是合法 JSON
- `Content-Type` 与请求体格式不匹配

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

- 云端 LLM 调用失败
- 图片生成模型调用失败
- SQLite 写入失败
- 服务端内部处理异常

## 10. 健康检查

### `GET /health`

用于检查后端服务是否可用。

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "service": "voxcanvas-backend",
    "status": "ok"
  }
}
```
