# appunvs

> AI 驱动的移动应用生成 SaaS。用户对 AI 说一句话，AI 写 React Native 代码，
> 一键发布，扫码在另一台设备里原生运行。

面向中国市场。后端全栈 Go + SQLite + Redis；前端 **iOS + Android 原生**
（Swift / SwiftUI · Kotlin / Compose），AI 生成的 bundle 在设备内嵌的
独立 Hermes 沙箱里跑；AI 层走 DeepSeek（OpenAI 兼容协议，可切 火山 Ark / 阿里百炼 / 智谱 / Kimi）。

v1 只做移动端（iOS + Android）；桌面 / 浏览器留到 v2 看市场反馈。

## 角色与核心概念

| 术语 | 含义 |
| --- | --- |
| **Provider** | 通过 Chat 与 AI 协作编辑代码的设备；拥有 Box |
| **Connector** | 扫码挂载已发布 Box、在 Stage 内运行其 bundle 的设备 |
| **Box** | provider 创建的项目单位；底层是一份 bare git repo |
| **Bundle** | Box 的不可变构建产物（`(box_id, version)` 唯一，内容寻址 sha256） |
| **Stage** | 端内加载并运行 bundle 的隔离容器；移动端 = 独立 Hermes runtime |
| **Pair** | 一次性短码，把一个 connector 设备绑到某个已发布 Box |
| **SubRuntime** | host app 内嵌的第二个 `jsi::Runtime`；专门跑 AI bundle，与 host UI 完全隔离 |

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
        ┌───────────────┼───────────────┐                   │
        │                               │                   │
   ┌─────────────────┐         ┌─────────────────┐          │
   │  iOS host app   │         │ Android host app│          │
   │  Swift+SwiftUI  │         │ Kotlin+Compose  │          │
   │                 │         │                 │          │
   │  ┌───────────┐  │         │  ┌───────────┐  │          │
   │  │SubRuntime │◀─┼─────────┼──│SubRuntime │◀─┼──────────┘
   │  │ Hermes JS │  │         │  │ Hermes JS │  │       (load AI bundle)
   │  └───────────┘  │         │  └───────────┘  │
   └─────────────────┘         └─────────────────┘
```

host UI（Chat / Stage / Profile）用平台原生组件（SwiftUI / Compose）；
AI 生成的 RN bundle 在 host 内嵌的 SubRuntime（独立 `jsi::Runtime`）里运行，
与 host JS heap 完全隔离。

## 仓库布局

```
appunvs/
├── appunvs/                     # host 应用（上架 App Store / Play）
│   ├── ios/                     # Swift + SwiftUI Xcode 工程（XcodeGen）
│   └── android/                 # Kotlin + Compose Gradle 工程
├── runtime/                     # RN 0.85 SDK 工程；产 RuntimeSDK.xcframework + runtime.aar
│   ├── README.md                # 三出口 / 三消费者表
│   ├── MODULES.md               # AI bundle 可用的 native module 白名单
│   ├── ARCHITECTURE.md          # 三路径对比（WebView / 整 app reload / Sub-Hermes）
│   ├── version.json             # runtime SDK 版本
│   ├── src/                     # host JS 入口 + HostBridge 类型 + dev 测试 harness
│   ├── ios/                     # RN init 自带的 iOS 子工程
│   ├── android/                 # RN init 自带的 Android 子工程
│   ├── packaging/               # build-ios.sh / build-android.sh → SDK artifacts
│   └── sandbox/                 # relay 编译 AI bundle 用的 docker 镜像 + metro 白名单
├── relay/                       # Go 后端
│   └── internal/
│       ├── auth/                # JWT（session / device / namespace 三种）
│       ├── handler/             # /box /pair /ai/turn /ws /billing /schema
│       ├── hub/                 # WebSocket 连接注册与广播
│       ├── stream/              # Redis Stream（24h 补偿窗口）
│       ├── sequencer/           # 全局 seq 分配
│       ├── store/               # SQLite（users · boxes · bundles · ai_turns）
│       ├── box/                 # build + publish 用例编排
│       ├── workspace/           # 每 Box 一个 bare git repo（go-git）
│       ├── pairing/             # Redis 短码
│       ├── artifact/            # bundle 存储接口（LocalFS / TOS / R2 / S3）
│       ├── sandbox/             # 构建接口（LocalStub / Metro / Modal）
│       ├── ai/                  # OpenAI 兼容 agent loop + 工具集合
│       ├── billing/             # 订阅 + 额度门
│       └── pb/                  # wire schema 的 Go 镜像
├── shared/proto/                # canonical wire schema（按模块拆分）
└── docs/
    ├── architecture.md
    ├── conventions.md
    ├── protocol.md
    ├── auth.md
    └── providers.md
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
  backend:   deepseek
  api_key:   ${DEEPSEEK_API_KEY}
  max_iters: 10
  max_tokens: 8000
