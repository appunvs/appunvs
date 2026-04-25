# appunvs

> AI 驱动的跨端应用生成 SaaS。用户对 AI 说一句话，AI 写 React Native 代码，
> 一键发布，扫码在任意设备（browser / desktop / mobile）里打开运行。

面向中国市场。后端全栈 Go + SQLite + Redis，前端单栈 Expo + React Native，
AI 层走 DeepSeek（OpenAI 兼容协议，可切 火山 Ark / 阿里百炼 / 智谱 / Kimi）。

## 角色与核心概念

| 术语 | 含义 |
| --- | --- |
| **Provider** | 通过 Chat 与 AI 协作编辑代码的设备；拥有 Box |
| **Connector** | 扫码挂载已发布 Box、在 Stage 内运行其 bundle 的设备 |
| **Box** | provider 创建的项目单位；底层是一份 bare git repo |
| **Bundle** | Box 的不可变构建产物（`(box_id, version)` 唯一，内容寻址 sha256） |
| **Stage** | 端内加载并运行 bundle 的隔离容器（web iframe / native WebView） |
| **Pair** | 一次性短码，把一个 connector 设备绑到某个已发布 Box |

状态机与协议细节见 [docs/conventions.md](docs/conventions.md) · [docs/protocol.md](docs/protocol.md)。

## 组件

```
                ┌──────────────────────── Relay (Go) ───────────────────────┐
                │  auth · /ws · sequencer · stream · ai · sandbox           │
                │  artifact · pairing · billing · box · workspace(git)      │
                │  SQLite · Redis · LocalFS/TOS · per-box bare git repos    │
                └───────┬───────────────────────────────────┬───────────────┘
                        │                                   │
                   /ws + HTTP                         bundle CDN
                        │                                   │
        ┌───────────────┼──────────────┬────────────────────┴───────┐
        │               │              │                            │
    browser          desktop         mobile        ...             ...
    Expo Web       Tauri 2         Expo Dev
                   (wraps Web)      Client

    ── 一份 RN 代码在 app/ 同时编译为三端 ──
```

三端从同一个 `app/` 工程输出，使用 expo-router + react-native-web；desktop 由
Tauri 2 套壳加载 Expo Web 静态导出，mobile 通过 Expo Dev Client 原生打包。

## 仓库布局

```
appunvs/
├── app/                  # Expo monorepo（三端共用前端）
│   └── README.md         # 三端构建脚本 · Stage runtime 契约
├── desktop/
│   └── src-tauri/        # Tauri 2 原生壳（Rust）
├── relay/                # Go 后端
│   └── internal/
│       ├── auth/         # JWT 签发（session / device / namespace 三种）
│       ├── handler/      # /box /pair /ai/turn /ws /billing /schema 路由
│       ├── hub/          # WebSocket 连接注册与广播
│       ├── stream/       # Redis Stream（24h 补偿窗口）
│       ├── sequencer/    # 全局 seq 分配
│       ├── store/        # SQLite（users · boxes · bundles · ai_turns · …）
│       ├── box/          # build + publish 用例编排
│       ├── workspace/    # 每 Box 一个 bare git repo（go-git）
│       ├── pairing/      # Redis 短码（Crockford-base32 · GETDEL）
│       ├── artifact/     # bundle 存储接口（LocalFS 默认；TOS/R2/S3 可插）
│       ├── sandbox/      # 构建接口（LocalStub 默认；Metro/Modal 可插）
│       ├── ai/           # OpenAI 兼容 agent loop + 工具集合
│       ├── billing/      # 订阅 + 额度门
│       └── pb/           # wire schema 的 Go 镜像
├── shared/proto/         # canonical wire schema（按模块拆分：auth/box/pair/sync/ai/…）
└── docs/
    ├── architecture.md   # 组件拓扑 · 数据流 · Stage 契约 · 实施次序
    ├── conventions.md    # 术语 · 状态机 · 命名约定
    ├── protocol.md       # HTTP + WebSocket 接口
    └── auth.md           # 鉴权细节
```

## AI agent

AI 层走 **OpenAI 兼容协议**，内置 5 家国内供应商的 base_url + 默认模型；
`ai.backend` 直接填 provider id 即可切换。完整清单、切换示例、增加新供应商
的步骤见 [docs/providers.md](docs/providers.md)。

| Provider ID | 厂商 | 默认模型 |
| --- | --- | --- |
| `deepseek` | DeepSeek | `deepseek-chat`（可换 `deepseek-reasoner`） |
| `volcengine` | 火山方舟 | 需指定**接入点 id** `ep-...` |
| `moonshot` | Moonshot Kimi | `kimi-k2-turbo-preview` |
| `zhipu` | 智谱 GLM | `glm-4.6` |
| `dashscope` | 阿里百炼（Qwen） | `qwen3-coder-plus` |

```yaml
# config.yaml — 最小配置
ai:
  backend:   deepseek              # stub / deepseek / volcengine / moonshot / zhipu / dashscope
  api_key:   ${DEEPSEEK_API_KEY}
  # base_url / model 留空时用 registry 默认；覆盖只在自建代理或换模型时需要
  max_iters: 10
  max_tokens: 8000
```

**工具集合**（agent 可调用）：

- `fs_read(path)` — 读 workspace HEAD 一个文件
- `fs_write(path, content)` — 写一个文件，一次 git commit
- `list_files()` — 枚举 HEAD 下所有文件
- `publish_box(entry_point?)` — 构建 HEAD、上传 bundle、置为 PUBLISHED

对话协议见 [docs/protocol.md §/ai/turn](docs/protocol.md)。流式 SSE 下发：
`token` / `tool_call` / `tool_result` / `finished` / `error` 帧。

