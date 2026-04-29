# Pricing strategy

> **状态**：v3 —— 三量纲（LLM / Sandbox / Storage），LLM 走 RMB 实时成本，Sandbox / Storage 仍按次/数量计；BYOK 改为付基础订阅、LLM 自付。数字 v0 草案。
> 等真上线收集到使用数据后微调。

---

## 出发点

- **三量纲、订阅 + 硬墙、不做按量超额**。
  - **LLM**（token 成本）—— 实时 RMB-cents 折算，因为同一模型不同 turn token 用量差 100×、跨模型差 70×，固定 turn 计数会爆样本
  - **Sandbox**（每次 publish 触发的 metro build）—— 按次计，每次成本 ~¥0.10 方差小
  - **Storage**（active box 数）—— 点状硬上限，不是窗口
- **5d 滚动 + 月度兜底**（仅适用于 LLM）—— 5d ≈ 一个 builder 自然 burst 周期（周二开始做到周五完成 MVP），比"小时级窗口"友好（不打断 flow），比单纯月限有节奏感（避免 day 1 烧光憋 29 天）。月限只在每个 5d 都跑满时才生效，作为最终兜底
- **不做按量超额**：超出即暂停到该窗口复位，不存在"用户睡醒欠 ¥500"
- 国内市场对**月固定费**接受度高，对纯按量焦虑 —— 订阅锚定 + 硬墙正好对上
- **BYOK 仍是付费档**（约 Pro 价位的 60%），只是 LLM 部分自付；Sandbox / Storage 由我们承担成本，不能白嫖

> 关于"为什么不抄 Cursor 直接 USD 透明计费"：Cursor 看似 1:1 透明，实际嵌了 ~20% markup + 吃订阅 breakage。我们沿用同样的 markup 嵌入策略（隐形 20%）但坚持硬墙 —— 不做 opt-in 按量超额，避免账单焦虑。

> 不做按量超额的取舍：放弃了一部分重度 builder 用户的"加钱继续"路径。补救：Max / Max+ 档位 LLM 余额定得宽松（¥80~¥200 等价 token），覆盖 95% 用户；剩下 5% 重度用户引导走 BYOK 通道（自己付 LLM、自由用量）。

## 命名：Free / Pro / Max / Max+

四档清晰递进，覆盖从试用到团队的全谱系：

```
Free     ← 免费试用（仅 DeepSeek，不可换模型，体验产品本身）
Pro      ← 主力档（绝大多数付费用户；解锁全模型自由切换）
Max      ← 高频 / 重度用户（更高额度 + 优先 sandbox）
Max+     ← 团队 / 企业 / 给 Opus 类高端模型预留
```

## 五档草案（v0 数字）

> 说明：
> - **可用模型**：Free 只锁定 DeepSeek（试用）；**Pro 及以上自由切换全部模型**（DeepSeek / Claude / GPT / Gemini）
> - **LLM 余额**单位是 RMB **分**，按 token 真实成本扣减（实时 cost = `tokens × 模型单价 × 1.20 markup`）。每模型一次 turn 扣多少分由真实 token 量决定 —— DeepSeek 短对话可能扣 1 分，Opus 长上下文一次扣 1500 分都正常
> - LLM 用 **5d 滚动主限 + 月度兜底**：任一触达即暂停到该窗口复位
> - **Sandbox**（每次 publish_box 触发）按月计数，**Storage**（active box 数）是点状硬上限
> - BYOK 不计 LLM 余额（cost 在用户自己 key 上），但 Sandbox / Storage 仍占平台成本，所以 BYOK 是**付费档**（不再是 ¥0）

| 档 | 月费 | LLM / 5d | LLM / 月 | Sandbox / 月 | Storage（active） | 可用模型 | 触达上限 |
|---|---|---|---|---|---|---|---|
| **Free** | ¥0 | ¥0.50 | ¥1 | 5 | 1 | **仅 DeepSeek** | 暂停到下个 5d / 引导升 Pro |
| **Pro** | ¥30 | ¥15 | ¥30 | 50 | 5 | **全模型自由切换** | 暂停到下个窗口 / 升 Max |
| **Max** | ¥80 | ¥40 | ¥80 | 200 | 20 | 全模型 + sandbox 优先队列 | 暂停 / 升 Max+ |
| **Max+** | ¥200 | 不限 | ¥200 | 500 | 不限 | + Opus 优先 / 私有部署 / team | 暂停到下月 / 联系销售 |
| **BYOK** | **¥18** | LLM 不计 | LLM 不计 | 50 | 5 | 自带 key 任意模型 | Sandbox / Storage 走 Pro 配额 |

