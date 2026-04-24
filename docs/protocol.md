# 消息协议

所有端与 relay 之间使用统一 JSON 消息格式。术语定义见
[conventions.md](conventions.md)；架构与组件拓扑见 [architecture.md](architecture.md)。

**Schema 源**：[`shared/proto/appunvs.proto`](../shared/proto/appunvs.proto)。
所有语言的 wire 类型从 proto 生成或手工镜像，并由 drift test 守护一致。
JSON 线上格式使用 canonical protojson（启用 `UseProtoNames`），字段名保持
`snake_case`；枚举序列化为短名小写（`ROLE_PROVIDER → "provider"`、
`PUBLISH_STATE_PUBLISHED → "published"`、`BUILD_STATE_SUCCEEDED → "succeeded"`）。

## 1. 数据消息（WebSocket）

`Message` 是端 ↔ relay 在 `/ws` 上的基本数据帧，承担用户表的 upsert / delete /
schema 变更广播。**和 Stage 流程无关**——Stage / Box / Pair 走独立 HTTP +
独立 WS 事件类型，详见 §3。

```json
{
  "seq": 1024,
  "device_id": "device_abc",
  "user_id": "user_xyz",
  "namespace": "user_xyz",
  "role": "provider",
  "op": "upsert",
  "table": "records",
  "payload": { "id": "r1", "data": "..." },
  "ts": 1714000000000
}
```

## 2. HTTP 接口

### 2.1 账号 / 设备

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/auth/signup` | 注册账号 |
| `POST` | `/auth/login`  | 登录 |
| `POST` | `/auth/register` | 注册设备，换发 device JWT |
| `GET`  | `/auth/me` | 当前账号 + 设备列表 |

### 2.2 Box / Stage（device JWT）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST`   | `/box`                | 创建 Box（draft 状态） |
| `GET`    | `/box`                | 列出当前 namespace 下所有 Box |
| `GET`    | `/box/:id`            | 详情，含 `current` BundleRef |
| `POST`   | `/box/:id/publish`    | 触发构建并发布；body 是 `{entry_point, files}` 源码快照（v1） |
| `DELETE` | `/box/:id`            | 归档 Box |

`POST /box/:id/publish` 同步返回新 BundleRef；下个迭代会改成异步：返回
202 + 通过 §3 的 `box_version_update` 事件推送结果。

### 2.3 Pairing（device JWT）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/pair`                | provider 申请短码绑定 box；body：`{box_id, ttl_sec ≤ 900}` |
| `POST` | `/pair/:code/claim`    | connector 兑换短码；返回 `{box_id, bundle, namespace_token}` |

短码字符集：Crockford-base32 去掉 `0/1/I/O`，长度 8。一次性消费。

### 2.4 AI（device JWT，**待实装**）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/ai/turn` | 提交一轮对话；返回 SSE 流（`ChatTurnEvent` 帧） |

frame 形态见 §3.2。

### 2.5 Artifact（公开，无 auth）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/_artifacts/:hash` | 直接拉取 bundle 字节（仅 LocalFS 后端；生产换 CDN 签名 URL） |

## 3. WebSocket 事件类型

`/ws` 在原 `Message` 之上，新增以下面向 Stage / Box 的事件类型。所有事件
共享同一连接，靠 `kind` 字段区分。

### 3.1 `box_version_update`

provider 发布新版本时，relay 向所有当前订阅了该 box 的 connector 推送：

```json
{
  "kind": "box_version_update",
  "box_id": "box_abc...",
  "bundle": {
    "version": "v1714000000-abc...",
    "uri": "https://relay.example/_artifacts/<hash>",
    "content_hash": "sha256:...",
    "size_bytes": 12345,
    "build_state": "succeeded",
    "built_at": 1714000000000,
    "expires_at": 1714001800000
  }
}
```

connector 收到后，比对 `content_hash`，不同则触发 Stage 重新加载。

### 3.2 `chat_turn_event`

`/ai/turn` 返回的流帧。每帧只填 `token / tool_call / tool_res / finished /
error` 之一：

```json
{ "kind": "chat_turn_event", "turn_id": "...", "token":    { "text": "Hello" } }
{ "kind": "chat_turn_event", "turn_id": "...", "tool_call":{ "call_id": "1", "name": "fs_write", "args_json": "{...}" } }
{ "kind": "chat_turn_event", "turn_id": "...", "tool_res": { "call_id": "1", "result_json": "{...}" } }
{ "kind": "chat_turn_event", "turn_id": "...", "finished": { "stop_reason": "end_turn" } }
```

## 4. 流程

### 4.1 Provider 编辑 → 发布

1. provider 在 Chat tab 输入需求 → `POST /ai/turn`（流式）
2. AI agent 在 relay 内通过 tool calls（`fs_read / fs_write / build_bundle / publish_box`）改源码、跑构建
3. AI 终态调用 `publish_box`：内部走 `box.Service.BuildAndPublish`
   1. sandbox 构建（v1 用 `LocalStub`；后续替换为 Metro / Modal / Firecracker）
   2. artifact 内容寻址写入（v1 LocalFS；生产 TOS / R2 / S3）
   3. `app_bundles` 入库 + `app_boxes.current_version` 更新
   4. 向所有已订阅 connector 推 `box_version_update`

### 4.2 Connector 配对 → 加载 Stage

1. provider 在 Profile tab 对一个 published Box 点 Pair → 后台 `POST /pair`，返回短码
2. connector 扫码或手输 → 进入 `/pair/:code` 路由 → 后台 `POST /pair/:code/claim`
3. relay 用 Redis `GETDEL` 原子兑换短码（一次性），返回 `{box_id, bundle, namespace_token}`
4. connector 把 active box 设为返回值 → 路由跳转到 Stage tab → `StageRuntime` 加载 `bundle.uri`
5. connector 通过 `/ws` 订阅 `box_version_update`（订阅消息形态待定，见 §5）

### 4.3 断线补偿

数据消息（§1）的 `last_seq` 补偿与原协议相同。Stage 不依赖 seq；
重连后通过 `GET /box/:id` 拉一次最新 `current` 即可。

## 5. 待定 / 下一片

- §3.2 chat_turn_event 的 SSE 编码（`event:` 标签 vs JSON 行流）
- §4.2 步骤 5：connector 订阅 `box_version_update` 的 WS 子协议消息
- `namespace_token`：为 connector 颁发 box-scoped JWT，使其 WS 订阅可被 relay 校验
- artifact URL 的真签名（HMAC + expires），替代 LocalFS 的明文 URL

## 6. 不变量

- 字段名 `snake_case`；枚举短名小写
- `seq` 仅由 relay 分配，端在数据流上禁止伪造
- `box_id` / `version` / `short_code` 全部由 relay 生成
- bundle 字节内容不可变；任何修改产生新 `version` + 新 `content_hash`
- artifact `uri` 永远是短期可访问；客户端**禁止**长期缓存
- 跨 namespace 的消息 / Box 不得互相可见；relay 在每条入口检查
