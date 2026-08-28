// knowledge 知识库索引入口，将 docs 目录下的 Markdown 文件建立向量索引
package main

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/ericthz/zebra-ops/internal/biz/knowledge_index_pipeline"
)

func main() {
	ctx := context.Background()
	// 初始化知识库索引管道（含向量存储连接等）
	if _, err := knowledge_index_pipeline.BuildKnowledgeIndexing(ctx); err != nil {
		panic(err)
	}
	// 遍历 docs 目录，逐个对 Markdown 文件建立索引
	err := filepath.WalkDir("./docs", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk dir failed: %w", err)
		}
		// 跳过目录，只处理文件
		if d.IsDir() {
			return nil
		}
		// 跳过非 Markdown 文件
		if !strings.HasSuffix(path, ".md") {
			fmt.Printf("[skip] not a markdown file: %s\n", path)
			return nil
		}
		fmt.Printf("[start] indexing file: %s\n", path)
		// 对单个文件执行索引重建（分块、向量化、写入存储）
		if err := knowledge_index_pipeline.RebuildSource(ctx, path); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
}
