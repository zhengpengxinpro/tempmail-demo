#!/bin/bash

# ========================================
# 临时邮箱系统 - 自动部署脚本
# ========================================
# 用途：在服务器上自动部署和更新应用
# 使用方法：./deploy.sh

set -e  # 遇到错误立即退出

# ========================================
# 配置变量
# ========================================
PROJECT_NAME="tempmail"
DEPLOY_PATH="/opt/tempmail"
GITHUB_REPO="https://github.com/zhengpengxinpro/tempmail-demo.git"  # 修改为你的仓库地址
BRANCH="main"  # 或 master

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# ========================================
# 日志函数
# ========================================
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# ========================================
# 检查依赖
# ========================================
check_dependencies() {
    log_info "检查依赖..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装，请先安装 Docker"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        log_error "Docker Compose 未安装，请先安装 Docker Compose"
        exit 1
    fi
    
    log_info "依赖检查通过"
}

# ========================================
# 创建部署目录
# ========================================
setup_directory() {
    log_info "设置部署目录: $DEPLOY_PATH"
    
    if [ ! -d "$DEPLOY_PATH" ]; then
        sudo mkdir -p "$DEPLOY_PATH"
        sudo chown -R $USER:$USER "$DEPLOY_PATH"
        log_info "部署目录已创建"
    else
        log_info "部署目录已存在"
    fi
}

# ========================================
# 拉取最新代码
# ========================================
pull_code() {
    log_info "拉取最新代码..."
    
    cd "$DEPLOY_PATH"
    
    if [ -d ".git" ]; then
        log_info "更新现有仓库..."
        git fetch origin
        git reset --hard origin/$BRANCH
        git pull origin $BRANCH
    else
        log_info "克隆新仓库..."
        git clone -b $BRANCH $GITHUB_REPO .
    fi
    
    log_info "代码更新完成"
}

# ========================================
# 检查环境变量文件
# ========================================
check_env_file() {
    log_info "检查环境变量文件..."
    
    cd "$DEPLOY_PATH/go"
    
    if [ ! -f ".env.production" ]; then
        log_error ".env.production 文件不存在！"
        log_info "请复制 .env.production.example 并配置："
        log_info "  cp .env.production.example .env.production"
        log_info "  vim .env.production"
        exit 1
    fi
    
    log_info "环境变量文件检查通过"
}

# ========================================
# 运行数据库迁移
# ========================================
run_migrations() {
    log_info "运行数据库迁移..."
    
    cd "$DEPLOY_PATH/go"
    
    # 等待数据库就绪
    log_info "等待数据库启动..."
    sleep 10
    
    # 使用 Docker 运行迁移
    docker-compose exec -T postgres psql -U tempmail -d tempmail -f /docker-entrypoint-initdb.d/001_initial_schema.up.sql || log_warn "迁移可能已执行"
    
    log_info "数据库迁移完成"
}

# ========================================
# 构建和启动服务
# ========================================
deploy_services() {
    log_info "构建和启动 Docker 服务..."
    
    cd "$DEPLOY_PATH/go"
    
    # 停止旧容器
    docker-compose down || true
    
    # 构建新镜像
    docker-compose build --no-cache
    
    # 启动服务
    docker-compose --env-file .env.production up -d
    
    log_info "服务启动完成"
}

# ========================================
# 健康检查
# ========================================
health_check() {
    log_info "执行健康检查..."
    
    MAX_RETRIES=30
    RETRY_COUNT=0
    
    while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
        if curl -f http://localhost:8080/health > /dev/null 2>&1; then
            log_info "✅ 应用健康检查通过！"
            return 0
        fi
        
        RETRY_COUNT=$((RETRY_COUNT + 1))
        log_warn "等待应用启动... ($RETRY_COUNT/$MAX_RETRIES)"
        sleep 2
    done
    
    log_error "❌ 应用健康检查失败！"
    log_info "查看日志："
    docker-compose logs --tail=50 app
    exit 1
}

# ========================================
# 清理旧镜像
# ========================================
cleanup() {
    log_info "清理旧的 Docker 镜像..."
    docker image prune -f
    log_info "清理完成"
}

# ========================================
# 显示状态
# ========================================
show_status() {
    log_info "========================================="
    log_info "服务状态："
    docker-compose ps
    log_info "========================================="
    log_info "应用访问地址："
    log_info "  HTTP API: http://YOUR_SERVER_IP:8080"
    log_info "  SMTP: YOUR_SERVER_IP:25"
    log_info "========================================="
    log_info "常用命令："
    log_info "  查看日志: docker-compose logs -f app"
    log_info "  重启服务: docker-compose restart"
    log_info "  停止服务: docker-compose down"
    log_info "========================================="
}

# ========================================
# 主流程
# ========================================
main() {
    log_info "========================================="
    log_info "开始部署 $PROJECT_NAME"
    log_info "========================================="
    
    check_dependencies
    setup_directory
    pull_code
    check_env_file
    deploy_services
    health_check
    cleanup
    show_status
    
    log_info "========================================="
    log_info "✅ 部署完成！"
    log_info "========================================="
}

# 执行主流程
main
