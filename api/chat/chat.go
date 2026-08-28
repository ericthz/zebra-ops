// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

// Package chat 定义聊天模块的控制器接口，由 GoFrame 自动生成
package chat

import (
	"context"

	v1 "github.com/ericthz/zebra-ops/api/chat/v1"
)

// IChatV1 聊天模块 V1 版本接口，定义所有对话相关操作
type IChatV1 interface {
	// Chat 普通对话，同步返回完整回答
	Chat(ctx context.Context, req *v1.ChatReq) (res *v1.ChatRes, err error)
	// ChatStream 流式对话，通过 SSE 实时推送 token
	ChatStream(ctx context.Context, req *v1.ChatStreamReq) (res *v1.ChatStreamRes, err error)
	// FileUpload 文件上传，保存并返回文件元信息
	FileUpload(ctx context.Context, req *v1.FileUploadReq) (res *v1.FileUploadRes, err error)
	// AIOps AI 运维分析，处理告警并返回诊断结果
	AIOps(ctx context.Context, req *v1.AIOpsReq) (res *v1.AIOpsRes, err error)
}
