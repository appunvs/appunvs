# Competitive landscape — AI app builders

四款常被提到的「AI builder」产品，加上 appunvs 自身，按**输入 / 构建 / 产物 / 预览**四个维度对比。

| 产品 | 输入 | 生成/构建 | 产物类型 | 预览承载 |
|---|---|---|---|---|
| **Lovable** | 响应式网页里 chat | Modal Firecracker | Vite / React **web** | sandbox URL iframe |
| **v0** | iOS app 里 chat | Vercel Sandbox | Next.js **web** components / pages | sandbox URL WebView |
| **Bolt.new** | 响应式网页里 chat | StackBlitz WebContainer（浏览器内 Node）| Vite **web app**（默认；任意 npm 框架） | 同页 iframe |
| **a0.dev** | builder app 里 chat | 云端 EAS 式 build | 真 **RN bundle** | 自家 tester app（替代 Expo Go） |
| **appunvs** | 移动 app 里 chat（Boxes tab → drill-down）| relay 调 sandbox（v0：本地 Docker 跑 Metro；后续 ECI / Modal / Firecracker）| 真 **RN bundle**（Hermes-ready）| 同 app 内 Stage tab 的 `RuntimeView`（per-instance Hermes 隔离） |

## 阵营划分

按**产物类型**分两派：

- **Web 派** —— Lovable / v0 / Bolt.new
- **RN 派** —— a0.dev / appunvs

这条线决定了几乎一切下游选择（UI lib 选型、预览方式、native 能力可达性、bundle size 约束），见 [stage-bundle-ui.md](stage-bundle-ui.md)。

## appunvs 与其他四家的关键差异

- **同 app 内 chat + 预览，没 WebView 也没扫码 / Expo Go 跳转**
  - Lovable / v0 / Bolt 都通过 WebView/iframe 加载 sandbox URL；a0.dev 走"另开一个 tester app + push"。
  - appunvs 的 Stage tab 是宿主原生 SDK（`runtime/`）暴露的 `RuntimeView`，AI bundle 直接挂在里面跑，**不经过 WebView**，也不要求另装一个 app。
- **chat → publish → 自动热重载，原生闭环**
  - chat → publish → relay 推 `/box/events` SSE → host 收事件 → `RuntimeView.loadBundle` 自动重挂，全程无感。
  - 等价于 Lovable / v0 / Bolt 那种 "publish 之后 iframe 自动 reload"，但发生在**原生 app 内，没浏览器环境**。
- **per-Box 隔离 Hermes**
  - 每个 RuntimeView 自带独立 Hermes runtime，跨 bundle 不会泄漏 JS state（reset = destroy）。
  - 对应 Lovable / Bolt 在容器或 WebContainer 里跑独立 web sandbox。
- **AI bundle 与 host 的隔离边界**
  - 桥（`@appunvs/host`）是有限 surface（storage / network / publish / subscribe），AI bundle 拿不到 device token，只能拿 namespace token（见 [auth.md](auth.md)）。
  - 对比 a0.dev 的 tester app，那种模式 AI bundle 等价拥有完整 RN runtime；appunvs 用 metro allowlist + Tier 1 模块白名单两道墙。
- **跨 app 数据同步**
  - appunvs 的 namespace 协议（`/ws`）让同一用户在多个 app 之间共享数据 —— 跨 box 协作。Lovable / v0 / Bolt / a0 都没这层，每个生成 app 自管自家数据。

## 取舍 / 风险

| 维度 | appunvs 选择 | 代价 | 对位的同行做了啥 |
|---|---|---|---|
| 同 app 内预览（vs WebView） | 用户不用切应用，体验最连贯 | 必须打通 RN brownfield 嵌入；版本升级要重新发 host shell | Lovable/v0/Bolt 都靠 iframe 偷懒；a0 靠另装 tester app |
| 真 RN bundle（vs web） | 性能 / 原生组件直接可用 | 不能跑任意 JS / CSS 库；受 Tier 1 allowlist 限制；UI lib 蹭不到 web shadcn 生态 | a0 同路；Lovable/v0/Bolt 直接吃 web 红利 |
| 沙箱在 relay 容器（vs FaaS） | v0 单机部署最低成本能跑 | 规模化要换 ECI / Modal / Firecracker，已经预留 `Sandbox` 抽象 | Lovable→Modal Firecracker；v0→Vercel Sandbox；Bolt→浏览器内 WebContainer |
| 桥 surface 有限（vs 完整 RN runtime） | 安全边界清晰，AI 写不出"逃逸到宿主"的代码 | AI bundle 能力受限，比如想用 camera 现在没接口 | a0 等价完整 RN，权限风险换灵活性 |

## 当下完成度（截至 2026-04）

- 同 app 内 chat：✅ #28、#29 落地（Boxes tab 替换 Chat：#39）
- per-Box Hermes 隔离 RuntimeView：✅ #17（D3.c.{1,2,3}）
- chat 驱动 publish → 自动热重载 stage：✅ #30 + #31
- Tier 1 native modules：✅ #19（D3.d）
- HostBridge 真实现（storage / network / publish）：✅ #20–28
- subscribe（实时协作）：deferred，等具体产品需求
- 真 Metro 构建（取代 LocalStub）：✅ #31
- 多 AI provider（OpenAI-compat + Anthropic）：✅ #34
- Stage bundle UI 库（`@appunvs/ui`）：deferred 到 dogfood 之后；策略见 [stage-bundle-ui.md](stage-bundle-ui.md)
