// recall 演示向量检索流程，从知识库中召回与查询相关的文档片段
package main

import (
	"context"
	"fmt"

	"github.com/ericthz/zebra-ops/internal/components/vectorstore"
)

func main() {
	ctx := context.Background()
	// 创建向量检索器，负责连接向量数据库并执行相似度搜索
	r, err := vectorstore.NewRetriever(ctx)
	if err != nil {
		panic(err)
	}
	query := "Zebra Ops 服务为什么下线？"
	// 执行检索，返回与查询语义最相关的文档片段
	docs, err := r.Retrieve(ctx, query)
	if err != nil {
		panic(err)
	}
	fmt.Println("Q：", query)
	for _, doc := range docs {
		fmt.Println("A：", doc.Content)
	}
	fmt.Println("Done", len(docs))
}
