# AI Ops SSE 化改造说明

> 记录将 AI 运维分析（`/api/ai_ops`）从同步阻塞改为 SSE 流式推送的分析与实现方案。

---

## 1. 现状

- `POST /api/ai_ops` 为同步请求：Controller（`internal/controller/chat/chat_v1_ai_ops.go`）阻塞调用 `plan_execute_replan.BuildPlanAgent`，等待整个 Plan-Execute-RePlan 流程结束后一次性返回 `{ result, detail[] }`。
- 单次分析为分钟级（本地小模型 2-5 分钟，线上模型更短但仍属长任务）。
- 前端 `aiops.js` 通过 `fetch` 等待响应，期间仅显示静态"分析中"遮罩，无任何进度信息。
- 但 `BuildPlanAgent`（`plan_execute_replan.go`）内部**本就用 `iter.Next()` 迭代事件流**（planner → 每次工具调用 → replanner → 报告），只是把事件攒成 `detail []string` 到最后返回。

## 2. 问题

| 问题 | 说明 |
|---|---|
| 无进度反馈 | 长任务期间用户只能干等，不知道当前处于哪一步 |
| 断连丢结果 | 同步请求连接一旦中断，整个分析结果丢失，且已完成的 Agent 运行被浪费 |
| 与 chat_stream 不一致 | 对话已有 SSE 流式基建，AI Ops 仍是同步，体验割裂 |

## 3. 为什么 SSE 更合理

1. **长任务**：分钟级操作适合服务端推送，前端实时看到"正在查询告警 / 正在检索文档 / 正在查询日志 / 生成报告"等进度。
2. **事件流现成**：biz 层已经是事件流驱动，只需"边迭代边推"，改造成本低。
3. **基建可复用**：后端已有 `internal/service/sse.go`（Hub/Client）与 `/api/chat_stream` 模式；前端已有 SSE 解析逻辑（`chat.js`）。

## 4. SSE 事件设计

> 注意：AI Ops 输出是整份 Markdown 报告，非逐 token 流。SSE 推送的是**进度事件 + 结尾完成事件**。
> 所有 data 均 JSON 编码为单行，避免多行文本破坏 SSE 帧结构。

| 事件 | data 载荷 | 说明 |
|---|---|---|
| `connected` | `{status, client_id}` | 连接建立（Hub 自带） |
| `status` | `{"text": "..."}` | 阶段提示（如"开始执行 AI 运维分析..."、"执行过程中出现异常，正在自动重试..."） |
| `step` | `{"text": "..."}` | 单步明细（对应原 `detail[]` 的每一项） |
| `done` | `{"report": "..."}` | 最终报告，携带完整 Markdown |
| `error` | `{"text": "..."}` | 失败提示 |

## 5. 实现方案

### 5.1 后端

- `plan_execute_replan.go`：新增 `BuildPlanAgentStream(ctx, query, onStep func(step string))`，在事件循环中为每个清洗后的步骤调用 `onStep`；原 `BuildPlanAgent` 委托给它并传 `nil` 回调，保持兼容。
- 新增 DTO：`AIOpsStreamReq`（`path:"/ai_ops_stream" method:"post"`，字段 `Id`）、`AIOpsStreamRes`（空）。
- 新增 Controller：`AIOpsStream`——`hub.Create` 建立 SSE 连接 → 调用 `BuildPlanAgentStream`，回调内 `SendToClient("step", json(...))`；成功后写报告入会话记忆并发 `done`；失败发 `error`。
- 路由：GoFrame 通过 `g.Meta` 自动注册 `/api/ai_ops_stream`，无需改 `main.go`。

### 5.2 前端

- `aiops.js`：`sendAIOpsRequest` 改为 `fetch` + `ReadableStream` 逐行解析（对齐 `chat.js` 的 SSE 解析），按事件更新：
  - `status` / `step` → 更新加载消息文本为当前进度，并累计 `details[]`
  - `done` → `updateAIOpsMessage(message, report, details)` 渲染报告与折叠步骤
  - `error` → 展示错误文本

## 6. 注意事项

| 点 | 处理 |
|---|---|
| 多行文本 | step/report 统一 JSON 编码为单行，避免破坏 SSE 帧 |
| 重试边界 | 最多 3 次尝试在 Handler 内完成，全部失败才发 `error`；重试会追加发送 `status` 提示 |
| 心跳 | 长任务建议定时发心跳（注释行 `: ping`），防止代理超时断开（当前模型较短，可后续补充） |
| 会话记忆 | 报告仍在 `done` 前写入会话记忆，语义不变 |
| 中间件 | `ResponseMiddleware` 会在 SSE 末尾追加 `{message,data}` JSON，前端解析到 `done` 后即返回，不受影响（与 chat_stream 同行为） |
