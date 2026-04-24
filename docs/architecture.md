# 架构与实施计划

appunvs 是 Creator 运营的、AI 驱动的跨端应用生成 SaaS：

- **Provider** 用 AI 对话编辑代码，发布出一份 RN bundle
- **Connector** 扫码 / 短码挂载到该 bundle，在 Stage 内运行它
- **End-user data**（AI 生成的 app 里用户产生的 records）通过 relay 原有的
  namespace-scoped Message 协议在所有 connector 之间实时同步

三端（browser / desktop / mobile）共享一份 React Native 代码；relay 承担
鉴权、AI agent、构建、artifact 托管、配对、**Box 工作区（git）** 与 **namespace 数据同步**。

## 组件拓扑

```
                ┌────────────────────────── Relay (Go 1.22+) ──────────────────────────┐
                │                                                                       │
                │  auth · /ws · sequencer · stream · ai · sandbox · artifact · pairing  │
                │  billing · box · workspace(git)                                       │
                │                                                                       │
                │  SQLite: users · devices · schemas · boxes · bundles · ai_turns · ... │
                │  Redis:  seq · Stream(records, 24h) · pair codes                      │
                │  Disk:   per-box bare git repos (workspace/)                          │
                │  Object: bundle blobs (LocalFS / TOS / R2), content-addressed         │
                └──────────┬───────────────────────────────────────┬────────────────────┘
                           │  /ws (Message envelope: upsert/delete)│
                           │       provider + connector + both     │
            ┌──────────────┼───────────────┬───────────────────────┴──────┐
            │              │               │                              │
        provider       connector       connector                          …
        (Chat)         (Stage)         (Stage)
            │              │               │
            │         runs AI-generated bundle
            │         bundle SDK ── namespace_token ── /ws ── namespace 数据
            │
     device 本地 cache（MMKV / expo-sqlite）
```

## 仓库布局

```
appunvs/
├── app/                  # Expo monorepo (browser + desktop frontend + mobile)
├── desktop/
│   └── src-tauri/        # Tauri 2 native shell wrapping Expo Web export
├── relay/
│   └── internal/
│       ├── auth/         # JWT signer; session / device / namespace flavors
│       ├── handler/      # /box, /pair, /keys, /billing, /ws, /schema
│       ├── hub/          # WebSocket connection registry
│       ├── stream/       # Redis Stream for /ws catchup
│       ├── sequencer/    # global seq INCR
│       ├── store/        # SQLite (users, devices, schemas, boxes, bundles, ai_turns)
│       ├── box/          # box.Service: workspace commit → build → artifact → publish
│       ├── workspace/    # per-Box bare git repo (go-git); AI fs tools write here
│       ├── pairing/      # Redis SET-NX / GETDEL short codes
│       ├── artifact/     # bundle storage (LocalFS today; TOS/S3/R2 swap-in)
│       ├── sandbox/      # build orchestrator (LocalStub today; Metro/Modal next)
│       ├── ai/           # Anthropic-style agent loop (StubEngine today)
│       ├── billing/      # Stripe checkout + quota gate
│       └── pb/           # Go mirrors of shared/proto (drift-tested)
├── shared/proto/         # canonical wire schema (appunvs.proto)
└── docs/                 # ← this doc, protocol.md, conventions.md, auth.md
```

## 核心对象

| 对象 | 住哪 | 特性 |
| --- | --- | --- |
| **Box**       | `app_boxes` 表 | provider 拥有的项目单位；`state = draft | published | archived` |
| **Workspace** | relay 磁盘的 bare git repo（`workspace/<box_id>/`） | AI 每轮 `fs_write` → 一个 commit；build 读 HEAD snapshot |
| **Bundle**    | `app_bundles` 表 + 对象存储 | `(box_id, version)` 唯一；内容寻址 sha256；immutable |
| **AI Turn**   | `ai_turns` 表 | 一整轮对话（user text + tool calls + final），JSON 原样存 |
| **Pair code** | Redis `pair:<code>` | 8 位 Crockford-alnum，TTL ≤ 15 分钟，一次性 GETDEL |
| **Records（AI 生成 app 的运行时数据）** | relay SQLite + Redis Stream | 原有 Message 协议，namespace-scoped 广播 |

## 三条独立数据流

### 1. 编辑流（provider Chat）

```
provider Chat ── POST /ai/turn ──► relay/ai (agent loop; Anthropic)
                                       │
                                       ▼ tool: fs_write
                                  workspace.Repo.Commit (git commit on main)
                                       │
                                       ▼ tool: publish_box
                                  box.Service.BuildAndPublish
                                       │
                            ┌──────────┴──────────┐
                            ▼                     ▼
                    sandbox.Build(src)     artifact.Put(bytes)
                    (Metro / LocalStub)    (LocalFS / TOS)
                            │                     │
                            └──────────┬──────────┘
                                       ▼
                            app_bundles  +  app_boxes.current_version
                                       │
                                       ▼
                            ai_turns.Insert（含 tokens_in/out）
```

