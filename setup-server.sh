#!/bin/bash

# ========================================
# 临时邮箱系统 - 服务器初始化脚本
# ========================================
# 用途：一键安装 Docker、克隆代码、配置环境
# 使用方法：curl -fsSL <script-url> | bash

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 配置变量
DEPLOY_PATH="/opt/tempmail"
GITHUB_REPO="https://github.com/zhengpengxinpro/tempmail-demo.git"
BRANCH="main"

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# 显示欢迎信息
echo "========================================="
log_info "临时邮箱系统 - 服务器初始化"
log_info "这个脚本将会："
echo "  1. 检查并安装 Docker"
echo "  2. 检查并安装 Docker Compose"
echo "  3. 克隆代码仓库"
echo "  4. 创建环境变量配置文件"
echo "  5. 配置防火墙（可选）"
echo "========================================="
echo ""

# 检查是否为 root 用户
if [ "$EUID" -ne 0 ]; then 
    log_warn "建议使用 root 用户运行此脚本"
    log_info "如果遇到权限问题，请使用: sudo bash $0"
    echo ""
fi

# ========================================
# 步骤1：安装 Docker
# ========================================
log_step "步骤 1/5: 检查 Docker 安装"

if command -v docker &> /dev/null; then
    DOCKER_VERSION=$(docker --version)
    log_info "Docker 已安装: $DOCKER_VERSION"
else
    log_info "Docker 未安装，开始安装..."
    
    # 检测操作系统
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID
    else
        log_error "无法检测操作系统类型"
        exit 1
    fi
    
    case $OS in
        ubuntu|debian)
            log_info "检测到 Ubuntu/Debian 系统"
            apt-get update
            apt-get install -y ca-certificates curl gnupg lsb-release
            
            # 添加 Docker 官方 GPG key
            mkdir -p /etc/apt/keyrings
            curl -fsSL https://download.docker.com/linux/$OS/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
            
            # 添加 Docker 仓库
            echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/$OS $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
            
            # 安装 Docker
            apt-get update
            apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
            ;;
            
        centos|rhel|fedora)
            log_info "检测到 CentOS/RHEL/Fedora 系统"
            yum install -y yum-utils
            yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
            yum install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
            ;;
            
        *)
            log_error "不支持的操作系统: $OS"
            log_info "请手动安装 Docker: https://docs.docker.com/engine/install/"
            exit 1
            ;;
    esac
    
    # 启动 Docker 服务
    systemctl enable docker
    systemctl start docker
    
    # 验证安装
    if docker --version &> /dev/null; then
        log_info "✅ Docker 安装成功！"
    else
        log_error "❌ Docker 安装失败"
        exit 1
    fi
fi

# ========================================
# 步骤2：检查 Docker Compose
# ========================================
log_step "步骤 2/5: 检查 Docker Compose"

if docker compose version &> /dev/null; then
    COMPOSE_VERSION=$(docker compose version)
    log_info "Docker Compose 已安装: $COMPOSE_VERSION"
elif command -v docker-compose &> /dev/null; then
    COMPOSE_VERSION=$(docker-compose --version)
    log_info "Docker Compose 已安装（旧版）: $COMPOSE_VERSION"
else
    log_error "Docker Compose 未找到"
    log_info "尝试安装 Docker Compose Plugin..."
    
    if [ "$OS" = "ubuntu" ] || [ "$OS" = "debian" ]; then
        apt-get install -y docker-compose-plugin
    elif [ "$OS" = "centos" ] || [ "$OS" = "rhel" ]; then
        yum install -y docker-compose-plugin
    fi
    
    if docker compose version &> /dev/null; then
        log_info "✅ Docker Compose 安装成功！"
    else
        log_error "❌ Docker Compose 安装失败"
        exit 1
    fi
fi

# ========================================
# 步骤3：克隆代码仓库
# ========================================
log_step "步骤 3/5: 准备代码仓库"

# 检查 git 是否安装
if ! command -v git &> /dev/null; then
    log_info "Git 未安装，开始安装..."
    if [ "$OS" = "ubuntu" ] || [ "$OS" = "debian" ]; then
        apt-get install -y git
    elif [ "$OS" = "centos" ] || [ "$OS" = "rhel" ]; then
        yum install -y git
    fi
fi

# 创建部署目录
log_info "创建部署目录: $DEPLOY_PATH"
mkdir -p $DEPLOY_PATH

# 克隆或更新代码
if [ -d "$DEPLOY_PATH/.git" ]; then
    log_info "代码仓库已存在，更新代码..."
    cd $DEPLOY_PATH
    git fetch origin
    git reset --hard origin/$BRANCH
    git pull origin $BRANCH
