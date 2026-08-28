// ai_ops 智能告警运维分析入口，演示基于 Plan Agent 的告警诊断流程
package main

import (
	"context"
	"fmt"

	"github.com/ericthz/zebra-ops/internal/biz/plan_execute_replan"
)

func main() {
	ctx := context.Background()
	// 定义告警分析任务的 Prompt，指导 Agent 按步骤执行告警查询、文档检索、日志分析和总结
	query := `
"1. 你是一个智能的服务告警运维分析助手,首先调用工具query_prometheus_alerts获取所有活跃的告警。"
"2. 分别根据告警的名称调用工具query_internal_docs，获取告警名对应的处理方案。"
"3. 完全遵循内部文档的内容进行查询和分析,不允许使用文档外的任何信息。"
"4. 涉及到时间的参数都需要先通过工具get_current_time获取当前时间,再结合用户的时间要求进行传参。"
"5. 涉及到日志的查询,需要先通过日志工具获取相关日志信息，参数必须携带地域和日志主题。"
"6. 分别将告警对应查询到的信息进行总结分析,最后汇总所有告警和总结。"`
	// 构建并执行 Plan Agent，获取最终回答和执行详情
	resp, detail, err := plan_execute_replan.BuildPlanAgent(ctx, query)
	if err != nil {
		panic(err)
	}
	fmt.Println("----- Final Response -----")
	fmt.Println(resp)
	fmt.Println("----- Final detail -----")
	fmt.Println(detail)
}
