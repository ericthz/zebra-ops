package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestIsSessionError(t *testing.T) {
	cases := []struct {
		msg  string
		want bool
	}{
		{"Session not found. The SSE connection may have been closed.", true},
		{"SSE connection may have been closed. Please reconnect.", true},
		{"transport error: request failed with status 400", true},
		{"no tool call", false},
		{"tool respond not found in toolsNode indexes", false},
		{"", false},
	}
	for _, c := range cases {
		if got := isSessionError(errFromString(c.msg)); got != c.want {
			t.Errorf("isSessionError(%q) got %v want %v", c.msg, got, c.want)
		}
	}
}

func TestShortErr(t *testing.T) {
	// 短错误原样返回
	if got := shortErr("short error"); got != "short error" {
		t.Errorf("short error got %q", got)
	}
	// 长错误按 rune 截断且不 panic（含中文多字节字符）
	long := strings.Repeat("中文错误内容", 50)
	got := shortErr(long)
	if got == "" {
		t.Error("长错误截断不应为空")
	}
	if len([]rune(got)) > 205 {
		t.Errorf("截断后应接近200 rune, got %d", len([]rune(got)))
	}
	if !strings.HasSuffix(got, "...") {
		t.Errorf("截断后应以...结尾, got %q", got)
	}
}

func TestGracefulClsResultHandler(t *testing.T) {
	ctx := context.Background()

	// 错误结果 → 优雅文本
	errResult := &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: "The specified log topic does not exist"},
		},
	}
	out, err := gracefulClsResultHandler(ctx, "SearchLog", errResult)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if out == nil || out.IsError {
		t.Error("错误结果应转为非错误结果")
	}
	if len(out.Content) == 0 {
		t.Fatal("输出应包含文本内容")
	}
	text := ""
	if tc, ok := out.Content[0].(mcp.TextContent); ok {
		text = tc.Text
	}
	if !strings.Contains(text, "日志查询未返回数据") {
		t.Errorf("优雅文本应包含提示, got: %q", text)
	}
	if !strings.Contains(text, "topic does not exist") {
		t.Errorf("优雅文本应包含原因, got: %q", text)
	}

	// 成功结果 → 原样返回
	okResult := &mcp.CallToolResult{
		IsError: false,
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: `{"logs":[]}`}},
	}
	out2, err := gracefulClsResultHandler(ctx, "SearchLog", okResult)
	if err != nil || out2 != okResult {
		t.Error("成功结果应原样返回")
	}

	// nil 结果
	out3, err := gracefulClsResultHandler(ctx, "SearchLog", nil)
	if err != nil || out3 != nil {
		t.Error("nil 结果应原样返回 nil")
	}
}

type errString string

func (e errString) Error() string { return string(e) }

func errFromString(s string) error {
	if s == "" {
		return nil
	}
	return errString(s)
}
