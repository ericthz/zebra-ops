package plan_execute_replan

import (
	"context"
	"strings"

	"github.com/ericthz/zebra-ops/internal/components/llm"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
)

// NewRePlanAgent 创建并返回一个重规划器 Agent，负责在执行计划遇到问题时重新调整计划。
// 使用 DeepSeek Think 模型作为推理引擎，能够根据执行结果和反馈修正后续计划。
func NewRePlanAgent(ctx context.Context) (adk.Agent, error) {
	// 使用 DeepSeek Think 模型进行重规划推理
	model, err := llm.NewDeepSeekThinkModel(ctx)
	if err != nil {
		return nil, err
	}
	return planexecute.NewReplanner(ctx, &planexecute.ReplannerConfig{
		ChatModel:  model,
		GenInputFn: buildClsAwareReplannerInputFn(),
		NewPlan:    NewFlexiblePlan,
	})
}

// buildClsAwareReplannerInputFn 构造自定义 replanner 输入生成函数。
// 在 executed_steps 末尾追加 CLS 工具失败时的行为指引，防止 replanner 因 CLS 失败反复循环。
func buildClsAwareReplannerInputFn() planexecute.GenModelInputFn {
	return func(ctx context.Context, in *planexecute.ExecutionContext) ([]adk.Message, error) {
		planContent, err := in.Plan.MarshalJSON()
		if err != nil {
			return nil, err
		}
		var stepsBuilder strings.Builder
		for _, s := range in.ExecutedSteps {
			stepsBuilder.WriteString("Step: " + s.Step + "\nResult: " + s.Result + "\n\n")
		}
		executedSteps := stepsBuilder.String()

		// 检测数据是否齐备
		hasAlerts := strings.Contains(executedSteps, "query_prometheus_alerts") && strings.Contains(executedSteps, "\"success\": true")
		hasDocs := strings.Contains(executedSteps, "query_internal_docs") && strings.Contains(executedSteps, "\"content\":")
		// 日志查询是否已尝试（无论成功还是优雅失败）
		hasLogsAttempted := strings.Contains(executedSteps, "SearchLog") ||
			strings.Contains(executedSteps, "DescribeLogContext") ||
			strings.Contains(executedSteps, "DescribeLogHistogram")
		hasClsFailure := strings.Contains(executedSteps, "工具调用失败") ||
			strings.Contains(executedSteps, "日志查询未返回数据")

		if hasAlerts && hasDocs && hasLogsAttempted {
			// 告警 + 文档已获取，日志查询已尝试（结果可能为空），数据足够生成报告 → 强制 respond
			executedSteps += `
[强制指令] 日志查询已尝试完成（可能因日志主题不存在而未返回数据，这是预期行为，不要重试）。
当前已获取的数据足够生成报告：
1. 告警信息：活跃告警清单（服务下线、接口失败率过高、对账差异、地域不匹配等）
2. 处理方案：内部文档已提供各告警的处理步骤
请立即调用 respond 工具生成最终报告，不要再调用 plan 工具。
报告应包含：告警清单、根因分析、处理方案、结论。`
		} else if hasClsFailure {
			executedSteps += "\n[提示] 日志查询失败为预期情况，请根据已有信息尽量完成任务。\n"
		}

		msgs, err := planexecute.ReplannerPrompt.Format(ctx, map[string]any{
			"plan":           string(planContent),
			"input":          formatInput(in.UserInput),
			"executed_steps": executedSteps,
			"plan_tool":      "plan",
			"respond_tool":   "respond",
		})
		if err != nil {
			return nil, err
		}
		return msgs, nil
	}
}

// formatInput 将用户输入消息格式化为字符串
func formatInput(input []adk.Message) string {
	var sb strings.Builder
	for _, msg := range input {
		sb.WriteString(msg.Content)
		sb.WriteString("\n")
	}
	return sb.String()
}
