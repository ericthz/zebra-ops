// llm_tool 演示将 MCP 工具和自定义工具绑定到 ChatModel 的完整流程
package main

import (
	"context"
	"fmt"

	"github.com/ericthz/zebra-ops/internal/components/tools"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/gogf/gf/v2/frame/g"
)

func main() {
	ctx := context.Background()
	// 创建 ChatModel（密钥从配置文件读取，可被环境变量 QUICK_CHAT_MODEL_API_KEY 覆盖）
	config := &openai.ChatModelConfig{
		APIKey:  g.Cfg().MustGetEffective(ctx, "quick_chat_model.api_key").String(),
		Model:   g.Cfg().MustGetEffective(ctx, "quick_chat_model.model").String(),
		BaseURL: g.Cfg().MustGetEffective(ctx, "quick_chat_model.base_url").String(),
	}
	chatModel, err := openai.NewChatModel(ctx, config)
	if err != nil {
		panic(err)
	}
	// 获取日志 MCP 工具，用于查询日志信息
	toolList, err := tools.GetLogMcpTool()
	if err != nil {
		panic(err)
	}
	// 获取当前时间工具，用于查询时间相关参数
	timeTool, err := tools.NewGetCurrentTimeTool()
	if err != nil {
		panic(err)
	}
	// 合并所有工具
	toolList = append(toolList, timeTool)
	// 提取工具描述信息（ToolInfo），用于绑定到模型
	toolInfos := make([]*schema.ToolInfo, 0)
	var info *schema.ToolInfo
	for _, todoTool := range toolList {
		info, err = todoTool.Info(ctx)
		if err != nil {
			panic(err)
		}
		toolInfos = append(toolInfos, info)
	}

	// 将工具绑定到 ChatModel，使模型具备 function calling 能力
	err = chatModel.BindTools(toolInfos)
	if err != nil {
		panic(err)
	}

	// 创建处理链，将 ChatModel 作为唯一节点
	chain := compose.NewChain[[]*schema.Message, *schema.Message]()
	chain.AppendChatModel(chatModel, compose.WithNodeName("chat_model"))

	// 编译处理链，生成可执行的 Agent
	agent, err := chain.Compile(ctx)
	if err != nil {
		panic(err)
	}
	// 发送示例请求，询问模型可用的工具列表
	resp, err := agent.Invoke(ctx, []*schema.Message{
		{
			Role:    schema.User,
			Content: "告诉我你有哪些工具可以使用",
		},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(resp.Content)
}
