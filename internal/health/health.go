package health

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/heptiolabs/healthcheck"
	"go.uber.org/zap"

	"tempmail/backend/internal/storage"
)

// HealthChecker 健康检查器
type HealthChecker struct {
	health healthcheck.Handler
	store  storage.Store
	logger *zap.Logger
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(store storage.Store, logger *zap.Logger) *HealthChecker {
	hc := &HealthChecker{
		health: healthcheck.NewHandler(),
		store:  store,
		logger: logger,
	}

	// 添加健康检查
	hc.addChecks()

	return hc
}

// addChecks 添加健康检查
func (hc *HealthChecker) addChecks() {
	// 数据库连接检查
	hc.health.AddLivenessCheck("database", func() error {
		return hc.store.Health()
	})

	// Redis 连接检查（如果支持）
	if rateLimitStore, ok := hc.store.(storage.RateLimitRepository); ok {
		hc.health.AddLivenessCheck("redis", func() error {
			_, err := rateLimitStore.GetRateLimit("health_check")
			return err
		})
	}

	// 系统资源检查
	hc.health.AddLivenessCheck("system", func() error {
		// 这里可以添加系统资源检查
		// 例如：内存使用率、磁盘空间等
		return nil
	})
}

// Handler 返回健康检查处理器
func (hc *HealthChecker) Handler() http.Handler {
	return hc.health
}

// CheckHealth 执行健康检查
func (hc *HealthChecker) CheckHealth() map[string]string {
	results := make(map[string]string)

	// 检查数据库
	if err := hc.store.Health(); err != nil {
		results["database"] = fmt.Sprintf("ERROR: %v", err)
	} else {
		results["database"] = "OK"
	}

	// 检查 Redis
	if rateLimitStore, ok := hc.store.(storage.RateLimitRepository); ok {
		if _, err := rateLimitStore.GetRateLimit("health_check"); err != nil {
			results["redis"] = fmt.Sprintf("ERROR: %v", err)
		} else {
			results["redis"] = "OK"
		}
	} else {
		results["redis"] = "NOT_AVAILABLE"
	}

	// 系统状态
	results["system"] = "OK"
	results["timestamp"] = time.Now().Format(time.RFC3339)

	return results
}

// DatabaseHealthCheck 数据库健康检查
func DatabaseHealthCheck(db *sql.DB) healthcheck.Check {
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		return db.PingContext(ctx)
	}
}

// RedisHealthCheck Redis 健康检查
func RedisHealthCheck(store storage.RateLimitRepository) healthcheck.Check {
	return func() error {
		_, err := store.GetRateLimit("health_check")
		return err
	}
}
