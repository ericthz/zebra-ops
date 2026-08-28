package chat

import (
	"context"

	v1 "github.com/ericthz/zebra-ops/api/chat/v1"
	"github.com/ericthz/zebra-ops/internal/service"
)

// Chat 非流式聊天接口：执行完整对话并一次性返回结果
func (c *ControllerV1) Chat(ctx context.Context, req *v1.ChatReq) (res *v1.ChatRes, err error) {
	// 调用 service 层执行对话，包含 RAG 检索和 ReAct Agent 处理
	out, err := service.Chat(ctx, req.Id, req.Question)
	if err != nil {
		return nil, err
	}
	return &v1.ChatRes{Answer: out.Content}, nil
}
