#!/bin/bash

# 分布式追踪演示环境服务启动脚本
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
    echo -e "${PURPLE}[START]${NC} $1"
}

# 显示欢迎信息
show_welcome() {
    echo "======================================================================"
    echo -e "${BLUE}🚀 SWIT 框架分布式追踪演示服务启动脚本${NC}"
    echo "======================================================================"
    echo -e "${YELLOW}本脚本将启动分布式追踪演示环境的所有服务：${NC}"
    echo "  📊 Jaeger 分布式追踪后端"
    echo "  🛍️  订单服务 (Order Service)"
    echo "  💳 支付服务 (Payment Service)"
    echo "  📦 库存服务 (Inventory Service)"
    echo
}

# 显示使用说明
show_usage() {
    echo "Usage: $0 [options] [service...]"
    echo
    echo "Options:"
    echo "  -h, --help          显示帮助信息"
    echo "  -d, --detach        后台模式启动服务"
    echo "  -f, --force         强制重建并启动服务"
    echo "  -w, --wait          启动后等待服务就绪"
    echo "  --timeout=SECONDS   服务就绪等待超时时间 (默认: 120秒)"
    echo
    echo "Services (可选，默认启动所有服务):"
    echo "  jaeger              只启动 Jaeger"
    echo "  order-service       只启动订单服务"
    echo "  payment-service     只启动支付服务"
    echo "  inventory-service   只启动库存服务"
    echo
    echo "Examples:"
    echo "  $0                              # 启动所有服务"
    echo "  $0 -d                          # 后台模式启动所有服务"
    echo "  $0 jaeger order-service        # 只启动 Jaeger 和订单服务"
    echo "  $0 -f -w --timeout=60          # 强制重建，启动并等待服务就绪"
}

# 加载配置
load_config() {
    if [ -f "$CONFIG_FILE" ]; then
        log_info "正在加载配置文件: $CONFIG_FILE"
        source "$CONFIG_FILE"
    else
        log_warning "配置文件不存在: $CONFIG_FILE，使用默认配置"
        # 默认配置
        export JAEGER_UI_PORT=16686
        export JAEGER_COLLECTOR_PORT=14268
        export ORDER_SERVICE_PORT=8081
        export PAYMENT_SERVICE_PORT=8082
        export INVENTORY_SERVICE_PORT=8083
    fi
}

# 检查依赖
check_dependencies() {
    log_header "检查系统依赖"
    
    # 检查 Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装或不在 PATH 中"
        log_info "请访问 https://docs.docker.com/get-docker/ 安装 Docker"
        exit 1
    fi
    
    # 检查 Docker Compose
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        log_error "Docker Compose 未安装或不在 PATH 中"
        log_info "请访问 https://docs.docker.com/compose/install/ 安装 Docker Compose"
        exit 1
    fi
    
    # 检查 Docker 服务状态
    if ! docker info &> /dev/null; then
        log_error "Docker 服务未运行"
        log_info "请启动 Docker 服务后重试"
        exit 1
    fi
    
    log_success "依赖检查完成"
}

