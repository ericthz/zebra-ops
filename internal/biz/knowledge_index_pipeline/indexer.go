package knowledge_index_pipeline

import (
	"context"

	"github.com/ericthz/zebra-ops/internal/components/vectorstore"

	"github.com/cloudwego/eino/components/indexer"
)

// newIndexer 创建并返回一个新的向量索引器实例，用于将文档写入向量存储（Redis/Milvus）
func newIndexer(ctx context.Context) (idr indexer.Indexer, err error) {
	// 委托给 vectorstore 组件完成实际的索引器初始化
	return vectorstore.NewIndexer(ctx)
}
