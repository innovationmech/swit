#!/bin/bash

# 分布式追踪演示环境搭建脚本
# Copyright (c) 2024 SWIT Framework Authors

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CONFIG_FILE="$PROJECT_ROOT/scripts/config/demo-config.env"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_header() {
    echo -e "${PURPLE}[SETUP]${NC} $1"
}

# 显示欢迎信息
show_welcome() {
    echo "======================================================================"
    echo -e "${BLUE}🚀 SWIT 框架分布式追踪演示环境搭建脚本${NC}"
    echo "======================================================================"
    echo -e "${YELLOW}本脚本将自动搭建完整的分布式追踪演示环境，包括：${NC}"
    echo "  📊 Jaeger 分布式追踪后端"
    echo "  🛍️  订单服务 (Order Service)"
    echo "  💳 支付服务 (Payment Service)"
    echo "  📦 库存服务 (Inventory Service)"
    echo "  🐳 Docker 容器化环境"
    echo ""
}

# 检查系统依赖
check_dependencies() {
    log_header "检查系统依赖..."
    
    local missing_deps=()
    
    # 检查 Docker
    if ! command -v docker &> /dev/null; then
        missing_deps+=("docker")
    else
        log_success "Docker 已安装: $(docker --version | head -n1)"
    fi
    
    # 检查 Docker Compose
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        missing_deps+=("docker-compose")
    else
        if command -v docker-compose &> /dev/null; then
            log_success "Docker Compose 已安装: $(docker-compose --version)"
        else
            log_success "Docker Compose 已安装: $(docker compose version)"
        fi
    fi
    
    # 检查 curl
    if ! command -v curl &> /dev/null; then
        missing_deps+=("curl")
    else
        log_success "curl 已安装"
    fi
    
    # 检查 jq
    if ! command -v jq &> /dev/null; then
        log_warning "jq 未安装，部分脚本功能可能受限"
    else
        log_success "jq 已安装"
    fi
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        log_error "缺少必要依赖: ${missing_deps[*]}"
        echo "请安装缺少的依赖后重试："
        echo ""
        echo "Ubuntu/Debian:"
        echo "  sudo apt-get update"
        echo "  sudo apt-get install docker.io docker-compose curl jq"
        echo ""
        echo "macOS:"
        echo "  brew install --cask docker"
        echo "  brew install jq"
        echo ""
        exit 1
    fi
}

