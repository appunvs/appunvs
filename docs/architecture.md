# 架构与实施计划

appunvs 是一个 Creator 运营的、AI 驱动的跨端应用生成 SaaS：

- **Provider** 用 AI 对话编辑代码，发布出一份 RN bundle
- **Connector** 通过扫码 / 短码挂载到该 bundle，在 Stage 内运行

三端（browser / desktop / mobile）共享一份 React Native 代码；relay 承担
鉴权、AI agent、构建、artifact 托管、配对。

## 组件拓扑

```
                      ┌──────────────────────────────────────────────┐
                      │               Relay (Go 1.22+)               │
                      │  auth · /ws · sequencer · ai · sandbox       │
                      │  artifact · pairing · billing · stripe       │
                      │      Redis  +  SQLite  +  Object Store       │
                      └─────────┬─────────────────────┬──────────────┘
                                │                     │
                  HTTPS + WSS   │                     │   Bundle CDN (artifact)
                                │                     │
        ┌───────────────────────┼─────────────────────┼─────────────────────┐
        │                       │                     │                     │
   ┌───────────┐           ┌───────────┐         ┌───────────┐         ┌──────────┐
   │  browser  │           │  desktop  │         │  mobile   │   ...   │  others  │
   │ Expo Web  │           │  Tauri 2  │         │ Expo Dev  │         │          │
   │ (RN-Web)  │           │  + Rust   │         │  Client   │         │          │
   └───────────┘           └───────────┘         └───────────┘         └──────────┘

      ┌──── single React Native + expo-router codebase (app/) ────┐
      │ Tabs: Chat (AI) · Stage (isolated runtime) · Profile      │
      └───────────────────────────────────────────────────────────┘
```

## 仓库布局

```
appunvs/
├── app/                  # Expo monorepo (browser + desktop frontend + mobile)
│   └── …                 # see app/README.md
├── desktop/
│   └── src-tauri/        # Tauri 2 native shell, points frontendDist at app/dist
├── relay/                # Go service
│   └── internal/
│       ├── auth/         # JWT signer, /auth/* handlers
│       ├── handler/      # /box, /pair, /keys, /billing, /ws, /schema
│       ├── hub/          # WebSocket connection registry
│       ├── stream/       # Redis Stream for /ws catchup
│       ├── sequencer/    # global seq INCR
│       ├── store/        # SQLite (users, devices, boxes, bundles, …)
│       ├── box/          # box.Service: build + persist + fanout
│       ├── pairing/      # short-code Issue/Claim via Redis SET-NX / GETDEL
│       ├── artifact/     # bundle storage (LocalFS today; TOS/S3/R2 next)
│       ├── sandbox/      # build orchestrator (LocalStub today; Metro next)
│       ├── ai/           # Anthropic-style agent loop (StubEngine today)
│       ├── billing/      # Stripe checkout + quota gate
│       └── pb/           # Go mirrors of shared/proto (drift-tested)
├── shared/proto/         # canonical wire schema (appunvs.proto)
└── docs/
    ├── architecture.md   # ← this file
    ├── conventions.md    # 术语与状态机
    ├── protocol.md       # HTTP + WebSocket 接口 / 流程
    └── auth.md           # 鉴权细节
```

## 核心数据流

### 编辑流（provider 上的 Chat tab）

```
provider/Chat ──POST /ai/turn──► relay/ai (agent loop)
                                    │
                                    ▼
                          tool: fs_read / fs_write
                                    │
                                    ▼
                          tool: build_bundle ──► sandbox.Builder
                                                    │
                                                    ▼
                                              artifact.Store.Put (sha256)
                                                    │
                                                    ▼
                                              app_bundles (build_state=succeeded)
```

### 发布流

```
provider/Chat or Profile ──POST /box/:id/publish──► box.Service.BuildAndPublish
                                                       │
                                                       ▼
                                          (re-run sandbox + artifact + DB)
                                                       │
                                                       ▼
                                  app_boxes.current_version = vNNN
                                  app_boxes.state = published
                                                       │
                                                       ▼
                                  hub.Broadcast(BoxVersionUpdate)
                                                       │
                                                       ▼
                                       all subscribed connectors
                                       reload Stage if hash changed
```

