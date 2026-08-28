// Package plan_execute_replan 实现基于计划执行与重规划的 Agent 工作流，
// 包含规划器（Planner）、执行器（Executor）、重规划器（Replanner）三个核心组件。
package plan_execute_replan

import (
	"context"

	"github.com/ericthz/zebra-ops/internal/components/llm"
	"github.com/ericthz/zebra-ops/internal/components/tools"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/compose"
)

// NewExecutor 创建并返回一个执行器 Agent，负责根据计划调用工具逐步执行任务。
// 执行器挂载了日志查询、Prometheus 告警查询、内部文档查询、当前时间获取等工具，
// 并使用 DeepSeek Quick 模型作为推理引擎。
func NewExecutor(ctx context.Context) (adk.Agent, error) {
	// 日志查询工具（MCP）——仅保留 AI Ops 流程所需的 CLS 工具子集，
	// 减少工具数量可降低小模型生成工具调用 XML 的出错率
	mcpTool, err := tools.GetLogMcpTool()
	if err != nil {
		return nil, err
	}
	toolList := tools.FilterLogMcpTools(mcpTool)
	// Prometheus 告警查询工具
	alertTool, err := tools.NewPrometheusAlertsQueryTool()
	if err != nil {
		return nil, err
	}
	toolList = append(toolList, alertTool)
	// 内部文档查询工具
	docsTool, err := tools.NewQueryInternalDocsTool()
	if err != nil {
		return nil, err
	}
	toolList = append(toolList, docsTool)
	// 当前时间获取工具
	timeTool, err := tools.NewGetCurrentTimeTool()
	if err != nil {
		return nil, err
	}
	toolList = append(toolList, timeTool)
	// 兜底 respond 工具：防止模型在执行器中误调用 respond（属于 replanner 的工具）导致报错
	respondTool, err := tools.NewRespondFallbackTool()
	if err != nil {
		return nil, err
	}
	toolList = append(toolList, respondTool)
	// 使用 DeepSeek Quick 模型作为执行器的推理模型
	execModel, err := llm.NewDeepSeekQuickModel(ctx)
	if err != nil {
		return nil, err
	}
	return planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
		Model: execModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: toolList,
			},
		},
		MaxIterations: 999999,
	})
}
