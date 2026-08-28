package callbacks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/callbacks"
)

// LogCallbackConfig 日志回调配置，控制日志输出的详细程度
type LogCallbackConfig struct {
	Detail bool // 是否输出详细信息（入参/出参）
	Debug  bool // 是否以格式化 JSON 输出，仅在 Detail=true 时生效
}

// LogCallback 创建一个日志回调处理器，用于在组件运行开始和结束时打印日志
func LogCallback(config *LogCallbackConfig) callbacks.Handler {
	// 若未传入配置，使用默认配置（开启详细输出）
	if config == nil {
		config = &LogCallbackConfig{
			Detail: true,
		}
	}

	builder := callbacks.NewHandlerBuilder()
	builder.OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
		fmt.Printf("[view start]:[%s:%s:%s]\n", info.Component, info.Type, info.Name)
		if config.Detail {
			var b []byte
			if config.Debug {
				b, _ = json.MarshalIndent(input, "", "  ")
			} else {
				b, _ = json.Marshal(input)
			}
			fmt.Printf("%s\n", string(b))
		}
		return ctx
	})
	builder.OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
		fmt.Printf("[view end]:[%s:%s:%s]\n", info.Component, info.Type, info.Name)
		return ctx
	})
	return builder.Build()
}
