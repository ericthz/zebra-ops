package memory

import (
	"sync"
	"time"

	"github.com/cloudwego/eino/schema"
)

// 会话 TTL 与容量上限：防止内存无限增长
const (
	MemoryTTL     = 24 * time.Hour
	MaxEntries    = 10000
	maxWindowSize = 6
)

// SimpleMemoryMap 存储所有会话内存的全局映射表，key 为会话 ID
var SimpleMemoryMap = make(map[string]*SimpleMemory)

// mu 保护 SimpleMemoryMap 的并发访问锁
var mu sync.Mutex

// GetSimpleMemory 获取会话内存，不存在则创建。
// 每次访问会惰性清理过期条目，并限制总条目数，防止内存泄漏。
func GetSimpleMemory(id string) *SimpleMemory {
	mu.Lock()
	defer mu.Unlock()
	now := time.Now()

	// 惰性清理超过 TTL 的会话
	for k, m := range SimpleMemoryMap {
		if now.Sub(m.lastAccess) > MemoryTTL {
			delete(SimpleMemoryMap, k)
		}
	}
	// 超过上限时淘汰最久未访问的条目
	if len(SimpleMemoryMap) >= MaxEntries {
		var oldestID string
		var oldestTime time.Time
		for k, m := range SimpleMemoryMap {
			if oldestID == "" || m.lastAccess.Before(oldestTime) {
				oldestID = k
				oldestTime = m.lastAccess
			}
		}
		delete(SimpleMemoryMap, oldestID)
	}

	if mem, ok := SimpleMemoryMap[id]; ok {
		mem.lastAccess = now
		return mem
	}
	newMem := &SimpleMemory{
		ID:            id,
		Messages:      []*schema.Message{},
		MaxWindowSize: maxWindowSize,
		lastAccess:    now,
	}
	SimpleMemoryMap[id] = newMem
	return newMem
}

// SimpleMemory 简单会话内存，维护消息滑动窗口，支持并发安全访问
type SimpleMemory struct {
	ID            string            `json:"id"`
	Messages      []*schema.Message `json:"messages"`
	MaxWindowSize int               // 最大消息窗口大小
	lastAccess    time.Time         // 最后访问时间，用于 TTL 淘汰
	mu            sync.Mutex        // 保护 Messages 的并发锁
}

// SetMessages 添加消息到会话内存，超出窗口大小时丢弃最早的消息
func (c *SimpleMemory) SetMessages(msg *schema.Message) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Messages = append(c.Messages, msg)
	if len(c.Messages) > c.MaxWindowSize {
		// 确保成对丢弃消息，保持对话配对关系
		// 计算需要丢弃的消息数量（必须是偶数）
		excess := len(c.Messages) - c.MaxWindowSize
		if excess%2 != 0 {
			excess++ // 确保丢弃偶数条消息
		}
		// 丢弃前面的消息，保持对话配对
		c.Messages = c.Messages[excess:]
	}
}

// GetMessages 获取会话中的所有消息，并更新最后访问时间
func (c *SimpleMemory) GetMessages() []*schema.Message {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastAccess = time.Now()
	return c.Messages
}
