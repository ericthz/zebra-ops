package chat_pipeline

import (
	"context"

	"github.com/ericthz/zebra-ops/internal/components/llm"

	"github.com/cloudwego/eino/components/model"
)

// newChatModel 创建聊天模型实例
// 返回一个支持工具调用的 DeepSeek 快速模型
func newChatModel(ctx context.Context) (cm model.ToolCallingChatModel, err error) {
	return llm.NewDeepSeekQuickModel(ctx)
}
