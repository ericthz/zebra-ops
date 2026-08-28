# Zebra Ops · 技术文档

> 面向开发：系统架构、模块划分、管线实现、API 契约、配置与部署、可观测性。

---

## 1. 系统概览

Zebra Ops 基于 **GoFrame v2** 构建 HTTP 服务，使用 **cloudwego/eino** 作为 AI 编排框架（Graph / ReAct / Plan-Execute-RePlan / ADK prebuilt），对接 LLM、Embedding、Milvus 向量库与 CLS 日志 MCP 工具，提供「RAG 知识库对话 + 文档上传建库 + AI 告警分析」三类能力。

---

## 2. 技术栈

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
| 语言 | Go 1.26+ |

---

## 3. 目录结构与分层

```
internal/
 ├── controller/chat/       # HTTP Handler（参数校验 + 转发 service）
 ├── service/               # 业务编排（chat / upload / sse）
 ├── biz/
 │   ├── chat_pipeline/            # 对话图（RAG + ReAct）
 │   ├── knowledge_index_pipeline/ # 文档索引图
 │   └── plan_execute_replan/      # AI Ops（Plan-Execute-RePlan）
 ├── components/            # llm / embedder / vectorstore / loader / tools / callbacks
 ├── memory/                # 内存会话（滑动窗口 + TTL）
 ├── middleware/            # CORS / request_id / 访问日志 / 指标 / 统一响应
 └── observability/         # 指标注册表 + 健康检查
```

请求链路：`middleware → controller/chat → service → biz → components`。

---

## 4. 后端架构

### 4.1 中间件链（`cmd/server/main.go`）

注册顺序：`CORSMiddleware → RequestIDMiddleware → AccessLogMiddleware → MetricsMiddleware → ResponseMiddleware`。

- `ResponseMiddleware` 位于链尾，先 `Next()` 再写入统一响应 `{ message, data }`（`internal/middleware/middleware.go`）。
- `RequestIDMiddleware` 生成/透传 `X-Request-Id`；`AccessLogMiddleware` 输出 slog JSON（含 method/path/status/duration_ms/remote_addr）。

### 4.2 对话管线（`internal/biz/chat_pipeline`）

Eino 有向图 `BuildChatAgent`（`orchestration.go`）：

```
START
 ├─ InputToRag ──> MilvusRetriever ─┐
 ├─ InputToChat ────────────────────┼─> ChatTemplate ──> ReactAgent ──> END
                                    （系统 prompt + 历史 + RAG 上下文）
```

- 节点：`InputToRag` / `InputToChat` / `MilvusRetriever` / `ChatTemplate` / `ReactAgent`，编译模式 `AllPredecessor`。
- ReAct Agent（`flow.go` 的 `newReactAgentLambda`）`MaxStep = 25`，工具调用模型为 Quick 模型（`NewDeepSeekQuickModel`）。
- 会话历史（`internal/memory/memory.go`）：窗口 6 条（成对丢弃保配对）、`MemoryTTL = 24h`、容量上限 10000 条（`SimpleMemoryMap` 全局存储，惰性淘汰最久未访问）。

### 4.3 文档索引管线（`internal/biz/knowledge_index_pipeline`）

```
FileLoader ──> MarkdownSplitter（按 # 标题切分，uuid 生成 ID）──> MilvusIndexer
```

- 切分器 `markdown.NewHeaderSplitter`，按 `#` 标题层级切分，无固定 token size / overlap 参数（`transformer.go`）。
- 覆盖更新 `RebuildSource`（`reindex.go`）：取首个文档 `metadata._source` 构造过滤表达式 `metadata["_source"] == "..."`，删除旧数据后重建；上传与 `cmd/knowledge` 共用。

### 4.4 AI Ops 管线（`internal/biz/plan_execute_replan`）

基于 eino `adk/prebuilt/planexecute.New` 组装（`plan_execute_replan.go` 的 `BuildPlanAgent`）：

- 外层 `MaxIterations = 10`；Planner / Replanner 使用 Think 模型（`NewDeepSeekThinkModel`），Executor 使用 Quick 模型（`NewDeepSeekQuickModel`，内部工具循环上限极大值，实际由报告产出提前终止）。
- **Executor 工具集（10 个，`executor.go`）**：
  1. `FilterLogMcpTools(GetLogMcpTool())` — CLS 工具子集（6 个）
  2. `query_prometheus_alerts`
  3. `query_internal_docs`
  4. `get_current_time`
  5. `respond` 兜底工具
