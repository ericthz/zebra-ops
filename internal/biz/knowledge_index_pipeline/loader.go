package knowledge_index_pipeline

import (
	"context"

	"github.com/cloudwego/eino-ext/components/document/loader/file"
	"github.com/cloudwego/eino/components/document"
)

// newLoader 创建并返回一个新的文件加载器实例，用于从本地文件系统读取文档
func newLoader(ctx context.Context) (ldr document.Loader, err error) {
	// TODO 根据实际需求修改组件配置
	config := &file.FileLoaderConfig{}
	ldr, err = file.NewFileLoader(ctx, config)
	if err != nil {
		// 初始化失败时直接返回错误
		return nil, err
	}
	return ldr, nil
}
