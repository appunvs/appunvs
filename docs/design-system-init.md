# Design System 初始化指引（host shell）

给接手 design system 工作的 Claude 会话用。**只看 host shell**（appunvs/ios + appunvs/android），不要碰 runtime/、relay/，也不要改 AI bundle 的样式契约（@appunvs/host 等等，那是另一个 deferred 话题）。

---

## 0. 先读这五个文件

按顺序读，读完就有完整上下文：

1. `CLAUDE.md` — 项目术语 + 仓库布局
2. `docs/architecture.md` § 实施次序 — 已合入 vs 下一片
3. `appunvs/ios/Runtime/Theme/Theme.swift` — iOS 现有 token（color / spacing / radius / typography）
4. `appunvs/android/app/src/main/java/com/appunvs/runtime/theme/{Color,Theme,Typography}.kt` — Android 对位
5. `appunvs/{ios,android}/.../screens/TokensPreview*` — token 预览屏，DEBUG 入口在 Profile 底部

---

## 1. 当前状态（你不需要从零搭）

**已经有的 token 词汇表**（两端语义名严格对齐，DS 工作只调**值**不改**结构**）：

| 类别 | tokens |
|---|---|
| Color | `brandDark` `brandLight` `brandPale` `textPrimary` `textSecondary` `bgPage` `bgCard` `bgInput` `borderDefault` `semanticSuccess` `semanticWarning` `semanticDanger` `semanticInfo` —— 每个都是 light / dark 双值 |
| Spacing | `xs s m l xl xxl xxxl huge`（`4 8 12 16 20 24 32 48`） |
| Radius | `s m l xl pill`（`6 10 12 14 999`） |
| Typography | `display title heading body bodyEmphasis caption label mono` —— iOS 是 SwiftUI `Font`（映射 Apple Dynamic Type），Android 是 `TextStyle` |

**已经有的工具**：
- iOS：`.appFont(Typography.body)` modifier
- Android：`Text(style = AppType.body)`
- 两端都有 `TokensPreviewView` / `TokensPreviewScreen` —— DEBUG build 下 Profile 底部「Design tokens →」入口

**两端落地差异**（要警惕）：
- iOS 用 Apple Dynamic Type，字号会跟用户系统设置浮动；Android 是硬 sp 值。视觉对齐**只能在默认 size 下保证**。
- iOS 颜色用程序化 `UIColor { trait in ... }` 解析，避开 asset catalog；Android 走 Compose `LocalAppColors` CompositionLocal。

---

## 2. 工作流（一定要走这条路）

1. 改 token：编辑 `Theme.swift` 或 `Typography.kt`
2. 跑 DEBUG build → Profile tab → 底部「Design tokens →」
3. 一屏看到所有 token 渲染效果，立即调
4. 满意了再去具体 screen 里把 `.font(.body)` `13.sp` 等硬编码替换成 token 引用

**不要**直接改具体 screen 而不验证 token 全局效果。**不要**新增 token 而不同时在两端添加。

---

## 3. v0 交付目标（按优先级）

### 必做

1. **品牌方向决定**：当前是青绿色系（`#0B505A` brand）。和用户对一下要不要换。换的话只改 `Theme.swift` + `Color.kt` 的 hex 值，token 名不动。
2. **Typography 微调**：当前 `body=17sp`、`heading=17sp/SemiBold`，跟 iOS Dynamic Type `.body` `.title3` 对齐。看是否合适或要改 scale（比如 16/18/20/24）。
3. **Migrate scattered call sites**：`grep` 找 `\.font\(\.` 和 `\.sp\b` 把硬编码替换成 token：
   ```bash
   grep -rn "\.font(\." appunvs/ios/Runtime/{UI,Screens}/ | grep -v "appFont\|Typography"
   grep -rn "\.sp\b" appunvs/android/app/src/main/java/com/appunvs/runtime/{ui,screens}/ | grep -v "AppType"
   ```
   每替换一处先看 TokensPreview 确认 token 视觉对，然后才动 screen 文件。

### 可选（等 v0 落地之后再说）

4. **加 elevation / shadow tokens**：当前几乎没用 shadow。如果设计语言要立体感，加 `Elevation.{none, low, medium, high}`，两端实现（iOS：`.shadow()` 参数；Android：`Modifier.shadow(elevation.dp)`）。
5. **加 motion duration tokens**：`Motion.{fast, normal, slow}`（150 / 250 / 400ms），约束动画时长。
6. **Component 变体规范**：`Card` / `Bubble` / `Badge` 等组件加 variant 参数（primary / secondary / outline / ghost），iOS + Android 各落一遍。

---

## 4. 硬约束

- **不改 AI bundle 那一摊**：`runtime/`、`@appunvs/host` TypeScript 契约、AI bundle 用的样式 —— 全部不动。那是后续独立工作（看 `docs/competitive-landscape.md` 关于 shadcn / gluestack 的讨论）。
- **不改 relay / Go**：跟 design 无关。
- **不能破坏 dark mode**：每个 color token 必须 light + dark 双值。验证方法：iOS 在 simulator 切 Appearance；Android 在 Profile 里有 SYSTEM/LIGHT/DARK 三选一。
- **iOS Dynamic Type 不能丢**：iOS Typography 不要从 `Font.body` 之类 Apple 语义字体改成硬 `.system(size: 17)`，会失去无障碍字号缩放。
- **不引入新组件库**：不要装 SwiftUI Introspect、不要装 Material Symbols、不要装第三方设计库。当前栈是裸 SwiftUI + 裸 Compose Material3。

---

## 5. 验证清单

PR 提交前检查：

- [ ] TokensPreview 在两端跑通，所有 token 渲染正常
- [ ] iOS：在 simulator 切 Appearance（light/dark）token 都正常
- [ ] Android：Profile 里切 SYSTEM/LIGHT/DARK token 都正常
- [ ] 既有的所有 screen（Login / Chat / Stage / Profile）视觉无 regression
- [ ] CI 全绿（包括 `runtime SDK iOS (UI test)` 和 `runtime SDK Android (instrumented)` —— DS 改不应该破坏 D3.c.4 fixture 测试）
- [ ] 没有引入新 npm / pod / gradle 依赖

---

## 6. 起手 prompt 模板

新会话开场可以直接说：

> 读 `docs/design-system-init.md` 然后开始 v0 design system 工作。品牌方向：[填这里 —— 比如「保持青绿但更冷一点，对标 Linear / Notion 的克制感」]。优先做必做 1-3 项，可选 4-6 留给后续 PR。
