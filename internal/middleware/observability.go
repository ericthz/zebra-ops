package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/ericthz/zebra-ops/internal/observability"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

// ctxKey context key 类型，用于存储 request_id
type ctxKey struct{}

// RequestIDMiddleware 为每个请求生成 request_id 并注入 context 与响应头
func RequestIDMiddleware(r *ghttp.Request) {
	rid := r.GetHeader("X-Request-Id")
	if rid == "" {
		rid = uuid.NewString()
	}
	r.SetCtx(context.WithValue(r.Context(), ctxKey{}, rid))
	r.Response.Header().Set("X-Request-Id", rid)
	r.Middleware.Next()
}

// RequestIDFromCtx 从 context 取出 request_id
func RequestIDFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKey{}).(string); ok {
		return v
	}
	return ""
}

// AccessLogMiddleware 结构化访问日志（slog JSON）
func AccessLogMiddleware(r *ghttp.Request) {
	start := time.Now()
	r.Middleware.Next()

	status := r.Response.Status
	if status == 0 {
		status = 200
	}
	slog.Info("http_request",
		"request_id", RequestIDFromCtx(r.Context()),
		"method", r.Method,
		"path", r.URL.Path,
		"status", status,
		"duration_ms", time.Since(start).Milliseconds(),
		"remote_addr", r.RemoteAddr,
	)
}

// MetricsMiddleware 采集请求计数与耗时直方图
func MetricsMiddleware(r *ghttp.Request) {
	start := time.Now()
	r.Middleware.Next()

	status := r.Response.Status
	if status == 0 {
		status = 200
	}
	labels := map[string]string{
		"method": r.Method,
		"path":   r.URL.Path,
		"status": itoa(status),
	}
	observability.Default.Inc("http_requests_total", "Total number of HTTP requests.", labels)
	observability.Default.Observe("http_request_duration_seconds", "HTTP request latency.", labels, time.Since(start))
}

// itoa 将整数转换为字符串，避免引入 strconv 依赖
func itoa(v int) string {
	if v == 0 {
		return "200"
	}
	const digits = "0123456789"
	if v < 10 {
		return string(digits[v])
	}
	var buf [3]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = digits[v%10]
		v /= 10
	}
	return string(buf[i:])
}
