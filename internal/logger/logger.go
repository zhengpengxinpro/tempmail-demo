package logger

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Config 日志配置
type Config struct {
	Level       string `mapstructure:"level"`
	Development bool   `mapstructure:"development"`
	LogFile     string `mapstructure:"log_file"`
	MaxSize     int    `mapstructure:"max_size"` // MB
	MaxBackups  int    `mapstructure:"max_backups"`
	MaxAge      int    `mapstructure:"max_age"` // days
	Compress    bool   `mapstructure:"compress"`
}

// NewLogger 创建日志记录器
func NewLogger(cfg Config) (*zap.Logger, error) {
	// 设置日志级别
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}

	// 创建编码器配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 创建编码器
	var encoder zapcore.Encoder
	if cfg.Development {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// 创建写入器
	var writeSyncer zapcore.WriteSyncer

	if cfg.LogFile != "" {
		// 确保日志目录存在
		logDir := filepath.Dir(cfg.LogFile)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, err
		}

		// 配置日志轮转
		lumberjackLogger := &lumberjack.Logger{
			Filename:   cfg.LogFile,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		}

		// 同时输出到文件和控制台
		writeSyncer = zapcore.NewMultiWriteSyncer(
			zapcore.AddSync(lumberjackLogger),
			zapcore.AddSync(os.Stdout),
		)
	} else {
		writeSyncer = zapcore.AddSync(os.Stdout)
	}

	// 创建核心
	core := zapcore.NewCore(encoder, writeSyncer, level)

	// 创建日志记录器
	var logger *zap.Logger
	if cfg.Development {
		logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	} else {
		logger = zap.New(core, zap.AddCaller())
	}

	return logger, nil
}

// NewDevelopmentLogger 创建开发环境日志记录器
func NewDevelopmentLogger() *zap.Logger {
	config := Config{
		Level:       "debug",
		Development: true,
	}

	logger, err := NewLogger(config)
	if err != nil {
		// 如果创建失败，返回默认日志记录器
		return zap.NewNop()
	}

	return logger
}

// NewProductionLogger 创建生产环境日志记录器
func NewProductionLogger(logFile string) *zap.Logger {
	config := Config{
		Level:       "info",
		Development: false,
		LogFile:     logFile,
		MaxSize:     100, // 100MB
		MaxBackups:  3,
		MaxAge:      28, // 28 days
		Compress:    true,
	}

	logger, err := NewLogger(config)
	if err != nil {
		// 如果创建失败，返回默认日志记录器
		return zap.NewNop()
	}

	return logger
}
