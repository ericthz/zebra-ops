// server 应用主入口，初始化 HTTP 服务并注册路由、中间件和可观测性端点
package main

import (
	"embed"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ericthz/zebra-ops/internal/controller/chat"
	"github.com/ericthz/zebra-ops/internal/middleware"
	"github.com/ericthz/zebra-ops/internal/observability"
	"github.com/ericthz/zebra-ops/internal/service"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gctx"
)

//go:embed static/index.html static/css/* static/js/* static/favicon.svg
var frontendFS embed.FS

func main() {
	// 日志目录，默认 logs/，可通过 LOG_DIR 环境变量覆盖
	logDir := "logs"
	if v := os.Getenv("LOG_DIR"); v != "" {
		logDir = v
	}
	_ = os.MkdirAll(logDir, 0755)
	logFile, err := os.OpenFile(filepath.Join(logDir, "zebra-ops.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("failed to open log file: %v", err)
	}
	defer logFile.Close()

	// slog 同时输出到 stdout + 日志文件
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	slog.SetDefault(slog.New(slog.NewJSONHandler(multiWriter, nil)))

	// 标准 log 包也输出到同一目标（覆盖 log.Printf 调用）
	log.SetOutput(multiWriter)

	ctx := gctx.New()
	// 上传目录可通过配置文件 file_dir 覆盖
	if v, err := g.Cfg().Get(ctx, "file_dir"); err == nil && v.String() != "" {
		service.UploadDir = v.String()
	}

	// 创建 HTTP 服务实例
	s := g.Server()
	// 注册 /api 路由组，挂载中间件和控制器
	s.Group("/api", func(group *ghttp.RouterGroup) {
		group.Middleware(middleware.CORSMiddleware)      // 跨域处理
		group.Middleware(middleware.RequestIDMiddleware) // 请求 ID 注入
		group.Middleware(middleware.AccessLogMiddleware) // 访问日志
		group.Middleware(middleware.MetricsMiddleware)   // 指标采集
		group.Middleware(middleware.ResponseMiddleware)  // 统一响应格式
		group.Bind(chat.NewV1())                         // 绑定聊天控制器
	})

	// 健康检查端点（存活探针）
	s.BindHandler("/healthz", func(r *ghttp.Request) {
		r.Response.Write("ok")
	})
	// 就绪探针，检查依赖服务是否就绪
	s.BindHandler("/readyz", func(r *ghttp.Request) {
		if err := observability.CheckReadiness(r.Context()); err != nil {
			r.Response.WriteStatus(503, "not ready: "+err.Error())
			return
		}
		r.Response.Write("ok")
	})
	// Prometheus 指标采集端点
	s.BindHandler("/metrics", func(r *ghttp.Request) {
		r.Response.Header().Set("Content-Type", "text/plain; version=0.0.4")
		r.Response.Write(observability.Default.Render())
	})

	// 嵌入前端静态文件，挂载到根路径
	sub, err := fs.Sub(frontendFS, "static")
	if err != nil {
		g.Log().Fatal(ctx, err)
	}
	s.BindHandler("/*", func(r *ghttp.Request) {
		http.FileServer(http.FS(sub)).ServeHTTP(r.Response.Writer, r.Request)
	})

	// 启动 HTTP 服务，监听 6872 端口
	s.SetPort(6872)
	s.Run()
}
