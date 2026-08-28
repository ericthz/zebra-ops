package llm

import (
	"context"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/gogf/gf/v2/frame/g"
)

// NewDeepSeekThinkModel 创建 DeepSeek 深度思考模型实例，适用于需要复杂推理的场景
func NewDeepSeekThinkModel(ctx context.Context) (cm model.ToolCallingChatModel, err error) {
	model, err := g.Cfg().GetEffective(ctx, "think_chat_model.model")
	if err != nil {
		return nil, err
	}
	api_key, err := g.Cfg().GetEffective(ctx, "think_chat_model.api_key")
	if err != nil {
		return nil, err
	}
	base_url, err := g.Cfg().GetEffective(ctx, "think_chat_model.base_url")
	if err != nil {
		return nil, err
	}
	config := &openai.ChatModelConfig{
		Model:   model.String(),
		APIKey:  api_key.String(),
		BaseURL: base_url.String(),
	}
	cm, err = openai.NewChatModel(ctx, config)
	if err != nil {
		return nil, err
	}
	return cm, nil
}

// NewDeepSeekQuickModel 创建 DeepSeek 快速模型实例，适用于需要低延迟响应的场景
func NewDeepSeekQuickModel(ctx context.Context) (cm model.ToolCallingChatModel, err error) {
	model, err := g.Cfg().GetEffective(ctx, "quick_chat_model.model")
	if err != nil {
		return nil, err
	}
	api_key, err := g.Cfg().GetEffective(ctx, "quick_chat_model.api_key")
	if err != nil {
		return nil, err
	}
	base_url, err := g.Cfg().GetEffective(ctx, "quick_chat_model.base_url")
	if err != nil {
		return nil, err
	}
	config := &openai.ChatModelConfig{
		Model:   model.String(),
		APIKey:  api_key.String(),
		BaseURL: base_url.String(),
	}
	cm, err = openai.NewChatModel(ctx, config)
	if err != nil {
		return nil, err
	}
	return cm, nil
}
