package smtp

import (
	"sync"
	"time"
)

// ConnectionLimiter SMTP 连接限流器
type ConnectionLimiter struct {
	maxConns    int
	current     int
	mu          sync.Mutex
	rateLimiter *RateLimiter
}

// NewConnectionLimiter 创建连接限流器
//
// 参数:
//   - maxConns: 最大并发连接数
//   - maxRate: 每秒最大新建连接数
func NewConnectionLimiter(maxConns, maxRate int) *ConnectionLimiter {
	return &ConnectionLimiter{
		maxConns:    maxConns,
		rateLimiter: NewRateLimiter(maxRate),
	}
}

// Acquire 获取连接许可
//
// 返回值:
//   - bool: 是否获取成功
func (l *ConnectionLimiter) Acquire() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// 检查连接数限制
	if l.current >= l.maxConns {
		return false
	}
	
	// 检查速率限制
	if !l.rateLimiter.Allow() {
		return false
	}
	
	l.current++
	return true
}

// Release 释放连接
func (l *ConnectionLimiter) Release() {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	if l.current > 0 {
		l.current--
	}
}

// Current 当前连接数
func (l *ConnectionLimiter) Current() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.current
}

// RateLimiter 速率限制器（令牌桶算法）
type RateLimiter struct {
	rate       int
	tokens     int
	maxTokens  int
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(rate int) *RateLimiter {
	return &RateLimiter{
		rate:       rate,
		tokens:     rate,
		maxTokens:  rate,
		lastRefill: time.Now(),
	}
}

// Allow 检查是否允许
func (r *RateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// 补充令牌
	now := time.Now()
	elapsed := now.Sub(r.lastRefill).Seconds()
	newTokens := int(elapsed * float64(r.rate))
	
	if newTokens > 0 {
		r.tokens = min(r.maxTokens, r.tokens+newTokens)
		r.lastRefill = now
	}
	
	// 消耗令牌
	if r.tokens > 0 {
		r.tokens--
		return true
	}
	
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
