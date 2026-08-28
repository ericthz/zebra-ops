package chat

import (
	"context"
	"errors"
	"io"
	"strings"

	v1 "github.com/ericthz/zebra-ops/api/chat/v1"
	"github.com/ericthz/zebra-ops/internal/biz/chat_pipeline"
	"github.com/ericthz/zebra-ops/internal/components/callbacks"
	"github.com/ericthz/zebra-ops/internal/memory"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/gogf/gf/v2/frame/g"
)

// ChatStream 流式聊天接口：通过 SSE 实时推送 AI 对话回复
func (c *ControllerV1) ChatStream(ctx context.Context, req *v1.ChatStreamReq) (res *v1.ChatStreamRes, err error) {
	id := req.Id
	msg := req.Question

	// 创建 SSE 连接客户端，用于后续向客户端推送流式消息

	client, err := c.hub.Create(ctx, g.RequestFromCtx(ctx))
	if err != nil {
		return nil, err
	}

	userMessage := &chat_pipeline.UserMessage{
		ID:      id,
		Query:   msg,
		History: memory.GetSimpleMemory(id).GetMessages(),
	}

	// 构建聊天 Agent 并启动流式处理
	runner, err := chat_pipeline.BuildChatAgent(ctx)
	sr, err := runner.Stream(ctx, userMessage, compose.WithCallbacks(callbacks.LogCallback(nil)))
	if err != nil {
		// 流式调用失败，通过 SSE 通知客户端错误信息
		client.SendToClient("error", err.Error())
		return nil, err
	}
	defer sr.Close()

	// 累积完整响应用于写入会话内存
	var fullResponse strings.Builder

	// 流结束后将完整对话写入会话内存
	defer func() {
		completeResponse := fullResponse.String()
		if completeResponse != "" {
			memory.GetSimpleMemory(id).SetMessages(schema.UserMessage(msg))
			memory.GetSimpleMemory(id).SetMessages(schema.SystemMessage(completeResponse))
		}
	}()

	// 循环接收流式响应块并逐个推送给客户端
	for {
		chunk, err := sr.Recv()
		if errors.Is(err, io.EOF) {
			// 流结束，通知客户端完成
			client.SendToClient("done", "Stream completed")
			return &v1.ChatStreamRes{}, nil
		}
		if err != nil {
			// 接收出错，通知客户端错误信息
			client.SendToClient("error", err.Error())
			return &v1.ChatStreamRes{}, nil
		}
		fullResponse.WriteString(chunk.Content)
		client.SendToClient("message", chunk.Content)
	}
}
