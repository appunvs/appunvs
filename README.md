# appunvs

AI-driven cross-platform app generator. Provider talks to an AI in Chat,
publishes a React Native bundle, and a Connector device pairs by short
code / QR to load the bundle inside an isolated Stage.

## 结构

```
appunvs/
├── app/                  # Expo monorepo (browser + desktop frontend + mobile)
│   └── README.md         # tabs, Stage runtime contract, build scripts
├── desktop/
│   └── src-tauri/        # Tauri 2 native shell wrapping the Expo Web export
├── relay/                # Go service (auth · ws · ai · sandbox · artifact · pairing · billing)
├── shared/proto/         # canonical wire schema (appunvs.proto)
└── docs/
    ├── architecture.md   # 组件拓扑 · 数据流 · Stage 契约 · 实施次序
    ├── conventions.md    # 术语 · 状态机 · 命名约定
    ├── protocol.md       # HTTP + WebSocket 接口
    └── auth.md           # 鉴权细节
```

## 角色

- **Provider** — 通过 AI 对话编辑代码，发布 RN bundle；拥有 Box 的设备
- **Connector** — 扫码挂载已发布的 Box，在 Stage 内运行；只读 + 状态写回
- **Stage** — 端内的隔离 JS runtime（Web 用 iframe，Native 当前用
  WebView，下一片换独立 Hermes）；UI tab 同名

## 三端共一栈

`app/` 是单一 Expo + React Native 工程，三端共用：

| Surface  | Build |
| --- | --- |
| Browser  | `npm --prefix app run web:dev` / `web:export` |
| Desktop  | `cd desktop/src-tauri && cargo tauri dev` (套住 `app/dist`) |
| Mobile   | `npx expo run:ios` / `run:android`（dev client，OTA 走 `expo-updates`） |

## 共享约定

所有端与 relay 必须遵循 [docs/conventions.md](docs/conventions.md) 的术语
与 [docs/protocol.md](docs/protocol.md) 的消息格式。任何跨端改动先更新这
两份文档与 `shared/proto/appunvs.proto`，再落到具体实现。