else
    log_info "克隆代码仓库..."
    git clone -b $BRANCH $GITHUB_REPO $DEPLOY_PATH
    cd $DEPLOY_PATH
fi

log_info "✅ 代码仓库准备完成"

# ========================================
# 步骤4：创建环境变量配置
# ========================================
log_step "步骤 4/5: 配置环境变量"

if [ -f "$DEPLOY_PATH/.env.production" ]; then
    log_warn ".env.production 已存在，跳过创建"
else
    log_info "创建 .env.production 配置文件..."
    
    # 生成随机密钥
    JWT_SECRET=$(openssl rand -base64 32 | tr -d '\n')
    POSTGRES_PASSWORD=$(openssl rand -base64 16 | tr -d '\n')
    REDIS_PASSWORD=$(openssl rand -base64 16 | tr -d '\n')
    
    cat > $DEPLOY_PATH/.env.production << EOF
# ========================================
# 临时邮箱系统 - 生产环境配置
# ========================================
# 自动生成于: $(date)

# JWT 配置
JWT_SECRET=$JWT_SECRET
JWT_ISSUER=tempmail-production
JWT_ACCESS_EXPIRY=24h
JWT_REFRESH_EXPIRY=168h

# PostgreSQL 配置
POSTGRES_DB=tempmail
POSTGRES_USER=tempmail
POSTGRES_PASSWORD=$POSTGRES_PASSWORD

# 数据库连接池
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m

# Redis 配置
REDIS_PASSWORD=$REDIS_PASSWORD
REDIS_DB=0

# SMTP 配置（修改为你的域名）
SMTP_DOMAIN=temp.mail
ALLOWED_DOMAINS=temp.mail,tempmail.dev

# 邮箱策略
MAILBOX_DEFAULT_TTL=24h
MAILBOX_MAX_PER_IP=10

# 服务端口
HTTP_PORT=8080
SMTP_PORT=25

# CORS 配置（生产环境请设置具体域名）
CORS_ALLOWED_ORIGINS=*

# 日志配置
LOG_LEVEL=info
LOG_DEVELOPMENT=false
EOF

    log_info "✅ 环境变量配置文件已创建"
    log_warn "⚠️  重要：请编辑 $DEPLOY_PATH/.env.production"
    log_warn "   修改 SMTP_DOMAIN 和 ALLOWED_DOMAINS 为你的域名"
fi

# ========================================
# 步骤5：配置防火墙（可选）
# ========================================
log_step "步骤 5/5: 配置防火墙"

if command -v ufw &> /dev/null; then
    log_info "检测到 UFW 防火墙"
    read -p "是否配置防火墙开放 8080 和 25 端口？(y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        ufw allow 8080/tcp
        ufw allow 25/tcp
        log_info "✅ 防火墙规则已添加"
    else
        log_warn "跳过防火墙配置，请手动开放端口：8080 和 25"
    fi
elif command -v firewall-cmd &> /dev/null; then
    log_info "检测到 firewalld 防火墙"
    read -p "是否配置防火墙开放 8080 和 25 端口？(y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        firewall-cmd --permanent --add-port=8080/tcp
        firewall-cmd --permanent --add-port=25/tcp
        firewall-cmd --reload
        log_info "✅ 防火墙规则已添加"
    else
        log_warn "跳过防火墙配置，请手动开放端口：8080 和 25"
    fi
else
    log_warn "未检测到防火墙，请手动确保端口 8080 和 25 可访问"
fi

# ========================================
# 完成
# ========================================
echo ""
echo "========================================="
log_info "✅ 服务器初始化完成！"
echo "========================================="
echo ""
log_info "📋 接下来的步骤："
echo ""
echo "  1️⃣  编辑配置文件（修改域名）："
echo "     nano $DEPLOY_PATH/.env.production"
echo ""
echo "  2️⃣  启动服务："
echo "     cd $DEPLOY_PATH"
echo "     docker compose --env-file .env.production up -d"
echo ""
echo "  3️⃣  查看服务状态："
echo "     docker compose ps"
echo ""
echo "  4️⃣  查看日志："
echo "     docker compose logs -f app"
echo ""
echo "========================================="
log_info "🌐 服务地址："
echo "  HTTP API: http://YOUR_SERVER_IP:8080"
echo "  SMTP: YOUR_SERVER_IP:25"
echo "========================================="
echo ""
log_info "现在可以在 GitHub 触发 Actions 自动部署了！"
echo ""
