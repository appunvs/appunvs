# Runtime — Architecture Decision Records

This document archives the long-form "why" behind appunvs 的 AI bundle 运行时设计：
两张表，分别记录 **运行时形态选型** 与 **`runtime/` 三个出口的消费者**。

`runtime/` 本身在仓库里是 native bundle host（独立 Hermes + 已链接 native modules）
的承载目录；它由 host app 链接，向 sandbox 容器供 JS 包，向 AI agent 供类型 +
版本元信息。具体 build 指令未来落到 `runtime/README.md`，这里只承载架构决策。

参考：[docs/architecture.md](../docs/architecture.md) 给出了 relay / app /
shared 三大块的整体拓扑；本页是其中"Stage 沙箱契约"与"native isolated Hermes"
两条的详细取舍记录。

---

## 1. AI bundle 运行时的三条架构路径

为了在宿主 app 内运行 AI 生成的 RN bundle，我们对比了三种隔离方案：把 bundle
丢进 WebView（路径 1）、把整个 app 当 sandbox 重启（路径 2）、以及最终选定的
**Sub-Hermes + SDK** 路径——在宿主进程内拉起一个独立的 Hermes 子运行时，
通过 SDK 向 bundle 暴露受控 native module。下表是三条路径在七个工程维度上的
权衡。

| 维度 | WebView (路径 1) | 整 app reload (路径 2) | Sub-Hermes + SDK (你的方案) |
| --- | --- | --- | --- |
| 真原生渲染 | ❌ 浏览器引擎 | ✅ | ✅ |
| 单 app 一站式 | ✅ | ❌（要装两个 app） | ✅ |
| Reload 快 | ~500ms | ~2s（重启 app） | ~50–100ms（重 spawn sub runtime） |
| AI bundle 体积 | ~500KB（带 react-dom） | ~3-5MB（带 RN） | ~50-200KB（纯业务代码） |
| 动画 / 手势原生 | ❌ | ✅ | ✅ |
| 安全沙箱 | iframe（强） | 整 app（弱） | sub runtime（强） |
| 工程量 | 1-2 周 | 1 周 + 多一个 app | 6-12 月 |

**结论**：路径 3 的工程成本最高（6-12 个月），但只有它同时满足"真原生渲染 +
单 app 一站式 + 强沙箱 + 小 bundle"四个硬约束。路径 1 留作 web 端的 fallback；
路径 2 仅在早期 dogfood 阶段用过，不进 production。

---

## 2. `runtime/` 三个出口及其消费者

Sub-Hermes 路径决定后，`runtime/` 这个目录需要同时服务三类消费者：宿主 app
（要把 native runtime 链进 binary）、sandbox 容器（要在 Metro 解析时拿到正确
的 JS 入口与 lockfile）、以及 AI agent（要在 system prompt 里拿到当前可用
module 与类型信息）。下表锁定每个出口的形态、消费者与使用方式，避免后续
迭代时三者口径漂移。

| 出口 | 谁消费 | 怎么用 |
| --- | --- | --- |
| `Runtime.framework` / `runtime.aar` | host app（你的 app/ Expo 工程或将来的纯 native 工程） | 链接进 binary，提供 RN runtime + 已链接的 native modules |
| `js/` 包 | sandbox 容器 | npm install 它，Metro 用它的 lockfile / package.json 解析 import |
| `types/` + `version.json` | AI agent | system prompt 注入 `version: 1.4.2`、可用 module 清单；AI 用 .d.ts 做语义补全 |

**结论**：三个出口必须共用一个版本号（`version.json` 是 source of truth）；
`Runtime.framework` / `runtime.aar` 与 `js/` 包通过 CI 同步发布，AI agent
通过 `types/` + `version.json` 在每轮 turn 开始前刷新可用面，确保生成的
bundle 不引用宿主未链接的 module。