- **报告兜底与提前终止**（均在 `plan_execute_replan.go`）：
  - `extractRespond` 从 Replanner 的 `{ response: ... }` 提取；`extractRespondFromToolCalls` 从 Executor 误调用的 `respond` 参数提取。
  - `isReportContent` 标记报告（含「告警分析报告」/「告警运维分析报告」/「# 告警处理详情」）；`isCompleteReport` 判定完整度（长度 ≥ 500 且含「处理方案」）。
  - 事件循环中 `isCompleteReport(bestReport)` 为真即返回，避免无效 RePlan 轮次。
  - 兜底优先级：respond 报告 > `bestReport` > 末条 assistant 正文 > 末条消息。
- `flexiblePlan`（`plan_execute_replan.go`）：自定义 `UnmarshalJSON` 兼容小模型将 `steps` 输出为字符串化数组或单字符串，供 `NewFlexiblePlan` 喂给 Planner / Replanner。

### 4.5 工具集（`internal/components/tools`）

| 文件 | 工具 | 功能 | 服务端注册 |
|---|---|---|---|
| `get_current_time.go` | `get_current_time` | 当前时间（秒/毫秒/微秒） | 是 |
| `query_metrics_alerts.go` | `query_prometheus_alerts` | Prometheus 活跃告警（空 URL 降级 Mock） | 是 |
| `query_internal_docs.go` | `query_internal_docs` | 内部文档 RAG 检索 | 是 |
| `query_log.go` | `query_log`（CLS MCP SSE） | 腾讯云日志查询 | 是 |
| `respond_fallback.go` | `respond` | 兜底响应（避免 Executor 误调用报错） | 仅 AI Ops Executor |
| `mysql_crud.go` | `mysql_crud` | MySQL 查询/写入（DSN + SQL） | 否，仅 `cmd/llm_tool` / 测试 |

对话管线挂载全部 CLS 工具 + alerts + time + docs（无 `respond`）；AI Ops 管线经 `FilterLogMcpTools` 仅挂载 CLS 子集以降低小模型工具调用出错率。

### 4.6 MCP 对接（`internal/components/tools/query_log.go`）

- `GetLogMcpTool`：`sync.Once` 懒加载单例 `reconnectableToolGroup`；`doConnectLocked` 建立 mark3labs/mcp-go SSE 客户端，`Initialize` 后经 eino-ext `mcp.GetTools` 拉取工具列表（传入 `gracefulClsResultHandler`）。
- 会话过期重连：`reconnectableTool` 按 `toolName` 动态路由到 `group.getTool`；调用遇 `isSessionError`（匹配 Session not found / SSE closed / transport error）触发异步 `tryReconnect`，`reconnecting` 原子标志保证仅重建一次。
- 优雅降级 `gracefulClsResultHandler`：MCP 结果 `IsError` 时转为「日志查询未返回数据（原因：…）请基于已获取的内部文档处理方案继续分析，无需重试日志查询」文本结果，不终止 Agent。
- `FilterLogMcpTools` 保留的 CLS 子集（`clsAIOpsTools`）：`SearchLog` / `DescribeLogContext` / `DescribeLogHistogram` / `TextToSearchLogQuery` / `ConvertTimeStringToTimestamp` / `ConvertTimestampToTimeString`。

### 4.7 RAG / 向量层（`internal/components/vectorstore`）

- 维度 **2048**（`embedder.go` `dim = 2048`；`client.go` 字段 `vector` `dim: 2048`），度量 **L2**（`indexer.go` / `retriever.go`）。
- 集合名 `biz`，库名 `agent`；字段 `id`(VarChar 主键) / `vector`(FloatVector) / `content`(VarChar 8192) / `metadata`(JSON)。
- 检索 `TopK = 1`（`retriever.go`）；indexer `DocumentConverter` 将 `float64` 向量转 `float32` 写入 FloatVector，retriever `VectorConverter` 输出 `entity.FloatVector`，修正 eino-ext 默认「字节打包进 BinaryVector + Hamming」的语义错误；输出字段 `content`、`metadata`。

### 4.8 会话记忆（`internal/memory`）

内存存储 `SimpleMemoryMap`，`GetSimpleMemory` 返回滑动窗口实现：窗口 6 条（成对丢弃保持 user/assistant 配对）、`MemoryTTL = 24h` 惰性清理、容量上限 10000 条（超额淘汰最久未访问）。

---

## 5. API 契约

统一前缀 `/api`，统一响应 `{ message, data }`；`/healthz` `/readyz` `/metrics` 不在 `/api` 下。

