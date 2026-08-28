package tools

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	emcp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// 全局 MCP 连接组：懒加载单例，所有调用共享同一 SSE 连接，避免每次请求新建连接。
var (
	mcpGroupOnce sync.Once
	mcpGroup     *reconnectableToolGroup
	mcpGroupErr  error
)

// GetLogMcpTool 获取日志查询 MCP 工具列表，返回可重连的工具组包装。
// 连接懒加载建立一次后复用；当 MCP SSE 会话过期时自动重建连接，
// 工具调用自动路由到最新连接。该函数是并发安全的。
func GetLogMcpTool() ([]tool.BaseTool, error) {
	mcpGroupOnce.Do(func() {
		mcpGroup = &reconnectableToolGroup{}
		mcpGroupErr = mcpGroup.connect()
	})
	if mcpGroupErr != nil {
		return nil, mcpGroupErr
	}
	return mcpGroup.tools(), nil
}

// clsAIOpsTools AI Ops 流程实际需要的 CLS 工具子集。
// 减少工具数量可显著降低小模型（4b）生成工具调用 XML 的出错率。
var clsAIOpsTools = map[string]bool{
	"SearchLog":                    true,
	"DescribeLogContext":           true,
	"DescribeLogHistogram":         true,
	"TextToSearchLogQuery":         true,
	"ConvertTimeStringToTimestamp": true,
	"ConvertTimestampToTimeString": true,
}

