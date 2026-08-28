package chat_pipeline

import (
	"context"
	"time"
)

// newInputToRagLambda 将用户消息提取为 RAG 检索查询字符串
// Eino InvokableLambda 签名要求 ctx 和 opts，此处不使用
func newInputToRagLambda(_ context.Context, input *UserMessage, _ ...any) (output string, err error) {
	return input.Query, nil
}

// newInputToChatLambda 将用户消息转换为 ChatTemplate 所需的参数 map
// 包含 content、history、date 三个模板变量
func newInputToChatLambda(_ context.Context, input *UserMessage, _ ...any) (output map[string]any, err error) {
	return map[string]any{
		"content": input.Query,
		"history": input.History,
		"date":    time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}
