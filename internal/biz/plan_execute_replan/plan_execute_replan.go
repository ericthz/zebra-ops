package plan_execute_replan

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/cloudwego/eino-examples/adk/common/prints"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/schema"
	"github.com/gogf/gf/v2/frame/g"
)

// cleanDetail 清洗 Agent 原始输出，去掉用户无意义的元数据
func cleanDetail(raw string) string {
	s := raw
	// 逐行过滤，跳过元数据行
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		trim := strings.TrimSpace(line)
		// 跳过元数据
		if strings.HasPrefix(trim, "finish_reason:") ||
			strings.HasPrefix(trim, "usage:") ||
			strings.HasPrefix(trim, "reasoning content:") {
			continue
		}
		// 去掉 tool_calls 中 index[0x...] 前缀
		if strings.HasPrefix(trim, "index[0x") {
			if idx := strings.Index(trim, "]:"); idx != -1 {
				trim = strings.TrimSpace(trim[idx+2:])
			}
		}
		lines = append(lines, line)
	}
	s = strings.Join(lines, "\n")
	// 去掉连续空行
	s = regexp.MustCompile(`\n{3,}`).ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

// BuildPlanAgent 构建并运行完整的计划-执行-重规划 Agent 工作流。
// 它依次创建规划器、执行器、重规划器，组合为 PlanExecuteAgent，
// 然后将用户查询提交执行，迭代获取事件流，最终返回最后一条消息的文本内容及各步骤详情。
func BuildPlanAgent(ctx context.Context, query string) (string, []string, error) {
	return BuildPlanAgentStream(ctx, query, nil)
}

// BuildPlanAgentStream 在 BuildPlanAgent 基础上支持流式进度回调。
// onStep 会在每次产出步骤明细时被调用（可为 nil），用于 SSE 实时推送进度。
func BuildPlanAgentStream(ctx context.Context, query string, onStep func(step string)) (string, []string, error) {
	// 创建规划器 Agent
	planAgent, err := NewPlanner(ctx)
	if err != nil {
		return "", []string{}, err
	}
	// 创建执行器 Agent
	executeAgent, err := NewExecutor(ctx)
	if err != nil {
		return "", []string{}, err
	}
	// 创建重规划器 Agent
	replanAgent, err := NewRePlanAgent(ctx)
	if err != nil {
		return "", []string{}, err
	}
	// 组合三个组件为计划执行 Agent，最大迭代次数为 10
	planExecuteAgent, err := planexecute.New(ctx, &planexecute.Config{
		Planner:       planAgent,
		Executor:      executeAgent,
		Replanner:     replanAgent,
		MaxIterations: 10,
	})
	if err != nil {
		return "", []string{}, fmt.Errorf("build PlanExecuteAgent Error: %v", err)
	}
	// 使用 Runner 驱动 Agent 执行用户查询
	r := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent: planExecuteAgent,
	})
	iter := r.Query(ctx, query)
	var lastMessage adk.Message
	var lastAssistantText string
	var bestReport string
	var finalReport string
	var detail []string
	// 迭代处理 Agent 返回的事件流
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		fmt.Println("current event")
		prints.Event(event)
		// 收集有输出的事件，记录各步骤详情（清洗元数据后保留）
		if event.Output != nil {
			msg, _, err := adk.GetMessage(event)
			if err != nil {
				continue
			}
			lastMessage = msg
			cleaned := cleanDetail(msg.String())
			detail = append(detail, cleaned)
			if onStep != nil {
				onStep(cleaned)
			}
			// 重规划器通过 respond 工具输出最终报告，内容为 {"response":"..."}，
			// 优先提取该字段作为最终结果，避免返回中间的步骤消息
			if resp := extractRespond(msg.Content); resp != "" {
				finalReport = resp
			}
			// 执行器误调用兜底 respond 工具时，从工具调用参数中直接提取报告
			if resp := extractRespondFromToolCalls(msg); resp != "" {
				finalReport = resp
			}
			// 记录最后一条 assistant 正文（排除工具结果等非正文输出）
			content := strings.TrimSpace(msg.Content)
			if msg.Role == schema.Assistant && content != "" && !strings.HasPrefix(content, "{") {
				lastAssistantText = content
				// 记录包含报告标识的最长报告文本（兜底用）
				if isReportContent(content) && len(content) > len(bestReport) {
					bestReport = content
				}
			}
			// 报告已生成（executor 一轮即可完成数据收集与报告撰写），
			// 提前终止避免 replanner 反复重规划浪费大量时间。
			// 要求报告达到一定完整度（覆盖处理方案），避免中断在部分报告上。
			if finalReport != "" {
				return normalizeMarkdown(finalReport), detail, nil
			}
			if isCompleteReport(bestReport) {
				return normalizeMarkdown(bestReport), detail, nil
			}
		}
	}
	// 优先级：respond 报告 > 报告标识正文 > 最后一条 assistant 正文 > 最后一条消息
	if finalReport != "" {
		return normalizeMarkdown(finalReport), detail, nil
	}
	if strings.TrimSpace(bestReport) != "" {
		return normalizeMarkdown(bestReport), detail, nil
	}
	if strings.TrimSpace(lastAssistantText) != "" {
		return normalizeMarkdown(lastAssistantText), detail, nil
	}
	if lastMessage == nil {
		return "", []string{}, fmt.Errorf("get lastMessage Error")
	}
	return normalizeMarkdown(lastMessage.Content), detail, nil
}

