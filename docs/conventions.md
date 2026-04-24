# 领域语言约定

本文档是所有 appunvs 端（relay / mobile / browser / desktop）共享的术语表。
新增代码、接口、日志字段时，名称须与本表一致。

## 核心术语

| 术语 | 含义 |
| --- | --- |
| **Creator** | 平台方，运营 relay 服务 |
| **Relay** | Creator 提供的中心服务；负责身份认证、操作定序、WebSocket 转发、离线补偿 |
| **Namespace** | 数据隔离单元，通常等于 `user_id`；relay 按 namespace 广播消息 |

Relay 不存储任何业务数据，只作为定序与转发通道（Redis Stream TTL 24h）。

## Platform

端运行的宿主环境。

| 值 | 说明 |
| --- | --- |
| `browser` | 浏览器 |
| `desktop` | 桌面 app |
| `mobile`  | 移动 app |

## Tech Stack

每个 platform 对应的实现技术。

| 值 | 对应 platform |
| --- | --- |
| `svelte`  | browser |
| `tauri`   | desktop |
| `flutter` | mobile |

## Role

端在运行时相对 relay 的角色。

| 值 | 含义 |
| --- | --- |
| `provider`  | 拥有数据，向 relay 推送本地变更 |
| `connector` | 消费数据，订阅 relay 广播并可回传操作 |

规则：

- 同一个设备可**同时**是 provider 和 connector（`role = "both"`）
- Role 可在运行时切换
- 当前阶段 mobile（flutter）只实现 `provider`

## 标识

| 字段 | 说明 |
| --- | --- |
| `device_id` | 设备唯一标识，注册时由端生成并提交 |
| `user_id`   | 用户唯一标识，relay 在注册时返回，同时作为默认 namespace |
| `token`     | JWT（RS256）；端保存后在 WebSocket 握手时携带 |
| `seq`       | 全局递增序号，由 relay 通过 Redis `INCR` 分配 |

## 命名约定

- 字段名使用 `snake_case`（`device_id`、`last_seq`、`updated_at`）
- 枚举值使用小写（`provider`、`browser`、`upsert`）
- 时间戳统一使用毫秒 Unix 时间（`i64`）
