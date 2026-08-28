package plan_execute_replan

import (
	"context"

	"github.com/ericthz/zebra-ops/internal/components/llm"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
)

// NewPlanner 创建并返回一个规划器 Agent，负责根据用户查询生成执行计划。
// 使用 DeepSeek Think 模型作为推理引擎，具备较强的推理和规划能力。
func NewPlanner(ctx context.Context) (adk.Agent, error) {
	// 使用 DeepSeek Think 模型，具备深度推理能力
	planModel, err := llm.NewDeepSeekThinkModel(ctx)
	if err != nil {
		return nil, err
	}
	return planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
		ToolCallingChatModel: planModel,
		NewPlan:              NewFlexiblePlan,
	})
}
