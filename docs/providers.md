# AI Providers

appunvs 的 AI agent 同时支持三类后端：**Anthropic 原生 API**（Claude）、
**Gemini 原生 API**、以及一切 **OpenAI 兼容协议**端点（OpenAI 自己 / DeepSeek /
Moonshot / Zhipu / Dashscope / MiniMax 等）。OpenAI 兼容那一组共享同一个
engine（`OpenAIEngine`），通过 provider 注册表查 base_url + 默认 model；
Anthropic 与 Gemini 各有独立 engine（`AnthropicEngine` / `GeminiEngine`），
因为协议形态差得够多。

代码入口：[`relay/internal/ai/providers.go`](../relay/internal/ai/providers.go)（OpenAI-compat 注册表）/
[`anthropic_engine.go`](../relay/internal/ai/anthropic_engine.go) /
[`gemini_engine.go`](../relay/internal/ai/gemini_engine.go)

## 已内置的供应商

**原生 API（独立 engine）**：

| Backend | 厂商 | 默认模型 | 备注 |
| --- | --- | --- | --- |
| `anthropic` | Anthropic Claude | `claude-opus-4-1` | 当前默认指向 Anthropic 最新 Opus 别名；想省钱可手动切 `claude-sonnet-4-0` |
| `gemini` | Google Gemini | `gemini-2.5-pro` | 走 generativelanguage.googleapis.com，需要海外网络 |

**OpenAI-compat（共享 engine + 注册表）**：

| Provider ID | 厂商 | 默认模型 | 备注 |
| --- | --- | --- | --- |
| `openai` | OpenAI | `gpt-5.5` | OpenAI 官方 models 页当前推荐复杂推理与编码任务从 `gpt-5.5` 开始 |
| `deepseek` | DeepSeek | `deepseek-v4-pro` | `deepseek-chat` / `deepseek-reasoner` 已进入兼容别名阶段，并将在 2026-07-24 废弃 |
| `moonshot` | Moonshot（Kimi） | `kimi-k2.6` | Moonshot 首页当前标注的“最新最智能”模型 |
| `zhipu` | 智谱 GLM | `glm-4.7` | 智谱文档当前旗舰文本模型 |
| `dashscope` | 阿里百炼（Qwen） | `qwen3-coder-next` | 阿里 Qwen-Coder 文档当前首选推荐 |
| `minimax` | MiniMax（海螺） | `MiniMax-M2.7` | MiniMax 2026-03 发布的最新文本旗舰 |

每家的 API key 约定（方便环境变量统一）：

| Provider | 环境变量（约定） |
| --- | --- |
| Anthropic  | `ANTHROPIC_API_KEY` |
| Gemini     | `GEMINI_API_KEY` |
| OpenAI     | `OPENAI_API_KEY` |
| DeepSeek   | `DEEPSEEK_API_KEY` |
| Moonshot   | `MOONSHOT_API_KEY` |
| Zhipu      | `ZHIPU_API_KEY` |
| Dashscope  | `DASHSCOPE_API_KEY` |
| MiniMax    | `MINIMAX_API_KEY` |

> appunvs 本身读 `APPUNVS_AI_API_KEY` / `APPUNVS_AI_MODEL` 这种统一命名；
> 上面的变量只是各家 SDK / CLI 的默认约定，方便复用现有环境。

## 配置

`config.yaml`（或环境变量 `APPUNVS_AI_*`）：

```yaml
ai:
  backend:   deepseek              # 供应商 id，或 "stub" 关掉 AI 走回声
  api_key:   ${DEEPSEEK_API_KEY}
  # 以下三项都可选；省略就用 providers.go 里的默认值
  base_url:  ""                    # 只在自建代理 / 私有部署时覆盖
  model:     ""                    # 只在换模型时覆盖
  max_iters: 10
  max_tokens: 8000
```

### 常见切换示例

**默认 DeepSeek（什么都不用多配）**：

```yaml
ai:
  backend: deepseek
  api_key: ${DEEPSEEK_API_KEY}
```

**智谱 GLM**：

