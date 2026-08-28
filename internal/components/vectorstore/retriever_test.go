package vectorstore

import (
	"context"
	"strings"
	"testing"
)

// isInfraUnavailable 判断是否因环境未就绪（Milvus 未启动 / embedding 未配置）而跳过集成测试
func isInfraUnavailable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// Milvus 不可达
	if strings.Contains(msg, "connect") || strings.Contains(msg, "connection") ||
		strings.Contains(msg, "refused") || strings.Contains(msg, "deadline") || strings.Contains(msg, "timeout") {
		return true
	}
	// embedding 模型未配置（测试环境无 config.yaml）
	return strings.Contains(msg, "model parameter")
}

func TestNewMilvusRetriever(t *testing.T) {
	ctx := context.Background()
	r, err := NewRetriever(ctx)
	if err != nil {
		if isInfraUnavailable(err) {
			t.Skipf("Milvus not available, skip integration test: %v", err)
		}
		t.Fatalf("NewMilvusRetriever returned error: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil retriever")
	}
}

func TestMilvusRetrieve_NoError(t *testing.T) {
	ctx := context.Background()
	r, err := NewRetriever(ctx)
	if err != nil {
		if isInfraUnavailable(err) {
			t.Skipf("Milvus not available, skip integration test: %v", err)
		}
		t.Fatalf("NewMilvusRetriever returned error: %v", err)
	}

	docs, err := r.Retrieve(ctx, "Zebra Ops 服务为什么下线？")
	if err != nil {
		if isInfraUnavailable(err) {
			t.Skipf("environment not ready (embedding not configured), skip: %v", err)
		}
		t.Fatalf("Retrieve returned error: %v", err)
	}
	// 知识库为空时允许返回 0 条，仅验证调用链路无错误
	t.Logf("retrieved %d docs", len(docs))
}
