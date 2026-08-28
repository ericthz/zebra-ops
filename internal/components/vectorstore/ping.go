package vectorstore

import (
	"context"
	"fmt"
	"time"

	cli "github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/gogf/gf/v2/frame/g"
)

// Ping 探测 Milvus 连通性（短超时、无副作用，用于就绪检查）
func Ping(ctx context.Context) error {
	milvusAddr := "localhost:19530"
	if v, err := g.Cfg().GetEffective(ctx, "milvus_url"); err == nil && v.String() != "" {
		milvusAddr = v.String()
	}

	probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	c, err := cli.NewClient(probeCtx, cli.Config{Address: milvusAddr, DBName: "default"})
	if err != nil {
		return fmt.Errorf("connect milvus failed: %w", err)
	}
	defer c.Close()
	return nil
}
