# Zebra Ops

English | [简体中文](./README.zh-CN.md)

Zebra Ops is an operations assistant service built on **GoFrame v2** and the **cloudwego/eino** AI orchestration framework. It provides three capabilities: RAG knowledge-base Q&A, document vectorization/indexing, and Plan-Execute-RePlan multi-turn Agent-based alert analysis.

## Features

| Capability | Description | Endpoint |
|---|---|---|
| Quick chat | Request-response chat with RAG retrieval augmentation + ReAct Agent tool calls | `POST /api/chat` |
| Streaming chat | SSE token-by-token output | `POST /api/chat_stream` |
| Knowledge base management | Upload documents → auto split, vectorize, index; same-source file overwrite update | `POST /api/upload` |
| AI ops analysis | Fetch alerts → retrieve runbooks → query related logs → generate structured report (SSE real-time progress) | `POST /api/ai_ops_stream` |
| Agent toolset | Time / internal doc retrieval / Tencent Cloud CLS log MCP / Prometheus alerts | Inside ReAct and Plan-Execute-RePlan |

## System Architecture

### Layers & request flow

```
HTTP request
  → middleware (CORS → RequestID → AccessLog → Metrics → Response)
  → controller/chat (DTO validation)
  → service (business orchestration)
  → biz (AI pipelines: chat_pipeline / knowledge_index_pipeline / plan_execute_replan)
  → components (llm / embedder / vectorstore / loader / tools)
```

- Unified response envelope: `{ "message": "...", "data": { ... } }` (`internal/middleware/middleware.go`).
- Config loading: GoFrame config component (`g.Cfg().GetEffective`); secrets do not enter the repo.

### Chat pipeline `internal/biz/chat_pipeline`

Eino directed graph (`BuildChatAgent` in `orchestration.go`):

```
START
 ├─ InputToRag ──> MilvusRetriever ─┐
 ├─ InputToChat ────────────────────┼─> ChatTemplate ──> ReactAgent ──> END
                                    (system prompt + history + RAG context)
```

- `InputToRag` / `InputToChat` run in parallel, aggregated via `AllPredecessor`.
- ReAct Agent (`newReactAgentLambda` in `flow.go`) binds all tools, `MaxStep = 25`, tool calls use the Quick model.
- Session history is held in memory (`internal/memory`): sliding window of 6 messages (paired drop to preserve pairs) + 24h TTL + 10000-message capacity cap.

### Document indexing pipeline `internal/biz/knowledge_index_pipeline`

```
FileLoader ──> MarkdownSplitter (split by # headings, uuid for doc ID) ──> MilvusIndexer
```

- Overwrite update: `RebuildSource` filters and deletes old data by `metadata._source` then rebuilds (shared by upload and CLI).

### AI ops analysis pipeline `internal/biz/plan_execute_replan`

Standard **Plan → Execute → RePlan** multi-turn Agent (`BuildPlanAgent` in `plan_execute_replan.go`, based on eino `adk/prebuilt/planexecute`):

- Planner / Replanner: Think model (strong reasoning, breaks down long chains).
- Executor: Quick model (low-latency tool calls) + toolset; outer `MaxIterations = 10`.
- Executor toolset (10 total): 6 CLS MCP tool subset (`FilterLogMcpTools`) + `query_prometheus_alerts` + `query_internal_docs` + `get_current_time` + `respond` fallback tool.
- Report fallback: in the event loop, `isCompleteReport` validates report completeness and terminates early; if the Executor mistakenly calls `respond` or the Replanner returns the report directly, both are captured via `extractRespond` / `extractRespondFromToolCalls`. Fallback priority: respond report > `bestReport` > last assistant message body.
- `flexiblePlan`: tolerates small models outputting `steps` as a stringified array or single string; custom deserialization feeds the Planner / Replanner.

### Agent toolset `internal/components/tools`

| Tool | Description | Registration scope |
|---|---|---|
| `get_current_time` | Current time (sec/ms/μs) | chat / AI Ops |
| `query_internal_docs` | Internal doc RAG retrieval | chat / AI Ops |
| `query_log` | Tencent Cloud CLS log MCP (SSE) | chat / AI Ops |
| `query_prometheus_alerts` | Prometheus active alerts | chat / AI Ops |
| `respond` | Fallback response tool (avoid Executor miscall errors) | AI Ops Executor only |
| `mysql_crud` | MySQL query/write | offline CLI only (`cmd/llm_tool`), not registered server-side |

### MCP integration `internal/components/tools/query_log.go`

