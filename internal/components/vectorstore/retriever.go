package vectorstore

import (
	"context"

	"github.com/ericthz/zebra-ops/internal/components/embedder"

	"github.com/cloudwego/eino-ext/components/retriever/milvus"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

// NewRetriever 创建 Milvus 向量检索器实例，用于根据查询向量从知识库中检索相似文档
func NewRetriever(ctx context.Context) (rtr retriever.Retriever, err error) {
	cli, err := Client(ctx)
	if err != nil {
		return nil, err
	}
	eb, err := embedder.DoubaoEmbedding(ctx)
	if err != nil {
		return nil, err
	}
	r, err := milvus.NewRetriever(ctx, &milvus.RetrieverConfig{
		Client:      cli,
		Collection:  MilvusCollectionName,
		VectorField: "vector",
		TopK:        1,
		Embedding:   eb,
		// 必须返回 content/metadata 字段，否则默认 DocumentConverter 拿不到内容
		OutputFields: []string{"content", "metadata"},
		// FloatVector 字段必须用 L2 度量，替代默认 HAMMING
		MetricType: entity.L2,
		// 自定义向量转换：输出 FloatVector（float32），替代默认的 BinaryVector 字节打包
		VectorConverter: func(ctx context.Context, vectors [][]float64) ([]entity.Vector, error) {
			vec := make([]entity.Vector, 0, len(vectors))
			for _, v := range vectors {
				fv := make(entity.FloatVector, len(v))
				for i, x := range v {
					fv[i] = float32(x)
				}
				vec = append(vec, fv)
			}
			return vec, nil
		},
	})
	if err != nil {
		return nil, err
	}
	return r, nil
}
