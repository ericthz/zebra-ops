# Zebra Ops

[English](./README.md) | 简体中文

Zebra Ops 是基于 **GoFrame v2** 与 **cloudwego/eino** AI 编排框架构建的运维助手服务，提供 RAG 知识库对话、文档向量化建库、以及基于 Plan-Execute-RePlan 多轮 Agent 的告警分析三类能力。

## 功能特性

| 能力 | 说明 | 入口 |
|---|---|---|
| 快速对话 | 请求-响应式对话，RAG 检索增强 + ReAct Agent 工具调用 | `POST /api/chat` |
| 流式对话 | SSE 逐 token 输出 | `POST /api/chat_stream` |
| 知识库管理 | 上传文档自动切分、向量化、入库，同源文件覆盖更新 | `POST /api/upload` |
| AI 运维分析 | 拉取告警 → 检索处理手册 → 查询关联日志 → 生成结构化报告（SSE 实时进度） | `POST /api/ai_ops_stream` |
| Agent 工具集 | 时间 / 内部文档检索 / 腾讯云 CLS 日志 MCP / Prometheus 告警 | ReAct 与 Plan-Execute-RePlan 内 |

## 系统架构

### 分层与请求链路

```
HTTP 请求
  → middleware（CORS → RequestID → AccessLog → Metrics → Response）
  → controller/chat（DTO 校验）
  → service（业务编排）
  → biz（AI 管线：chat_pipeline / knowledge_index_pipeline / plan_execute_replan）
  → components（llm / embedder / vectorstore / loader / tools）
```

- 统一响应封装：`{ "message": "...", "data": { ... } }`（`internal/middleware/middleware.go`）。
- 配置读取：GoFrame 配置组件（`g.Cfg().GetEffective`），密钥不进入版本库。

### 对话流水线 `internal/biz/chat_pipeline`

Eino 有向图（`orchestration.go` 的 `BuildChatAgent`）：

```
START
 ├─ InputToRag ──> MilvusRetriever ─┐
 ├─ InputToChat ────────────────────┼─> ChatTemplate ──> ReactAgent ──> END
                                    （系统 prompt + 历史 + RAG 上下文）
```

- `InputToRag` / `InputToChat` 并行，`AllPredecessor` 聚合模式。
- ReAct Agent（`flow.go` 的 `newReactAgentLambda`）绑定全部工具，`MaxStep = 25`，工具调用使用 Quick 模型。
- 会话历史存于内存（`internal/memory`）：滑动窗口 6 条（成对丢弃保配对）+ 24h TTL + 10000 条容量上限。

### 文档索引流水线 `internal/biz/knowledge_index_pipeline`

```
FileLoader ──> MarkdownSplitter（按 # 标题切分，uuid 生成文档 ID）──> MilvusIndexer
```

- 覆盖更新：`RebuildSource` 按 `metadata._source` 过滤删除旧数据后重建（上传与 CLI 共用）。

### AI 运维分析流水线 `internal/biz/plan_execute_replan`

标准 **Plan → Execute → RePlan** 多轮 Agent（`plan_execute_replan.go` 的 `BuildPlanAgent`，基于 eino `adk/prebuilt/planexecute`）：

- Planner / Replanner：Think 模型（强推理，拆解长链路）。
- Executor：Quick 模型（低延迟工具调用）+ 工具集；外层 `MaxIterations = 10`。
- Executor 工具集（共 10 个）：6 个 CLS MCP 工具子集（`FilterLogMcpTools`）+ `query_prometheus_alerts` + `query_internal_docs` + `get_current_time` + `respond` 兜底工具。
- 报告兜底：事件循环中通过 `isCompleteReport` 校验报告完整性并提前终止；若 Executor 误调用 `respond` 或 Replanner 直接返回报告，均经 `extractRespond` / `extractRespondFromToolCalls` 捕获，兜底优先级为 respond 报告 > `bestReport` > 末条 assistant 正文。
- `flexiblePlan`：兼容小模型将 `steps` 输出为字符串化数组或单字符串的情形，自定义反序列化后喂给 Planner / Replanner。

### Agent 工具集 `internal/components/tools`

| 工具 | 说明 | 注册范围 |
|---|---|---|
| `get_current_time` | 当前时间（秒/毫秒/微秒） | 对话 / AI Ops |
| `query_internal_docs` | 内部文档 RAG 检索 | 对话 / AI Ops |
| `query_log` | 腾讯云 CLS 日志 MCP（SSE） | 对话 / AI Ops |
| `query_prometheus_alerts` | Prometheus 活跃告警 | 对话 / AI Ops |
| `respond` | 兜底响应工具（避免 Executor 误调用报错） | 仅 AI Ops Executor |
| `mysql_crud` | MySQL 查询/写入 | 仅离线 CLI（`cmd/llm_tool`），服务端不注册 |

### MCP 集成 `internal/components/tools/query_log.go`