- `GetLogMcpTool` lazily loads a singleton `reconnectableToolGroup` via `sync.Once`, establishing a mark3labs/mcp-go SSE client; after `Initialize` it pulls the tool list via eino-ext `mcp.GetTools`.
- Auto-reconnect on session expiry: `reconnectableTool` dynamically routes to the latest connection by `toolName`; on a session error (`isSessionError`) it triggers async `tryReconnect`, and the `reconnecting` atomic flag ensures only one rebuild.
- Graceful degradation: when a CLS tool returns `IsError`, `gracefulClsResultHandler` converts it to the text "log query returned no data (reason: …) please continue analysis based on the internal-docs solution already obtained", without interrupting the Agent.

### RAG retrieval `internal/components/vectorstore`

- Vector: `text-embedding-v4`, **2048-dim FloatVector**, `L2` metric (Milvus collection `biz`, database `agent`).
- Splitting: `markdown.NewHeaderSplitter` splits by `#` heading hierarchy, uuid for chunk ID.
- Retrieval: TopK = 1; indexer / retriever custom converter outputs `float32` vectors, fixing eino-ext's default "byte-packed BinaryVector + Hamming" semantic error.
- Output fields: `content`, `metadata`.

## Tech Stack

| Layer | Tech |
|---|---|
| HTTP | GoFrame v2.10.2 (api / controller / service layering) |
| AI orchestration | cloudwego/eino v0.9.15 (Graph + ReAct Agent + ADK prebuilt) |
| LLM | OpenAI-compatible API (default Volcengine Ark DeepSeek-V4-flash, Think / Quick dual-model) |
| Embedding | OpenAI-compatible API (default Alibaba Bailian text-embedding-v4, 2048-dim) |
| Vector DB | Milvus 2.5.10 (server), Go SDK v2.4.2 (FloatVector + L2) |
| Agent tools | Tencent Cloud CLS log MCP (SSE) + self-built Prometheus / time / doc tools |
| Observability | slog JSON logs + self-built Prometheus metrics + health checks |
| Frontend | Plain HTML/CSS/JS, SSE streaming, light/dark themes, DOMPurify against XSS, `//go:embed` into binary |
| CI | GitHub Actions (vet + build + test) |
| Language | Go 1.26+ |

## Directory Structure

```
Zebra Ops
├── .github/workflows/         # CI (vet + build + test)
├── api/chat/v1/               # GoFrame API DTO + g.Meta route tags
├── cmd/
│   ├── server/                # HTTP service entry (main.go + embedded static frontend)
│   └── chat/ knowledge/ recall/ ai_ops/ llm_tool/   # CLI tools
├── internal/
│   ├── controller/chat/       # HTTP Handler (param validation + forward)
│   ├── service/               # Business orchestration (chat / upload / sse)
│   ├── biz/
│   │   ├── chat_pipeline/            # Chat graph (RAG + ReAct)
│   │   ├── knowledge_index_pipeline/ # Document index graph
│   │   └── plan_execute_replan/      # AI Ops (Plan-Execute-RePlan)
│   ├── components/           # llm / embedder / vectorstore / loader / tools / callbacks
│   ├── memory/               # In-memory session (sliding window + TTL)
│   ├── middleware/           # CORS / request_id / access log / metrics / unified response
│   └── observability/        # Metrics registry + health checks
├── manifest/
│   ├── config/               # config.yaml.example (config.yaml gitignored)
│   ├── docker/               # Dockerfile (multi-stage)
│   └── deploy/milvus/        # Milvus docker-compose
├── docs/                     # Docs and knowledge-base seed docs
├── mock/                     # Mock alert data (alerts.json gitignored)
└── storage/uploads/          # Uploaded files on disk (generated at runtime)
```

## API Reference

Common prefix `/api`, unified response `{ "message": "...", "data": { ... } }`; `/healthz` `/readyz` `/metrics` are not under the `/api` prefix.

| Method | Path | Input | Response data | Description |
|---|---|---|---|---|
| POST | `/api/chat` | `{ Id, Question }` | `{ answer }` | Quick chat (RAG + ReAct) |
| POST | `/api/chat_stream` | `{ Id, Question }` | SSE event stream | Streaming chat (events: `connected` / `message` / `done` / `error`) |
| POST | `/api/upload` | multipart (`file`) | `{ fileName, filePath, fileSize }` | Document upload & index (overwrite update) |
| POST | `/api/ai_ops` | `{ Id }` | `{ result, detail[] }` | AI alert analysis (sync, Plan Agent) |
| POST | `/api/ai_ops_stream` | `{ Id }` | SSE event stream | AI alert analysis (SSE real-time progress + report) |
| GET | `/healthz` | - | `ok` | Liveness probe |
| GET | `/readyz` | - | `ok` / 503 | Readiness probe (short-timeout Milvus probe) |
| GET | `/metrics` | - | Prometheus text | Metrics output |

