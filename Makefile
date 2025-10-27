# 临时邮箱系统 - Makefile

.PHONY: help build clean test dev prod docker migrate

# 默认目标
help:
	@echo "临时邮箱系统 - 可用命令:"
	@echo ""
	@echo "  build     - 构建生产版本"
	@echo "  clean     - 清理构建文件"
	@echo "  test      - 运行测试"
	@echo "  dev       - 启动开发环境"
	@echo "  prod      - 启动生产环境"
	@echo "  docker    - 构建Docker镜像"
	@echo "  migrate   - 运行数据库迁移"
	@echo "  deps      - 安装依赖"
	@echo ""

# 构建
build:
	@echo "🔨 构建应用..."
	@go build -ldflags="-w -s" -o server ./cmd/server
	@go build -ldflags="-w -s" -o migrate ./cmd/migrate
	@echo "✅ 构建完成"

# 清理
clean:
	@echo "🧹 清理构建文件..."
	@rm -f server migrate api main
	@rm -f *.exe *.exe~
	@rm -f *.log
	@rm -f coverage.out coverage.html
	@rm -rf tmp/
	@echo "✅ 清理完成"

# 测试
test:
	@echo "🧪 运行测试..."
	@go test -v ./...
	@echo "✅ 测试完成"

# 基准测试
bench:
	@echo "⚡ 运行基准测试..."
	@go test -bench=. -benchmem ./internal/storage/memory
	@echo "✅ 基准测试完成"

# 覆盖率测试
coverage:
	@echo "📊 生成覆盖率报告..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ 覆盖率报告生成完成: coverage.html"

# 开发环境
dev:
	@echo "🛠️  启动开发环境..."
ifeq ($(OS),Windows_NT)
	@./dev.bat
else
	@./dev.sh
endif

# 生产环境
prod: build
	@echo "🚀 启动生产环境..."
ifeq ($(OS),Windows_NT)
	@./start.bat
else
	@./start.sh
endif

# Docker构建
docker:
	@echo "🐳 构建Docker镜像..."
	@docker build -t tempmail-backend .
	@echo "✅ Docker镜像构建完成"

# 生产Docker构建
docker-prod:
	@echo "🐳 构建生产Docker镜像..."
	@docker build -f Dockerfile.prod -t tempmail-backend:prod .
	@echo "✅ 生产Docker镜像构建完成"

# 数据库迁移
migrate:
	@echo "📊 运行数据库迁移..."
	@go run ./cmd/migrate up
	@echo "✅ 迁移完成"

# 安装依赖
deps:
	@echo "📦 安装依赖..."
	@go mod tidy
	@go mod download
	@echo "✅ 依赖安装完成"

# 代码格式化
fmt:
	@echo "🎨 格式化代码..."
	@go fmt ./...
	@echo "✅ 代码格式化完成"

# 代码检查
lint:
	@echo "🔍 代码检查..."
	@go vet ./...
	@echo "✅ 代码检查完成"

# 安全检查
security:
	@echo "🔒 安全检查..."
	@go list -json -m all | nancy sleuth
	@echo "✅ 安全检查完成"

# 完整检查
check: fmt lint test
	@echo "✅ 所有检查完成"

# 发布准备
release: clean deps check build
	@echo "🎉 发布准备完成"