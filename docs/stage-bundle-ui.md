# Stage bundle UI strategy

> **状态**：策略已定，**实施 deferred**。等真用户先 dogfood，确认 Stage bundle 实际怎么被 AI 用，再开干。
>
> 本 doc 把战略写下来，避免下次回到这个话题时重新推一遍。

---

## 问题

AI 在 Stage bundle 里写 RN UI 代码 —— 用什么 UI 框架 / 库 / 风格？

四个候选：

| 候选 | 形态 |
|---|---|
| A. **`@appunvs/ui`（自家做）** | 一个 token-aware RN 组件库，专门为 AI 写代码场景设计 |
| B. **NativeWind + rn-primitives** | 把 Tailwind className 塞进 RN |
| C. **Tamagui** | 自家 DSL，编译期优化，跨 web+native |
| D. **react-native-paper / NativeBase** | 标准 Material 组件库 |

## 为什么不能直接抄竞品

参考 [competitive-landscape.md](competitive-landscape.md) 的阵营划分：

| 产品 | 产物 | UI 栈 |
|---|---|---|
| **Lovable** | Vite/React **web** | shadcn/ui + Tailwind（事实上的 web AI builder 标配） |
| **v0** | Next.js **web** | shadcn/ui + Tailwind |
| **Bolt.new** | Vite **web** | 任意 npm 包，AI 自己装；shadcn 居多 |
| **a0.dev** | 真 **RN bundle** | 没公开；推测裸 RN style 或自家内部 lib |
| **appunvs** | 真 **RN bundle** | 待定（本 doc 的话题） |

**Lovable / v0 / Bolt 选 shadcn+Tailwind 不是因为"shadcn 设计好"，是因为他们的产物就是 web，shadcn 是 web AI 写代码的事实标准** —— 蹭训练数据红利。

我们和 a0.dev 一样产物是 RN，**蹭不到这个红利**。

## RN 端为什么 NativeWind 风险大

NativeWind 是把 Tailwind className 翻译成 RN style 的方案。但 RN ≠ web CSS：

- AI 训练数据是 **web Tailwind**，写出来的 className 一半在 RN 上是哑的：
  - `hover:bg-blue-500` —— RN 没 hover，**编译过、运行时静默无效果**
  - `grid grid-cols-3` —— RN 没 grid
  - `sticky top-0` —— RN 没 position:sticky
  - 复杂 gradient / transform / animation 多数行为不一致
- AI 编译器（metro / TS）这层完全检测不出 —— `className: string`，啥字符串都过；运行时也不报错
- 调试体验：「为啥这个不工作？」「啊是 RN，没有这个属性」「我怎么知道？」

NativeWind v4 的论坛常驻 issue。开源 lib 不是不能用，但**给 AI 用尤其危险** —— AI 没办法学到「这个类在 RN 里不能用」。

## 选定方向：`@appunvs/ui` + 给 AI 看的文档

### 核心思路

写一个**专为 AI 在 Stage bundle 里调用而设计**的小 RN 组件库。AI 不写样式，只 compose。

```tsx
// AI 写的代码长这样：
import { Button, Card, Stack, Text } from '@appunvs/host/ui';

export default function Counter() {
  const [n, setN] = useState(0);
  return (
    <Card>
      <Stack space="m" align="center">
        <Text variant="display">{n}</Text>
        <Button variant="primary" onPress={() => setN(n + 1)}>
          +1
        </Button>
      </Stack>
    </Card>
  );
}
```

### 为什么这条路是对的

1. **兼容性 100% 我们说了算** —— 每个组件手写 RN，能用什么属性可控；没有 NativeWind 那种「className 接受任何字符串但只一半生效」陷阱
2. **AI 写得对率高** —— API 简单（5-10 个 prop / 组件），错位 prop 名 TypeScript 直接报错
3. **bundle 几乎零增重** —— 自家代码 tree-shake 干净
4. **token 直连** —— 组件内部从 `host().tokens` 拿 token，跟 host shell 视觉天然一致；用户在 host 改主题色 → AI bundle 立刻跟随
5. **质量 + 排查路径闭环** —— AI 写错 → 翻 SKILL 文档看用法 → 修组件 / 修文档；不依赖任何外部 lib 升级

### 组件清单（v0 范围，按需增量）

最小集，~12-15 个组件：