# 检查端口可用性
check_ports() {
    log_header "检查端口可用性..."
    
    local required_ports=(16686 14268 14250 8081 8083 9081 9082 9083)
    local occupied_ports=()
    
    for port in "${required_ports[@]}"; do
        if lsof -i ":$port" >/dev/null 2>&1; then
            occupied_ports+=($port)
        fi
    done
    
    if [ ${#occupied_ports[@]} -ne 0 ]; then
        log_warning "以下端口已被占用: ${occupied_ports[*]}"
        echo "请关闭占用端口的进程或修改配置文件中的端口设置"
        echo "继续执行可能会导致服务启动失败"
        echo ""
        read -p "是否继续执行？(y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "用户取消执行"
            exit 0
        fi
    else
        log_success "所有必要端口都可用"
    fi
}

# 创建必要的目录
create_directories() {
    log_header "创建必要目录..."
    
    local dirs=(
        "$PROJECT_ROOT/data/logs"
        "$PROJECT_ROOT/data/jaeger"
        "$PROJECT_ROOT/scripts/config"
    )
    
    for dir in "${dirs[@]}"; do
        if [ ! -d "$dir" ]; then
            mkdir -p "$dir"
            log_success "创建目录: $dir"
        fi
    done
}

# 创建配置文件
create_config_files() {
    log_header "创建配置文件..."
    
    # 创建演示配置文件
    if [ ! -f "$CONFIG_FILE" ]; then
        cat > "$CONFIG_FILE" << 'EOF'
# SWIT 分布式追踪演示环境配置
# 服务端口配置
ORDER_SERVICE_PORT=8081
ORDER_SERVICE_GRPC_PORT=9081
PAYMENT_SERVICE_GRPC_PORT=9082
INVENTORY_SERVICE_PORT=8083
INVENTORY_SERVICE_GRPC_PORT=9083
JAEGER_UI_PORT=16686
JAEGER_COLLECTOR_PORT=14268

# 测试配置
LOAD_TEST_REQUESTS=100
LOAD_TEST_CONCURRENCY=10
DEMO_DELAY_SECONDS=2

# Jaeger 配置
JAEGER_ENDPOINT=http://localhost:14268/api/traces
JAEGER_SAMPLING_RATE=1.0

# 日志级别
LOG_LEVEL=info

# 追踪配置
TRACING_ENABLED=true
TRACING_SAMPLING_RATE=1.0
EOF
        log_success "创建配置文件: $CONFIG_FILE"
    fi
}

# 拉取 Docker 镜像
pull_docker_images() {
    log_header "拉取 Docker 镜像..."
    
    cd "$PROJECT_ROOT"
    
    local images=(
        "jaegertracing/all-in-one:1.42"
        "golang:1.26.1-alpine"
        "alpine:latest"
    )
    
    for image in "${images[@]}"; do
        log_info "拉取镜像: $image"
        docker pull "$image"
    done
    
    log_success "所有镜像拉取完成"
}

# 构建服务镜像
build_service_images() {
    log_header "构建服务镜像..."
    
    cd "$PROJECT_ROOT"
    
    # 使用 docker-compose 构建镜像
    if command -v docker-compose &> /dev/null; then
        docker-compose build --parallel
    else
        docker compose build --parallel
    fi
    
    log_success "服务镜像构建完成"
}

# 启动服务
start_services() {
    log_header "启动服务..."
    
    cd "$PROJECT_ROOT"
    
    # 首先启动 Jaeger
    log_info "启动 Jaeger 追踪后端..."
    if command -v docker-compose &> /dev/null; then
        docker-compose up -d jaeger
    else
        docker compose up -d jaeger
    fi
    
    # 等待 Jaeger 启动
    log_info "等待 Jaeger 启动完成..."
    local max_wait=60
    local wait_time=0
    
    while [ $wait_time -lt $max_wait ]; do
        if curl -s http://localhost:16686 > /dev/null; then
            break
        fi
        sleep 2
        wait_time=$((wait_time + 2))
        echo -n "."
    done
    echo ""
    
    if [ $wait_time -ge $max_wait ]; then
        log_error "Jaeger 启动超时"
        exit 1
    fi
    
    log_success "Jaeger 启动成功"
    
    # 启动所有服务
    log_info "启动所有微服务..."
    if command -v docker-compose &> /dev/null; then
        docker-compose up -d
    else
        docker compose up -d
    fi
    
    log_success "所有服务启动命令已执行"
}

# 等待服务就绪
wait_for_services() {
    log_header "等待服务就绪..."
    
    local services=(
        "http://localhost:8081/health|订单服务"
        "http://localhost:8083/health|库存服务"
        "http://localhost:16686|Jaeger UI"
    )
    
    local max_wait=120
    local all_ready=false
    
    for i in $(seq 1 $max_wait); do
        local ready_count=0
        
        for service_info in "${services[@]}"; do
            IFS='|' read -r url name <<< "$service_info"
            
            if curl -sf "$url" > /dev/null 2>&1; then
                ready_count=$((ready_count + 1))
            fi
        done
        
        if [ $ready_count -eq ${#services[@]} ]; then
            all_ready=true
            break
        fi
        
        if [ $((i % 10)) -eq 0 ]; then
            log_info "等待服务就绪... ($i/${max_wait}s)"
        fi
        
        sleep 1
    done
    
    if [ "$all_ready" = false ]; then
        log_warning "部分服务可能未完全启动，请稍后检查服务状态"
        return 1
    fi
    
    log_success "所有服务已就绪"
    return 0
}

# 验证部署
verify_deployment() {
    log_header "验证部署状态..."
    
    local failed_services=()
    
    # 检查 Jaeger UI
    if curl -sf http://localhost:16686 > /dev/null; then
        log_success "✅ Jaeger UI: http://localhost:16686"
    else
        log_error "❌ Jaeger UI 不可访问"
        failed_services+=("Jaeger")
    fi
    
    # 检查订单服务
    if curl -sf http://localhost:8081/health > /dev/null; then
        log_success "✅ 订单服务: http://localhost:8081"
    else
        log_error "❌ 订单服务不可访问"
        failed_services+=("Order Service")
    fi
    
    # 检查库存服务
    if curl -sf http://localhost:8083/health > /dev/null; then
        log_success "✅ 库存服务: http://localhost:8083"
    else
        log_error "❌ 库存服务不可访问"
        failed_services+=("Inventory Service")
    fi
    
    # 检查支付服务 (gRPC，使用 docker inspect)
    if docker ps --filter "name=payment-service" --filter "status=running" | grep -q payment-service; then
        log_success "✅ 支付服务: gRPC :9082"
    else
        log_error "❌ 支付服务未运行"
        failed_services+=("Payment Service")
    fi
    
    if [ ${#failed_services[@]} -eq 0 ]; then
        return 0
    else
        log_error "以下服务验证失败: ${failed_services[*]}"
        return 1
    fi
}

# 显示完成信息
show_completion() {
    echo ""
    echo "======================================================================"
    echo -e "${GREEN}🎉 分布式追踪演示环境搭建完成！${NC}"
    echo "======================================================================"
    echo ""
    echo -e "${YELLOW}📊 服务访问地址：${NC}"
    echo "  🔍 Jaeger UI:    http://localhost:16686"
    echo "  🛍️  订单服务:     http://localhost:8081"
    echo "  📦 库存服务:     http://localhost:8083"
    echo "  💳 支付服务:     gRPC :9082"
    echo ""
    echo -e "${YELLOW}🚀 快速测试：${NC}"
    echo "  # 创建订单（触发完整调用链）"
    echo "  curl -X POST http://localhost:8081/api/v1/orders \\"
    echo "    -H \"Content-Type: application/json\" \\"
    echo "    -d '{\"customer_id\": \"customer-123\", \"product_id\": \"product-456\", \"quantity\": 2, \"amount\": 99.99}'"
    echo ""
    echo -e "${YELLOW}📋 有用的命令：${NC}"
    echo "  ./scripts/health-check.sh    # 检查服务状态"
    echo "  ./scripts/demo-scenarios.sh  # 运行演示场景"
    echo "  ./scripts/logs.sh            # 查看服务日志"
    echo "  ./scripts/cleanup.sh         # 清理环境"
    echo ""
    echo -e "${BLUE}📚 更多信息请参考: docs/user-guide.md${NC}"
    echo ""
}

# 主函数
main() {
    show_welcome
    
    # 切换到项目根目录
    cd "$PROJECT_ROOT"
    
    # 执行设置步骤
    check_dependencies
    check_ports
    create_directories
    create_config_files
    pull_docker_images
    build_service_images
    start_services
    
    # 等待服务就绪
    if wait_for_services; then
        if verify_deployment; then
            show_completion
            exit 0
        else
            log_error "部署验证失败，请检查服务日志"
            echo "使用以下命令查看详细日志："
            echo "  docker-compose logs"
            exit 1
        fi
    else
        log_warning "服务启动可能需要更多时间，请稍后运行健康检查："
        echo "  ./scripts/health-check.sh"
        exit 0
    fi
}

# 信号处理
cleanup_on_exit() {
    log_info "接收到中断信号，清理并退出..."
    exit 1
}

trap cleanup_on_exit SIGINT SIGTERM

# 执行主函数
main "$@"