## 技术栈

| 层 | 选型 |
| --- | --- |
| 前端（三端） | Expo SDK 53+ · React Native 0.76+ · expo-router · react-native-web |
| 桌面壳 | Tauri 2（wrap Expo Web 静态导出） |
| 移动端发布 | Expo Dev Client · 自建 OTA（不走 EAS） |
| 后端语言 | Go 1.22+ |
| Web / WS | Gin · gorilla/websocket |
| 持久化 | SQLite（modernc，无 CGO） + Redis 7+ |
| Git 工作区 | go-git（bare repo per box） |
| AI 提供商 | DeepSeek / 火山 Ark / 阿里百炼 / 智谱 / Moonshot（任选，OpenAI 兼容） |
| 对象存储 | LocalFS（dev）· 火山引擎 TOS / Cloudflare R2 / S3（生产） |
| 计费 | Stripe（海外）· 国内支付通道预留接口 |

## 定价

| 档位 | 月价 | 对话额度 | Box 数 | 其它 |
| --- | --- | --- | --- | --- |
| Free | ¥0 | 轻使用 | 1 | 发布带水印 / bundle 24h 过期 |
| **Pro** | **¥39**（¥399/年） | 100 轮/天 | 10 | 永久发布 · 去水印 · 自定义子域名 |
| Max | ¥199 | 500 轮/天 | 不限 | 优先队列 · 团队协作（未来） |
| Enterprise | 面议 | — | — | 私有化部署 · SLA · 自备 key |

实际额度与定价以 `relay/internal/store/billing.go`（`CanonicalPlans`）为准。

## 本地开发

**前置条件**：Go 1.22+、Redis（本机 `redis-server` 或 Docker）、Node 18+、
可选 Tauri build 依赖（GTK/WebKit）

```bash
# 1. relay
cd relay
redis-server &                             # or: docker run -p 6379:6379 redis:7
go run ./cmd/server                        # 默认端口 :8080

# 2. 前端（dev mode）
cd app
npm install
npm run web:dev                            # :8081 → http://localhost:8081

# 3. 桌面壳（可选）
cd desktop/src-tauri
cargo tauri dev

# 4. 开真 AI 需要 DeepSeek key
export APPUNVS_AI_BACKEND=deepseek
export APPUNVS_AI_API_KEY=sk-...
# 不设上面两个变量时默认走 stub 引擎（回声模式，方便 UI / 协议调试）
```

## 当前实装状态

| 模块 | 状态 |
| --- | --- |
| 账号 / 设备 / JWT（三种 token 类型） | ✅ 生产就绪 |
| /box · /pair · /billing · /ws · /schema HTTP 路由 | ✅ 生产就绪 |
| Box + git workspace + Bundle 发布管线 | ✅ 生产就绪 |
| `/ai/turn` SSE + DeepSeek agent loop + 4 个工具 | ✅ 已合并（需 API key） |
| `ai_turns` 持久化 + 多轮对话重放 | ✅ 已合并 |
| Expo 三端骨架（Chat / Stage / Profile） | ✅ 骨架已就位；Chat 需接 /ai/turn |
| **sandbox.Builder** 真实 Metro 构建 | 🚧 `LocalStub` 占位 |
| **artifact.Store** 火山 TOS 后端 | 🚧 `LocalFS` 默认 |
| Stage Native 独立 Hermes runtime | 🚧 `react-native-webview` fallback |
| WS `box_version_update` fanout | 🚧 handler 预留，未接订阅表 |
| AI token 预算门（按 namespace / 按日） | 🚧 未开始 |
| 自建 OTA（Expo Updates 替代） | 🚧 未开始 |
| 微信支付 / 支付宝 | 🚧 未开始（国内接入） |

## 实施次序

- [x] 共享 proto + 文档约定
- [x] relay 骨架（auth / ws / schema / billing）
- [x] 前端单栈收敛到 Expo（替代原 Flutter + SvelteKit + Svelte+Tauri）
- [x] Box / Bundle / Pair 管线 + namespace_token
- [x] git workspace + ai_turns + AI agent loop + /ai/turn SSE
- [ ] **app/ Chat 接入 /ai/turn 流**（下一步）
- [ ] 真 Metro 构建（替代 LocalStub）
- [ ] 火山引擎 TOS 适配（替代 LocalFS）
- [ ] AI 预算门 + 按档位限流
- [ ] WS 级 box_version_update fanout
- [ ] Stage native 独立 Hermes
- [ ] 自建 OTA

## 文档

- [docs/architecture.md](docs/architecture.md) — 组件拓扑、三条数据流、Stage 契约、实施次序
- [docs/protocol.md](docs/protocol.md) — HTTP / WebSocket 接口与事件
- [docs/conventions.md](docs/conventions.md) — 术语、状态机、命名约定、token 类型
- [docs/auth.md](docs/auth.md) — 账号与设备鉴权细节
- [docs/providers.md](docs/providers.md) — AI 供应商清单 / 配置 / 新增步骤
- [app/README.md](app/README.md) — 前端三端构建 · Stage runtime 契约
- [desktop/README.md](desktop/README.md) — Tauri 壳的 dev / build

## License

MIT（见 [LICENSE](LICENSE)）。随项目进入商业化阶段可能调整；当前状态下任何人
都可以 fork / 自建 / 商用，不需要告知。

## 共享约定

所有端与 relay 必须遵循 [docs/conventions.md](docs/conventions.md) 的术语和
[docs/protocol.md](docs/protocol.md) 的消息格式。任何跨端改动先更新这两份文
档与 `shared/proto/*.proto`（按模块拆分），再落到具体实现。
