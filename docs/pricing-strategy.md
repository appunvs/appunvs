# Pricing strategy

> **状态**：方向确定（**Claude 式订阅档位 + 用量上限** + BYOK 单独通道），数字 v0 草案。
> 等真上线收集到使用数据后微调。

---

## 出发点

- **学 Claude，不学 Cursor**：四档订阅 + 每档硬上限。用完即停（提示等下个周期或升档），不做按量超额。
  - 实现简单：只数 turn，不需要 USD ledger / Stripe metered / 实时折算
  - 用户心智简单：每档"我能做 N 次"清晰可数，不用算账单
  - 防失控：不存在"用户睡一觉醒来欠 ¥500"的可能性
  - 成本可控：硬墙保证毛利不被尾部用户吃光
- 国内市场对**月固定费**接受度高，对纯按量焦虑 —— Claude 模式正好对上
- BYOK 单独通道：把 LLM 成本最大头转嫁给极客用户，我们只赚基础设施

> 不做按量超额（Cursor 式）的取舍：放弃了一部分重度 builder 用户的"加钱继续"路径。补救：Max / Max+ 档位本身定得宽松（5000+ turn），覆盖 95% 用户；剩下 5% 重度用户引导走 BYOK 通道（自己付 LLM、自由用量）。

## 命名：Free / Pro / Max / Max+

跟 Anthropic Claude 订阅档位对齐。我们后面主打 Claude 路径，命名一致 = 用户不用做心智映射。

```
Free     ← 免费试用（仅 DeepSeek，不可换模型，体验产品本身）
Pro      ← 主力档（绝大多数付费用户；解锁全模型自由切换）
Max      ← 高频 / 重度用户（更高额度 + 优先 sandbox）
Max+     ← 团队 / 企业 / 给 Opus 类高端模型预留
```

## 四档草案（v0 数字）

> 说明：
> - **可用模型**：Free 只锁定 DeepSeek（试用）；**Pro 及以上自由切换全部模型**（DeepSeek / Claude / GPT / Gemini）。这是 Pro 的核心价值，不是 Max 才解锁。
> - **AI turn / 月** 是硬上限，对所有模型一视同仁（不区分 DeepSeek 一次还是 Opus 一次都按 1 turn）；后端做 token 软限：单 turn 烧 token 超过基线 10× 按 2 turn 算，防滥用 / 长文档塞爆。
> - 硬上限触达后**直接暂停到下个计费周期**（月度滚动），UI 引导升档。BYOK 用户 LLM 部分不计入 turn 上限。

| 档 | 月费 | AI turn / 月 | active box | publish_box / 月 | 可用模型 | 触达上限 |
|---|---|---|---|---|---|---|
| **Free** | ¥0 | 50 | 1 | 5 | **仅 DeepSeek**（不可切换） | 暂停到下月 / 引导升 Pro |
| **Pro** | ¥30 | 500 | 5 | 50 | **全模型自由切换** | 暂停 / 升 Max |
| **Max** | ¥80 | 2000 | 20 | 200 | 全模型 + sandbox 优先队列 | 暂停 / 升 Max+ |
| **Max+** | ¥200 | 5000 | 不限 | 500 | + Opus 优先 / 私有部署 / team | 暂停 / 联系销售 |
| **BYOK** | ¥0 | LLM 不计 | 5 | 50 | 自带 key 任意模型 | sandbox 用完按 Pro 价升级 |

> 数字 v0 草案。等 dogfood + 真用户数据再调，重要的是结构。

## 计费单位选择

混用三个：

- **AI turn / 月** —— 主限制，跨模型统一计数（用户能直观理解"我能聊 N 次"）
- **active box 数量** —— 心理感知强（"我能开 N 个项目"）的硬限制
- **publish_box / 月** —— 独立配额，每次触发 sandbox 真实算力

LLM token 不直接暴露给用户（波动太大、不直观）；后端对单 turn 做 token 软限（10× 算 2 turn）防滥用。

