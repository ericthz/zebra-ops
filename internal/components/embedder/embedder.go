package embedder

import (
	"context"
	"log"

	eino_openai "github.com/cloudwego/eino-ext/components/embedding/openai"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/gogf/gf/v2/frame/g"
)

// DoubaoEmbedding 创建向量化服务实例，支持 OpenAI 兼容 API（如 Ollama、DashScope）
func DoubaoEmbedding(ctx context.Context) (eb embedding.Embedder, err error) {
	model, err := g.Cfg().GetEffective(ctx, "embedding_model.model")
	if err != nil {
		return nil, err
	}
	apiKey, err := g.Cfg().GetEffective(ctx, "embedding_model.api_key")
	if err != nil {
		return nil, err
	}
	baseURL, err := g.Cfg().GetEffective(ctx, "embedding_model.base_url")
	if err != nil {
		return nil, err
	}

	// 向量维度固定为 2048，需与 Milvus 集合 schema 保持一致
	dim := 2048
	embedder, err := eino_openai.NewEmbedder(ctx, &eino_openai.EmbeddingConfig{
		Model:       model.String(),
		APIKey:      apiKey.String(),
		BaseURL:     baseURL.String(),
		Dimensions:  &dim,
	})
	if err != nil {
		log.Printf("new embedder error: %v\n", err)
		return nil, err
	}
	return embedder, nil
}
