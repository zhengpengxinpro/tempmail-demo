package main

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	gosmtp "github.com/emersion/go-smtp"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"tempmail/backend/internal/auth"
	jwtpkg "tempmail/backend/internal/auth/jwt"
	"tempmail/backend/internal/config"
	"tempmail/backend/internal/domain"
	"tempmail/backend/internal/health"
	"tempmail/backend/internal/logger"
	"tempmail/backend/internal/monitoring"
	"tempmail/backend/internal/service"
	"tempmail/backend/internal/smtp"
	"tempmail/backend/internal/storage"
	"tempmail/backend/internal/storage/filesystem"
	"tempmail/backend/internal/storage/hybrid"
	"tempmail/backend/internal/storage/memory"
	httptransport "tempmail/backend/internal/transport/http"
	"tempmail/backend/internal/websocket"
)

// main 启动同时包含 HTTP API 与 SMTP 的综合服务。
func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	// 设置 Gin 模式（基于开发环境标志）
	if !cfg.Log.Development {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// 初始化日志系统
	log, err := logger.NewLogger(logger.Config{
		Level:       cfg.Log.Level,
		Development: cfg.Log.Development,
		LogFile:     "",
		MaxSize:     100,
		MaxBackups:  3,
		MaxAge:      28,
		Compress:    true,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	log.Info("starting tempmail server",
		zap.String("version", "0.8.2-beta"),
		zap.String("log_level", cfg.Log.Level),
		zap.Bool("development", cfg.Log.Development),
	)

	// 初始化存储层
	var store storage.Store

	// 根据配置选择存储类型
	if cfg.Database.Type != "" && cfg.Database.DSN != "" {
		// 使用数据库存储
		store, err = initializeDatabaseStorage(cfg, log)
		if err != nil {
			panic(fmt.Sprintf("failed to initialize database storage: %v", err))
		}
		log.Info("using database storage", zap.String("type", cfg.Database.Type))
	} else {
		// 使用内存存储（开发环境）
		store = memory.NewStore(cfg.Mailbox.DefaultTTL)
		log.Info("using memory storage (development mode)", zap.Duration("ttl", cfg.Mailbox.DefaultTTL))
	}

	// 初始化监控系统
	metrics := monitoring.NewMetrics()
	// 注意：promauto 已经自动注册了指标，不需要手动调用 RegisterCustomMetrics()

	// 初始化健康检查
	healthChecker := health.NewHealthChecker(store, log)

	// 初始化告警系统
	alertManager := monitoring.NewAlertManager(log)
	alertManager.AddReceiver(monitoring.NewLogAlertReceiver(log))
	alertManager.AddRule(monitoring.HighMemoryUsageRule(512.0)) // 512MB
	alertManager.AddRule(monitoring.DatabaseConnectionRule(store))

	log.Info("monitoring system initialized")

	// 初始化文件系统存储（用于邮件内容和附件）
	fsStorePath := cfg.Storage.Path
	if fsStorePath == "" {
		fsStorePath = "./data/mail-storage" // 默认路径
	}
	fsStore, err := filesystem.NewStore(fsStorePath)
	if err != nil {
		log.Warn("failed to initialize filesystem storage, continuing without it", zap.Error(err))
		fsStore = nil
	} else {
		log.Info("filesystem storage initialized", zap.String("path", fsStorePath))
	}

	// 初始化服务层
	mailboxService := service.NewMailboxService(store, store, cfg)
	messageService := service.NewMessageService(store)
	// 设置文件系统存储
	if fsStore != nil {
		messageService.SetFilesystemStore(fsStore)
	}
	aliasService := service.NewAliasService(store, store, cfg)
	searchService := service.NewSearchService(store)
	webhookService := service.NewWebhookService(store)
	tagService := service.NewTagService(store) // 初始化标签服务
	userDomainService := service.NewUserDomainService(store, cfg)
	systemDomainService := service.NewSystemDomainService(store, cfg) // 初始化系统域名服务
	apiKeyService := service.NewAPIKeyService(store)                  // 初始化API Key服务
	configService := service.NewConfigService(store)                  // 初始化系统配置服务

	// 设置邮箱服务和用户域名服务的关联（避免循环依赖）
	mailboxService.SetUserDomainService(userDomainService)

	// 初始化管理服务（需要转换配置）
	domainConfig := &domain.Config{
		AllowedDomains: cfg.Mailbox.AllowedDomains,
	}
	adminService := service.NewAdminService(store, domainConfig)

	// 初始化认证服务
	authService := auth.NewService(store)
	jwtManager := jwtpkg.NewManager(
		cfg.JWT.Secret,
		cfg.JWT.Issuer,
		cfg.JWT.AccessExpiry,
		cfg.JWT.RefreshExpiry,
	)

	log.Info("JWT configuration",
		zap.String("issuer", cfg.JWT.Issuer),
		zap.Duration("access_expiry", cfg.JWT.AccessExpiry),
		zap.Duration("refresh_expiry", cfg.JWT.RefreshExpiry),
	)

	// 初始化系统域名（从配置文件自动导入）
	initializeSystemDomains(systemDomainService, cfg, log)

	// 创建默认管理员用户（仅用于开发测试）
	if cfg.Log.Development {
		createDefaultAdmin(store, log)
	}

	// 创建 WebSocket Hub
	// 使用 CORS 配置的允许来源列表、JWT密钥和邮箱存储
	wsHub := websocket.NewHub(cfg.CORS.AllowedOrigins, cfg.JWT.Secret, store)

	// 创建 HTTP 服务器
	httpAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	router := httptransport.NewRouter(httptransport.RouterDependencies{
		Config:              cfg,
		MailboxService:      mailboxService,
		MessageService:      messageService,
		AliasService:        aliasService,
		SearchService:       searchService,  // 添加搜索服务
		WebhookService:      webhookService, // 添加 Webhook 服务
		TagService:          tagService,     // 添加标签服务
		AuthService:         authService,
		AdminService:        adminService,
		UserDomainService:   userDomainService,
		SystemDomainService: systemDomainService, // 添加系统域名服务
		APIKeyService:       apiKeyService,       // 添加API Key服务
		ConfigService:       configService,       // 添加系统配置服务
		JWTManager:          jwtManager,
		WebSocketHub:        wsHub,
		Store:               store,
		Logger:              log,
	})

	// 添加额外的健康检查和监控端点
	// 注意：/health 端点已在 router.go 中注册

	// 健康检查处理器（用于 Kubernetes 等）
	router.GET("/health/live", gin.WrapH(healthChecker.Handler()))
	router.GET("/health/ready", gin.WrapH(healthChecker.Handler()))

	// Prometheus 指标端点
	router.GET("/metrics", gin.WrapH(metrics.HTTPHandler()))

	httpServer := &http.Server{
		Addr:              httpAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// 创建 SMTP 服务器（支持动态域名配置）
	smtpBackend := smtp.NewBackend(mailboxService, messageService, aliasService, systemDomainService, userDomainService, wsHub, fsStore)
	smtpServer := gosmtp.NewServer(smtpBackend)
	smtpServer.Addr = cfg.SMTP.BindAddr
	smtpServer.Domain = cfg.SMTP.Domain
	smtpServer.AllowInsecureAuth = cfg.Log.Development // 仅在开发模式允许不安全认证
	smtpServer.ReadTimeout = 10 * time.Second
	smtpServer.WriteTimeout = 10 * time.Second
	smtpServer.MaxMessageBytes = 10 * 1024 * 1024 // 10MB
	smtpServer.MaxRecipients = 50

	// 信号处理
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	group, groupCtx := errgroup.WithContext(ctx)

	// HTTP 服务器 goroutine
	group.Go(func() error {
		log.Info("starting HTTP server", zap.String("address", httpAddr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server error", zap.Error(err))
			return err
		}
		return nil
	})

	// SMTP 服务器 goroutine
	group.Go(func() error {
		log.Info("starting SMTP server",
			zap.String("address", cfg.SMTP.BindAddr),
			zap.String("domain", cfg.SMTP.Domain),
		)
		if err := smtpServer.ListenAndServe(); err != nil {
			log.Error("SMTP server error", zap.Error(err))
			return err
		}
		return nil
	})

	// 定时清理过期邮箱 goroutine
	group.Go(func() error {
		ticker := time.NewTicker(1 * time.Hour) // 每小时执行一次
		defer ticker.Stop()

		log.Info("starting expired mailbox cleanup task", zap.Duration("interval", 1*time.Hour))

		for {
			select {
			case <-groupCtx.Done():
				log.Info("cleanup task stopped")
				return nil
			case <-ticker.C:
				count, err := store.DeleteExpiredMailboxes()
				if err != nil {
					log.Error("failed to cleanup expired mailboxes", zap.Error(err))
				} else if count > 0 {
					log.Info("expired mailboxes cleaned up", zap.Int("count", count))
				}
			}
		}
	})

	// 定时清理未验证的系统域名 goroutine
	group.Go(func() error {
		ticker := time.NewTicker(1 * time.Hour) // 每小时执行一次
		defer ticker.Stop()

		log.Info("starting unverified system domains cleanup task", zap.Duration("interval", 1*time.Hour))

		for {
			select {
			case <-groupCtx.Done():
				log.Info("unverified domains cleanup task stopped")
				return nil
			case <-ticker.C:
				count, err := systemDomainService.CleanupUnverifiedDomains()
				if err != nil {
					log.Error("failed to cleanup unverified system domains", zap.Error(err))
				} else if count > 0 {
					log.Info("unverified system domains cleaned up", zap.Int("count", count))
				}
			}
		}
	})

	// 定时重试失败的 Webhook 投递 goroutine
	group.Go(func() error {
		ticker := time.NewTicker(5 * time.Minute) // 每5分钟执行一次
		defer ticker.Stop()

		log.Info("starting webhook retry task", zap.Duration("interval", 5*time.Minute))

		for {
			select {
			case <-groupCtx.Done():
				log.Info("webhook retry task stopped")
				return nil
			case <-ticker.C:
				if err := webhookService.RetryFailedDeliveries(); err != nil {
					log.Error("failed to retry webhook deliveries", zap.Error(err))
				}
			}
		}
	})

	// WebSocket Hub goroutine
	group.Go(func() error {
		log.Info("starting WebSocket hub")
		wsHub.Run(groupCtx)
		return nil
	})

	// 监控服务 goroutine
	group.Go(func() error {
		log.Info("starting monitoring services")

		// 启动健康检查（暂时注释掉，因为方法不存在）
		// healthChecker.StartPeriodicHealthCheck(groupCtx, 30*time.Second)

		// 启动告警监控
		alertManager.StartMonitoring(groupCtx, 1*time.Minute)

		return nil
	})

	// 优雅关闭 goroutine
	group.Go(func() error {
		<-groupCtx.Done()
		log.Info("shutdown signal received, gracefully shutting down...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 关闭 HTTP 服务器
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Error("HTTP server shutdown error", zap.Error(err))
		}

		// 关闭 SMTP 服务器
		if err := smtpServer.Close(); err != nil {
			log.Warn("SMTP server close warning", zap.Error(err))
		}

		log.Info("servers stopped")
		return nil
	})

	// 等待所有 goroutine 完成
	if err := group.Wait(); err != nil && err != context.Canceled {
		log.Fatal("server error", zap.Error(err))
	}

	log.Info("server exited cleanly")
}

// initializeSystemDomains 初始化系统域名
//
// 从配置文件中读取允许的域名列表，自动添加到系统域名中
// 如果域名已存在，则跳过；如果不存在，则创建为已验证并激活状态
//
// 参数:
//   - systemDomainService: 系统域名服务
//   - cfg: 配置对象
//   - log: 日志记录器
func initializeSystemDomains(systemDomainService *service.SystemDomainService, cfg *config.Config, log *zap.Logger) {
	log.Info("initializing system domains from configuration",
		zap.Strings("domains", cfg.Mailbox.AllowedDomains),
	)

	// 获取现有系统域名
	existingDomains, err := systemDomainService.ListSystemDomains()
	if err != nil {
		log.Error("failed to list existing system domains", zap.Error(err))
		return
	}

	// 创建域名映射，快速检查是否已存在
	existingDomainMap := make(map[string]bool)
	for _, d := range existingDomains {
		existingDomainMap[d.Domain] = true
	}

	// 遍历配置中的域名，自动添加到系统中
	for _, domainName := range cfg.Mailbox.AllowedDomains {
		// 如果域名已存在，跳过
		if existingDomainMap[domainName] {
			log.Debug("system domain already exists, skipping",
				zap.String("domain", domainName),
			)
			continue
		}

		// 添加新的系统域名（自动设置为已验证并激活）
		log.Info("adding new system domain from configuration",
			zap.String("domain", domainName),
		)

		// 直接创建为已验证状态（配置文件中的域名默认信任）
		now := time.Now().UTC()
		sysDomain := &domain.SystemDomain{
			ID:           fmt.Sprintf("config-%s", domainName), // 使用特殊ID标识配置域名
			Domain:       domainName,
			Status:       domain.SystemDomainStatusVerified,
			VerifyToken:  "auto-configured",
			VerifyMethod: "configuration",
			VerifiedAt:   &now,
			LastCheckAt:  &now,
			CreatedAt:    now,
			CreatedBy:    "system",
			IsActive:     true,
			IsDefault:    false, // 可以在后台手动设置默认域名
			MailboxCount: 0,
			Notes:        "自动从配置文件导入",
		}

		// 保存到存储
		if err := systemDomainService.GetStore().SaveSystemDomain(sysDomain); err != nil {
			log.Error("failed to save system domain",
				zap.String("domain", domainName),
				zap.Error(err),
			)
			continue
		}

		log.Info("system domain added successfully",
			zap.String("domain", domainName),
		)
	}

	// 设置第一个域名为默认域名（如果还没有默认域名）
	hasDefault := false
	for _, d := range existingDomains {
		if d.IsDefault {
			hasDefault = true
			break
		}
	}

	if !hasDefault && len(cfg.Mailbox.AllowedDomains) > 0 {
		defaultDomain := cfg.Mailbox.AllowedDomains[0]

		// 查找该域名的ID
		updatedDomains, err := systemDomainService.ListSystemDomains()
		if err == nil {
			for _, d := range updatedDomains {
				if d.Domain == defaultDomain {
					if err := systemDomainService.SetDefaultDomain(d.ID); err != nil {
						log.Error("failed to set default domain",
							zap.String("domain", defaultDomain),
							zap.Error(err),
						)
					} else {
						log.Info("set default system domain",
							zap.String("domain", defaultDomain),
						)
					}
					break
				}
			}
		}
	}

	log.Info("system domains initialization completed")
}

// createDefaultAdmin 创建默认管理员用户（仅用于开发测试）
func createDefaultAdmin(store storage.Store, log *zap.Logger) {
	email := "admin@tempmail.local"
	password := "Admin123456!"
	username := "admin"

	// 检查管理员是否已存在
	if _, err := store.GetUserByEmail(email); err == nil {
		log.Info("默认管理员用户已存在，跳过创建", zap.String("email", email))
		return
	}

	// 哈希密码
	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		log.Error("无法哈希密码", zap.Error(err))
		return
	}

	// 创建超级管理员用户
	user := &domain.User{
		ID:              fmt.Sprintf("super-admin-001"),
		Email:           email,
		Username:        username,
		PasswordHash:    hashedPassword,
		Role:            domain.RoleSuper,
		Tier:            domain.TierEnterprise,
		IsActive:        true,
		IsEmailVerified: true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := store.CreateUser(user); err != nil {
		log.Error("创建默认管理员失败", zap.Error(err))
		return
	}

	log.Warn("✅ 默认管理员用户已创建（仅用于开发环境）",
		zap.String("email", email),
		zap.String("password", password),
		zap.String("role", string(domain.RoleSuper)),
	)
}

// initializeDatabaseStorage 初始化数据库存储
func initializeDatabaseStorage(cfg *config.Config, log *zap.Logger) (storage.Store, error) {
	log.Info("initializing database storage",
		zap.String("database_type", cfg.Database.Type),
		zap.String("redis_address", cfg.Redis.Address),
	)

	// 使用混合存储（SQL + Redis）
	store, err := hybrid.NewStoreWithType(
		cfg.Database.Type,
		cfg.Database.DSN,
		cfg.Redis.Address,
		cfg.Redis.Password,
		cfg.Redis.DB,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create hybrid store: %w", err)
	}

	log.Info("database storage initialized successfully",
		zap.String("database_type", cfg.Database.Type),
	)

	return store, nil
}