- `GetLogMcpTool` 通过 `sync.Once` 懒加载单例 `reconnectableToolGroup`，建立 mark3labs/mcp-go SSE 客户端，`Initialize` 后经 eino-ext `mcp.GetTools` 拉取工具列表。
- 会话过期自动重连：`reconnectableTool` 按 `toolName` 动态路由到最新连接；调用遇会话错误（`isSessionError`）触发异步 `tryReconnect`，`reconnecting` 原子标志保证仅重建一次。
- 优雅降级：CLS 工具返回 `IsError` 时经 `gracefulClsResultHandler` 转为「日志查询未返回数据（原因：…）请基于已获取的内部文档处理方案继续分析」文本结果，不中断 Agent。

### RAG 检索 `internal/components/vectorstore`

- 向量：`text-embedding-v4`，**2048 维 FloatVector**，`L2` 度量（Milvus 集合 `biz`，库 `agent`）。
- 切分：`markdown.NewHeaderSplitter` 按 `#` 标题层级切分，uuid 生成分片 ID。
- 检索：TopK = 1；indexer / retriever 自定义 converter 输出 `float32` 向量，修正 eino-ext 默认「字节打包进 BinaryVector + Hamming」的语义错误。
- 输出字段：`content`、`metadata`。

## 技术栈

| 层 | 技术 |
|---|---|
| HTTP | GoFrame v2.10.2（api / controller / service 分层） |
| AI 编排 | cloudwego/eino v0.9.15（Graph + ReAct Agent + ADK prebuilt） |
| LLM | OpenAI 兼容接口（默认火山方舟 DeepSeek-V4-flash，Think / Quick 双模型） |
| Embedding | OpenAI 兼容接口（默认阿里百炼 text-embedding-v4，2048 维） |
| 向量库 | Milvus 2.5.10（服务端），Go SDK v2.4.2（FloatVector + L2） |
| Agent 工具 | 腾讯云 CLS 日志 MCP（SSE）+ 自研 Prometheus / 时间 / 文档工具 |
| 可观测性 | slog JSON 日志 + 自研 Prometheus 指标 + 健康检查 |
| 前端 | 原生 HTML/CSS/JS，SSE 流式，亮/暗双主题，DOMPurify 防 XSS，`//go:embed` 嵌入二进制 |
| CI | GitHub Actions（vet + build + test） |
| 语言 | Go 1.26+ |

## 目录结构

```
Zebra Ops
├── .github/workflows/         # CI（vet + build + test）
├── api/chat/v1/               # GoFrame API DTO + g.Meta 路由标签
├── cmd/
│   ├── server/                # HTTP 服务入口（main.go + 内嵌 static 前端）
│   └── chat/ knowledge/ recall/ ai_ops/ llm_tool/   # 命令行工具
├── internal/
│   ├── controller/chat/       # HTTP Handler（参数校验 + 转发）
│   ├── service/               # 业务编排（chat / upload / sse）
│   ├── biz/
│   │   ├── chat_pipeline/            # 对话图（RAG + ReAct）
│   │   ├── knowledge_index_pipeline/ # 文档索引图
│   │   └── plan_execute_replan/      # AI Ops（Plan-Execute-RePlan）
│   ├── components/           # llm / embedder / vectorstore / loader / tools / callbacks
│   ├── memory/               # 内存会话（滑动窗口 + TTL）
│   ├── middleware/           # CORS / request_id / 访问日志 / 指标 / 统一响应
│   └── observability/        # 指标注册表 + 健康检查
├── manifest/
│   ├── config/               # config.yaml.example（config.yaml 已 gitignore）
│   ├── docker/               # Dockerfile（multi-stage）
│   └── deploy/milvus/        # Milvus docker-compose
├── docs/                     # 文档与知识库种子文档
├── mock/                     # 模拟告警数据（alerts.json 已 gitignore）
└── storage/uploads/          # 上传文件落盘（运行时生成）
```

## API 参考

统一前缀 `/api`，统一响应 `{ "message": "...", "data": { ... } }`；`/healthz` `/readyz` `/metrics` 不在 `/api` 前缀下。

| 方法 | 路径 | 入参 | 响应 data | 说明 |
|---|---|---|---|---|
| POST | `/api/chat` | `{ Id, Question }` | `{ answer }` | 快速对话（RAG + ReAct） |
| POST | `/api/chat_stream` | `{ Id, Question }` | SSE 事件流 | 流式对话（事件：`connected` / `message` / `done` / `error`） |
| POST | `/api/upload` | multipart（`file`） | `{ fileName, filePath, fileSize }` | 文档上传建库（覆盖更新） |
| POST | `/api/ai_ops` | `{ Id }` | `{ result, detail[] }` | AI 告警分析（同步，Plan Agent） |
| POST | `/api/ai_ops_stream` | `{ Id }` | SSE 事件流 | AI 告警分析（SSE 实时进度 + 报告） |
| GET | `/healthz` | - | `ok` | 存活探针 |
| GET | `/readyz` | - | `ok` / 503 | 就绪探针（短超时探测 Milvus） |
| GET | `/metrics` | - | Prometheus 文本 | 指标输出 |

