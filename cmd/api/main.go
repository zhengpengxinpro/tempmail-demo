package main

// @title TempMail Backend API
// @version 0.9.0
// @description TempMail 后端 API 文档
// @contact.name API Support
// @contact.email support@example.com
// @BasePath /
// @schemes http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description 使用格式：Bearer {token}
// @securityDefinitions.apikey MailboxToken
// @in header
// @name X-Mailbox-Token
// @description 邮箱访问令牌
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
// @description 兼容层 API Key

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"tempmail/backend/internal/auth"
	jwtpkg "tempmail/backend/internal/auth/jwt"
	"tempmail/backend/internal/config"
	"tempmail/backend/internal/domain"
	"tempmail/backend/internal/logger"
	"tempmail/backend/internal/service"
	"tempmail/backend/internal/storage/filesystem"
	"tempmail/backend/internal/storage/memory"
	httptransport "tempmail/backend/internal/transport/http"
	"tempmail/backend/internal/websocket"

	_ "tempmail/backend/docs" // Swagger docs
)

// main 是后端 HTTP 服务的程序入口（仅 HTTP API，不含 SMTP）。
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
	logCfg := logger.Config{
		Level:       cfg.Log.Level,
		Development: cfg.Log.Development,
	}
	log, err := logger.NewLogger(logCfg)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	defer log.Sync()
	log.Info("starting tempmail API server",
		zap.String("version", "0.8.2-beta"),
		zap.String("log_level", cfg.Log.Level),
		zap.Bool("development", cfg.Log.Development),
	)

	// 初始化存储层（使用内存存储用于开发）
	store := memory.NewStore(cfg.Mailbox.DefaultTTL)
	log.Info("using memory storage", zap.Duration("ttl", cfg.Mailbox.DefaultTTL))

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

	// 创建 WebSocket Hub
	// 使用 CORS 配置的允许来源列表、JWT密钥和邮箱存储
	wsHub := websocket.NewHub(cfg.CORS.AllowedOrigins, cfg.JWT.Secret, store)

	// 创建 HTTP 路由
	router := httptransport.NewRouter(httptransport.RouterDependencies{
		Config:              cfg,
		MailboxService:      mailboxService,
		MessageService:      messageService,
		AliasService:        aliasService,
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

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	server := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// 信号处理
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 启动 WebSocket Hub
	go func() {
		log.Info("starting WebSocket hub")
		wsHub.Run(ctx)
	}()

	// 启动 HTTP 服务器
	go func() {
		log.Info("API server listening", zap.String("address", addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	<-ctx.Done()
	log.Info("shutdown signal received, gracefully shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown error", zap.Error(err))
	} else {
		log.Info("server stopped cleanly")
	}
}
