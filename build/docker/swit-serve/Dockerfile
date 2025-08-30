FROM golang:1.23.12-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git curl

WORKDIR /app

COPY . .

# Install buf tool for proto generation
RUN go install github.com/bufbuild/buf/cmd/buf@latest

# Generate protobuf code
RUN cd api && buf dep update && buf generate

# 创建输出目录结构，与构建脚本保持一致
RUN mkdir -p _output/build/swit-serve/linux/amd64

# Download dependencies
RUN go mod tidy && go mod download

# 使用与构建脚本相同的构建参数
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags "-w -s -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.gitCommit=${GIT_COMMIT}" \
    -o _output/build/swit-serve/linux/amd64/swit-serve \
    ./cmd/swit-serve

FROM alpine:latest  

WORKDIR /root/

COPY --from=builder /app/_output/build/swit-serve/linux/amd64/swit-serve .

COPY ./swit.yaml /root/swit.yaml

EXPOSE 8080

LABEL org.opencontainers.image.source=https://github.com/innovationmech/swit

ENTRYPOINT ["./swit-serve"]

CMD ["serve"]
