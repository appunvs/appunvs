# Pricing strategy

> **状态**：方向确定（**Cursor 式 subscription + 按量超量** + BYOK 单独通道），数字 v0 草案。
> 等真上线收集到使用数据后微调。

---

## 出发点

- **学 Cursor，不学 Claude**：月订阅是基础盘，订阅含的是一笔"用量额度"（按 LLM 实际消耗折算），用完不限速、不降级，**直接进按量计费**；想避免就升档。
- 我们的目标用户是**专业开发者 / 重度构建用户**（appunvs 是产品工具，不是聊天玩具）。这类用户对"明码按量"接受度比"突然限流到下周一"高得多 —— 等不起。Claude.ai / ChatGPT 选 rate-limit 是因为他们的 80% 是 chatter，限不住的话 1% 重度用户能吃光全部毛利；我们用户分布更接近 Cursor，重度构建是常态而不是异常，得给他们一条"加钱继续"的合规通道。
- 国内市场对**月固定费**接受度高，对纯按量焦虑 —— 所以保留订阅作为锚定，按量只在用完订阅额度后启用，且**默认要用户主动开启**（防误烧）。
- 我们成本结构和竞品略不同：BYOK 让用户自带 LLM key 是杠杆 —— 成本最大头转嫁给用户，我们只赚基础设施。

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
> - **订阅含额度**用 Cursor 式的"美元额度"（折算后端真实 LLM 成本 + sandbox 算力）。用户面板看到的是**「已用 / 月度额度」百分比**，超出后按 1.0× 直接计费（即不加价、不降速）。
> - "不限 LLM" 适用于 BYOK：用户自带 key 时 LLM 成本不计入额度，只扣 sandbox / publish 配额。

| 档 | 月费 | 含订阅额度 | active box | publish_box / 月 | 可用模型 | 超量策略 |
|---|---|---|---|---|---|---|
| **Free** | ¥0 | ¥3 折算（DeepSeek ~50 turn） | 1 | 5 | **仅 DeepSeek**（不可切换） | 暂停到下月 / 引导升 Pro |
| **Pro** | ¥30 | ¥30 折算（按所选模型自动折算 turn 数） | 5 | 50 | **全模型自由切换** | 默认进按量 1.0×；可关 |
| **Max** | ¥80 | ¥80 折算 | 20 | 200 | 全模型 + sandbox 优先队列 | 默认进按量 1.0×；可关 |
| **Max+** | ¥200 | ¥200 折算 | 不限 | 500 | + Opus 优先 / 私有部署 / team | 默认进按量 1.0×；可关 |
| **BYOK** | ¥0 | LLM 成本归 0 | 5 | 50 | 自带 key 任意模型 | sandbox 用完按 Pro 价 |

> 数字 v0 草案。"额度"以美元等值折算成 RMB；当真实使用数据进来时，会把折算系数定到让 Pro 用户在 DeepSeek 下能达到 ~500 turn / Claude Sonnet 下 ~150 turn / Opus 下 ~30 turn 的目标量。等 dogfood + 真用户数据再调，重要的是结构。

## 计费单位选择

订阅含的是一笔**额度（美元等值，按真实成本折算）**，UI 上以三种方式呈现：

- **额度百分比** —— 主显示，"本月已用 60% / ¥30"。直观、和模型无关。
- **按模型展开的 turn 估算** —— 用户切到 DeepSeek 时显示"剩余约 200 turn"；切到 Claude Sonnet 时显示"剩余约 60 turn"。让用户对**模型差价**有直觉。
- **active box 数量** —— 独立计数，硬限制（不可超额）。

`publish_box` 占独立配额（每次触发 sandbox 真实算力，跟 LLM 额度分开）。

## 为什么选 Cursor 模式而不是 Claude 模式

| 维度 | Claude.ai (rate-limit) | Cursor (metered overage) | appunvs 选择 |
|---|---|---|---|
| 用完后行为 | 限速 / 等下个时段 | 可继续，按量计费 | **Cursor** |
| 用户画像匹配 | 80% 是聊天用户 | 80% 是开发用户，有交付压力 | 我们更像 Cursor |
| 重度用户 | 偶发滥用，可被限制 | 高强度构建是常态 | 必须放行 |
| 1% 重度用户毛利风险 | 限速兜底，不会爆 | 按量收回成本 | 按量足够 |
| 用户心智 | "用尽就停"很清晰 | "可继续付费"更专业 | 专业感优先 |

简单结论：**rate-limit 适合 chatter（边际成本低、用量分布尾巴薄）；metered overage 适合 builder（边际成本高、有重度长尾）**。我们做的是工具，目标用户是 builder。

> 注：默认**进按量需用户主动开启**（设置页 / 订阅页一键开关，类似 Cursor 的 "Allow usage-based pricing"）。关闭则用完即停 —— 给保守用户保留 Claude 风格的"硬墙"体验。

## BYOK 单独通道

