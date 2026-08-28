package chat_pipeline

import (
	"context"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// BuildChatAgent 构建聊天智能体的编排图
// 将用户输入、RAG 检索、聊天模板和 ReAct 智能体组合成一个完整的对话流程
func BuildChatAgent(ctx context.Context) (r compose.Runnable[*UserMessage, *schema.Message], err error) {
	// 定义图中各节点的名称常量
	const (
		InputToRag      = "InputToRag"      // 输入到 RAG 的转换节点
		ChatTemplate    = "ChatTemplate"    // 聊天模板节点
		ReactAgent      = "ReactAgent"      // ReAct 智能体节点
		MilvusRetriever = "MilvusRetriever" // Milvus 向量检索节点
		InputToChat     = "InputToChat"     // 输入到聊天的转换节点
	)

	// 创建一个新的编排图，输入为 UserMessage，输出为 Message
	g := compose.NewGraph[*UserMessage, *schema.Message]()

	// 添加节点：将用户消息转换为 RAG 查询字符串
	_ = g.AddLambdaNode(InputToRag, compose.InvokableLambdaWithOption(newInputToRagLambda), compose.WithNodeName("UserMessageToRag"))

	// 创建并添加聊天模板节点
	chatTemplateKeyOfChatTemplate, err := newChatTemplate()
	if err != nil {
		return nil, err
	}
	_ = g.AddChatTemplateNode(ChatTemplate, chatTemplateKeyOfChatTemplate)

	// 创建并添加 ReAct 智能体节点
	reactAgentKeyOfLambda, err := newReactAgentLambda(ctx)
	if err != nil {
		return nil, err
	}
	_ = g.AddLambdaNode(ReactAgent, reactAgentKeyOfLambda, compose.WithNodeName("ReActAgent"))

	// 创建并添加 Milvus 向量检索节点
	milvusRetrieverKeyOfRetriever, err := newRetriever(ctx)
	if err != nil {
		return nil, err
	}
	// 注意下面的 output key 设置，把查询出来的设置为了documents，匹配 ChatTemplate 里面说prompt
	_ = g.AddRetrieverNode(MilvusRetriever, milvusRetrieverKeyOfRetriever, compose.WithOutputKey("documents"))

	// 添加节点：将用户消息转换为包含内容、历史和日期的 map
	_ = g.AddLambdaNode(InputToChat, compose.InvokableLambdaWithOption(newInputToChatLambda), compose.WithNodeName("UserMessageToChat"))

	// 定义节点间的边，构建完整的处理流程
	// START -> InputToRag: 开始节点连接到 RAG 输入转换
	_ = g.AddEdge(compose.START, InputToRag)
	// START -> InputToChat: 开始节点同时连接到聊天输入转换
	_ = g.AddEdge(compose.START, InputToChat)
	// ReactAgent -> END: ReAct 智能体输出为最终结果
	_ = g.AddEdge(ReactAgent, compose.END)
	// InputToRag -> MilvusRetriever: RAG 输入转换后进行向量检索
	_ = g.AddEdge(InputToRag, MilvusRetriever)
	// MilvusRetriever -> ChatTemplate: 检索结果传递给聊天模板
	_ = g.AddEdge(MilvusRetriever, ChatTemplate)
	// InputToChat -> ChatTemplate: 聊天输入也传递给聊天模板
	_ = g.AddEdge(InputToChat, ChatTemplate)
	// ChatTemplate -> ReactAgent: 模板填充后传递给 ReAct 智能体
	_ = g.AddEdge(ChatTemplate, ReactAgent)

	// 编译图，设置图名称和节点触发模式（所有前驱节点完成才触发）
	r, err = g.Compile(ctx, compose.WithGraphName("ChatAgent"), compose.WithNodeTriggerMode(compose.AllPredecessor))
	if err != nil {
		return nil, err
	}
	return r, err
}