> 数字 v0 草案。
> - LLM 月度余额 ≈ 月费扣除 sandbox/storage 摊销后剩下的部分（Pro ¥30 月费里 ~¥15 是 LLM 余额、~¥10 摊到 sandbox / 存储 / relay、~¥5 毛利）
> - BYOK ¥18 = Sandbox / Storage / Relay 真实成本 + 一点点利润（用户自付 LLM 那部分省下来的，作为我们卖基础设施的对价）
> - Max+ 取消 LLM 5d 限制只保留月限，覆盖企业级"集中冲刺一周"场景

## 三量纲（LLM / Sandbox / Storage）

每个量纲独立硬墙、独立计算、UI 独立显示。一个量纲耗尽不影响另外两个：LLM 配额满了 chat 暂停，但 publish 还能跑；Sandbox 满了不能发新版本，但 chat 不受影响。

### LLM —— 实时 RMB-cents 计费

每次 turn 完成时，引擎按 provider 返回的真实 token 量算成本：

```
cost_cents = ⌈(tokens_in × in_price + tokens_out × out_price) × 1.20⌉
```

- 价格表（`internal/usage/pricing.go`）：每模型每方向 RMB/M tokens，季度刷新
- markup `1.20` = 平台 20% 加价，覆盖 sandbox / relay / storage / 客服等不直接收费的成本
- ceil 到分，最低 1 分（避免成功 turn 因 rounding 免费搭便车）
- 历史 turn 不重算：cost 在 turn 完成时就落库到 `ai_turns.cost_cents`，价格表后续调整不影响

为什么不直接暴露 USD/RMB 余额：
- **UI 选择**：实际 cost 用户看得到（chat 完每条显示"本次消耗 ¥0.18"），但**总余额**用同一单位（"剩余 ¥12.80 / ¥15"）显示
- 跟 Cursor 现行模式一致，区别在于我们不开 opt-in 按量超额，用完即停

### Sandbox —— 按次计数

每次 `publish_box` 调用触发一次 sandbox docker → metro → bundle，落 1 行 `app_bundles`。配额按月归零。

为什么不也用 RMB：
- 单次成本方差小（¥0.05~¥0.30），按次更直观
- 失败的 build 也算 publish（消耗 docker 时间）—— 简化判断
- 用户 UI："还剩 32 次 publish"比"还剩 ¥3.20"清晰

### Storage —— 点状硬上限

`active_boxes`（state ≠ archived）的总数。新建 box 触达上限即拒绝，归档老 box 释放配额。

为什么不按 MB：
- 用户决策粒度是"项目数"不是"占多少 MB"
- bundle 平均 ~3MB，box 数是合理的存储成本代理

## BYOK ¥18/月

**为什么单独存在**：
- 吸引种子用户 / 极客用户 —— 这些人愿意自己申请 Anthropic 账号、付跨境费
- 我们 LLM 成本归零，只承担 sandbox + relay + 存储成本
- 病毒传播载体：BYOK 用户在社区分享 "appunvs 让我自带 key 用 Claude 做 RN app"，比付费用户传播力强
- **覆盖重度用户的"加钱继续"诉求**：放弃了 LLM 按量超额后，重度 builder 的逃生口在这里 —— BYOK 之下他们 token 用量不限

**BYOK 用户付什么**：
- 月费 **¥18**（不再是 ¥0）—— 因为我们仍然提供 Sandbox 编译 + Storage + Relay 同步等基础设施，这些都是真成本
- LLM 部分自付：用户在设置页填 Anthropic / OpenAI / Gemini 的 key（AES-GCM at rest，relay 转发时使用，不写日志）
- Sandbox / Storage 配额：50 publish + 5 box（与 Pro 同档）
- 想超过 5 个 box？升级到 Pro/Max（保留 BYOK，仅买更多 Sandbox / Storage 配额）

