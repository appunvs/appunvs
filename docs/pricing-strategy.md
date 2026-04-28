# Pricing strategy

> **状态**：方向确定（subscription + 用量上限 + 超量两条路 + BYOK 单独通道），数字 v0 草案。
> 等真上线收集到使用数据后微调。

---

## 出发点

- 学 OpenAI / Anthropic：**月订阅是基础盘，用量是天花板，超量给两条路（升级 / 按量）**，不要纯免费、也不要纯按量
- 国内市场对**月固定费**接受度高，对纯按量焦虑
- 我们成本结构和竞品略不同：BYOK 让用户自带 LLM key 是杠杆 —— 成本最大头转嫁给用户，我们只赚基础设施

## 命名：Free / Pro / Max / Max+

跟 Anthropic Claude 订阅档位对齐。我们后面主打 Claude 路径，命名一致 = 用户不用做心智映射。

```
Free     ← 免费引流，每天能玩
Pro      ← 主力档（绝大多数付费用户）
Max      ← 高频 / 重度用户
Max+     ← 团队 / 企业 / 给 Opus 类高端模型预留
```

## 四档草案（v0 数字）

| 档 | 月费 | AI message / 月 | active box | publish_box / 月 | 可用模型 | 超量策略 |
|---|---|---|---|---|---|---|
| **Free** | ¥0 | 50 | 1 | 5 | DeepSeek 默认 | 阻塞到下月 / 升级提示 |
| **Pro** | ¥30 | 500 | 5 | 50 | DeepSeek + Claude Sonnet | 0.05 元/条 OR 升 Max |
| **Max** | ¥80 | 2000 | 20 | 200 | + Claude Opus | 0.04 元/条 OR 升 Max+ |
| **Max+** | ¥200 | 5000 | 不限 | 500 | + 优先 sandbox 队列 + 私有部署/team workspace | 0.03 元/条 |
| **BYOK** | ¥0 | 不限 LLM | 5 | 50 | 自带 key 任意模型 | sandbox 配额按 Pro 算 |

> 数字是 v0 草案，等 dogfood + 真用户数据再调。重要的是结构。

## 计费单位选择

混用两个：

- **AI message / turn** —— 用户能直观理解（"我能聊 N 次"）
- **active box 数量** —— 心理感知强（"我能开 N 个项目"）的硬限制

LLM token 不直接暴露给用户（波动太大、不直观），但**后端做软限制**：单条 message 烧 token 超过某个倍数（比如 10×）按 2 条算 —— 防滥用 / 长文档塞爆。

`publish_box` 单独算（每次触发 sandbox 真实算力）。

## 超量两条路 vs OpenAI 降级模式

OpenAI 默认是「Plus 用 80 条降级到 mini」。我们选**直接按量**：

- 弹窗"已用 500/500 → [升级 Max] [继续按 0.05 元/条] [等下月]"
- 用户明确知道每条消息成本，不会被偷偷降级
- 国内用户对"明码标价"接受度比"自动降级"高（被坑过太多次自动开通的服务）

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

- ❌ **纯按量**：用户每次 chat 都担心烧钱，停留时长锐减
- ❌ **纯订阅不限量**：1% 重度用户烧光所有利润；Bolt.new 早期吃过这亏
- ❌ **海外档位（$20）直接对标人民币 ¥150**：国内付费墙过高，¥30 是更合适的锚定
- ❌ **太多档**：超过 4 档用户决策瘫痪。Free / Pro / Max / Max+ 已经是上限
- ❌ **限制 Free 用户用 publish**：publish 是产品核心 a-ha，让 Free 用户至少能完成一次完整 dogfood

## 上线节奏（不用一次到位）

**Phase 1（dogfood）**：完全免费，无任何用量限制 —— 收数据，看真实用量分布
**Phase 2（小范围内测）**：上 Free + Pro 两档，¥30 试水
**Phase 3（公开发布）**：四档全开 + BYOK 单独通道
**Phase 4（运营调优）**：根据真实数据调 Pro 包含的 message 数（500 可能太多 / 太少）+ 考虑年付折扣（年付 8.5 折是行业惯例）

## 竞品定价快速对比

⚠️ **数字可能已变动**。具体看官网。

| 产品 | 免费层 | 付费起步 | 计费单位 | 主要超量策略 |
|---|---|---|---|---|
| **Lovable** | 每天 ~5 message | ~$20/月（Starter） | "credit"（一条 message 扣 X） | 升级 / 等下月 |
| **v0** | 少量 generation | ~$20/月 | generation + token | 升级 |
| **Bolt.new** | 每天 ~150-200K token | ~$20/月（Pro） | LLM token 直计 | 升级 / 加 Pro+ |
| **a0.dev** | beta 试用 | 推测 ~$20/月 | message / generation | 不太透明 |
| **appunvs** | 每月 50 message + 1 box | ¥30/月（Pro） | message + box + publish | 按量 OR 升级 |

我们的差异点：
- **国内价位**（¥30 ≈ 美元 $4，是 Lovable 的 1/5）—— 国内市场更可负担
- **BYOK 通道**给到 LLM 完全免费，竞品没人有
- **active box 数量**作为限制维度 —— 用户感知强，竞品没人这么算
- **超量直接按量**（明码），不偷偷降级
