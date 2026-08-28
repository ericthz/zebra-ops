package chat_pipeline

import (
	"context"

	"github.com/ericthz/zebra-ops/internal/components/vectorstore"

	"github.com/cloudwego/eino/components/retriever"
)

// newRetriever 创建向量检索器实例
// 从 Milvus 向量存储中检索相关文档
func newRetriever(ctx context.Context) (rtr retriever.Retriever, err error) {
	return vectorstore.NewRetriever(ctx)
}
