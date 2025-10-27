# 数据库配置说明

## 快速开始

### 1. 内存存储（默认，适合开发）

不需要任何数据库配置，直接运行：

```bash
go run ./cmd/server
```

数据存储在内存中，重启后会丢失。

### 2. MySQL 5.7+ 存储（生产环境）

#### 创建数据库
```sql
CREATE DATABASE tempmail CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'tempmail'@'%' IDENTIFIED BY 'your_password';
GRANT ALL PRIVILEGES ON tempmail.* TO 'tempmail'@'%';
FLUSH PRIVILEGES;
```

#### 运行迁移
```bash
# 安装 MySQL driver
go get github.com/go-sql-driver/mysql@latest

# 运行迁移
go run cmd/migrate/main.go \
  -type=mysql \
  -dsn='tempmail:your_password@tcp(localhost:3306)/tempmail?parseTime=true&charset=utf8mb4' \
  -action=up
```

#### 配置 .env
```bash
TEMPMAIL_DATABASE_TYPE=mysql
TEMPMAIL_DATABASE_DSN=tempmail:your_password@tcp(localhost:3306)/tempmail?parseTime=true&charset=utf8mb4
```

### 3. PostgreSQL 存储（生产环境）

#### 创建数据库
```sql
CREATE DATABASE tempmail;
CREATE USER tempmail WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE tempmail TO tempmail;
```

#### 运行迁移
```bash
go run cmd/migrate/main.go \
  -type=postgres \
  -dsn='postgres://tempmail:your_password@localhost:5432/tempmail?sslmode=disable' \
  -action=up
```

#### 配置 .env
```bash
TEMPMAIL_DATABASE_TYPE=postgres
TEMPMAIL_DATABASE_DSN=postgres://tempmail:your_password@localhost:5432/tempmail?sslmode=disable
```

## 默认管理员账户

- 邮箱: `admin@tempmail.local`
- 密码: `admin123`
- **请立即修改密码！**

## 详细文档

查看完整文档：[DATABASE_SETUP.md](DATABASE_SETUP.md)