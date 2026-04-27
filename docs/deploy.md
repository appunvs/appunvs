# Deploy & dogfood guide

最小路径：在你自己的 mac/pc 起一个 relay + redis + sandbox image，手机 app 走 LAN 连过来。20 分钟内能跑通 chat → AI 写代码 → publish → Stage 热重载的完整闭环。

后续上云的部分在文末「公网 / 多机部署」。

---

## 1. 前置依赖

| 工具 | 装哪儿 | 用途 |
|---|---|---|
| `go` ≥ 1.24 | 你的 dev 机 | 编译 relay |
| `docker` | 你的 dev 机 | 跑 redis + sandbox image |
| `node` 22 + `npm` | 你的 dev 机 | 构建 sandbox image（里面装 metro / RN deps） |
| AI provider key | env var | 让 chat 真的能调模型 |
| iOS simulator / Android emulator / 真机 | 同 wifi | 装 host app 验证 |

AI provider key 三选一（按可获得性排序）：
- **DeepSeek**（[platform.deepseek.com](https://platform.deepseek.com)）—— 国内付款方便，便宜；走 OpenAI-compat 协议
- **Anthropic Claude**（[console.anthropic.com](https://console.anthropic.com)）—— 跨境付款 + 国外网络；写代码工具调用最强
- **火山 Ark / 阿里百炼 / Moonshot / 智谱**—— 国内同 DeepSeek 形态

---

## 2. 五步起跑

### 2.1 克隆仓库 + 准备 .env

```bash
git clone <repo>
cd appunvs/relay
cp .env.example .env  # 没有就直接 vim .env
```

`.env` 内容示例（**不要 commit**）：

```bash
# 选一种 backend
APPUNVS_AI_BACKEND=anthropic
APPUNVS_AI_API_KEY=sk-ant-xxx
APPUNVS_AI_MODEL=claude-sonnet-4-6   # 可选

# 或者用 DeepSeek
# APPUNVS_AI_BACKEND=deepseek
# APPUNVS_AI_API_KEY=sk-xxx
# APPUNVS_AI_MODEL=deepseek-chat

# 第一次跑想确认线路通就用 stub，chat 会回声
# APPUNVS_AI_BACKEND=stub
```

### 2.2 一键拉起 relay + redis + sandbox image

```bash
bash scripts/dev-up.sh
```

脚本会：
1. 起 redis 容器（如已运行则跳过）
2. 检查 `appunvs/sandbox:latest` 镜像，缺则调 `runtime/packaging/build-sandbox.sh` 现 build（**第一次 5-8 分钟，npm install RN deps**）
3. `go run ./cmd/server` 启动 relay
4. 输出你 mac 的 LAN IP，即 host app 该连的地址

成功的 startup log 关键行：

```
INFO  ai engine wired      backend=anthropic   model=claude-sonnet-4-6
INFO  sandbox wired        backend=docker      image=appunvs/sandbox:latest
INFO  stage pipeline wired artifact_backend=local
INFO  relay listening      addr=:8080
```

任一行报 fatal 就别急着改 host app —— 修对了再继续。常见报错见 §5。

### 2.3 验 relay 通

另开终端：

```bash
curl http://localhost:8080/health
# 期望: ok
```

### 2.4 配置 host app endpoint

iOS：
```swift
// appunvs/ios/Runtime/Net/Config.swift
static let relayBaseURL = URL(string: "http://192.168.x.x:8080")!  // 你的 LAN IP
```

Android：
```kotlin
// appunvs/android/.../net/NetConfig.kt
const val relayBaseURL = "http://192.168.x.x:8080"
```

> **注意** Android 9+ 默认禁明文 HTTP，先加 `android:usesCleartextTraffic="true"` 到 AndroidManifest 的 `<application>`，仅 dev 用，prod 别留。

### 2.5 装 host app + 跑

```bash
# iOS：开 Xcode
open appunvs/ios/Runtime.xcworkspace
# Cmd-R 跑模拟器或真机

# Android
cd appunvs/android && gradle installDebug
```

注册一个账号（Login 屏 → 注册 tab），创建一个 box，进 Chat tab 发条消息。

**期望**：AI 回复流式输出 → 触发 `publish_box` 工具 → relay log 里看到 `BuildAndPublish` → 30s 后 box 列表自动刷新 → 切到 Stage tab 看新 bundle 渲染（fixture 默认是黑底白字 "Hello from D3.c"，AI 写新代码后变 AI 写的内容）

---

## 3. 三种 AI backend 速查

| Backend | env var 配法 | 模型默认 |
|---|---|---|
| `stub` | 只设 `APPUNVS_AI_BACKEND=stub` | 无 —— chat 回声 |
| `anthropic` | `APPUNVS_AI_BACKEND=anthropic` + `APPUNVS_AI_API_KEY=sk-ant-...` | `claude-sonnet-4-6` |
| `deepseek` | `APPUNVS_AI_BACKEND=deepseek` + `APPUNVS_AI_API_KEY=sk-...` | `deepseek-chat` |
| `volcengine` | `..._BACKEND=volcengine` + `..._API_KEY=...` + `..._MODEL=ep-...` | 无（必填） |
| `moonshot` / `zhipu` / `dashscope` | 同上模式 | 各家固定值 |
| 自定义 OpenAI-compat | `..._BACKEND=openai-compatible` + `..._BASE_URL=...` + `..._MODEL=...` + `..._API_KEY=...` | 无（必填） |

---

## 4. 文件 / 数据落在哪

| 数据 | 路径（默认） | 备注 |
|---|---|---|
| SQLite（账号、box 元、AI turns） | `relay/data/relay.db` | 单机够用；备份直接拷文件 |
| Box git workspaces | `relay/data/workspaces/<box_id>/` | bare repo，每个 box 一个 |
| 编出来的 bundle artifacts | `relay/data/artifacts/<sha256>/` | content-addressed |
| Redis 数据 | `appunvs-redis` 容器内 `/data` | 仅 Stream + seq；丢了不致命 |

清空一切重来：`rm -rf relay/data && docker rm -f appunvs-redis`

---

## 5. 排障 checklist

| 症状 | 原因 | 修法 |
|---|---|---|
| `sandbox.DockerBuilder: image "appunvs/sandbox:latest" not found` | 没 build sandbox image | `bash runtime/packaging/build-sandbox.sh` |
| `sandbox.DockerBuilder: docker not on PATH` | 容器化部署忘装 docker CLI；本机部署没装 | 直接装 docker 或换 `APPUNVS_SANDBOX_BACKEND=stub` |
| `ai: AnthropicConfig.APIKey required` | 没设 `APPUNVS_AI_API_KEY` | `.env` 写好 / export |
| `ai: turn aborted ... 401 Unauthorized` | API key 错或失效 | 控制台重生成 key |
| `redis ping failed at startup`（warn） | redis 没起 | `docker ps | grep appunvs-redis` |
| host app 连不上 relay | LAN IP 写错 / mac 防火墙拦了 8080 / Android 没开 cleartext | 见 §2.4 |
| chat 流式但 publish 后 Stage 不刷新 | sandbox 真的 build 失败了，bundle URL 没更新 | 看 relay log 里 `box.publish` 那段 |
| iOS 真机无法装 | 没配开发者账号 / 没 trust developer | Xcode → Signing & Capabilities |

---

## 6. 公网 / 多机部署（暂未必要）

LAN 跑通之后可以考虑：

- **走 tunnel 让外部访问 dev 机**：[cloudflared](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/) 或 [ngrok](https://ngrok.com/) 5 分钟，免维护
- **上云单 VM**：阿里云 ECS / 腾讯云 CVM，直接 `git clone` + 这套脚本同样能跑；Caddy 在前面挂 HTTPS（[Caddyfile 例子](#caddy-example)）
- **多实例**：当前 `box.Events` 是 in-memory，多 relay 实例间事件不会 fanout —— 上多机前先把它换成 Redis pub/sub（独立 PR）

### Caddy example

```caddyfile
your.domain.com {
    reverse_proxy localhost:8080
}
```

`caddy run --config Caddyfile` 即可，Let's Encrypt 证书自动签发。

---

## 7. 你 dogfood 时建议这样验

1. **stub 先跑通线路**（无 AI key，无 sandbox image）—— 注册、创建 box、发 chat、看回声。证明 host ↔ relay ↔ redis 链路通。
2. **加 sandbox + AI key 后再跑一遍**：发 "把 RuntimeRoot 改成红色" 之类小指令，看 chat 输出 + box 列表 refresh + Stage 重挂 bundle，验证完整闭环。
3. **把任何踩到的问题加进 §5 的 checklist** —— 这表会越来越值钱。