**风险**：
- 极客用户找到不付费替代方案（自建 relay 开源版）
- 缓解：开源 relay 不含 host shell + RuntimeSDK 的便利（每次手动编译 / 手动分发），愿意折腾的就让他们折腾，他们本来也不会变成大客户

> 早期版本的"BYOK ¥0/月"是错的设计 —— 平台不能白送 sandbox + storage + relay。¥18/月把 BYOK 变成"正常付费用户的 LLM 子模式"，平台不亏，用户拿到自由模型选择 + 不限 token。

## 成本测算（v3：LLM 实时计费）

Pro ¥30 用户的 ¥30 是怎么花掉的：

| 项 | 一个月成本 / 配额 | 备注 |
|---|---|---|
| LLM 余额（含在订阅）| **¥15**（即用户能消耗的 token cost ceiling）| 按 cost_cents 实时扣，超出即暂停 |
| Sandbox build（50 publish × ~¥0.10）| **¥5** | 阿里云 ECI / 本地 docker，按秒计 |
| Storage（5 box × bundle CDN + 元数据）| **¥1** | 对象存储 + CDN 摊销 |
| Relay shared infra（auth / /ws / /box/events）| **¥2** | 单机摊销 |
| **平台真实变动成本** | **~¥23** | |
| **毛利** | **~¥7（23%）** | 紧但不亏 |
| **+ breakage**（30% 用户用不完 LLM 余额）| **~¥4-5** | 真实毛利提升到 35-40% |

为什么之前 v2 的"亏损 ¥170"消失了：
- v2 用整数 turn 计数，全程 Opus 跑满 500 turn 不会 trip cap
- v3 用 RMB 余额：Opus 一次 cost ~¥1.40，¥15 余额只够 ~10 次 Opus，**额度自然用完即停**
- 用户想多用 Opus 就升档（Max ¥80 给到 ¥40 余额可跑 ~30 次 Opus）

Max+ ¥200 给重度用户：含 ¥120 LLM 余额，按 Opus 跑约 80 次，按 Sonnet 跑约 500 次，按 DeepSeek 不限速。私有部署 / team 协作功能是 Max+ 的核心溢价。

BYOK ¥18 的成本拆解：
| 项 | 月成本 |
|---|---|
| Sandbox 50 publish | ~¥5 |
| Storage 5 box | ~¥1 |
| Relay | ~¥2 |
| **平台真实成本** | **~¥8** |
| BYOK 月费 | **¥18** |
| **毛利** | **~¥10（55%）** |

BYOK 反而毛利更高 —— 因为不需要平台垫付 LLM 余额。这鼓励重度 builder 走这条路（自付 LLM 拿到无上限 token 用量），平台也乐见。

## 不要做的事

- ❌ **纯按量**（无订阅锚定）：用户每次 chat 都担心烧钱，停留时长锐减
- ❌ **opt-in 按量超额**（Cursor 模式）：增加客服压力 + 用户账单焦虑；坚持硬墙
- ❌ **纯订阅不限量**：1% 重度用户烧光所有利润；Bolt.new 早期吃过这亏
- ❌ **海外档位（$20）直接对标人民币 ¥150**：国内付费墙过高，¥30 是更合适的锚定
- ❌ **太多档**：超过 5 档用户决策瘫痪。Free / Pro / Max / Max+ / BYOK 已经是上限
- ❌ **Free 用户开放模型切换**：Free 是试用，不是低配版 —— 能切到 Claude / GPT 就没人付费了
- ❌ **限制 Free 用户用 publish**：publish 是产品核心 a-ha，让 Free 用户至少能完成一次完整 dogfood
- ❌ **三量纲塞进一个 RMB 池子**：LLM 高方差、Sandbox 低方差、Storage 持续累计 —— 成本结构不同就该独立量纲
- ❌ **BYOK 免月费**：早期错误设计，把 sandbox/storage/relay 真成本白送了；¥18 是平衡点

## 上线节奏