// FilterLogMcpTools 仅保留 AI Ops 流程所需的 CLS 工具。
func FilterLogMcpTools(tools []tool.BaseTool) []tool.BaseTool {
	filtered := make([]tool.BaseTool, 0, len(tools))
	for _, t := range tools {
		info, err := t.Info(context.Background())
		if err != nil || info == nil {
			continue
		}
		if clsAIOpsTools[info.Name] {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// reconnectableToolGroup 封装 MCP SSE 连接与工具列表，支持断连后重建。
// 重建后旧的 reconnectableTool 实例通过 toolName 动态路由到新工具。
type reconnectableToolGroup struct {
	mu           sync.RWMutex
	cli          client.MCPClient
	toolMap      map[string]tool.BaseTool // name → 当前最新工具
	toolNames    []string                 // 保持顺序
	reconnecting atomic.Bool
}

// connect 建立 MCP SSE 连接并加载工具列表。
func (grp *reconnectableToolGroup) connect() error {
	grp.mu.Lock()
	defer grp.mu.Unlock()
	return grp.doConnectLocked()
}

// doConnectLocked 实际建立连接（调用方需持有写锁）。
func (grp *reconnectableToolGroup) doConnectLocked() error {
	mcpUrl, err := g.Cfg().GetEffective(context.Background(), "mcp_url")
	if err != nil {
		return err
	}
	ctx := context.Background()
	cli, err := client.NewSSEMCPClient(mcpUrl.String())
	if err != nil {
		return err
	}
	if err = cli.Start(ctx); err != nil {
		return err
	}
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{Name: "zebra-ops", Version: "0.1.0"}
	if _, err = cli.Initialize(ctx, initReq); err != nil {
		return err
	}
	rawTools, err := emcp.GetTools(ctx, &emcp.Config{
		Cli:                   cli,
		ToolCallResultHandler: gracefulClsResultHandler,
	})
	if err != nil {
		return err
	}
	grp.cli = cli
	grp.toolMap = make(map[string]tool.BaseTool, len(rawTools))
	grp.toolNames = make([]string, 0, len(rawTools))
	for _, t := range rawTools {
		info, _ := t.Info(context.Background())
		if info != nil {
			grp.toolMap[info.Name] = t
			grp.toolNames = append(grp.toolNames, info.Name)
		}
	}
	g.Log().Info(ctx, "[MCP] SSE 连接建立成功, 工具数:", len(grp.toolNames))
	return nil
}

// tryReconnect 尝试重建连接（同一时刻只执行一次）。
func (grp *reconnectableToolGroup) tryReconnect() {
	if !grp.reconnecting.CompareAndSwap(false, true) {
		return
	}
	defer grp.reconnecting.Store(false)

	g.Log().Info(context.Background(), "[MCP] 重建 SSE 连接...")
	grp.mu.Lock()
	defer grp.mu.Unlock()

	if grp.cli != nil {
		grp.cli.Close()
	}
	grp.cli = nil
	grp.toolMap = nil
	grp.toolNames = nil

	if err := grp.doConnectLocked(); err != nil {
		g.Log().Error(context.Background(), "[MCP] 重建连接失败:", err.Error())
	}
}

// getTool 根据名称获取当前最新工具（读锁）。
func (grp *reconnectableToolGroup) getTool(name string) (tool.BaseTool, bool) {
	grp.mu.RLock()
	defer grp.mu.RUnlock()
	t, ok := grp.toolMap[name]
	return t, ok
}

// tools 返回所有 reconnectableTool 包装（首次创建时调用）。
func (grp *reconnectableToolGroup) tools() []tool.BaseTool {
	grp.mu.RLock()
	defer grp.mu.RUnlock()
	result := make([]tool.BaseTool, 0, len(grp.toolNames))
	for _, name := range grp.toolNames {
		result = append(result, &reconnectableTool{group: grp, toolName: name})
	}
	return result
}

// reconnectableTool 包装单个 MCP 工具，每次调用动态路由到 group 中的最新工具。
// 即使 group 重建了连接，旧的 reconnectableTool 实例仍能正常工作。
type reconnectableTool struct {
	group    *reconnectableToolGroup
	toolName string
}

func (t *reconnectableTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	inner, ok := t.group.getTool(t.toolName)
	if !ok {
		return nil, fmt.Errorf("tool %s not found", t.toolName)
	}
	return inner.Info(ctx)
}

func (t *reconnectableTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	inner, ok := t.group.getTool(t.toolName)
	if !ok {
		g.Log().Warning(context.Background(), "[MCP] tool not found:", t.toolName)
		return "工具不可用: " + t.toolName, nil
	}
	it, ok := inner.(tool.InvokableTool)
	if !ok {
		return "工具不支持调用", nil
	}
	out, err := it.InvokableRun(ctx, argumentsInJSON, opts...)
	if err != nil {
		// 注意：错误可能很短，截断前必须安全处理，避免 slice bounds panic
		g.Log().Info(context.Background(), "[MCP] tool call error, tool:", t.toolName, "err:", shortErr(err.Error()))
		if isSessionError(err) {
			g.Log().Info(context.Background(), "[MCP] 检测到会话过期，触发异步重连, tool:", t.toolName)
			go t.group.tryReconnect()
		}
		return "工具调用失败: " + err.Error(), nil
	}
	return out, nil
}

func isSessionError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Session not found") ||
		strings.Contains(msg, "SSE connection may have been closed") ||
		strings.Contains(msg, "transport error")
}

// gracefulClsResultHandler 将 CLS 工具的错误结果转换为优雅的文本结果返回，
// 避免以 Go error 终止 Agent，同时让模型把"日志查询失败/主题不存在"当作
// 可继续分析的数据缺失，而不是需要重试的硬错误。
func gracefulClsResultHandler(ctx context.Context, name string, result *mcp.CallToolResult) (*mcp.CallToolResult, error) {
	if result == nil || !result.IsError {
		return result, nil
	}
	// 提取错误文本
	var textParts []string
	for _, c := range result.Content {
		if tc, ok := c.(mcp.TextContent); ok && strings.TrimSpace(tc.Text) != "" {
			textParts = append(textParts, tc.Text)
		}
	}
	text := strings.Join(textParts, "\n")
	if text == "" {
		text = "MCP tool call failed"
	}
	// 日志查询类工具失败时，返回"未查询到日志"的优雅结果，引导模型基于文档继续分析
	graceful := fmt.Sprintf("日志查询未返回数据（原因：%s）。这通常表示该日志主题暂无匹配日志或主题不存在。请基于已获取的内部文档处理方案继续分析，无需重试日志查询。", shortErr(text))
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: graceful}},
	}, nil
}

// shortErr 截断错误文本，保留有用信息（按 rune 截断，避免切断多字节字符）
func shortErr(s string) string {
	s = strings.TrimSpace(s)
	const maxLen = 200
	runes := []rune(s)
	if len(runes) > maxLen {
		return string(runes[:maxLen]) + "..."
	}
	return s
}
