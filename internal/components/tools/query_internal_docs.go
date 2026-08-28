package tools

import (
	"context"
	"encoding/json"

	"github.com/ericthz/zebra-ops/internal/components/vectorstore"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// QueryInternalDocsInput 内部文档查询的输入参数
type QueryInternalDocsInput struct {
	Query string `json:"query" jsonschema:"description=The query string to search in internal documentation for relevant information and processing steps"`
}

// NewQueryInternalDocsTool 创建内部文档检索工具，通过 RAG 方式从知识库中搜索相关文档
func NewQueryInternalDocsTool() (tool.InvokableTool, error) {
	t, err := utils.InferOptionableTool(
		"query_internal_docs",
		"Use this tool to search internal documentation and knowledge base for relevant information. It performs RAG (Retrieval-Augmented Generation) to find similar documents and extract processing steps. This is useful when you need to understand internal procedures, best practices, or step-by-step guides stored in the company's documentation.",
		func(ctx context.Context, input *QueryInternalDocsInput, opts ...tool.Option) (output string, err error) {
			rr, err := vectorstore.NewRetriever(ctx)
			if err != nil {
				return "", err
			}
			resp, err := rr.Retrieve(ctx, input.Query)
			if err != nil {
				return "", err
			}
			respBytes, err := json.Marshal(resp)
			if err != nil {
				return "", err
			}
			output = string(respBytes)
			return output, nil
		})
	if err != nil {
		return nil, err
	}
	return t, nil
}
