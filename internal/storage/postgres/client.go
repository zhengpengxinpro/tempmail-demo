package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"tempmail/backend/internal/config"
)

// Client 封装 PostgreSQL 连接池
type Client struct {
	pool *pgxpool.Pool
	log  *zap.Logger
}

// New 创建新的 PostgreSQL 客户端
func New(cfg *config.DatabaseConfig) (*Client, error) {
	if cfg.DSN == "" {
		return nil, fmt.Errorf("database DSN is required")
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database DSN: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.MaxOpenConns)
	poolConfig.MinConns = int32(cfg.MaxIdleConns)
	poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// 测试连接
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log := zap.NewNop() // 临时使用空日志
	log.Info("connected to PostgreSQL",
		zap.Int("max_conns", cfg.MaxOpenConns),
		zap.Int("min_conns", cfg.MaxIdleConns),
	)

	return &Client{
		pool: pool,
		log:  log,
	}, nil
}

// Pool 返回底层的连接池
func (c *Client) Pool() *pgxpool.Pool {
	return c.pool
}

// Close 关闭数据库连接池
func (c *Client) Close() {
	c.pool.Close()
	c.log.Info("PostgreSQL connection closed")
}

// Ping 测试数据库连接
func (c *Client) Ping(ctx context.Context) error {
	return c.pool.Ping(ctx)
}

// Stats 返回连接池统计信息
func (c *Client) Stats() *pgxpool.Stat {
	return c.pool.Stat()
}
