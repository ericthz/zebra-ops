package vectorstore

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	cli "github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

var (
	clientMu    sync.Mutex
	cached      cli.Client
	lastChecked time.Time
)

// clientHealthInterval 健康检查间隔：在该间隔内复用缓存连接，不重复探测
const clientHealthInterval = 30 * time.Second

// Client 返回复用的 Milvus 客户端（懒初始化 + 定时健康检查，失效自动重建）。
// 相比每次请求新建连接，避免 gRPC 连接泄漏与握手开销。
func Client(ctx context.Context) (cli.Client, error) {
	clientMu.Lock()
	defer clientMu.Unlock()

	if cached != nil && time.Since(lastChecked) < clientHealthInterval {
		return cached, nil
	}
	if cached != nil {
		checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		_, err := cached.CheckHealth(checkCtx)
		cancel()
		if err == nil {
			lastChecked = time.Now()
			return cached, nil
		}
		_ = cached.Close()
		cached = nil
	}
	c, err := NewClient(ctx)
	if err != nil {
		return nil, err
	}
	cached = c
	lastChecked = time.Now()
	return cached, nil
}

// NewClient 创建新的 Milvus 客户端连接，自动初始化数据库和集合结构
func NewClient(ctx context.Context) (cli.Client, error) {
	// Milvus 地址从配置读取，可用环境变量 MILVUS_URL 覆盖
	milvusAddr := "localhost:19530"
	if v, err := g.Cfg().GetEffective(ctx, "milvus_url"); err == nil && v.String() != "" {
		milvusAddr = v.String()
	}

	// Milvus 连接设置超时，避免 Milvus 不可用时无限阻塞
	dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 1. 先连接default数据库
	defaultClient, err := cli.NewClient(dialCtx, cli.Config{
		Address: milvusAddr,
		DBName:  "default",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to default database: %w", err)
	}
	// 2. 检查agent数据库是否存在，不存在则创建
	databases, err := defaultClient.ListDatabases(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}
	agentDBExists := false
	for _, db := range databases {
		if db.Name == MilvusDBName {
			agentDBExists = true
			break
		}
	}
	if !agentDBExists {
		err = defaultClient.CreateDatabase(ctx, MilvusDBName)
		if err != nil {
			return nil, fmt.Errorf("failed to create agent database: %w", err)
		}
	}

	// 3. 创建连接到agent数据库的客户端
	agentClient, err := cli.NewClient(dialCtx, cli.Config{
		Address: milvusAddr,
		DBName:  MilvusDBName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent database: %w", err)
	}
	// 4. 检查biz collection是否存在，不存在则创建
	collections, err := agentClient.ListCollections(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	bizCollectionExists := false
	for _, collection := range collections {
		if collection.Name == MilvusCollectionName {
			bizCollectionExists = true
			break
		}
	}

	if bizCollectionExists {
		// 尝试加载已存在的集合，加载失败则删除重建
		err = agentClient.LoadCollection(ctx, MilvusCollectionName, false)
		if err != nil {
			_ = agentClient.DropCollection(ctx, MilvusCollectionName)
			bizCollectionExists = false
		}
	}

	if !bizCollectionExists {
		// 创建biz collection的schema
		schema := &entity.Schema{
			CollectionName: MilvusCollectionName,
			Description:    "Business knowledge collection",
			Fields:         fields,
		}

		err = agentClient.CreateCollection(ctx, schema, entity.DefaultShardNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to create biz collection: %w", err)
		}

		// 为id字段创建autoindex索引
		idIndex, err := entity.NewIndexAUTOINDEX(entity.L2)
		if err != nil {
			return nil, fmt.Errorf("failed to create id index: %w", err)
		}
		err = agentClient.CreateIndex(ctx, MilvusCollectionName, "id", idIndex, false)
		if err != nil {
			return nil, fmt.Errorf("failed to create id index: %w", err)
		}

		// 为content字段创建autoindex索引
		contentIndex, err := entity.NewIndexAUTOINDEX(entity.L2)
		if err != nil {
			return nil, fmt.Errorf("failed to create content index: %w", err)
		}
		err = agentClient.CreateIndex(ctx, MilvusCollectionName, "content", contentIndex, false)
		if err != nil {
			return nil, fmt.Errorf("failed to create content index: %w", err)
		}

		// 为vector字段创建autoindex索引（FloatVector 用 L2 度量）
		vectorIndex, err := entity.NewIndexAUTOINDEX(entity.L2)
		if err != nil {
			return nil, fmt.Errorf("failed to create vector index: %w", err)
		}
		err = agentClient.CreateIndex(ctx, MilvusCollectionName, "vector", vectorIndex, false)
		if err != nil {
			return nil, fmt.Errorf("failed to create vector index: %w", err)
		}

		// 集合必须加载到内存后才能搜索
		err = agentClient.LoadCollection(ctx, MilvusCollectionName, false)
		if err != nil {
			return nil, fmt.Errorf("failed to load collection: %w", err)
		}
	}

	// 关闭default数据库连接
	defaultClient.Close()

	return agentClient, nil
}

// Milvus 数据库与集合名（唯一真相源）
const (
	MilvusDBName         = "agent"
	MilvusCollectionName = "biz"
)

var fields = []*entity.Field{
	{
		Name:     "id",
		DataType: entity.FieldTypeVarChar,
		TypeParams: map[string]string{
			"max_length": "256",
		},
		PrimaryKey: true,
	},
	{
		Name:     "vector", // 与 embedder 输出维度一致（float 2048 维）
		DataType: entity.FieldTypeFloatVector,
		TypeParams: map[string]string{
			"dim": "2048",
		},
	},
	{
		Name:     "content",
		DataType: entity.FieldTypeVarChar,
		TypeParams: map[string]string{
			"max_length": "8192",
		},
	},
	{
		Name:     "metadata",
		DataType: entity.FieldTypeJSON,
	},
}