| 方法 | 路径 | 请求 | 响应 data | 说明 |
|---|---|---|---|---|
| POST | `/api/chat` | `{ Id, Question }` | `{ answer }` | 快速对话（RAG + ReAct） |
| POST | `/api/chat_stream` | `{ Id, Question }` | SSE 事件流 | 流式对话（事件：`connected` / `message` / `done` / `error`） |
| POST | `/api/upload` | multipart（`file`） | `{ fileName, filePath, fileSize }` | 文档上传建库（覆盖更新） |
| POST | `/api/ai_ops` | `{ Id }` | `{ result, detail[] }` | AI 告警分析（Plan Agent） |
| GET | `/healthz` | - | `ok` | 存活探针 |
| GET | `/readyz` | - | `ok` / 503 | 就绪探针（探测 Milvus） |
| GET | `/metrics` | - | Prometheus 文本 | 指标输出 |

---

## 6. 配置

配置文件 `manifest/config/config.yaml`（模板 `config.yaml.example`），经 GoFrame 配置组件读取，敏感字段在本地 `config.yaml` 填写（该文件已 gitignore）。服务进程仅读取操作系统环境变量 `LOG_DIR`。

| config 键 | 环境变量 | 默认值 | 说明 |
|---|---|---|---|
| `server.address` | - | `:6872` | HTTP 监听（代码 `SetPort(6872)` 同值） |
| `think_chat_model.api_key` | `THINK_CHAT_MODEL_API_KEY` | 空 | Think 模型密钥 |
| `think_chat_model.base_url` | - | `https://ark.cn-beijing.volces.com/api/v3` | 火山方舟端点 |
| `think_chat_model.model` | - | `DeepSeek-V4-flash` | Think 模型 |
| `quick_chat_model.*` | `QUICK_CHAT_MODEL_API_KEY` | 同上 | Quick 模型 |
| `embedding_model.*` | `EMBEDDING_MODEL_API_KEY` | 阿里百炼 | Embedding（2048 维） |
| `file_dir` | `FILE_DIR` | `./storage/uploads` | 上传落盘 |
| `milvus_url` | `MILVUS_URL` | `localhost:19530` | Milvus 地址 |
| `mcp_url` | - | `http://localhost:3000/sse` | CLS 日志 MCP SSE |
| `prometheus_url` | `PROMETHEUS_URL` | 空 | Prometheus API（空则 Mock） |

独立 CLS 日志 MCP 服务（`cls-mcp-server`）经环境变量 `TRANSPORT` / `PORT` / `TENCENTCLOUD_SECRET_ID` / `TENCENTCLOUD_SECRET_KEY` / `TENCENTCLOUD_REGION` 配置，与 Go 服务配置相互独立。

---

## 7. 可观测性

| 端点 / 机制 | 说明 |
|---|---|
| `/healthz` | 存活探针，返回 `ok` |
| `/readyz` | 就绪探针，经 `vectorstore.Ping` 3s 超时探测 Milvus，失败返回 503 |
| `/metrics` | 自研零依赖 Prometheus 注册表（counter + histogram），文本格式 |
| 指标 | `http_requests_total`（标签 method/path/status）、`http_request_duration_seconds`（直方图桶 0.01–10s） |
| 访问日志 | `X-Request-Id` + slog JSON（`method/path/status/duration_ms/remote_addr`） |

---

## 8. 构建与部署

### 8.1 本地启动

```bash
cd manifest/deploy/milvus && docker-compose up -d   # Attu: http://localhost:8000
cp manifest/config/config.yaml.example manifest/config/config.yaml   # 填入真实值
go run ./cmd/server          # 访问 http://localhost:6872
go run ./cmd/knowledge       # 索引 docs/ 种子文档
```

### 8.2 镜像

`manifest/docker/Dockerfile`（multi-stage）：Go 编译 + 精简运行镜像，静态资源内嵌。

---

## 9. 前端架构（`cmd/server/static`）

模块化原生 JS（`ZebraOpsApp` prototype），内嵌至二进制：

| 文件 | 职责 |
|---|---|
| `app.js` | 主类 + 会话 ID 生成（`session_...` + 时间戳），`apiBaseUrl = /api` |
| `ui.js` | DOM 引用、事件绑定、模式切换、通知、遮罩、滚动、侧栏 |
| `chat.js` | 消息收发、`/api/chat` 与 `/api/chat_stream` SSE 解析、消息 DOM、复制/赞/踩/重试 |
| `history.js` | 会话 localStorage 保存/加载/删除（上限 50 条） |
| `markdown.js` | marked + DOMPurify 安全渲染 + highlight.js 高亮 |
| `upload.js` | 文件校验（`.txt/.md/.markdown`，≤50MB）与上传 |
| `aiops.js` | AI Ops 触发、报告渲染、可折叠步骤明细 |
| `icons.js` | 内联 SVG 图标系统 |
| `init.js` | 初始化入口 + 全局状态 |

---

## 10. CI

`.github/workflows/ci.yml`：push / PR 触发 `go vet` + `go build` + `go test`，Milvus 集成测试在无环境时自动跳过。
