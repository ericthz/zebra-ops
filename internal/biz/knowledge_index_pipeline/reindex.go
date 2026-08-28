package knowledge_index_pipeline

import (
	"context"
	"fmt"
	"strings"

	"github.com/ericthz/zebra-ops/internal/components/callbacks"
	"github.com/ericthz/zebra-ops/internal/components/loader"
	"github.com/ericthz/zebra-ops/internal/components/vectorstore"

	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/compose"
)

// RebuildSource 删除指定 _source 的旧数据并重建索引，实现文件级覆盖更新。
// 供上传接口与 knowledge 命令行工具共用，避免重复实现。
func RebuildSource(ctx context.Context, path string) error {
	// 构建知识索引 DAG 流程
	r, err := BuildKnowledgeIndexing(ctx)
	if err != nil {
		return err
	}

	// 通过文件加载器读取指定路径的文档
	l, err := loader.NewFileLoader(ctx)
	if err != nil {
		return err
	}
	docs, err := l.Load(ctx, document.Source{URI: path})
	if err != nil {
		return err
	}
	// 确保至少加载到一个文档
	if len(docs) == 0 {
		return fmt.Errorf("no docs loaded from %s", path)
	}

	// 获取 Milvus 客户端，用于后续的删除和查询操作
	cli, err := vectorstore.Client(ctx)
	if err != nil {
		return err
	}
	// 从第一个文档的元数据中提取 _source 标识
	source := docs[0].MetaData["_source"]

	// 使用 Milvus 过滤表达式查询同 _source 的旧数据 ID
	expr := fmt.Sprintf(`metadata["_source"] == "%s"`, source)
	queryResult, err := cli.Query(ctx, vectorstore.MilvusCollectionName, []string{}, expr, []string{"id"})
	if err != nil {
		return err
	}
	// 从查询结果中提取需要删除的文档 ID
	var idsToDelete []string
	for _, column := range queryResult {
		if column.Name() != "id" {
			continue
		}
		for i := 0; i < column.Len(); i++ {
			id, err := column.GetAsString(i)
			if err == nil {
				idsToDelete = append(idsToDelete, id)
			}
		}
	}
	// 批量删除旧数据
	if len(idsToDelete) > 0 {
		deleteExpr := fmt.Sprintf(`id in ["%s"]`, strings.Join(idsToDelete, `","`))
		if err = cli.Delete(ctx, vectorstore.MilvusCollectionName, "", deleteExpr); err != nil {
			return fmt.Errorf("delete existing data failed: %w", err)
		}
	}

	// 调用索引 DAG 流程，重新加载并索引文件内容
	ids, err := r.Invoke(ctx, document.Source{URI: path}, compose.WithCallbacks(callbacks.LogCallback(nil)))
	if err != nil {
		return fmt.Errorf("invoke index graph failed: %w", err)
	}
	// 输出索引完成信息，包括文件路径和生成的分片数量
	fmt.Printf("[done] indexing file: %s, len of parts: %d\n", path, len(ids))
	return nil
}
