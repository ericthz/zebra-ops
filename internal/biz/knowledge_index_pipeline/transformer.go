package knowledge_index_pipeline

import (
	"context"

	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/markdown"
	"github.com/cloudwego/eino/components/document"
	"github.com/google/uuid"
)

// newDocumentTransformer 创建并返回一个 Markdown 文档分割器，按标题层级拆分文档
func newDocumentTransformer(ctx context.Context) (tfr document.Transformer, err error) {
	// 配置分割规则：按一级标题（#）分割，保留标题文本，为每个分片生成唯一 ID
	config := &markdown.HeaderConfig{
		Headers: map[string]string{
			"#": "title",
		},
		TrimHeaders: false,
		IDGenerator: func(ctx context.Context, originalID string, splitIndex int) string {
			// 使用 UUID 为每个分片生成全局唯一标识
			return uuid.New().String()
		},
	}
	// 基于配置创建 Markdown 标题分割器实例
	tfr, err = markdown.NewHeaderSplitter(ctx, config)
	if err != nil {
		return nil, err
	}
	return tfr, nil
}