**Phase 1（dogfood，已完成）**：`cfg.Pricing.Enabled=false`，配额计数照跑、UI 照显示，但不硬墙
**Phase 2（小范围内测）**：开启 LLM 配额硬墙，先上 Free + Pro 两档；继续 dogfood 数据收集 Sandbox / Storage 真实分布
**Phase 3（公开发布）**：五档全开 + BYOK 通道；Stripe 订阅打通，users.plan 列入 Phase D 落地
**Phase 4（运营调优）**：
- 根据真实数据调价格表 markup（用户打满 Pro 的 LLM 月余额比例 ≥ 70% 是健康，太低则毛利过高、太高则用户体感差）
- 调每档 LLM 余额（Pro ¥15 / 5d 是否合适）
- 年付折扣（行业惯例 8.5 折）

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
| **appunvs** | ¥0.50 LLM / 5d + 1 box（仅 DeepSeek） | ¥30/月（Pro，全模型 + ¥15 LLM/5d） | LLM RMB-cents + Sandbox 次数 + Storage box 数 | **暂停到下个窗口 / 升档**（三量纲硬墙，无按量）|

我们的差异点：
- **国内价位**（¥30 ≈ 美元 $4，是 Lovable / Cursor 的 1/5）—— 国内市场更可负担
- **三量纲独立计费**：LLM / Sandbox / Storage 各管各的，一个用完不影响另外两个；竞品要么单池要么强行混算
- **LLM 实时 cost-cents**（跟 Cursor 一致透明度）+ **不开按量**（坚持 Claude 式硬墙，免账单焦虑）—— 跨模式取长补短
- **BYOK 通道** ¥18/月：用户自付 LLM、不限 token，平台只赚 sandbox / 基础设施
- **5d 滚动主限 + 月度兜底**：竞品要么纯月限（节奏感差）、要么 5h / 每日（builder 反工作流）—— 5d 单独成立
- **付费门槛清晰**：Free 锁 DeepSeek 试体验，Pro 即解锁全模型自由切换

## 实现复杂度

| 模块 | 按量模式 | 我们的模式（v3 三量纲硬墙）|
|---|---|---|
| LLM 单位 | 实时 USD ledger | **`ai_turns.cost_cents`** —— 引擎在 turn 完成时落库；查询是单 SUM |
| 模型价格表 | 每模型每方向 USD/1M | 同样需要 —— 在 `internal/usage/pricing.go`，季度刷新 |
| Stripe | 订阅 + metered usage | **仅订阅**（subscription only） |
| Ledger 表 | accrued_usd_cents 实时累加 | 不需要 —— `ai_turns.cost_cents` 已经是 ledger |
| 用户 UI | USD% + 按模型 turn 估算 + 按量开关 | 三栏并排（LLM / Sandbox / Storage），各自 used / limit |
| 按量结算 | 月底批量 + 错误处理 + 退款流程 | **不存在** |
| 客服压力 | "为什么我被扣了 ¥X？"问询 | **极低**（硬墙清晰，每条 chat 实时显示 cost）|

### 落地阶段（已完成 / 进行中 / 待办）

| Phase | 状态 | 内容 |
|---|---|---|
| A | ✅ 已合并 #45 | `store.Turns.CountByNamespace`（早期版本，按行计数）+ `internal/usage` 包初版 |
| B | ✅ 已合并 #46 | `/ai/turn` 入口前查配额，超额 429 + Retry-After |
| /usage/me | ✅ 已合并 #48 | Profile UI 进度条数据接口 |
| **v3 重构** | 🔄 当前 PR | LLM 改 cost_cents、`Plan` 三量纲、价格表、`CheckLLM/Sandbox/Storage` 拆分 |
| C | ⏳ 后续 | 在 `box.Service.BuildAndPublish` / `box.Create` 接 `CheckSandbox` / `CheckStorage` |
| D | ⏳ 后续 | Stripe 订阅 + Webhook → `users.plan` 列；`PlanFor` 闭包改成 DB lookup |
| E | ⏳ 后续 | BYOK 通道（`users.provider_keys` AES-GCM 加密 + 引擎转发逻辑）|

`/usage/me` 接口契约（v3）：

```json
{
  "plan":    { "id": "pro", "label": "Pro" },
  "llm":     { "used_cents_last_5d": 900, "used_cents_this_month": 1800,
               "budget_cents_per_5d": 1500, "budget_cents_per_month": 3000 },
  "sandbox": { "used_this_month": 18, "per_month": 50 },
  "storage": { "active": 4, "active_limit": 5 }
}
```

`-1` 在任意 limit 字段表示无上限（Max+ 不限 5d、BYOK 不限 LLM、Max+ 不限 Storage）。
