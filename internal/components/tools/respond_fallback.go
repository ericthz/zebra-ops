package tools

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// RespondInput 兜底 respond 工具的输入参数
type RespondInput struct {
	Response string `json:"response" jsonschema:"description=最终报告内容"`
}

// NewRespondFallbackTool 创建兜底的 respond 工具。
// respond 本属于重规划器（Replanner）的工具，但小模型（4b）经常在计划中看到
// "调用 respond" 后误在执行器中调用它。这里为执行器注册一个同名的兜底工具，
// 被调用时返回引导文本，避免 "tool respond not found in toolsNode indexes" 报错中断 Agent。
func NewRespondFallbackTool() (tool.InvokableTool, error) {
	t, err := utils.InferOptionableTool(
		"respond",
		"提交最终回答。注意：本工具仅用于收集报告内容，调用后会直接作为最终报告返回。",
		func(ctx context.Context, input *RespondInput, opts ...tool.Option) (string, error) {
			if input == nil || input.Response == "" {
				return "请直接以文本形式输出最终告警运维分析报告，无需再调用本工具。", nil
			}
			return input.Response, nil
		})
	if err != nil {
		return nil, err
	}
	return t, nil
}
