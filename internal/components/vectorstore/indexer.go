package vectorstore

import (
	"context"
	"encoding/json"

	"github.com/ericthz/zebra-ops/internal/components/embedder"

	"github.com/cloudwego/eino-ext/components/indexer/milvus"
	"github.com/cloudwego/eino/schema"
)

// floatRow 与 Milvus biz 集合 schema 对齐的行结构体（Vector 为 float32 向量）
type floatRow struct {
	ID       string    `json:"id" milvus:"name:id"`
	Content  string    `json:"content" milvus:"name:content"`
	Vector   []float32 `json:"vector" milvus:"name:vector"`
	Metadata []byte    `json:"metadata" milvus:"name:metadata"`
}

// NewIndexer 创建 Milvus 向量索引器实例，用于将文档向量化后写入向量数据库
func NewIndexer(ctx context.Context) (*milvus.Indexer, error) {
	cli, err := Client(ctx)
	if err != nil {
		return nil, err
	}
	eb, err := embedder.DoubaoEmbedding(ctx)
	if err != nil {
		return nil, err
	}
	config := &milvus.IndexerConfig{
		Client:     cli,
		Collection: MilvusCollectionName,
		Fields:     fields,
		Embedding:  eb,
		// FloatVector 字段必须用 L2/IP/COSINE，不能用默认的 HAMMING（二进制度量）
		MetricType: milvus.L2,
		// 自定义行转换：将 float64 向量转成 float32 写入 FloatVector 字段，
		// 替代组件默认的"字节打包塞进 BinaryVector"行为（HAMMING 度量在 float 字节上语义错误）
		DocumentConverter: func(ctx context.Context, docs []*schema.Document, vectors [][]float64) ([]interface{}, error) {
			rows := make([]interface{}, 0, len(docs))
			for i, doc := range docs {
				metadata, err := json.Marshal(doc.MetaData)
				if err != nil {
					return nil, err
				}
				vec := make([]float32, len(vectors[i]))
				for j, v := range vectors[i] {
					vec[j] = float32(v)
				}
				rows = append(rows, &floatRow{
					ID:       doc.ID,
					Content:  doc.Content,
					Vector:   vec,
					Metadata: metadata,
				})
			}
			return rows, nil
		},
	}
	indexer, err := milvus.NewIndexer(ctx, config)
	if err != nil {
		return nil, err
	}
	return indexer, nil
}
