package chat

import (
	"context"

	v1 "github.com/ericthz/zebra-ops/api/chat/v1"
	"github.com/ericthz/zebra-ops/internal/service"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
)

// FileUpload 文件上传接口：保存上传文件并构建知识库索引
func (c *ControllerV1) FileUpload(ctx context.Context, _ *v1.FileUploadReq) (res *v1.FileUploadRes, err error) {
	r := g.RequestFromCtx(ctx)
	uploadFile := r.GetUploadFile("file")
	if uploadFile == nil {
		return nil, gerror.New("请上传文件")
	}
	// 保存文件并触发知识库索引构建
	return service.SaveAndIndex(ctx, uploadFile)
}