> 跟 Cursor 式 USD ledger 的差异：我们**不向用户呈现"美元/RMB 等值额度"**。turn 是统一单位，模型差价由档位结构吸收（Pro 用户多用 Opus 会更快到 500 turn 上限，但毛利保护由档位本身覆盖）。

## BYOK 单独通道

**为什么单独存在**：
- 吸引种子用户 / 极客用户 —— 这些人愿意自己申请 Anthropic 账号、付跨境费
- 我们 LLM 成本归零，只承担 sandbox + relay + 存储成本
- 病毒传播载体：BYOK 用户在社区分享 "appunvs 让我自带 key 用 Claude 做 RN app"，比付费用户传播力强
- **覆盖按量重度用户**：放弃了 Cursor 式按量后，重度 builder 的逃生口在这里

**BYOK 用户付什么**：
- 月费 0
- LLM 部分自付（用户在设置页填 Anthropic / OpenAI / Gemini 的 key，relay 转发不存储 key 明文）
- sandbox / publish 配额按 Pro 档发（5 box / 50 publish / month）
- 想超过这个量？升级到 Pro / Max（保留 BYOK，仅买更多 sandbox 配额），或自建 relay（开源版）

**风险**：
- 极客用户白嫖完 5 个 box 后流失
- 缓解：BYOK 渠道明确写"sandbox 配额仍占成本"，超出引导付 ¥30 解锁更多 box

## 成本测算（DeepSeek 基线）

中度用户（月内 500 turn + 50 publish）：

| 项 | 单价 | 月成本 |
|---|---|---|
| DeepSeek token | ~¥0.001 / 1K input + ¥0.002 / 1K output | ~¥3-5（按 avg 3K token / turn） |
| 阿里云 ECI sandbox | ~¥0.10 / publish（按秒计 ~30s） | ~¥5 |
| Artifact 存储 | LocalFS / 对象存储 ~免费 | ~¥0.5 |
| Relay 共享 | 单机摊销 | ~¥1 |
| **合计 marginal cost** | | **~¥10-12** |
| Pro 月费 | | **¥30** |
| **毛利** | | **~60%** |

中等用户毛利 60%。**注意**：用户切到 Claude Sonnet / Opus 时 token 成本会大幅上升 —— Claude Sonnet 单价 ~30× DeepSeek，Opus ~70×。如果 Pro 用户全程用 Opus 跑满 500 turn，token 成本可达 ~¥200，**亏损 ¥170**。

应对（不开按量的前提下）：
- **后端按模型加权 turn 计数**：用 Sonnet 一次 = 3 turn，用 Opus 一次 = 10 turn（Cursor 式 multiplier 但不暴露给用户）。Pro 用户用 Opus 实际上限就是 50 turn，亏损被吸收
- 或者**简单粗暴：Free / Pro 限制只能用 DeepSeek + Sonnet，Max 才解锁 Opus**（违反前面"Pro 全模型"承诺，需要权衡）

> 决策：v0 用**模型加权 turn 计数（隐式）**，不破坏"Pro 全模型"心智。具体倍数等真实成本数据回来再调。

Max+ ¥200 给 Opus 用户：放更宽的 multiplier（如 Opus 一次 = 5 turn），但 Max+ 用户量极少（典型 < 5%），平均毛利仍可保持 50%+。

## 不要做的事

- ❌ **纯按量**：用户每次 chat 都担心烧钱，停留时长锐减
- ❌ **纯订阅不限量**：1% 重度用户烧光所有利润；Bolt.new 早期吃过这亏
- ❌ **海外档位（$20）直接对标人民币 ¥150**：国内付费墙过高，¥30 是更合适的锚定
- ❌ **太多档**：超过 4 档用户决策瘫痪。Free / Pro / Max / Max+ 已经是上限
- ❌ **Free 用户开放模型切换**：Free 是试用，不是低配版 —— 能切到 Claude / GPT 就没人付费了。Free 必须只 DeepSeek。
- ❌ **限制 Free 用户用 publish**：publish 是产品核心 a-ha，让 Free 用户至少能完成一次完整 dogfood
- ❌ **暴露模型加权 multiplier 给用户**：用户一旦看到"Opus = 10 turn"就觉得被宰；隐式扣减、UI 显示"剩余 N turn"即可

