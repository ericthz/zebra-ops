package chat

import (
	"context"
	"errors"

	"github.com/cloudwego/eino/schema"
	"github.com/gogf/gf/v2/frame/g"
	v1 "github.com/ericthz/zebra-ops/api/chat/v1"
	"github.com/ericthz/zebra-ops/internal/biz/plan_execute_replan"
	"github.com/ericthz/zebra-ops/internal/memory"
)

// AIOps 智能运维分析接口：调用 Plan Agent 分析活跃告警并生成运维报告
func (c *ControllerV1) AIOps(ctx context.Context, req *v1.AIOpsReq) (res *v1.AIOpsRes, err error) {
	// 构建告警分析的 Agent 查询指令，包含获取告警、查询文档、分析根因、生成报告的完整流程
	query := `
"1. 你是一个智能的服务告警分析助手,首先调用工具query_prometheus_alerts获取所有活跃的告警。"
"2. 分别根据告警的名称调用工具query_internal_docs，获取告警名对应的处理方案。"
"3. 完全遵循内部文档的内容进行查询和分析,不允许使用文档外的任何信息。"
"4. 涉及到时间的参数都需要先通过工具get_current_time获取当前时间,再结合工具的时间要求进行传参。"
"5. 涉及到日志的查询,需要先通过日志工具获取相关日志信息，参数必须携带地域和日志主题。如果日志工具调用失败（如主题不存在、会话过期等），不要重试，直接使用已有信息继续分析。"
"6. 当所有必要的文档和告警信息都已获取后，基于已有信息直接撰写并输出完整的告警运维分析报告，不要等待所有工具都成功。日志查询失败不影响报告生成。"
"7. 分别将告警对应查询到的信息进行总结分析,生成告警运维分析报告，格式如下：
告警分析报告
---
# 告警处理详情
## 活跃告警清单
## 告警根因分析N(第N个告警)
## 处理方案执行N(第N个告警)
## 结论
`

	// 调用 Plan Agent 执行告警分析。
	// 4b 本地小模型存在偶发不稳定（规划器不产出工具调用、工具调用 XML 格式错误等），
	// 快速失败时重试，成功或产出报告即返回。
	var resp string
	var detail []string
	const maxAttempts = 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, detail, err = plan_execute_replan.BuildPlanAgent(ctx, query)
		if err == nil && resp != "" {
			break
		}
		if attempt < maxAttempts {
			g.Log().Warning(ctx, "AI Ops 执行失败，重试", "attempt", attempt, "err", err, "respLen", len(resp))
		}
	}
	if err != nil {
		return nil, err
	}
	// Agent 返回空结果视为内部错误
	if resp == "" {
		return nil, errors.New("内部错误")
	}
	res = &v1.AIOpsRes{
		Result: resp,
		Detail: detail,
	}
	// 将分析报告写入后端会话记忆，使后续对话框追问能引用该报告上下文
	if req.Id != "" {
		memory.GetSimpleMemory(req.Id).SetMessages(schema.SystemMessage(resp))
	}
	return res, nil

}