示例：

```bash
curl -X POST http://localhost:6872/api/chat \
  -H 'Content-Type: application/json' \
  -d '{"Id": "session_1", "Question": "Zebra Ops 服务为什么下线？"}'
```

## 配置

### 配置加载机制

服务配置通过 GoFrame 配置组件从 `manifest/config/config.yaml` 读取。该文件已被 `.gitignore` 忽略，不入库；从 `config.yaml.example` 复制后填入本地实际值，敏感字段（API Key）在本地 `config.yaml` 中填写，密钥不进入版本库。

服务进程读取的操作系统环境变量仅 `LOG_DIR`（日志目录）。

独立运行的 CLS 日志 MCP 服务（`cls-mcp-server`）通过环境变量 `TRANSPORT` / `PORT` / `TENCENTCLOUD_SECRET_ID` / `TENCENTCLOUD_SECRET_KEY` / `TENCENTCLOUD_REGION` 配置，可用 `.env` + `export` 注入，与 Go 服务配置相互独立。

### 配置项清单

| config 键 | 环境变量 | 默认值 | 说明 |
|---|---|---|---|
| `server.address` | - | `:6872` | HTTP 监听（代码 `SetPort(6872)` 同值） |
| `think_chat_model.api_key` | `THINK_CHAT_MODEL_API_KEY` | 空 | Think 模型密钥（强推理/规划） |
| `think_chat_model.base_url` | - | `https://ark.cn-beijing.volces.com/api/v3` | 火山方舟 OpenAI 兼容端点 |
| `think_chat_model.model` | - | `DeepSeek-V4-flash` | Think 模型名 |
| `quick_chat_model.*` | `QUICK_CHAT_MODEL_API_KEY` | 同上 | Quick 模型（低延迟执行） |
| `embedding_model.api_key` | `EMBEDDING_MODEL_API_KEY` | 空 | 阿里百炼 DashScope 密钥 |
| `embedding_model.base_url` | - | `https://dashscope.aliyuncs.com/compatible-mode/v1` | Embedding 端点 |
| `embedding_model.model` | - | `text-embedding-v4` | Embedding 模型（2048 维） |
| `file_dir` | `FILE_DIR` | `./storage/uploads` | 上传落盘目录 |
| `milvus_url` | `MILVUS_URL` | `localhost:19530` | Milvus 地址 |
| `mcp_url` | - | `http://localhost:3000/sse` | CLS 日志 MCP SSE 端点 |
| `prometheus_url` | `PROMETHEUS_URL` | 空 | Prometheus API（留空则降级 Mock 告警） |

### Mock 告警模式

`query_prometheus_alerts` 在 `PROMETHEUS_URL` 为空时降级读取 `mock/alerts.json`，内置 4 类典型告警（服务下线 / 接口失败率过高 / 上下游对账差异 / 服务地域与资源地域不匹配），其 `activeAt` 按相对偏移动态生成，无需本地部署 Prometheus 即可完整体验 AI 运维分析。

## 可观测性

| 端点 / 机制 | 说明 |
|---|---|
| `/healthz` | 存活探针，返回 `ok` |
| `/readyz` | 就绪探针，3s 超时探测 Milvus 连通性，失败返回 503 |
| `/metrics` | 自研零依赖 Prometheus 注册表（counter + histogram），文本格式输出 |
| 指标 | `http_requests_total`（method/path/status）、`http_request_duration_seconds`（直方图桶 0.01–10s） |
| 访问日志 | `X-Request-Id` 生成/透传 + slog JSON 结构化输出（method/path/status/duration_ms/remote_addr） |

## 快速开始

### 1. 启动依赖（Milvus + Attu）

```bash
cd manifest/deploy/milvus
docker-compose up -d
# Milvus: localhost:19530；Attu 管理台: http://localhost:8000
```

### 2. 配置

```bash
cp manifest/config/config.yaml.example manifest/config/config.yaml
# 编辑 manifest/config/config.yaml，填入真实 API Key 与端点
```

### 3. 启动服务

```bash
go run ./cmd/server
# 监听 6872 端口，前端已内嵌到二进制，访问 http://localhost:6872
```

### 4. 命令行工具

```bash
go run ./cmd/knowledge   # 索引 docs/ 下 md 文档到知识库
go run ./cmd/chat        # 演示多轮对话
go run ./cmd/recall      # 演示向量召回
go run ./cmd/ai_ops      # 演示 AI 告警分析
go run ./cmd/llm_tool    # 演示 MCP + 自定义工具绑定
```

## 构建与部署

`manifest/docker/Dockerfile`（multi-stage）：Go 编译 + 精简运行镜像，静态资源内嵌，无需单独分发前端。

## 开发

- 单元测试：`go test ./...`
- CI（`.github/workflows/ci.yml`）：push / PR 触发 `go vet` + `go build` + `go test`，Milvus 集成测试在无环境时自动跳过。

## License

MIT
