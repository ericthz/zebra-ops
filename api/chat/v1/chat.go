// Package v1 定义聊天模块的 API 请求与响应结构
package v1

import (
	"github.com/gogf/gf/v2/frame/g"
)

// ChatReq 普通对话请求，包含会话 ID 和用户问题
type ChatReq struct {
	g.Meta   `path:"/chat" method:"post" summary:"对话"`
	Id       string // 会话标识
	Question string // 用户提问内容
}

// ChatRes 普通对话响应，返回模型生成的答案
type ChatRes struct {
	Answer string `json:"answer"` // 模型生成的回答
}

// ChatStreamReq 流式对话请求，支持服务端推送 token
type ChatStreamReq struct {
	g.Meta   `path:"/chat_stream" method:"post" summary:"流式对话"`
	Id       string // 会话标识
	Question string // 用户提问内容
}

// ChatStreamRes 流式对话响应（实际通过 SSE 推送，此处为空结构体）
type ChatStreamRes struct {
}

// FileUploadReq 文件上传请求，接收 multipart/form-data 格式文件
type FileUploadReq struct {
	g.Meta `path:"/upload" method:"post" mime:"multipart/form-data" summary:"文件上传"`
}

// FileUploadRes 文件上传响应，返回已保存文件的元信息
type FileUploadRes struct {
	FileName string `json:"fileName" dc:"保存的文件名"`   // 文件保存后的名称
	FilePath string `json:"filePath" dc:"文件保存路径"`   // 文件在服务器上的存储路径
	FileSize int64  `json:"fileSize" dc:"文件大小(字节)"` // 文件大小，单位字节
}

// AIOpsReq AI 运维请求，触发智能告警分析流程
type AIOpsReq struct {
	g.Meta `path:"/ai_ops" method:"post" summary:"AI运维"`
	Id     string `json:"id" dc:"会话ID，用于将分析报告写入后端记忆以便后续追问"` // 会话ID
}

// AIOpsRes AI 运维响应，返回分析结果和详情
type AIOpsRes struct {
	Result string   `json:"result"` // 整体分析结论
	Detail []string `json:"detail"` // 各告警的详细分析信息
}

// AIOpsStreamReq AI 运维流式请求，通过 SSE 推送分析进度与最终报告
type AIOpsStreamReq struct {
	g.Meta `path:"/ai_ops_stream" method:"post" summary:"AI运维(SSE)"`
	Id     string `json:"id" dc:"会话ID，用于将分析报告写入后端记忆以便后续追问"` // 会话ID
}

// AIOpsStreamRes AI 运维流式响应（实际通过 SSE 推送，此处为空结构体）
type AIOpsStreamRes struct {
}
