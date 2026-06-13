# VoxCanvas API 规范

## 1. 概述

VoxCanvas 后端为语音控制绘图工具提供会话创建、语音文本理解和图片生成能力。前端负责将用户语音识别为文本，并通过 HTTP JSON 接口提交给后端。

- 默认服务地址：`http://localhost:6060`
- 接口前缀：`/api/v1`
- 数据格式：`application/json; charset=utf-8`

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

## 3. 创建会话

### `POST /api/v1/session/start`

创建一次绘图会话，返回后端生成的会话 ID。

#### 请求

当前接口无需请求体。

#### 成功响应

HTTP 状态码：`200 OK`

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "session_id": "sess_20260612_235959123"
  }
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `data.session_id` | string | 会话 ID，格式为 `sess_YYYYMMDD_HHMMSSmmm` |

#### curl 示例

```bash
curl -X POST http://localhost:6060/api/v1/session/start
```

## 4. 理解绘图语音文本

### `POST /api/v1/draw/understand`

前端将语音识别得到的文本发送给后端。后端会调用云端 LLM 判断文本属于指令还是绘图需求。

- 如果是指令，返回 `op = "order"`。
- 如果是需求，返回 `op = "requirement"`，并将多轮需求精炼后保存到后端临时上下文 `dev` 中。
- 每次收到文本后，后端会异步写入 SQLite。
- 当触发“生成图片”指令时，后端会将当前 `dev` 内容发送给生成图模型，返回图片 base64，并将生成结果异步写入 SQLite。

#### 请求

HTTP Header：

```http
Content-Type: application/json
```

请求体：

```json
{
  "sentences": "画一只正在月光下散步的猫"
}
```

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `sentences` | string | 是 | 前端语音识别出的单句或一句完整指令 |

#### 成功响应：需求

HTTP 状态码：`200 OK`

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "op": "requirement",
    "content": "一只猫在月光下散步，画面氛围安静柔和"
  }
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `data.op` | string | 固定为 `requirement`，表示当前文本被识别为绘图需求 |
| `data.content` | string | LLM 精炼后的绘图需求；后端会同步更新 `dev` |

#### 成功响应：普通指令

HTTP 状态码：`200 OK`

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "op": "order",
    "content": "生成图片"
  }
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `data.op` | string | 固定为 `order`，表示当前文本被识别为指令 |
| `data.content` | string | LLM 校验后的指令内容 |

#### 成功响应：生成图片指令

HTTP 状态码：`200 OK`

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "op": "order",
    "content": "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJ..."
  }
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `data.op` | string | 固定为 `order` |
| `data.content` | string | 图片 base64 数据，不包含 `data:image/png;base64,` 前缀 |

#### 多轮需求精炼流程

1. 用户第一次发送需求，例如：“画一座海边小屋”。
2. 后端将文本发送至 LLM 精炼，结果保存到 `dev`，并返回 `requirement`。
3. 用户继续发送需求，例如：“加上夕阳和几只海鸥”。
4. 后端将旧 `dev` 与新文本一起发送至 LLM 精炼。
5. 后端用新的精炼结果覆盖 `dev`，并返回 `requirement`。
6. 用户发送“生成图片”后，后端使用当前 `dev` 调用生成图模型并返回 base64 图片。
7. 图片生成后，后端清空当前 `dev`。

#### curl 示例

提交绘图需求：

```bash
curl -X POST http://localhost:6060/api/v1/draw/understand \
  -H "Content-Type: application/json" \
  -d "{\"sentences\":\"画一只正在月光下散步的猫\"}"
```

触发生成图片：

```bash
curl -X POST http://localhost:6060/api/v1/draw/understand \
  -H "Content-Type: application/json" \
  -d "{\"sentences\":\"生成图片\"}"
```

## 5. 错误响应

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
- 服务端内部处理异常。
- 服务端内部处理异常。

## 7. 健康检查

### `GET /health`

用于检查后端服务是否可用。

成功响应示例：

```json
{
  "status": "ok"
}
```
