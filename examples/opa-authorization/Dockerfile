FROM golang:1.23-alpine AS builder

WORKDIR /build

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/opa-example ./examples/opa-authorization

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/opa-example .

# 创建策略目录
RUN mkdir -p /app/policies

EXPOSE 8080 9090

ENTRYPOINT ["/app/opa-example"]

