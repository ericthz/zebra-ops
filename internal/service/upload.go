package service

import (
	"context"
	"os"
	"path/filepath"

	v1 "github.com/ericthz/zebra-ops/api/chat/v1"
	"github.com/ericthz/zebra-ops/internal/biz/knowledge_index_pipeline"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gfile"
)

// UploadDir 上传文件落盘目录，与知识库种子文档 docs/ 分离。
// 默认值可通过配置文件 file_dir 覆盖。
var UploadDir = "./storage/uploads/"

// SaveAndIndex 保存上传文件并构建知识库索引（同 _source 旧数据覆盖更新）。
func SaveAndIndex(ctx context.Context, uploadFile *ghttp.UploadFile) (*v1.FileUploadRes, error) {
	// 确保上传目录存在，不存在则创建
	if !gfile.Exists(UploadDir) {
		if err := gfile.Mkdir(UploadDir); err != nil {
			return nil, gerror.Wrapf(err, "创建目录失败: %s", UploadDir)
		}
	}

	// 保存到 UploadDir 目录，文件名沿用原始文件名
	if _, err := uploadFile.Save(UploadDir, false); err != nil {
		return nil, gerror.Wrapf(err, "保存文件失败")
	}
	filePath := filepath.Join(UploadDir, uploadFile.Filename)
	// 获取保存后的文件信息，用于返回文件大小等元数据
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, gerror.Wrapf(err, "获取文件信息失败")
	}

	// 构建返回结果
	res := &v1.FileUploadRes{
		FileName: uploadFile.Filename,
		FilePath: filePath,
		FileSize: fileInfo.Size(),
	}
	// 触发知识库索引构建，更新或新增文档内容
	if err := knowledge_index_pipeline.RebuildSource(ctx, filePath); err != nil {
		return nil, gerror.Wrapf(err, "构建知识库失败")
	}
	return res, nil
}
