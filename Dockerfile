# 构建阶段
FROM golang:1.25-alpine AS builder

WORKDIR /app

# 安装构建依赖
RUN apk add --no-cache git ca-certificates tzdata

# 复制 go mod 文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o /app/server ./cmd/server

# 运行阶段
FROM alpine:latest

WORKDIR /root/

# 安装运行时依赖
RUN apk --no-cache add ca-certificates tzdata wget

# 从构建阶段复制二进制文件
COPY --from=builder /app/server .

# 暴露端口
EXPOSE 8080 25

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --spider -q http://localhost:8080/health || exit 1

# 运行应用
CMD ["./server"]
