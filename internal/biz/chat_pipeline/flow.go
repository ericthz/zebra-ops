package chat_pipeline

import (
	"context"

	"github.com/ericthz/zebra-ops/internal/components/tools"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
)

// newReactAgentLambda 创建 ReAct 智能体 Lambda 节点
// 配置了聊天模型和多种工具（日志查询、告警查询、时间获取、文档查询）
func newReactAgentLambda(ctx context.Context) (lba *compose.Lambda, err error) {
	// 初始化 ReAct 智能体配置，设置最大步数和工具返回配置
	config := &react.AgentConfig{
		MaxStep:            25,
		ToolReturnDirectly: map[string]struct{}{}}

	// 创建聊天模型实例作为工具调用模型
	chatModelIns11, err := newChatModel(ctx)
	if err != nil {
		return nil, err
	}
	config.ToolCallingModel = chatModelIns11

	// 添加日志查询 MCP 工具
	mcpTool, err := tools.GetLogMcpTool()
	if err != nil {
		return nil, err
	}
	config.ToolsConfig.Tools = mcpTool

	// 添加 Prometheus 告警查询工具
	alertTool, err := tools.NewPrometheusAlertsQueryTool()
	if err != nil {
		return nil, err
	}
	config.ToolsConfig.Tools = append(config.ToolsConfig.Tools, alertTool)

	// 添加当前时间获取工具
	timeTool, err := tools.NewGetCurrentTimeTool()
	if err != nil {
		return nil, err
	}
	config.ToolsConfig.Tools = append(config.ToolsConfig.Tools, timeTool)

	// 添加内部文档查询工具
	docsTool, err := tools.NewQueryInternalDocsTool()
	if err != nil {
		return nil, err
	}
	config.ToolsConfig.Tools = append(config.ToolsConfig.Tools, docsTool)

	// 创建 ReAct 智能体实例
	ins, err := react.NewAgent(ctx, config)
	if err != nil {
		return nil, err
	}

	// 将智能体包装为 Lambda，支持同步生成和流式输出
	lba, err = compose.AnyLambda(ins.Generate, ins.Stream, nil, nil)
	if err != nil {
		return nil, err
	}
	return lba, nil
}