```

## 技术栈

| 层 | 选型 |
| --- | --- |
| iOS host | Swift 5.10 · SwiftUI · iOS 16+ · XcodeGen |
| Android host | Kotlin 2.0 · Jetpack Compose · minSdk 24 / targetSdk 35 |
| AI bundle 运行时 | RN 0.85 · Hermes（独立 sub-runtime per Stage） |
| 后端语言 | Go 1.22+ |
| Web / WS | Gin · gorilla/websocket |
| 持久化 | SQLite（modernc，无 CGO）+ Redis 7+ |
| Git 工作区 | go-git（bare repo per box） |
| AI 提供商 | DeepSeek / 火山 Ark / 阿里百炼 / 智谱 / Moonshot |
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

**前置条件**：
- relay：Go 1.22+、Redis（本机或 Docker）
- iOS：macOS 14+、Xcode 16+、`brew install xcodegen`
- Android：JDK 17+、Android Studio Hedgehog+、Android SDK 35

```bash
# 1. relay
cd relay
redis-server &                             # or: docker run -p 6379:6379 redis:7
go run ./cmd/server                        # :8080

# 2. iOS host
cd appunvs/ios
xcodegen generate                          # 产出 Runtime.xcodeproj
open Runtime.xcodeproj                     # ⌘R 跑模拟器

# 3. Android host
cd appunvs/android
gradle wrapper --gradle-version 8.10       # 一次性
./gradlew installDebug                     # 装到连接的设备/模拟器

# 4. AI 真实模型
export APPUNVS_AI_BACKEND=deepseek
export APPUNVS_AI_API_KEY=sk-...
# 不设上面两个变量时默认走 stub 引擎（回声模式，方便协议调试）
```

## 当前实装状态

| 模块 | 状态 |
| --- | --- |
| 账号 / 设备 / JWT（三种 token 类型） | ✅ 生产就绪 |
| /box · /pair · /billing · /ws · /schema HTTP 路由 | ✅ 生产就绪 |
| Box + git workspace + Bundle 发布管线 | ✅ 生产就绪 |
| `/ai/turn` SSE + DeepSeek agent loop + 4 个工具 | ✅ 已合并 |
| `ai_turns` 持久化 + 多轮对话重放 | ✅ 已合并 |
| 5 家 LLM provider registry | ✅ 已合并 |
| `appunvs/ios` host 壳（XcodeGen + SwiftUI + 网络 + 邮箱密码登录） | ✅ |
| `appunvs/android` host 壳（Gradle + Compose + Retrofit + 邮箱密码登录） | ✅ |
| `runtime/` RN 0.85 SDK 工程骨架（src/ + packaging/ + sandbox/） | ✅ 本 PR |
| Native CI workflow（macos + ubuntu） | ✅ 本 PR |
| **三屏完整 UI 移植**（Bubble / ToolCall / BoxSwitcher 等） | 🚧 PR C |
| **网络客户端 + 状态层**（HTTP + AsyncStorage 等价 + Box 缓存） | 🚧 PR C |
| **SubRuntime native module**（独立 Hermes runtime） | 🚧 PR D |
| **HostBridge / 模块白名单** | 🚧 PR D |
| **sandbox.Builder 真 Metro 构建** | 🚧 |
| **artifact.Store 火山 TOS 后端** | 🚧 |
| WS `box_version_update` fanout | 🚧 |
| AI token 预算门 | 🚧 |

## 实施次序

- [x] 共享 proto + 文档约定
- [x] relay 骨架（auth / ws / schema / billing）
- [x] Box / Bundle / Pair 管线 + namespace_token
- [x] git workspace + ai_turns + AI agent loop + /ai/turn SSE
- [x] 5 家 LLM provider registry
- [x] proto 按模块拆分
- [x] **iOS + Android 原生工程骨架**（本 PR）
- [ ] **PR C：port UI 到 native**（Swift/SwiftUI + Kotlin/Compose 各一份组件库 + 三屏完整实现 + 网络层）
- [ ] **PR D：SubRuntime native module**（独立 Hermes 运行时 + Fabric surface 挂载 + bundle 加载/卸载）
- [ ] HostBridge 白名单 + AI bundle 工具暴露
- [ ] 真 Metro 构建（替代 LocalStub）
- [ ] 火山引擎 TOS 适配（替代 LocalFS）
- [ ] AI 预算门 + 按档位限流
- [ ] WS 级 box_version_update fanout

## 文档

- [docs/architecture.md](docs/architecture.md) — 组件拓扑、三条数据流、Stage 契约、实施次序
- [docs/protocol.md](docs/protocol.md) — HTTP / WebSocket 接口与事件
- [docs/conventions.md](docs/conventions.md) — 术语、状态机、命名约定、token 类型
- [docs/auth.md](docs/auth.md) — 账号与设备鉴权细节
- [docs/providers.md](docs/providers.md) — AI 供应商清单 / 配置 / 新增步骤
- [runtime/README.md](runtime/README.md) — 原生 host 工程 + 三出口结构
- [runtime/MODULES.md](runtime/MODULES.md) — AI bundle 可用的 native module 白名单

## License

MIT（见 [LICENSE](LICENSE)）。随项目进入商业化阶段可能调整；当前状态下任何人
都可以 fork / 自建 / 商用，不需要告知。

## 共享约定

所有端与 relay 必须遵循 [docs/conventions.md](docs/conventions.md) 的术语和
[docs/protocol.md](docs/protocol.md) 的消息格式。任何跨端改动先更新这两份文
档与 `shared/proto/*.proto`（按模块拆分），再落到具体实现。
