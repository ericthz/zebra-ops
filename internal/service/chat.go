package service

import (
	"context"

	"github.com/ericthz/zebra-ops/internal/biz/chat_pipeline"
	"github.com/ericthz/zebra-ops/internal/components/callbacks"
	"github.com/ericthz/zebra-ops/internal/memory"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// Chat 执行一次完整对话：RAG 检索 + ReAct Agent，并写入会话内存。
func Chat(ctx context.Context, id, msg string) (*schema.Message, error) {
	// 构建用户消息，包含历史上下文
	userMessage := &chat_pipeline.UserMessage{
		ID:      id,
		Query:   msg,
		History: memory.GetSimpleMemory(id).GetMessages(),
	}
	// 构建聊天 Agent 实例
	runner, err := chat_pipeline.BuildChatAgent(ctx)
	if err != nil {
		return nil, err
	}
	// 同步调用 Agent 执行对话
	out, err := runner.Invoke(ctx, userMessage, compose.WithCallbacks(callbacks.LogCallback(nil)))
	if err != nil {
		return nil, err
	}
	// 将对话结果写入会话内存，供后续对话使用
	memory.GetSimpleMemory(id).SetMessages(schema.UserMessage(msg))
	memory.GetSimpleMemory(id).SetMessages(schema.SystemMessage(out.Content))
	return out, nil
}