// isReportContent 判断文本是否包含报告标识，用于兜底时挑选最终报告。
func isReportContent(s string) bool {
	markers := []string{"告警分析报告", "告警运维分析报告", "# 告警处理详情"}
	for _, m := range markers {
		if strings.Contains(s, m) {
			return true
		}
	}
	return false
}

// isCompleteReport 判断报告是否达到可返回的完整度：
// 足够长，且包含"处理方案"段落（说明覆盖了各告警的处理步骤）。
func isCompleteReport(s string) bool {
	if len(s) < 500 {
		return false
	}
	return strings.Contains(s, "处理方案执行") || strings.Contains(s, "处理方案")
}

// normalizeMarkdown 清洗模型输出中的 markdown 嵌套标题。
// 模型常把模板字面量直接拼进标题，产生 "## # 告警处理详情"、"### ## 活跃告警清单"
// 这类重复的 # 标记，统一压缩为外层层级。
var reNestedHash = regexp.MustCompile(`^(\s*)(#{2,})\s*#+\s*`)

func normalizeMarkdown(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = reNestedHash.ReplaceAllString(line, "${1}${2} ")
	}
	return strings.Join(lines, "\n")
}

// extractRespond 从 replanner respond 工具输出中提取 response 字段。
// respond 输出为 JSON 字符串：{"response":"最终报告内容"}
func extractRespond(content string) string {
	var out struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		return ""
	}
	return strings.TrimSpace(out.Response)
}

// extractRespondFromToolCalls 从消息的工具调用中提取 respond 参数。
// 执行器误调用兜底 respond 工具时，报告内容在 ToolCalls[].Function.Arguments 中。
func extractRespondFromToolCalls(msg adk.Message) string {
	for _, tc := range msg.ToolCalls {
		if tc.Function.Name == "respond" {
			var args struct {
				Response string `json:"response"`
			}
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err == nil {
				if resp := strings.TrimSpace(args.Response); resp != "" {
					return resp
				}
			}
		}
	}
	return ""
}

// flexiblePlan 自定义 Plan 实现，兼容 4b 等小模型输出的两种格式：
//   - 标准格式：{"steps":["Step 1","Step 2"]}
//   - 字符串化格式：{"steps":"[\"Step 1\",\"Step 2\"]"}
//
// 默认 sonic.Unmarshal 在类型不匹配时会报错，这里做一层前置修正。
type flexiblePlan struct {
	Steps []string `json:"steps"`
}

func (p *flexiblePlan) FirstStep() string {
	if len(p.Steps) == 0 {
		return ""
	}
	return p.Steps[0]
}

func (p *flexiblePlan) MarshalJSON() ([]byte, error) {
	type planTyp flexiblePlan
	return json.Marshal((*planTyp)(p))
}

func (p *flexiblePlan) UnmarshalJSON(bytes []byte) error {
	type planTyp flexiblePlan

	// 先解析 steps 字段为 raw JSON
	var raw struct {
		Steps json.RawMessage `json:"steps"`
	}
	if err := json.Unmarshal(bytes, &raw); err != nil {
		return err
	}
	if raw.Steps == nil {
		return fmt.Errorf("missing steps field")
	}

	// 情况1：steps 是 JSON 数组 → 直接反序列化为 []string
	var arr []string
	if err := json.Unmarshal(raw.Steps, &arr); err == nil {
		p.Steps = arr
		return nil
	}

	// 情况2：steps 是字符串（stringified array）→ 解析字符串再转为 []string
	var s string
	if err := json.Unmarshal(raw.Steps, &s); err == nil {
		// 尝试将字符串解析为 JSON 数组
		if err2 := json.Unmarshal([]byte(s), &arr); err2 == nil {
			p.Steps = arr
			g.Log().Debug(context.Background(), "[flexiblePlan] 修复 stringified array 格式, 步骤数:", len(arr))
			return nil
		}
		// 字符串不是 JSON 数组，当作单个步骤
		p.Steps = []string{s}
		return nil
	}

	// 情况3：回退到标准反序列化
	return json.Unmarshal(bytes, (*planTyp)(p))
}

// NewFlexiblePlan 创建 flexiblePlan 实例，用于 PlannerConfig.NewPlan
func NewFlexiblePlan(_ context.Context) planexecute.Plan {
	return &flexiblePlan{}
}
