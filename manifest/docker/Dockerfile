# syntax=docker/dockerfile:1

# ========== build stage ==========
FROM golang:1.26-alpine AS builder
WORKDIR /src

# 依赖层缓存：先复制 go.mod / go.sum
COPY go.mod go.sum ./
RUN go mod download

# 构建 cmd/server（关闭 CGO，产出静态二进制）
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/server ./cmd/server

# ========== runtime stage ==========
FROM alpine:3.21
WORKDIR /app

# CA 证书（调用火山方舟 / 阿里百炼 HTTPS 需要）与 tzdata
RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /out/server /app/server
# 知识库种子文档
COPY docs /app/docs

ENV TZ=Asia/Shanghai
EXPOSE 6872

# 密钥 / 模型 URL 通过环境变量注入（参见 manifest/config/config.yaml.example）：
#   THINK_CHAT_MODEL_API_KEY / QUICK_CHAT_MODEL_API_KEY / EMBEDDING_MODEL_API_KEY / MCP_URL / FILE_DIR
ENTRYPOINT ["/app/server"]