# 检查端口占用
check_ports() {
    log_header "检查端口占用情况"
    
    local ports=(
        "$JAEGER_UI_PORT:Jaeger UI"
        "$JAEGER_COLLECTOR_PORT:Jaeger Collector"
        "$ORDER_SERVICE_PORT:Order Service"
        "$PAYMENT_SERVICE_PORT:Payment Service" 
        "$INVENTORY_SERVICE_PORT:Inventory Service"
    )
    
    local port_conflicts=()
    
    for port_info in "${ports[@]}"; do
        local port="${port_info%:*}"
        local service="${port_info#*:}"
        
        if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
            port_conflicts+=("$port ($service)")
        fi
    done
    
    if [ ${#port_conflicts[@]} -gt 0 ]; then
        log_warning "发现端口占用："
        for conflict in "${port_conflicts[@]}"; do
            echo "  - 端口 $conflict"
        done
        
        if [ "$FORCE_MODE" != "true" ]; then
            read -p "是否继续启动？占用的端口可能导致服务启动失败 (y/N): " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                log_info "启动已取消"
                exit 0
            fi
        fi
    else
        log_success "端口检查完成，无冲突"
    fi
}

# 启动服务
start_services() {
    log_header "启动分布式追踪演示服务"
    
    cd "$PROJECT_ROOT"
    
    # 构建 docker-compose 命令
    local compose_cmd="docker-compose"
    if docker compose version &> /dev/null; then
        compose_cmd="docker compose"
    fi
    
    local compose_args=()
    
    # 添加服务选择
    if [ ${#SELECTED_SERVICES[@]} -gt 0 ]; then
        compose_args+=("${SELECTED_SERVICES[@]}")
        log_info "启动选定服务: ${SELECTED_SERVICES[*]}"
    else
        log_info "启动所有服务"
    fi
    
    # 构建启动命令
    local start_cmd="$compose_cmd up"
    
    if [ "$DETACH_MODE" = "true" ]; then
        start_cmd="$start_cmd -d"
        log_info "使用后台模式启动"
    fi
    
    if [ "$FORCE_MODE" = "true" ]; then
        start_cmd="$start_cmd --build --force-recreate"
        log_info "强制重建容器"
    fi
    
    # 执行启动命令
    log_info "执行命令: $start_cmd ${compose_args[*]}"
    $start_cmd "${compose_args[@]}"
    
    if [ $? -eq 0 ]; then
        log_success "服务启动完成"
    else
        log_error "服务启动失败"
        exit 1
    fi
}

# 等待服务就绪
wait_for_services() {
    if [ "$WAIT_MODE" != "true" ]; then
        return 0
    fi
    
    log_header "等待服务就绪"
    
    local timeout_seconds=${TIMEOUT:-120}
    local start_time=$(date +%s)
    local services_ready=false
    
    local health_endpoints=(
        "http://localhost:$JAEGER_UI_PORT:Jaeger UI"
        "http://localhost:$ORDER_SERVICE_PORT/health:Order Service"
        "http://localhost:$PAYMENT_SERVICE_PORT/health:Payment Service"
        "http://localhost:$INVENTORY_SERVICE_PORT/health:Inventory Service"
    )
    
    while [ $services_ready = false ]; do
        local current_time=$(date +%s)
        local elapsed=$((current_time - start_time))
        
        if [ $elapsed -ge $timeout_seconds ]; then
            log_error "等待服务就绪超时 (${timeout_seconds}秒)"
            return 1
        fi
        
        local all_ready=true
        
        for endpoint_info in "${health_endpoints[@]}"; do
            local endpoint="${endpoint_info%:*}"
            local service="${endpoint_info#*:}"
            
            # 如果只启动了特定服务，跳过未选择的服务
            if [ ${#SELECTED_SERVICES[@]} -gt 0 ]; then
                local service_selected=false
                for selected in "${SELECTED_SERVICES[@]}"; do
                    if [[ "$service" == *"$selected"* ]]; then
                        service_selected=true
                        break
                    fi
                done
                if [ "$service_selected" = false ]; then
                    continue
                fi
            fi
            
            if ! curl -f -s "$endpoint" > /dev/null 2>&1; then
                all_ready=false
                break
            fi
        done
        
        if [ $all_ready = true ]; then
            services_ready=true
            log_success "所有服务已就绪 (用时: ${elapsed}秒)"
        else
            printf "."
            sleep 2
        fi
    done
    
    echo # 换行
}

# 显示访问信息
show_access_info() {
    log_header "服务访问信息"
    
    echo -e "${GREEN}🎉 分布式追踪演示环境启动成功！${NC}"
    echo
    echo "服务访问地址："
    echo -e "  📊 ${BLUE}Jaeger UI:${NC}         http://localhost:$JAEGER_UI_PORT"
    echo -e "  🛍️  ${BLUE}订单服务:${NC}          http://localhost:$ORDER_SERVICE_PORT"
    echo -e "  💳 ${BLUE}支付服务:${NC}          http://localhost:$PAYMENT_SERVICE_PORT"
    echo -e "  📦 ${BLUE}库存服务:${NC}          http://localhost:$INVENTORY_SERVICE_PORT"
    echo
    echo "健康检查端点："
    echo -e "  🛍️  ${BLUE}订单服务健康检查:${NC}    http://localhost:$ORDER_SERVICE_PORT/health"
    echo -e "  💳 ${BLUE}支付服务健康检查:${NC}    http://localhost:$PAYMENT_SERVICE_PORT/health"
    echo -e "  📦 ${BLUE}库存服务健康检查:${NC}    http://localhost:$INVENTORY_SERVICE_PORT/health"
    echo
    echo "快速测试："
    echo -e "${YELLOW}# 创建测试订单${NC}"
    echo "curl -X POST http://localhost:$ORDER_SERVICE_PORT/api/v1/orders \\"
    echo "  -H \"Content-Type: application/json\" \\"
    echo "  -d '{\"customer_id\": \"test-customer\", \"product_id\": \"test-product\", \"quantity\": 1}'"
    echo
    echo -e "${YELLOW}# 查看追踪数据${NC}"
    echo "在浏览器中打开: http://localhost:$JAEGER_UI_PORT"
    echo
    echo "其他有用命令："
    echo -e "  ${BLUE}查看服务日志:${NC}        ./scripts/logs.sh"
    echo -e "  ${BLUE}健康检查:${NC}            ./scripts/health-check.sh"
    echo -e "  ${BLUE}运行演示场景:${NC}        ./scripts/demo-scenarios.sh"
    echo -e "  ${BLUE}停止服务:${NC}            ./scripts/stop.sh"
    echo -e "  ${BLUE}清理环境:${NC}            ./scripts/cleanup.sh"
}

# 主函数
main() {
    # 默认参数
    DETACH_MODE="false"
    FORCE_MODE="false"
    WAIT_MODE="false"
    TIMEOUT=120
    SELECTED_SERVICES=()
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_usage
                exit 0
                ;;
            -d|--detach)
                DETACH_MODE="true"
                shift
                ;;
            -f|--force)
                FORCE_MODE="true"
                shift
                ;;
            -w|--wait)
                WAIT_MODE="true"
                shift
                ;;
            --timeout=*)
                TIMEOUT="${1#*=}"
                shift
                ;;
            --timeout)
                TIMEOUT="$2"
                shift 2
                ;;
            jaeger|order-service|payment-service|inventory-service)
                SELECTED_SERVICES+=("$1")
                shift
                ;;
            *)
                log_error "未知选项: $1"
                show_usage
                exit 1
                ;;
        esac
    done
    
    # 如果选择了后台模式，默认启用等待模式
    if [ "$DETACH_MODE" = "true" ] && [ "$WAIT_MODE" != "true" ]; then
        WAIT_MODE="true"
    fi
    
    show_welcome
    load_config
    check_dependencies
    check_ports
    start_services
    wait_for_services
    
    if [ "$DETACH_MODE" = "true" ]; then
        show_access_info
    else
        echo
        log_info "按 Ctrl+C 停止所有服务"
        echo
        show_access_info
    fi
}

# 捕获退出信号
trap 'echo; log_info "正在停止服务..."; docker-compose -f "$PROJECT_ROOT/docker-compose.yml" down; exit 0' SIGINT SIGTERM

# 执行主函数
main "$@"