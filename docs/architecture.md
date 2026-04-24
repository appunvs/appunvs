# 架构与实施计划

本文档索引 appunvs 四个实现任务以及它们之间的依赖关系。每个任务在独立分支推进。

## 组件拓扑

```
            ┌─────────────────────────┐
            │        Relay (Go)       │
            │  JWT · seq · WS · Redis │
            └──────────┬──────────────┘
                       │ WebSocket (JSON 协议, 见 docs/protocol.md)
        ┌──────────────┼──────────────┐
        │              │              │
  ┌───────────┐  ┌───────────┐  ┌───────────┐
  │  browser  │  │  desktop  │  │  mobile   │
  │  svelte   │  │   tauri   │  │  flutter  │
  │ prov+conn │  │ prov+conn │  │ provider  │
  └───────────┘  └───────────┘  └───────────┘
```

数据层：

- `browser` — wa-sqlite（IndexedDB 持久化）
- `desktop` — rusqlite（Rust 侧）
- `mobile`  — drift / SQLite

## Task 清单

| # | 目录 | 技术栈 | 角色 | 说明 |
| --- | --- | --- | --- | --- |
| 1 | `relay/`   | Go 1.22 + Gin + gorilla/websocket + Redis + JWT | —          | 中心服务 |
| 2 | `mobile/`  | Flutter 3.x + drift                             | provider   | 本地 SQLite，推送变更 |
| 3 | `browser/` | SvelteKit + wa-sqlite                           | prov+conn  | 纯 CSR |
| 4 | `desktop/` | Tauri 2 + rusqlite + tokio-tungstenite          | prov+conn  | 前端复用 browser UI，数据层在 Rust |

各任务详细需求见原始任务说明；本分支只落约定文档。

## 实施顺序

```
1. relay     → 跑通 WebSocket + 定序 + 广播
2. mobile    → Flutter provider 连上 relay，推送变更
3. browser   → Svelte provider + connector，验证跨端同步
4. desktop   → Tauri，复用 browser UI，Rust 数据层
```

每个任务完成后的验收：两个设备，一端写入，另一端实时收到变更。

## 共享不变量

无论在哪一端实现，下列约束必须成立：

1. 所有消息遵循 [docs/protocol.md](protocol.md) 的字段与流程
2. 术语与枚举值遵循 [docs/conventions.md](conventions.md)
3. `seq` 只能由 relay 分配；端仅用作本地顺序追踪
4. Relay 不感知业务 schema，`payload` 对 relay 透明
5. 断线重连必须携带 `last_seq`，以触发离线补偿
6. namespace 默认等于 `user_id`；跨 namespace 的消息不得泄漏
