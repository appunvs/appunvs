# 领域语言约定

本文档是 appunvs 共享的术语表。新增代码、接口、日志字段、proto 字段时，
名称必须与本表一致。

## 核心术语

| 术语 | 含义 |
| --- | --- |
| **Creator**   | 平台方，运营 relay 服务 |
| **Relay**     | 中心服务；负责身份认证、操作定序、WebSocket 转发、artifact 托管、AI agent、配对短码 |
| **Namespace** | 数据隔离单元，默认等于 `user_id`；relay 按 namespace 广播消息和限制可见性 |
| **Box**       | provider 创建的、由 AI 编辑产物的项目实体；产物为一份 RN JS bundle |
| **Stage**     | 端内运行 Box bundle 的隔离容器（沙箱）；UI tab 同名 |
| **Bundle**    | 一份不可变的构建产物（`(box_id, version)` 唯一），存储在 artifact 后端 |
| **Pair**      | 一次性短码，把一个 connector 设备绑定到某个已发布 Box |

## Platform

端运行的宿主环境。

| 值 | 说明 |
| --- | --- |
| `browser` | 浏览器 |
| `desktop` | 桌面 app（Tauri 2 包裹 Expo Web 静态导出） |
| `mobile`  | 移动 app（Expo Dev Client） |

## Tech Stack

三端用 **同一份代码**：Expo SDK 53+ / React Native 0.76+ / expo-router /
react-native-web。Desktop 通过 Tauri 2 加壳分发；mobile 通过 Expo Updates
做 OTA。

## Role

端在运行时相对 relay 的角色。

| 值          | 含义 |
| ----------- | --- |
| `provider`  | 拥有 Box；通过 Chat 与 AI agent 协作生成代码并发布 bundle |
| `connector` | 通过 Pair / 扫码挂载到一个 Box；在 Stage 内运行其 bundle |
| `both`      | 同一设备同时承担两种角色（默认情形） |

规则：

- Role 是每条 Message 的字段，可逐消息切换
- Box 的 `provider_device_id` 永远只有一个；其他设备只能以 connector 身份订阅

## 状态机

### PublishState（Box）
`draft` → `published` → `archived`

- `draft`：AI 在编辑；可以触发构建，但不会广播版本更新
- `published`：`current_version` 指向最近一次成功构建；可以发起 Pair
- `archived`：拒绝新的 publish / pair；历史 bundle 仍可查询

### BuildState（Bundle）
`queued` → `running` → `succeeded` | `failed`

- `failed` 写入 `build_log`（截断 4 KiB）
- 只有 `succeeded` 的 bundle 才能成为 `Box.current_version`

### RuntimeKind（Bundle）
v1 仅支持 `rn_bundle`：Metro 输出的 JS 包 + asset 清单。

## 标识

| 字段 | 说明 |
| --- | --- |
| `device_id`  | 设备唯一标识，端在首次启动生成并通过 SecureStore / localStorage 持久化 |
| `user_id`    | 用户唯一标识，relay 在 signup 时返回，同时作为默认 namespace |
| `box_id`     | Box 的唯一标识，relay 生成（`box_<24-hex>`） |
| `version`    | 构建版本，relay 生成（`v<unix>-<12-hex>`） |
| `short_code` | 配对短码，8 位 Crockford-without-0/1/I/O 字符 |
| `seq`        | 全局递增序号，由 relay 通过 Redis `INCR` 分配 |
| `token`      | JWT（RS256）；端保存后在 WebSocket / HTTP 头携带 |

## 命名约定

- 字段名使用 `snake_case`（`device_id`、`last_seq`、`updated_at`、`box_id`）
- 枚举值使用小写短形式（`provider`、`browser`、`published`、`rn_bundle`）
- 时间戳统一使用毫秒 Unix 时间（`int64`）
- 哈希统一使用 `sha256:<hex>` 字符串