### 2. 配对流（provider 生成短码 → connector 扫码）

```
provider ── POST /pair ──► Redis SET NX (pair:ABCDEF, TTL≤15m)
                                │
                                ▼  short_code + QR

connector ── POST /pair/ABCDEF/claim ──► relay (Redis GETDEL)
                                            │
                                            ▼
                            { box_id, bundle, namespace_token }
                                            │
                                            ▼
                            connector 加载 bundle.uri 到 Stage
                            bundle SDK 使用 namespace_token 连 /ws
```

### 3. 数据同步流（Stage 内的 AI 生成 app ↔ 多个 connector）

```
Stage bundle
   │
   │  bundle 内调 sync SDK → /ws（带 namespace_token）
   │
   ▼
Message 协议（seq · upsert · delete）── 原有 relay 通道
   │
   ▼
同 namespace 下其他 connector 实时收到广播
断线后重连用 last_seq 从 Redis Stream 补偿（24h 保留）
```

**这条是原 relay 设计的本体**——新增的 Box / Stage / Pair 只是提供了"哪个 bundle 被载入"的前置，数据层本身没变。

## Stage 沙箱契约

> 加载到 Stage 的 bundle **不得**触达宿主 app 的状态、token、MMKV、文件系统。

两层：

1. **运行时隔离**：
   - Web：`<iframe sandbox="allow-scripts">`（禁 `allow-same-origin`）
   - Native（今日）：`react-native-webview` + 禁 cookies / DOM storage
   - Native（目标）：自研 native module 跑独立 Hermes，无桥接模块
2. **数据凭证隔离**：Stage 里 bundle 拿到的 `namespace_token` 是 **box-scoped JWT**（claims: `uid / did / box_id / typ=namespace`），不能访问其他 Box 的数据；也没有宿主 device_token，不能调 `/box` 等管理接口。

## 可插拔抽象

| 接口 | v1 | v2+ |
| --- | --- | --- |
| `sandbox.Builder` | `LocalStub`（拼源文件） | Metro 子进程 → 之后 Modal / 自建 Firecracker |
| `artifact.Store`  | `LocalFS` | 火山引擎 TOS（S3 协议） → 多云 |
| `ai.Engine`       | `StubEngine`（回声） | `AnthropicEngine` + 多模型 router + fast-apply |

## 实施次序（已合入 vs 下一片）

**已合入**（PR #2）：
- `app/` Expo monorepo（Chat / Stage / Profile 三 tab + StageRuntime 契约）
- `relay/internal/box`、`artifact`、`sandbox`、`pairing`、`ai` 接口 + v1 stub
- `relay/internal/workspace`（**新**：go-git per-box bare repo）
- `relay/internal/store/turns`（**新**：ai_turns 表 + 迁移 v7）
- `auth.TokenNamespace` 与 `Signer.IssueNamespace`，`POST /pair/:code/claim` 发放 namespace_token
- `box.Service.BuildAndPublish` 在 workspace 存在时先提交 git，再从 HEAD 构建

**下一片候选**（按依赖排序）：

1. **AnthropicEngine + 工具集合**：`fs_read / fs_write / list_files / build_bundle / publish_box`；绑 `ai_turns` 做持久记忆 + 预算治理
2. **`/ai/turn` SSE 流路由**：工具调用 / token delta / finished 帧通过 SSE 下推
3. **真 Metro 构建**：`sandbox.LocalStub` → Metro 子进程（或 esbuild）
4. **bundle SDK**：Stage 里 bundle 用的最小 `sync`/`schema` 客户端（RN 包），帮 AI 生成的 app 直连 /ws
5. **`box_version_update` 的 WS fanout**（可选，launch 前再判断是否需要）
6. **artifact 切 TOS**（launch 前）
7. **native isolated Hermes**（launch 后判断）
8. **自建 OTA**（launch 后）

## 共享不变量

1. 所有消息遵循 [docs/protocol.md](protocol.md) 与 `shared/proto/appunvs.proto`
2. 术语、状态机、枚举值遵循 [docs/conventions.md](conventions.md)
3. `seq`、`box_id`、`version`、`short_code` 全部由 relay 生成
4. relay 不感知用户业务 schema；Message `payload` 对 relay 透明
5. Stage bundle 与宿主完全隔离（运行时 + 数据凭证两层）
6. **Stage bundle 只能持有 namespace token**；不能持有 device / session token
7. namespace 默认等于 `user_id`；跨 namespace 数据 / Box 不得泄漏
8. artifact 字节不可变；版本是唯一可变量
9. workspace git history 即 AI 编辑审计；revert 直接 `git reset`，不另建 undo
