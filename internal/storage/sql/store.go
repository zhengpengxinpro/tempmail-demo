package sql

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
	_ "github.com/lib/pq"              // PostgreSQL driver
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"tempmail/backend/internal/domain"
)

// Store SQL 数据库存储实现（支持 MySQL 5.7+ 和 PostgreSQL）
type Store struct {
	db         *sql.DB
	gormDB     *gorm.DB // GORM实例，用于迁移
	driverName string   // "mysql" or "postgres"
}

// NewStore 创建SQL数据库存储
func NewStore(
	driverName string,
	dsn string,
	maxOpenConns int,
	maxIdleConns int,
	connMaxLifetime time.Duration,
) (*Store, error) {
	// 验证驱动类型
	if driverName != "mysql" && driverName != "postgres" {
		return nil, fmt.Errorf("unsupported database driver: %s (supported: mysql, postgres)", driverName)
	}

	// 打开数据库连接
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)

	// 测试连接
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// 初始化GORM（用于自动迁移）
	var gormDB *gorm.DB
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	if driverName == "mysql" {
		gormDB, err = gorm.Open(mysql.New(mysql.Config{
			Conn: db,
		}), gormConfig)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to initialize GORM: %w", err)
		}
	}

	store := &Store{
		db:         db,
		gormDB:     gormDB,
		driverName: driverName,
	}

	// 自动执行数据库迁移
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return store, nil
}

// Close 关闭数据库连接
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Health 检查数据库健康状态
func (s *Store) Health() error {
	if s.db == nil {
		return fmt.Errorf("database connection is nil")
	}
	return s.db.Ping()
}

// migrate 执行数据库迁移（使用GORM AutoMigrate）
func (s *Store) migrate() error {
	if s.gormDB == nil {
		return nil
	}

	return s.gormDB.AutoMigrate(
		&domain.User{},
		&domain.Mailbox{},
		&domain.Message{},
		&domain.Attachment{},
		&domain.MailboxAlias{},
		&domain.UserDomain{},
		&domain.SystemDomain{},
		&domain.APIKey{},
		&domain.Webhook{},
		&domain.WebhookDelivery{},
		&domain.Tag{},
		&domain.MessageTag{},
		&domain.SystemConfig{},
	)
}

// placeholder 根据数据库类型返回占位符
func (s *Store) placeholder(n int) string {
	if s.driverName == "postgres" {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}
