package observability

import (
	"context"
	"fmt"
	"time"

	"github.com/ericthz/zebra-ops/internal/components/vectorstore"
)

// CheckLiveness 存活检查：进程活着即返回 ok
func CheckLiveness() bool {
	return true
}

// CheckReadiness 就绪检查：探测关键依赖（Milvus）是否可用。
// 使用短超时避免阻塞探针请求。
func CheckReadiness(ctx context.Context) error {
	probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := vectorstore.Ping(probeCtx); err != nil {
		return fmt.Errorf("milvus unavailable: %w", err)
	}
	return nil
}