Example:

```bash
curl -X POST http://localhost:6872/api/chat \
  -H 'Content-Type: application/json' \
  -d '{"Id": "session_1", "Question": "Why did the Zebra Ops service go down?"}'
```

## Configuration

### Config loading

Service config is read from `manifest/config/config.yaml` via the GoFrame config component. That file is gitignored and not in the repo; copy from `config.yaml.example` and fill in local values. Sensitive fields (API Key) are filled in the local `config.yaml`; secrets do not enter the repo.

The only OS environment variable the service process reads is `LOG_DIR` (log directory).

The standalone CLS log MCP service (`cls-mcp-server`) is configured via env vars `TRANSPORT` / `PORT` / `TENCENTCLOUD_SECRET_ID` / `TENCENTCLOUD_SECRET_KEY` / `TENCENTCLOUD_REGION`, injectable via `.env` + `export`, independent from the Go service config.

### Config items

| config key | env var | default | description |
|---|---|---|---|
| `server.address` | - | `:6872` | HTTP listen (code `SetPort(6872)` same value) |
| `think_chat_model.api_key` | `THINK_CHAT_MODEL_API_KEY` | empty | Think model key (strong reasoning/planning) |
| `think_chat_model.base_url` | - | `https://ark.cn-beijing.volces.com/api/v3` | Volcengine Ark OpenAI-compatible endpoint |
| `think_chat_model.model` | - | `DeepSeek-V4-flash` | Think model name |
| `quick_chat_model.*` | `QUICK_CHAT_MODEL_API_KEY` | same as above | Quick model (low-latency execution) |
| `embedding_model.api_key` | `EMBEDDING_MODEL_API_KEY` | empty | Alibaba Bailian DashScope key |
| `embedding_model.base_url` | - | `https://dashscope.aliyuncs.com/compatible-mode/v1` | Embedding endpoint |
| `embedding_model.model` | - | `text-embedding-v4` | Embedding model (2048-dim) |
| `file_dir` | `FILE_DIR` | `./storage/uploads` | Upload disk directory |
| `milvus_url` | `MILVUS_URL` | `localhost:19530` | Milvus address |
| `mcp_url` | - | `http://localhost:3000/sse` | CLS log MCP SSE endpoint |
| `prometheus_url` | `PROMETHEUS_URL` | empty | Prometheus API (empty → mock alerts fallback) |

### Mock alert mode

When `PROMETHEUS_URL` is empty, `query_prometheus_alerts` falls back to reading `mock/alerts.json`, which contains 4 typical alert types (service down / high interface failure rate / upstream-downstream reconciliation mismatch / service-region vs resource-region mismatch). Their `activeAt` is generated from relative offsets dynamically, so you can experience full AI ops analysis without deploying Prometheus locally.

## Observability

| Endpoint / mechanism | Description |
|---|---|
| `/healthz` | Liveness probe, returns `ok` |
| `/readyz` | Readiness probe, 3s timeout Milvus connectivity probe, returns 503 on failure |
| `/metrics` | Self-built zero-dependency Prometheus registry (counter + histogram), text format output |
| Metrics | `http_requests_total` (method/path/status), `http_request_duration_seconds` (histogram buckets 0.01–10s) |
| Access log | `X-Request-Id` generate/passthrough + slog JSON structured output (method/path/status/duration_ms/remote_addr) |

## Quick Start

### 1. Start dependencies (Milvus + Attu)

```bash
cd manifest/deploy/milvus
docker-compose up -d
# Milvus: localhost:19530; Attu console: http://localhost:8000
```

### 2. Configure

```bash
cp manifest/config/config.yaml.example manifest/config/config.yaml
# Edit manifest/config/config.yaml, fill in real API Key and endpoints
```

### 3. Start service

```bash
go run ./cmd/server
# Listens on port 6872, frontend embedded in binary, visit http://localhost:6872
```

### 4. CLI tools

```bash
go run ./cmd/knowledge   # Index md docs under docs/ into knowledge base
go run ./cmd/chat        # Demo multi-turn chat
go run ./cmd/recall      # Demo vector recall
go run ./cmd/ai_ops      # Demo AI alert analysis
go run ./cmd/llm_tool    # Demo MCP + custom tool binding
```

## Build & Deploy

`manifest/docker/Dockerfile` (multi-stage): Go compile + slim runtime image, static assets embedded, no separate frontend distribution needed.

## Development

- Unit tests: `go test ./...`
- CI (`.github/workflows/ci.yml`): push / PR triggers `go vet` + `go build` + `go test`; Milvus integration tests auto-skip when no environment.

## License

MIT
