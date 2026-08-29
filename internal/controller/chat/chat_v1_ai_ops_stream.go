package chat

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/schema"
	v1 "github.com/ericthz/zebra-ops/api/chat/v1"
	"github.com/ericthz/zebra-ops/internal/biz/plan_execute_replan"
	"github.com/ericthz/zebra-ops/internal/memory"
	"github.com/gogf/gf/v2/frame/g"
)

// sseJSON 将结构体编码为单行 JSON 字符串，避免多行文本破坏 SSE 帧结构
func sseJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return `{"text":"` + err.Error() + `"}`
	}
	return string(b)
}

// AIOpsStream AI 运维流式分析接口：通过 SSE 实时推送分析进度，结束后推送完整报告
func (c *ControllerV1) AIOpsStream(ctx context.Context, req *v1.AIOpsStreamReq) (res *v1.AIOpsStreamRes, err error) {
	// 建立 SSE 连接客户端
	client, err := c.hub.Create(ctx, g.RequestFromCtx(ctx))
	if err != nil {
		return nil, err
	}
	id := req.Id
	query := aiOpsPrompt()

	// 快速失败时自动重试（本地小模型偶发不稳定），成功或产出报告即返回
	const maxAttempts = 3
	var finalResp string
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		client.SendToClient("status", sseJSON(map[string]string{"text": "开始执行 AI 运维分析..."}))
		var resp string
		var aerr error
		resp, _, aerr = plan_execute_replan.BuildPlanAgentStream(ctx, query, func(step string) {
			client.SendToClient("step", sseJSON(map[string]string{"text": step}))
		})
		if aerr == nil && resp != "" {
			finalResp = resp
			break
		}
		if attempt < maxAttempts {
			client.SendToClient("status", sseJSON(map[string]string{"text": "执行过程中出现异常，正在自动重试..."}))
		} else {
			client.SendToClient("error", sseJSON(map[string]string{"text": fmt.Sprintf("AI 运维分析失败: %v", aerr)}))
			return &v1.AIOpsStreamRes{}, nil
		}
	}

	// 将分析报告写入后端会话记忆，使后续对话框追问能引用该报告上下文
	if id != "" {
		memory.GetSimpleMemory(id).SetMessages(schema.SystemMessage(finalResp))
	}

	client.SendToClient("done", sseJSON(map[string]string{"report": finalResp}))
	return &v1.AIOpsStreamRes{}, nil
}
