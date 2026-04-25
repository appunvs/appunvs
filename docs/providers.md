# AI Providers

appunvs 的 AI agent 走 OpenAI 兼容协议，支持通过配置切换后端供应商。内置
五家主流国内 LLM 的连接信息（BaseURL / 默认 model / API key 约定），但
任何其他 OpenAI 兼容端点都可以通过直接提供 `base_url` + `model` 接入。

代码入口：[`relay/internal/ai/providers.go`](../relay/internal/ai/providers.go)

## 已内置的供应商

| Provider ID | 厂商 | 默认模型 | 备注 |
| --- | --- | --- | --- |
| `deepseek` | DeepSeek | `deepseek-chat` | 默认选择，最便宜；`deepseek-reasoner` 可换思考模式 |
| `volcengine` | 火山方舟（Ark） | *需用户指定* | 用的是账号下自建的**接入点 id**（`ep-YYYYMMDD-xxx`），需在 Ark 控制台先创建 |
| `moonshot` | Moonshot（Kimi） | `kimi-k2-turbo-preview` | 长 context，工具调用稳 |
| `zhipu` | 智谱 GLM | `glm-4.6` | 国内 agent 生态里工具调用最稳的一档 |
| `dashscope` | 阿里百炼（Qwen） | `qwen3-coder-plus` | Qwen3 coder 原厂入口 |

每家的 API key 约定（方便环境变量统一）：

| Provider | 环境变量（约定） |
| --- | --- |
| DeepSeek   | `DEEPSEEK_API_KEY` |
| Volcengine | `ARK_API_KEY` |
| Moonshot   | `MOONSHOT_API_KEY` |
| Zhipu      | `ZHIPU_API_KEY` |
| Dashscope  | `DASHSCOPE_API_KEY` |

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
