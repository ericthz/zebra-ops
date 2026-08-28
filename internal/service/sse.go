package service

import (
	"context"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/container/gmap"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/util/guid"
)

// Client 表示一个 SSE 客户端连接，包含唯一标识和 HTTP 请求上下文
type Client struct {
	Id      string         // 客户端唯一标识
	Request *ghttp.Request // 关联的 HTTP 请求对象
}

// Hub 管理所有 SSE 客户端连接，使用线程安全的 Map 存储
type Hub struct {
	clients *gmap.StrAnyMap // 客户端连接映射表，key 为客户端 ID
}

// NewHub 创建 SSE 连接管理实例
func NewHub() *Hub {
	return &Hub{
		clients: gmap.NewStrAnyMap(true),
	}
}

// Create 创建新的 SSE 连接，设置必要的 HTTP 头并初始化客户端
func (s *Hub) Create(_ context.Context, r *ghttp.Request) (*Client, error) {
	// 设置 SSE 必要的 HTTP 头，确保浏览器正确处理事件流
	r.Response.Header().Set("Content-Type", "text/event-stream")
	r.Response.Header().Set("Cache-Control", "no-cache")
	r.Response.Header().Set("Connection", "keep-alive")
	r.Response.Header().Set("Access-Control-Allow-Origin", "*")

	// 生成或获取客户端 ID，优先使用请求中的 client_id
	clientId := r.Get("client_id", guid.S()).String()
	client := &Client{
		Id:      clientId,
		Request: r,
	}
	// 发送连接建立成功事件，通知客户端可以开始接收数据
	r.Response.Writefln("id: %s", clientId)
	r.Response.Writefln("event: connected")
	r.Response.Writefln("data: {\"status\": \"connected\", \"client_id\": \"%s\"}\n", clientId)
	r.Response.Flush()
	return client, nil
}

// SendToClient 向指定客户端发送 SSE 事件消息
func (c *Client) SendToClient(eventType, data string) bool {
	// 格式化 SSE 消息，包含时间戳作为 ID、事件类型和数据
	msg := fmt.Sprintf(
		"id: %d\nevent: %s\ndata: %s\n\n",
		time.Now().UnixNano(), eventType, data,
	)
	// 写入响应缓冲区并立即刷新，确保客户端实时接收
	c.Request.Response.Write(msg)
	c.Request.Response.Flush()
	return true
}
