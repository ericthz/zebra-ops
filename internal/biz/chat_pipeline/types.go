package chat_pipeline

import "github.com/cloudwego/eino/schema"

// UserMessage 用户消息结构体
// 包含消息 ID、查询内容和对话历史
type UserMessage struct {
	ID      string            `json:"id"`      // 消息唯一标识
	Query   string            `json:"query"`   // 用户查询内容
	History []*schema.Message `json:"history"` // 对话历史记录
}
