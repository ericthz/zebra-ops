package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestNewGetCurrentTimeTool(t *testing.T) {
	tool, err := NewGetCurrentTimeTool()
	if err != nil {
		t.Fatalf("NewGetCurrentTimeTool returned error: %v", err)
	}

	info, err := tool.Info(context.Background())
	if err != nil {
		t.Fatalf("tool.Info returned error: %v", err)
	}
	if info.Name != "get_current_time" {
		t.Fatalf("unexpected tool name: %s", info.Name)
	}

	out, err := tool.InvokableRun(context.Background(), "{}")
	if err != nil {
		t.Fatalf("InvokableRun returned error: %v", err)
	}

	var result struct {
		Success   bool   `json:"success"`
		Seconds   int64  `json:"seconds"`
		Timestamp string `json:"timestamp"`
		Message   string `json:"message"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nraw: %s", err, out)
	}
	if !result.Success {
		t.Fatalf("expected success=true, got %+v", result)
	}
	if result.Seconds <= 0 {
		t.Fatalf("expected positive unix seconds, got %d", result.Seconds)
	}
	if !strings.HasPrefix(result.Timestamp, "20") {
		t.Fatalf("unexpected timestamp format: %q", result.Timestamp)
	}
}

func TestNewMysqlCrudTool_RejectsEmptySQL(t *testing.T) {
	tool, err := NewMysqlCrudTool()
	if err != nil {
		t.Fatalf("NewMysqlCrudTool returned error: %v", err)
	}

	out, err := tool.InvokableRun(context.Background(), `{"dsn":"user:pass@tcp(127.0.0.1:3306)/db","sql":"","operate_type":"query"}`)
	if err == nil {
		t.Fatalf("expected error for empty sql, got output: %s", out)
	}
}

func TestNewMysqlCrudTool_RejectsNoOperateTypeForNonSelect(t *testing.T) {
	tool, err := NewMysqlCrudTool()
	if err != nil {
		t.Fatalf("NewMysqlCrudTool returned error: %v", err)
	}

	out, err := tool.InvokableRun(context.Background(), `{"dsn":"user:pass@tcp(127.0.0.1:3306)/db","sql":"DELETE FROM t"}`)
	if err == nil {
		t.Fatalf("expected error for DELETE without operate_type, got output: %s", out)
	}
}
