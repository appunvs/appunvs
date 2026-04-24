# 消息协议

所有端与 relay 之间使用统一 JSON 消息格式。术语定义见
[conventions.md](conventions.md)。

**Schema 源**：[`shared/proto/appunvs.proto`](../shared/proto/appunvs.proto)。
所有语言的 wire 类型从 proto 生成，严禁手写。JSON 线上格式使用 canonical
protojson（启用 `UseProtoNames`），字段名保持 `snake_case`；枚举序列化为
短名小写（`ROLE_PROVIDER → "provider"`）。
详见 [`shared/proto/README.md`](../shared/proto/README.md)。

## 消息结构

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

### 字段

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `seq`       | `i64`          | 全局递增序号，由 relay 分配；端发送时可省略 |
| `device_id` | `string`       | 设备 ID |
| `user_id`   | `string`       | 用户 ID |
| `namespace` | `string`       | 数据隔离单元，通常等于 `user_id` |
| `role`      | `provider \| connector` | 发送方当前角色 |
| `op`        | `upsert \| delete`       | 操作类型 |
| `table`     | `string`       | 业务表名（如 `records`） |
| `payload`   | `object`       | 业务数据；`upsert` 必含 `id`，`delete` 只需 `id` |
| `ts`        | `i64`          | 发送方本地毫秒时间戳 |

## HTTP 接口

### 设备注册

```
POST /auth/register
Content-Type: application/json

{ "device_id": "xxx", "platform": "browser|desktop|mobile" }
```

**响应**

```json
{ "token": "<JWT RS256>", "user_id": "xxx" }
```

### 健康检查

```
GET /health → 200 OK
```

## WebSocket 握手

```
GET /ws?token=<JWT>&last_seq=<number>
```

- relay 验证 JWT 并解析 `user_id`
- 将连接注册到对应 `namespace`
- 若携带 `last_seq`，从 Redis Stream 补发所有 `seq > last_seq` 的消息
- 心跳：30 秒 ping/pong，超时断开

## 消息流

### Provider → Relay → Namespace

1. provider 本地写入后，构造消息发送给 relay（不含 `seq`）
2. relay 通过 Redis `INCR` 分配全局 `seq`
3. 写入 Redis Stream（TTL 24h）
4. 广播给同 `namespace` 下所有在线设备（含发送者自己，用于确认 `seq`）

### Connector → Relay → Provider

1. connector 构造 `op = upsert | delete` 消息发送给 relay
2. relay 将消息转发给同 `namespace` 的 **provider**
3. provider 执行本地写入后，按照 "Provider → Relay" 流程推送变更（这一次广播让所有端收敛）

### 断线补偿

1. 端重连时带上本地最大 `seq` 作为 `last_seq`
2. relay 从 Redis Stream 补发缺失消息
3. 端按 `seq` 顺序应用，若检测到间隙应断开重连或主动请求补偿

## 约束

- 端**必须**按 `seq` 顺序处理 provider 广播；遇到不连续时不得直接跳过
- `payload` 内容对 relay 透明，relay 不做 schema 校验
- relay 不重写任何字段，只追加 `seq`（若端未提供）