| 组件 | 用途 |
|---|---|
| `Button` | primary / secondary / ghost / danger 几种 variant |
| `Text` | 暴露 Typography token：display / title / heading / body / caption / label |
| `Card` | 圆角白底容器 |
| `Stack` | 垂直 / 水平排布 + space token |
| `Input` | 单行 / 多行文本 |
| `Tabs` | 顶部 tab bar |
| `Dialog` | 模态框 |
| `Toast` | 顶部短消息 |
| `Avatar` | 圆形头像 |
| `Badge` | 状态徽章 |
| `Divider` | 分割线 |
| `Spinner` | loading |
| `Empty` | 空状态 |

逃生舱：AI 想自定义复杂样式时直接用 RN 原生 `View` / `Text` + `host().tokens` 拿原始 token 值，不强制每个像素都过组件库。

### 配套产物

```
runtime/src/ui/
├── index.ts                ← 暴露给 @appunvs/host/ui import
├── Button.tsx
├── Card.tsx
├── ... (其他组件)
├── tokens.ts               ← 桥接 host().tokens
├── SKILL.md                ← 给 Claude Skill 加载的 AI 指南
├── components.json         ← shadcn 风格组件清单
├── examples/               ← 给 AI 看的"标准用法"代码
│   ├── counter.tsx
│   ├── form-screen.tsx
│   ├── list-detail.tsx
│   └── ...
└── README.md               ← 给开发者读的常规文档
```

**`SKILL.md` 是关键**：
- 每个组件一段：什么场景用、prop 含义、**反例（不要这样写）**
- 在 Claude Code 等 agent 平台上注册成 Skill，AI 写 Stage bundle 时自动加载
- 反例特别重要 —— 蹭"AI 训练数据知道这种用法"那种隐性知识不可靠，反例显式写出来更稳

**`examples/` 里的代码是"AI 学习样本"**：
- 几个完整的标准 Stage bundle（counter / form / list-detail / settings 等）
- AI 接到任务时这些是 in-context 示例

### 开源策略

- 跟 appunvs 主仓一起开源，不另开 repo
- 加 LICENSE（MIT）
- README 顶部讲清楚定位：「a tiny RN UI lib designed to be written by AI agents（明确不是给人手写）」
- 对外发声有两层好处：
  - 帮其他做 RN AI builder 的人（生态价值）
  - 建技术品牌（招聘 / 认可度）
- 等成熟后可以独立 npm publish 成 `@appunvs/ui`，host bridge 那侧 alias 进来

## 实施顺序（不是现在做）

**dogfood 之前不动**。先用裸 RN style 跑通一段时间，搞清楚 AI 实际撞到什么 UI 痛点，再针对性建组件。

dogfood 之后的 PR 顺序：

| PR | 内容 |
|---|---|
| #1 | 加 `runtime/src/ui/` 骨架 + `Button` / `Text` / `Card` / `Stack` 四件套 + `SKILL.md` v0 |
| #2 | metro 沙箱解析 `@appunvs/host/ui` —— 跟 `@appunvs/host` 同样的 babel 解析路径 |
| #3 | 加 form 类组件：`Input` / `Select` / `Checkbox` |
| #4 | 加 overlay 类：`Dialog` / `Toast` / `Sheet` |
| #5 | 加 examples/ 里 5-7 个标准 Stage bundle 模板 |
| #6 | 独立 npm publish + 公开宣传 |

每个 PR 都带 SKILL.md 增量更新；reading order 是 SKILL.md 必读。

## 不要做的事

- ❌ NativeWind / Tailwind-on-RN —— 兼容性陷阱多
- ❌ Tamagui —— DSL AI 训练数据少，写错率高
- ❌ react-native-paper —— 太重 + 风格锁死 Material
- ❌ 把组件库做太大太通用 —— 我们要的是「AI 能写对的小组件」，不是「应付所有场景的大库」

## 监督指标

dogfood 期间观察这几条：

1. AI 第一次生成的 Stage bundle 能 build 成功率
2. AI 第一次生成的 bundle 视觉跟 host shell 调性一致的比例
3. publish 后用户主动改样式的比例（如果高，说明 AI 默认产出不好）
4. 哪些组件 / pattern 反复出现 —— 这是 v0 组件清单的真实信号

这些指标比"我们觉得设计漂不漂亮"更能驱动 `@appunvs/ui` 的演进方向。
