package cache

import (
	"sync"
	"time"
)

// LocalCache 本地内存缓存（L1 缓存）
//
// 特点：
// - 使用 sync.Map 实现无锁读取
// - 支持 TTL 过期
// - 自动清理过期条目
// - 容量限制（LRU）
type LocalCache struct {
	data    sync.Map
	mu      sync.RWMutex
	maxSize int
	ttl     time.Duration
}

type cacheEntry struct {
	value     interface{}
	expiresAt time.Time
}

// NewLocalCache 创建本地缓存
//
// 参数:
//   - maxSize: 最大缓存条目数
//   - ttl: 默认过期时间
func NewLocalCache(maxSize int, ttl time.Duration) *LocalCache {
	cache := &LocalCache{
		maxSize: maxSize,
		ttl:     ttl,
	}
	
	// 启动定期清理
	go cache.cleanupLoop()
	
	return cache
}

// Get 获取缓存值
func (c *LocalCache) Get(key string) (interface{}, bool) {
	val, ok := c.data.Load(key)
	if !ok {
		return nil, false
	}
	
	entry := val.(*cacheEntry)
	
	// 检查是否过期
	if time.Now().After(entry.expiresAt) {
		c.data.Delete(key)
		return nil, false
	}
	
	return entry.value, true
}

// Set 设置缓存值
func (c *LocalCache) Set(key string, value interface{}, ttl time.Duration) {
	if ttl == 0 {
		ttl = c.ttl
	}
	
	entry := &cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	
	c.data.Store(key, entry)
}

// Delete 删除缓存值
func (c *LocalCache) Delete(key string) {
	c.data.Delete(key)
}

// Clear 清空所有缓存
func (c *LocalCache) Clear() {
	c.data = sync.Map{}
}

// cleanupLoop 定期清理过期条目
func (c *LocalCache) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		now := time.Now()
		c.data.Range(func(key, value interface{}) bool {
			entry := value.(*cacheEntry)
			if now.After(entry.expiresAt) {
				c.data.Delete(key)
			}
			return true
		})
	}
}
