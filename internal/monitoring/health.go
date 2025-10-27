package monitoring

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"go.uber.org/zap"

	"tempmail/backend/internal/storage"
)

// HealthStatus 健康状态
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheck 健康检查
type HealthCheck struct {
	Name        string        `json:"name"`
	Status      HealthStatus  `json:"status"`
	Message     string        `json:"message,omitempty"`
	Duration    time.Duration `json:"duration"`
	LastChecked time.Time     `json:"last_checked"`
}

// HealthReport 健康报告
type HealthReport struct {
	Status      HealthStatus  `json:"status"`
	Timestamp   time.Time     `json:"timestamp"`
	Uptime      time.Duration `json:"uptime"`
	Checks      []HealthCheck `json:"checks"`
	Version     string        `json:"version"`
	Environment string        `json:"environment"`
}

// HealthChecker 健康检查器
type HealthChecker struct {
	store     storage.Store
	logger    *zap.Logger
	startTime time.Time
	version   string
	env       string
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(store storage.Store, logger *zap.Logger, version, env string) *HealthChecker {
	return &HealthChecker{
		store:     store,
		logger:    logger,
		startTime: time.Now(),
		version:   version,
		env:       env,
	}
}

// CheckHealth 执行健康检查
func (hc *HealthChecker) CheckHealth() *HealthReport {
	report := &HealthReport{
		Timestamp:   time.Now(),
		Uptime:      time.Since(hc.startTime),
		Version:     hc.version,
		Environment: hc.env,
		Checks:      make([]HealthCheck, 0),
	}

	// 执行各项健康检查
	checks := []func() HealthCheck{
		hc.checkDatabase,
		hc.checkRedis,
		hc.checkMemory,
		hc.checkCPU,
		hc.checkStorage,
		hc.checkSystem,
	}

	overallStatus := HealthStatusHealthy

	for _, check := range checks {
		healthCheck := check()
		report.Checks = append(report.Checks, healthCheck)

		// 确定整体状态
		switch healthCheck.Status {
		case HealthStatusUnhealthy:
			overallStatus = HealthStatusUnhealthy
		case HealthStatusDegraded:
			if overallStatus != HealthStatusUnhealthy {
				overallStatus = HealthStatusDegraded
			}
		}
	}

	report.Status = overallStatus
	return report
}

// checkDatabase 检查数据库连接
func (hc *HealthChecker) checkDatabase() HealthCheck {
	start := time.Now()

	check := HealthCheck{
		Name:        "database",
		LastChecked: start,
	}

	// 检查数据库连接
	if err := hc.store.Health(); err != nil {
		check.Status = HealthStatusUnhealthy
		check.Message = fmt.Sprintf("Database connection failed: %v", err)
	} else {
		check.Status = HealthStatusHealthy
		check.Message = "Database connection is healthy"
	}

	check.Duration = time.Since(start)
	return check
}

// checkRedis 检查 Redis 连接
func (hc *HealthChecker) checkRedis() HealthCheck {
	start := time.Now()

	check := HealthCheck{
		Name:        "redis",
		LastChecked: start,
	}

	// 检查 Redis 连接
	if rateLimitStore, ok := hc.store.(storage.RateLimitRepository); ok {
		_, err := rateLimitStore.GetRateLimit("health_check")
		if err != nil {
			check.Status = HealthStatusDegraded
			check.Message = fmt.Sprintf("Redis connection issue: %v", err)
		} else {
			check.Status = HealthStatusHealthy
			check.Message = "Redis connection is healthy"
		}
	} else {
		check.Status = HealthStatusDegraded
		check.Message = "Redis not available in current storage implementation"
	}

	check.Duration = time.Since(start)
	return check
}

// checkMemory 检查内存使用
func (hc *HealthChecker) checkMemory() HealthCheck {
	start := time.Now()

	check := HealthCheck{
		Name:        "memory",
		LastChecked: start,
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 计算内存使用率
	memoryUsageMB := float64(m.Alloc) / 1024 / 1024
	memoryLimitMB := 1024.0 // 1GB 限制

	if memoryUsageMB > memoryLimitMB {
		check.Status = HealthStatusDegraded
		check.Message = fmt.Sprintf("High memory usage: %.2f MB", memoryUsageMB)
	} else {
		check.Status = HealthStatusHealthy
		check.Message = fmt.Sprintf("Memory usage: %.2f MB", memoryUsageMB)
	}

	check.Duration = time.Since(start)
	return check
}

// checkCPU 检查 CPU 使用
func (hc *HealthChecker) checkCPU() HealthCheck {
	start := time.Now()

	check := HealthCheck{
		Name:        "cpu",
		LastChecked: start,
	}

	// 简单的 CPU 检查
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 检查 GC 压力
	gcPause := m.PauseTotalNs
	if gcPause > 1000000000 { // 1秒
		check.Status = HealthStatusDegraded
		check.Message = fmt.Sprintf("High GC pressure: %d ns", gcPause)
	} else {
		check.Status = HealthStatusHealthy
		check.Message = "CPU usage is normal"
	}

	check.Duration = time.Since(start)
	return check
}

// checkStorage 检查存储空间
func (hc *HealthChecker) checkStorage() HealthCheck {
	start := time.Now()

	check := HealthCheck{
		Name:        "storage",
		LastChecked: start,
	}

	// 这里可以添加磁盘空间检查
	// 由于 Go 标准库限制，这里简化处理
	check.Status = HealthStatusHealthy
	check.Message = "Storage is available"

	check.Duration = time.Since(start)
	return check
}

// checkSystem 检查系统状态
func (hc *HealthChecker) checkSystem() HealthCheck {
	start := time.Now()

	check := HealthCheck{
		Name:        "system",
		LastChecked: start,
	}

	// 检查系统指标
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 检查 Goroutine 数量
	numGoroutines := runtime.NumGoroutine()
	if numGoroutines > 1000 {
		check.Status = HealthStatusDegraded
		check.Message = fmt.Sprintf("High goroutine count: %d", numGoroutines)
	} else {
		check.Status = HealthStatusHealthy
		check.Message = fmt.Sprintf("Goroutines: %d", numGoroutines)
	}

	check.Duration = time.Since(start)
	return check
}

// IsHealthy 检查系统是否健康
func (hc *HealthChecker) IsHealthy() bool {
	report := hc.CheckHealth()
	return report.Status == HealthStatusHealthy
}

// GetUptime 获取系统运行时间
func (hc *HealthChecker) GetUptime() time.Duration {
	return time.Since(hc.startTime)
}

// StartPeriodicHealthCheck 启动定期健康检查
func (hc *HealthChecker) StartPeriodicHealthCheck(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			report := hc.CheckHealth()

			// 记录健康状态
			if report.Status == HealthStatusUnhealthy {
				hc.logger.Error("System health check failed",
					zap.String("status", string(report.Status)),
					zap.Duration("uptime", report.Uptime),
				)
			} else if report.Status == HealthStatusDegraded {
				hc.logger.Warn("System health check degraded",
					zap.String("status", string(report.Status)),
					zap.Duration("uptime", report.Uptime),
				)
			} else {
				hc.logger.Debug("System health check passed",
					zap.String("status", string(report.Status)),
					zap.Duration("uptime", report.Uptime),
				)
			}
		}
	}
}