### 配对流

```
provider/Profile ──POST /pair──► relay (Redis SET NX, TTL≤15m)
                       │
                       ▼
                 short_code "ABCD2345" + QR

connector/Camera scan ──/pair/ABCD2345──► relay (Redis GETDEL)
                                              │
                                              ▼
                                     {box_id, bundle, namespace_token}
                                              │
                                              ▼
                                     active box ← payload
                                     route → /(tabs)/stage
```

## Stage 沙箱契约

> 加载到 Stage 的 bundle **不得**触达宿主 app 的状态、token、MMKV、文件系统。

实现：

| 端 | 当前 | 目标 |
| --- | --- | --- |
| Web      | `<iframe sandbox="allow-scripts">`，禁 `allow-same-origin` | 不变 |
| Native   | `react-native-webview`，禁 cookies / DOM storage | 自研 native module 拉起独立 Hermes 实例，无桥接模块 |

WebView 在 v1 是临时退化，让 pair → fetch → render 整条链路先跑通。
真正的 RN-渲染 Stage 需要一个隔离的 JS runtime——这是单独一片工作，
契约（`StageRuntimeProps`）在切换时不变。

## Sandbox 抽象

`sandbox.Builder` 接口固定，便于阶梯式替换：

| 阶段 | 实现 | 备注 |
| --- | --- | --- |
| v1（当前） | `LocalStub`：把源文件拼接成假 bundle | 跑通 publish + artifact + DB 路径 |
| v2 | 本机 Docker 跑 Metro | 真 RN bundle；单机吞吐 |
| v3 | Modal / E2B 托管 sandbox | 弹性伸缩；按用调度 |
| v4 | 自建 Firecracker 池 | 成本下降；启动 < 1s |

## Artifact 抽象

`artifact.Store` 接口固定：

| 阶段 | 实现 |
| --- | --- |
| v1（当前） | `LocalFS`，relay 自己 `r.Static("/_artifacts", root)` 服务 |
| v2 | 火山引擎 TOS（S3 兼容）+ CDN |
| v3 | 多云：TOS（中国）+ Cloudflare R2（海外） |

## AI agent

`ai.Engine` 接口固定，初版是 `StubEngine`（回声）。生产实现：

- **直连 Anthropic Messages API**，Claude Opus 4.7 / Sonnet 4.6，server-side
  tool_use loop，工具集合 = `fs_read / fs_write / list_files / build_bundle /
  publish_box / run_test`
- 后续可前置一个 router，对补全类调用走 fast-apply 小模型，对
  reasoning 类调用走 Opus

## 实施次序（追加在已完成之后）

1. **Sandbox 真实化**：把 `LocalStub` 换成 Metro 子进程，输出真 RN bundle
2. **Artifact 切 TOS**：`artifact.Store` 加一个 TOS 后端，`config.artifact.backend`
   切到 `tos`
3. **AI agent 实装**：`ai.StubEngine` → `ai.AnthropicEngine`，工具集合的 5 个
   tool 各落实
4. **`/ai/turn` 路由 + SSE 编码**
5. **`box_version_update` 的 WS 子协议**：connector 订阅 + relay fanout
6. **Stage native isolated Hermes**：替代 WebView fallback
7. **Expo Updates 自建 OTA**（runner 升级，不走 EAS 付费）

## 共享不变量

不论哪一端 / 哪一阶段实现，下列约束必须成立：

1. 所有线上消息遵循 [docs/protocol.md](protocol.md) 与 `shared/proto/appunvs.proto`
2. 术语、状态机、枚举值遵循 [docs/conventions.md](conventions.md)
3. `seq` 只能由 relay 分配；`box_id` / `version` / `short_code` 同样
4. relay 不感知用户业务 schema；data message 的 `payload` 对 relay 透明
5. Stage bundle 与宿主完全隔离（§"Stage 沙箱契约"）
6. namespace 默认等于 `user_id`；跨 namespace 数据 / Box 不得泄漏
7. artifact 内容不可变；版本是唯一可变量
