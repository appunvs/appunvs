# appunvs

Cross-platform data sync skeleton built around a Creator-operated relay.

## 结构

```
appunvs/
├── docs/
│   ├── conventions.md   # 领域语言约定（术语、角色、平台）
│   ├── protocol.md      # Relay 与端之间的消息协议
│   └── architecture.md  # 四个实现任务与实施顺序
├── shared/
│   └── proto/           # protobuf schema（wire 类型单一源）
├── relay/               # Task 1 — Go relay (TODO)
├── mobile/              # Task 2 — Flutter provider (TODO)
├── browser/             # Task 3 — SvelteKit provider + connector (TODO)
└── desktop/             # Task 4 — Tauri provider + connector (TODO)
```

## 共享约定

所有端与 relay 必须遵循 [docs/conventions.md](docs/conventions.md) 的术语定义与
[docs/protocol.md](docs/protocol.md) 的消息格式。任何跨端改动先更新这两份文档，再
落到具体实现。

## 实施顺序

见 [docs/architecture.md](docs/architecture.md)。当前分支只建立共享约定，四个实现任务分别在独立分支推进。