```yaml
ai:
  backend: zhipu
  api_key: ${ZHIPU_API_KEY}
  # model 留空 → 默认 glm-4.7
```

**阿里百炼 Qwen3**：

```yaml
ai:
  backend: dashscope
  api_key: ${DASHSCOPE_API_KEY}
  # model 留空 → 默认 qwen3-coder-next；若偏通用对话可手动换 qwen3-max
```

**Moonshot Kimi**：

```yaml
ai:
  backend: moonshot
  api_key: ${MOONSHOT_API_KEY}
  # model 留空 → 默认 kimi-k2.6
```

**OpenAI（GPT-5.5）**：

```yaml
ai:
  backend: openai
  api_key: ${OPENAI_API_KEY}
  # model 留空 → 默认 gpt-5.5；更省钱可换 gpt-5.4-mini
```

**MiniMax 海螺**：

```yaml
ai:
  backend: minimax
  api_key: ${MINIMAX_API_KEY}
  # model 留空 → 默认 MiniMax-M2.7
```

**Anthropic Claude**（走原生 API，不在 registry 里）：

```yaml
ai:
  backend: anthropic
  api_key: ${ANTHROPIC_API_KEY}
  # model 留空 → 默认 claude-opus-4-1；省钱可换 claude-sonnet-4-0
```

**Google Gemini**（走原生 API，不在 registry 里）：

```yaml
ai:
  backend: gemini
  api_key: ${GEMINI_API_KEY}
  # model 留空 → 默认 gemini-2.5-pro
```

**自建 OpenAI 兼容端点（例如内部代理 / 未来新供应商）**：

```yaml
ai:
  backend:  openai-compatible        # 触发"原始模式"，跳过 registry
  base_url: https://your.proxy.example/v1
  api_key:  ${YOUR_API_KEY}
  model:    your-model-id
```

## 增加一个供应商

1. 在 `relay/internal/ai/providers.go` 里 `Providers` map 加一行：

   ```go
   "newprovider": {
       ID:        "newprovider",
       Name:      "New Provider Display Name",
       BaseURL:   "https://api.newprovider.example/v1",
       ModelChat: "recommended-chat-model-id",
       DocsURL:   "https://docs.newprovider.example",
       EnvAPIKey: "NEWPROVIDER_API_KEY",
   },
   ```

2. 若它不是严格 OpenAI 兼容（工具调用协议、字段名差别），先用 `curl` 或者
   go-openai 本地试一下：
   - `POST /chat/completions` + `tools: [...]` + `stream: true` 能不能正常出 `tool_calls` delta
   - 如果不能，暂时不要加进 registry；需要在 `openai_engine.go` 里加特殊
     case（比如厂商特有的流式帧分支）
3. 跑 `go test ./internal/ai/` ——`TestProviderRegistryShape` 会强制要求
   你把新供应商的必填字段（Name / BaseURL / EnvAPIKey）填全

## 换模型 vs 换供应商

| 目标 | 操作 |
| --- | --- |
| 同供应商换模型（如 `deepseek-v4-pro` → `deepseek-v4-flash`） | 只改 `ai.model` |
| 换供应商 | 改 `ai.backend` + 重新设 `ai.api_key` |
| 自定义端点 / 代理 | `backend: openai-compatible` + 显式 `base_url` + `model` |

## 路由（多模型调度）

**当前版本不做跨供应商运行时路由**——一个 relay 进程只绑定一个 Provider。
原因：

- MVP 阶段不同模型的成本/质量差异比路由复杂度更值得优化的点
- 多供应商路由需要处理 rate limit 熔断、跨厂商 retry、成本归集、降级策略
- 单 provider 已经能覆盖 90% 用户需求

需要"聪明路由"时的升级路径（不必现在做）：

1. 实例化多个 engine（每个 provider 一份），都实现 `Engine` 接口
2. 写一个 `RouterEngine` 组合它们，按 turn 的上下文特征（需要 reasoning？
   要视觉？fast-apply 类小编辑？）挑选委托目标
3. 路由表放 Redis 方便运行时调整

这一步的代码面积**不在现在的计划内**，`Config.Provider` 留出了钩子，
可以无痛接入。