## 上线节奏（不用一次到位）

**Phase 1（dogfood）**：完全免费，无任何用量限制 —— 收数据，看真实用量分布
**Phase 2（小范围内测）**：上 Free + Pro 两档，¥30 试水；turn 硬上限上线
**Phase 3（公开发布）**：四档全开 + BYOK 单独通道；按模型加权计数上线
**Phase 4（运营调优）**：根据真实数据调每档 turn 数 + multiplier 系数 + 考虑年付折扣（年付 8.5 折是行业惯例）

## 竞品定价快速对比

⚠️ **数字可能已变动**。具体看官网。

| 产品 | 免费层 | 付费起步 | 计费单位 | 主要超量策略 |
|---|---|---|---|---|
| **Claude.ai** | 每天少量 message | $20/月（Pro） | 5h 滚动窗口 message 限速 | 等下个 5h 窗口 / 升 Max |
| **ChatGPT Plus** | 受限 | $20/月 | message + token 限速 | 等限速复位 / 升 Pro |
| **Cursor** | 50 fast/月 | $20/月（Pro） | USD 余额 + opt-in 按量 | 按量继续（默认关） |
| **Lovable** | 每天 ~5 message | ~$20/月（Starter） | "credit"（一条 message 扣 X） | 升级 / 等下月 |
| **v0** | 少量 generation | ~$20/月 | generation + token | 升级 |
| **Bolt.new** | 每天 ~150-200K token | ~$20/月（Pro） | LLM token 直计 | 升级 / 加 Pro+ |
| **appunvs** | 50 turn + 1 box（仅 DeepSeek） | ¥30/月（Pro，全模型自由切换） | turn + box + publish | **暂停到下月 / 升档**（Claude 式硬墙） |

我们的差异点：
- **国内价位**（¥30 ≈ 美元 $4，是 Lovable / Cursor 的 1/5）—— 国内市场更可负担
- **BYOK 通道**给到 LLM 完全免费，竞品没人有 —— 同时是重度用户的逃生口（弥补无按量超额）
- **active box 数量**作为限制维度 —— 用户感知强，竞品没人这么算
- **付费门槛清晰**：Free 锁 DeepSeek 试体验，Pro 即解锁全模型（Claude / GPT / Gemini 自由切换），单点付费即可解锁产品全部能力
- **简单的 Claude 式硬墙**：不做按量超额，实现轻、用户决策无负担

## 实现复杂度（vs Cursor 模式）

选 Claude 模式让落地工作量大幅缩水：

| 模块 | Cursor 模式 | Claude 模式 |
|---|---|---|
| Turn 计数 | 必须实时折算 USD | **简单计数**（已有 store.Turns） |
| 模型价格表 | 每模型每方向 USD/1M | 仅作为后端 multiplier 表，UI 不显示 |
| Stripe | 订阅 + metered usage | **仅订阅**（subscription only） |
| Ledger 表 | accrued_usd_cents 实时累加 | turn 计数 monthly bucket 即可 |
| 用户 UI | USD% + 按模型 turn 估算 + 按量开关 | "已用 N / M turn" 一行 |
| 按量结算 | 月底批量 + 错误处理 + 退款流程 | **不存在** |
| 客服压力 | "为什么我被扣了 ¥X？"问询 | **极低**（硬墙清晰） |

落地顺序：
1. **Phase A**：在 `store.Turns` 上加 monthly aggregation（用户 + 月份 → turn count）
2. **Phase B**：`/ai/turn` 入口前查 quota，超额返回 `429 plan_exhausted`
3. **Phase C**：模型加权 multiplier（Opus 一次扣多次）
4. **Phase D**：Stripe 订阅 + Webhook 同步 plan
5. **Phase E**：BYOK 通道（用户 settings 存 key，relay 转发）

A/B 这周可做；C 等模型 catalog 扩到包含 Claude / GPT 后再加；D/E 等公开发布前。
