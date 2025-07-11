FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY . .

# 创建输出目录结构，与构建脚本保持一致
RUN mkdir -p _output/build/switctl/linux/amd64

# 使用与构建脚本相同的构建参数
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags "-w -s -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.gitCommit=${GIT_COMMIT}" \
    -o _output/build/switctl/linux/amd64/switctl \
    ./cmd/switctl/switctl.go

FROM alpine:latest  

WORKDIR /root/

COPY --from=builder /app/_output/build/switctl/linux/amd64/switctl .

LABEL org.opencontainers.image.source=https://github.com/innovationmech/swit

ENTRYPOINT ["./switctl"]

CMD ["--help"]
