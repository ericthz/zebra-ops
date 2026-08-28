// chat 演示基于 Chat Agent 的多轮对话流程，包含会话记忆管理
package main

import (
	"context"
	"fmt"

	"github.com/ericthz/zebra-ops/internal/biz/chat_pipeline"
	"github.com/ericthz/zebra-ops/internal/memory"

	"github.com/cloudwego/eino/schema"
)

func main() {
	ctx := context.Background()
	id := "111"
	// 构造第一轮用户消息，从记忆组件获取历史上下文
	userMessage := &chat_pipeline.UserMessage{
		ID:      id,
		Query:   "你好",
		History: memory.GetSimpleMemory(id).GetMessages(),
	}
	// 构建聊天 Agent
	runner, err := chat_pipeline.BuildChatAgent(ctx)
	if err != nil {
		panic(err)
	}
	// 第一次对话：发送 "你好" 并输出回答
	out, err := runner.Invoke(ctx, userMessage)
	if err != nil {
		panic(err)
	}
	answer := out.Content
	fmt.Println("Q: 你好")
	fmt.Println("A:", answer)
	// 将本轮对话写入记忆组件，供后续对话使用
	memory.GetSimpleMemory(id).SetMessages(schema.UserMessage("你好"))
	memory.GetSimpleMemory(id).SetMessages(schema.SystemMessage(out.Content))
	// 第二次对话：基于已有记忆追问时间
	userMessage = &chat_pipeline.UserMessage{
		ID:      id,
		Query:   "现在是几点",
		History: memory.GetSimpleMemory(id).GetMessages(),
	}
	out, err = runner.Invoke(ctx, userMessage)
	if err != nil {
		panic(err)
	}
	answer = out.Content
	fmt.Println("----------------")
	fmt.Println("Q: 现在是几点")
	fmt.Println("A:", answer)
}