**为什么单独存在**：
- 吸引种子用户 / 极客用户 —— 这些人愿意自己申请 Anthropic 账号、付跨境费
- 我们 LLM 成本归零，只承担 sandbox + relay + 存储成本
- 病毒传播载体：BYOK 用户在社区分享 "appunvs 让我自带 key 用 Claude 做 RN app"，比付费用户传播力强

**BYOK 用户付什么**：
- 月费 0
- sandbox / publish 配额按 Pro 档发（5 box / 50 publish）
- 想超过这个量？有两选：升级到 Pro / Max（继续 BYOK 但买 sandbox 量），或自建 relay（开源版本）

**风险**：
- 极客用户白嫖完 5 个 box 后流失
- 缓解：BYOK 渠道明确写"sandbox 配额仍占成本"，超出引导付 ¥30 解锁更多 box

## 成本测算（DeepSeek 基线）

中度用户（月内 500 message + 50 publish）：

| 项 | 单价 | 月成本 |
|---|---|---|
| DeepSeek token | ~¥0.001 / 1K input + ¥0.002 / 1K output | ~¥3-5（按 avg 3K token / message） |
| 阿里云 ECI sandbox | ~¥0.10 / publish（按秒计 ~30s） | ~¥5 |
| Artifact 存储 | LocalFS / 对象存储 ~免费 | ~¥0.5 |
| Relay 共享 | 单机摊销 | ~¥1 |
| **合计 marginal cost** | | **~¥10-12** |
| Pro 月费 | | **¥30** |
| **毛利** | | **~60%** |

中等用户毛利 60%。重度用户（用满 500 message）毛利会被压到 40%，但这种用户少。

Free 档预期亏本但可控（每用户 ~¥1-2 marginal cost）—— 当获客成本看。

Max+ ¥200 给 Opus 用户：Claude Opus token 单价是 DeepSeek 的 ~30x，但 Max+ 用户量极少（典型 < 5%），平均毛利仍可保持 50%+。

## 不要做的事

- ❌ **纯按量**（无订阅锚定）：用户每次 chat 都担心烧钱，停留时长锐减；Cursor 自己也是订阅 + 按量，不是纯按量
- ❌ **纯订阅不限量**：1% 重度用户烧光所有利润；Bolt.new 早期吃过这亏
- ❌ **默认强制开启按量**：用户必须主动同意，不然会被投诉"自动扣款"；Cursor 也是默认关、用户开
- ❌ **海外档位（$20）直接对标人民币 ¥150**：国内付费墙过高，¥30 是更合适的锚定
- ❌ **太多档**：超过 4 档用户决策瘫痪。Free / Pro / Max / Max+ 已经是上限
- ❌ **Free 用户开放模型切换**：Free 是试用，不是低配版 —— 能切到 Claude / GPT 就没人付费了。Free 必须只 DeepSeek。
- ❌ **限制 Free 用户用 publish**：publish 是产品核心 a-ha，让 Free 用户至少能完成一次完整 dogfood

## 上线节奏（不用一次到位）

**Phase 1（dogfood）**：完全免费，无任何用量限制 —— 收数据，看真实用量分布
**Phase 2（小范围内测）**：上 Free + Pro 两档，¥30 试水
**Phase 3（公开发布）**：四档全开 + BYOK 单独通道
**Phase 4（运营调优）**：根据真实数据调 Pro 包含的额度（¥30 折算多少 turn 合适）+ 接入按量超额开关 UI + 考虑年付折扣（年付 8.5 折是行业惯例）

## 竞品定价快速对比

⚠️ **数字可能已变动**。具体看官网。

| 产品 | 免费层 | 付费起步 | 计费单位 | 主要超量策略 |
|---|---|---|---|---|
| **Lovable** | 每天 ~5 message | ~$20/月（Starter） | "credit"（一条 message 扣 X） | 升级 / 等下月 |
| **v0** | 少量 generation | ~$20/月 | generation + token | 升级 |
| **Bolt.new** | 每天 ~150-200K token | ~$20/月（Pro） | LLM token 直计 | 升级 / 加 Pro+ |
| **a0.dev** | beta 试用 | 推测 ~$20/月 | message / generation | 不太透明 |
| **appunvs** | DeepSeek-only 试用，¥3 折算额度 + 1 box | ¥30/月（Pro，全模型自由切换） | 美元等值额度 + box + publish 配额 | **Cursor 式按量超量**（默认开关，可关） |

我们的差异点：
- **国内价位**（¥30 ≈ 美元 $4，是 Lovable / Cursor 的 1/5）—— 国内市场更可负担
- **BYOK 通道**给到 LLM 完全免费，竞品没人有
- **active box 数量**作为限制维度 —— 用户感知强，竞品没人这么算
- **付费门槛清晰**：Free 锁 DeepSeek 试体验，Pro 即解锁全模型（Claude / GPT / Gemini 自由切换），单点付费即可解锁产品全部能力
- **Cursor 式按量超量**（专业感）+ 默认要用户主动开启（防误烧）
