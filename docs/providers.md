# AI Providers

appunvs 的 AI agent 有**两条 engine 通道**：

1. **OpenAI-compatible 通道** —— 走 OpenAI Chat Completions 协议，覆盖国内国外大部分 LLM
2. **Anthropic 通道** —— 直连 Claude，独立 engine（Anthropic Messages API 格式不一样）

通过 `APPUNVS_AI_BACKEND` 切换。

代码入口：
- [`relay/internal/ai/providers.go`](../relay/internal/ai/providers.go) —— OpenAI-compat 通道的 provider 注册表
- [`relay/internal/ai/anthropic_engine.go`](../relay/internal/ai/anthropic_engine.go) —— Anthropic 独立 engine

## OpenAI-compat 通道：内置 provider

| Provider ID | 厂商 | 默认模型 | 备注 |
| --- | --- | --- | --- |
| `deepseek` | DeepSeek | `deepseek-chat` | 默认选择，最便宜；`deepseek-reasoner` 可换思考模式 |
| `openai` | OpenAI | `gpt-4o` | `o3-mini` / `o3` 可换 reasoning |
| `gemini` | Google Gemini | `gemini-2.5-pro` | 走 Google 提供的 OpenAI-compat shim；想用 native Gemini API 等独立 GeminiEngine |
| `volcengine` | 火山方舟（Ark） | *需用户指定* | 用的是账号下自建的**接入点 id**（`ep-YYYYMMDD-xxx`），需在 Ark 控制台先创建 |
| `moonshot` | Moonshot（Kimi） | `kimi-k2-turbo-preview` | 长 context，工具调用稳 |
| `zhipu` | 智谱 GLM | `glm-4.6` | 国内 agent 生态里工具调用最稳的一档 |
| `dashscope` | 阿里百炼（Qwen） | `qwen3-coder-plus` | Qwen3 coder 原厂入口 |

每家的 API key 约定（方便环境变量统一）：

| Provider | 环境变量（约定） |
| --- | --- |
| DeepSeek   | `DEEPSEEK_API_KEY` |
| OpenAI     | `OPENAI_API_KEY` |
| Gemini     | `GEMINI_API_KEY` |
| Volcengine | `ARK_API_KEY` |
| Moonshot   | `MOONSHOT_API_KEY` |
| Zhipu      | `ZHIPU_API_KEY` |
| Dashscope  | `DASHSCOPE_API_KEY` |

> appunvs 本身读 `APPUNVS_AI_API_KEY` / `APPUNVS_AI_MODEL` 这种统一命名；
> 上面的变量只是各家 SDK / CLI 的默认约定，方便复用现有环境。

## Anthropic 通道：直连 Claude

跟 OpenAI-compat 通道平行的另一条路 —— Anthropic Messages API 协议跟 OpenAI 不兼容（system prompt 是 top-level 字段、tool result 是 user message 内的 content block 等等），所以是独立 engine。

| Backend | 默认模型 | 备注 |
| --- | --- | --- |
| `anthropic` | `claude-sonnet-4-6` | 直连 Anthropic Messages API |

可换模型：`claude-opus-4-7` / `claude-haiku-4-5-*` / `claude-sonnet-4-5` 等，在 `APPUNVS_AI_MODEL` 里指定。

环境变量约定：`ANTHROPIC_API_KEY`（appunvs 仍读 `APPUNVS_AI_API_KEY`）。

切换示例：
```yaml
ai:
  backend: anthropic
  api_key: ${ANTHROPIC_API_KEY}
  # model 留空 → 默认 claude-sonnet-4-6
  # model: claude-opus-4-7        # 顶配，token 单价 ~30x DeepSeek
```

> 想用 Claude 又想要 OpenAI-compat 接口？通过 `openai-compatible` backend 指向第三方代理（如 [OpenRouter](https://openrouter.ai/)、[Helicone](https://helicone.ai/)）。但**直连 Anthropic 通道有 cache_control（system prompt + tools 缓存 ~90% 折扣）和原生 tool use 支持**，强烈推荐直连。

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

**OpenAI（GPT-4o / o3）**：

```yaml
ai:
  backend: openai
  api_key: ${OPENAI_API_KEY}
  # model 留空 → 默认 gpt-4o；要 reasoning 模型：
  # model: o3-mini
  # model: o3
```

**Google Gemini**：

```yaml
ai:
  backend: gemini
  api_key: ${GEMINI_API_KEY}
  # model 留空 → 默认 gemini-2.5-pro；要 flash 提速：
  # model: gemini-2.5-flash
```

> 走的是 Google 的 OpenAI-compat shim。如果你需要原生 Gemini 函数调用 / 多模态特性，等专门的 GeminiEngine。

**智谱 GLM**：

```yaml
ai:
  backend: zhipu
  api_key: ${ZHIPU_API_KEY}
  # model 留空 → 默认 glm-4.6；要 glm-4.6-air 就覆盖：
  # model: glm-4.6-air
```

**阿里百炼 Qwen3**：

```yaml
ai:
  backend: dashscope
  api_key: ${DASHSCOPE_API_KEY}
  # model 留空 → 默认 qwen3-coder-plus；要换成 qwen3-max：
  # model: qwen3-max
```

**火山方舟（Ark）**——必须指定 `model`：

```yaml
ai:
  backend: volcengine
  api_key: ${ARK_API_KEY}
  model:   ep-20260424-abc123          # 你在 Ark 控制台创建的接入点 id
```

**Moonshot Kimi**：

```yaml
ai:
  backend: moonshot
  api_key: ${MOONSHOT_API_KEY}
  # model 留空 → 默认 kimi-k2-turbo-preview
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
| 同供应商换模型（如 `deepseek-chat` → `deepseek-reasoner`） | 只改 `ai.model` |
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
