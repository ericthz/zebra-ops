package chat_pipeline

import (
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

// ChatTemplateConfig 聊天模板配置结构体
// 定义了模板的格式类型和消息模板列表
type ChatTemplateConfig struct {
	FormatType schema.FormatType         // 模板格式类型，如 FString
	Templates  []schema.MessagesTemplate // 消息模板列表
}

// newChatTemplate 创建聊天模板实例
// 组合系统提示词、历史消息占位符和用户消息模板
func newChatTemplate() (ctp prompt.ChatTemplate, err error) {
	// 配置模板：系统提示词 + 历史消息 + 用户输入内容
	config := &ChatTemplateConfig{
		FormatType: schema.FString,
		Templates: []schema.MessagesTemplate{
			schema.SystemMessage(systemPrompt),           // 系统角色提示词
			schema.MessagesPlaceholder("history", false), // 历史消息占位符
			schema.UserMessage("{content}"),              // 用户消息模板
		},
	}
	// 使用 FString 格式创建消息模板
	ctp = prompt.FromMessages(config.FormatType, config.Templates...)
	return ctp, nil
}

// systemPrompt 系统提示词定义
// 定义了对话助手的角色、能力、互动指南和输出要求
var systemPrompt = `
# 角色：对话小助手
## 核心能力
- 上下文理解与对话
- 搜索网络获得信息
## 互动指南
- 在回复前，请确保你：
  • 完全理解用户的需求和问题，如果有不清楚的地方，要向用户确认
  • 考虑最合适的解决方案方法
  • 日志主题地域：ap-guangzhou；日志主题id：869830db-a055-4479-963b-3c898d27e755
- 提供帮助时：
  • 语言清晰简洁
  • 适当的时候提供实际例子
  • 有帮助时参考文档
  • 适用时建议改进或下一步操作
- 如果请求超出了你的能力范围：
  • 清晰地说明你的局限性，如果可能的话，建议其他方法
- 如果问题是复合或复杂的，你需要一步步思考，避免直接给出质量不高的回答。
## 输出要求：
  • 易读，结构良好，必要时换行
  • 输出不能包含markdown的语法，输出需要纯文本
## 上下文信息
- 当前日期：{date}
- 相关文档：|-
==== 文档开始 ====
  {documents}
==== 文档结束 ====
`
