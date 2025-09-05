#!/bin/bash

# 分布式追踪演示环境服务停止脚本
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
    echo -e "${PURPLE}[STOP]${NC} $1"
}

# 显示欢迎信息
show_welcome() {
    echo "======================================================================"
    echo -e "${BLUE}🛑 SWIT 框架分布式追踪演示服务停止脚本${NC}"
    echo "======================================================================"
    echo -e "${YELLOW}本脚本将停止分布式追踪演示环境的服务：${NC}"
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
    echo "  -f, --force         强制停止服务 (相当于 docker-compose kill)"
    echo "  -r, --remove        停止后删除容器"
    echo "  -v, --volumes       同时删除相关数据卷"
    echo "  -n, --networks      同时删除相关网络"
    echo "  --timeout=SECONDS   服务停止超时时间 (默认: 30秒)"
    echo
    echo "Services (可选，默认停止所有服务):"
    echo "  jaeger              只停止 Jaeger"
    echo "  order-service       只停止订单服务"
    echo "  payment-service     只停止支付服务"
    echo "  inventory-service   只停止库存服务"
    echo
    echo "Examples:"
    echo "  $0                              # 优雅停止所有服务"
    echo "  $0 -f                          # 强制停止所有服务"
    echo "  $0 -r -v                       # 停止服务并删除容器和数据卷"
    echo "  $0 jaeger order-service        # 只停止 Jaeger 和订单服务"
    echo "  $0 --timeout=60 -f             # 60秒超时后强制停止"
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
        export ORDER_SERVICE_PORT=8081
        export PAYMENT_SERVICE_PORT=8082
        export INVENTORY_SERVICE_PORT=8083
    fi
}

# 检查 Docker 和 Docker Compose
check_docker() {
    log_header "检查 Docker 环境"
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装或不在 PATH 中"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        log_error "Docker Compose 未安装或不在 PATH 中"
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        log_error "Docker 服务未运行"
        exit 1
    fi
    
    log_success "Docker 环境检查完成"
}

# 检查运行中的服务
check_running_services() {
    log_header "检查运行中的服务"
    
    cd "$PROJECT_ROOT"
    
    local compose_cmd="docker-compose"
    if docker compose version &> /dev/null; then
        compose_cmd="docker compose"
    fi
    
    local running_services
    running_services=$($compose_cmd ps --services --filter "status=running" 2>/dev/null || echo "")
    
    if [ -z "$running_services" ]; then
        log_info "未发现运行中的分布式追踪服务"
        
        # 检查是否有相关容器在运行
        local related_containers
        related_containers=$(docker ps --format "table {{.Names}}" | grep -E "(jaeger|order-service|payment-service|inventory-service)" || echo "")
        
        if [ -n "$related_containers" ]; then
            log_warning "发现相关容器正在运行："
            echo "$related_containers"
            log_info "这些容器可能不是通过 docker-compose 启动的"
        else
            log_info "没有找到相关的运行中容器"
            exit 0
        fi
    else
        log_info "发现运行中的服务："
        echo "$running_services" | while IFS= read -r service; do
            echo "  - $service"
        done
    fi
}

# 停止服务
stop_services() {
    log_header "停止分布式追踪演示服务"
    
    cd "$PROJECT_ROOT"
    
    local compose_cmd="docker-compose"
    if docker compose version &> /dev/null; then
        compose_cmd="docker compose"
    fi
    
    # 构建停止命令
    local stop_cmd="$compose_cmd"
    
    if [ "$FORCE_MODE" = "true" ]; then
        stop_cmd="$stop_cmd kill"
        log_info "使用强制停止模式"
    else
        stop_cmd="$stop_cmd stop"
        if [ -n "$TIMEOUT" ]; then
            stop_cmd="$stop_cmd --timeout $TIMEOUT"
            log_info "使用 ${TIMEOUT} 秒超时时间"
        fi
        log_info "使用优雅停止模式"
    fi
    
    # 添加服务选择
    if [ ${#SELECTED_SERVICES[@]} -gt 0 ]; then
        stop_cmd="$stop_cmd ${SELECTED_SERVICES[*]}"
        log_info "停止选定服务: ${SELECTED_SERVICES[*]}"
    else
        log_info "停止所有服务"
    fi
    
    # 执行停止命令
    log_info "执行命令: $stop_cmd"
    if $stop_cmd; then
        log_success "服务停止完成"
    else
        log_error "服务停止失败"
        
        if [ "$FORCE_MODE" != "true" ]; then
            log_info "尝试使用强制停止模式..."
            if $compose_cmd kill "${SELECTED_SERVICES[@]}"; then
                log_success "服务强制停止完成"
            else
                log_error "服务强制停止也失败了"
                exit 1
            fi
        else
            exit 1
        fi
    fi
    
    # 删除容器
    if [ "$REMOVE_CONTAINERS" = "true" ]; then
        log_info "删除停止的容器..."
        local rm_cmd="$compose_cmd rm -f"
        if [ ${#SELECTED_SERVICES[@]} -gt 0 ]; then
            rm_cmd="$rm_cmd ${SELECTED_SERVICES[*]}"
        fi
        
        if $rm_cmd; then
            log_success "容器删除完成"
        else
            log_warning "容器删除失败，可能已经被删除"
        fi
    fi
}

# 清理资源
cleanup_resources() {
    if [ "$REMOVE_VOLUMES" = "true" ] || [ "$REMOVE_NETWORKS" = "true" ]; then
        log_header "清理相关资源"
        
        cd "$PROJECT_ROOT"
        
        local compose_cmd="docker-compose"
        if docker compose version &> /dev/null; then
            compose_cmd="docker compose"
        fi
        
        local down_cmd="$compose_cmd down"
        
        if [ "$REMOVE_VOLUMES" = "true" ]; then
            down_cmd="$down_cmd -v"
            log_info "将删除相关数据卷"
        fi
        
        if [ "$REMOVE_NETWORKS" = "true" ]; then
            down_cmd="$down_cmd --remove-orphans"
            log_info "将删除相关网络"
        fi
        
        if $down_cmd; then
            log_success "资源清理完成"
        else
            log_warning "资源清理部分失败"
        fi
    fi
}

# 显示停止后的状态
show_status() {
    log_header "服务停止状态"
    
    cd "$PROJECT_ROOT"
    
    local compose_cmd="docker-compose"
    if docker compose version &> /dev/null; then
        compose_cmd="docker compose"
    fi
    
    local running_services
    running_services=$($compose_cmd ps --services --filter "status=running" 2>/dev/null || echo "")
    
    if [ -z "$running_services" ]; then
        log_success "所有分布式追踪服务已停止"
    else
        log_warning "仍有服务在运行："
        echo "$running_services" | while IFS= read -r service; do
            echo "  - $service"
        done
    fi
    
    # 显示端口状态
    local ports=(
        "$JAEGER_UI_PORT:Jaeger UI"
        "$ORDER_SERVICE_PORT:Order Service"
        "$PAYMENT_SERVICE_PORT:Payment Service"
        "$INVENTORY_SERVICE_PORT:Inventory Service"
    )
    
    local ports_in_use=()
    for port_info in "${ports[@]}"; do
        local port="${port_info%:*}"
        local service="${port_info#*:}"
        
        if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
            ports_in_use+=("$port ($service)")
        fi
    done
    
    if [ ${#ports_in_use[@]} -gt 0 ]; then
        log_warning "以下端口仍被占用："
        for port_usage in "${ports_in_use[@]}"; do
            echo "  - 端口 $port_usage"
        done
        echo
        log_info "如果需要释放端口，请检查是否有其他程序在使用这些端口"
    else
        log_success "相关端口已释放"
    fi
    
    # 显示清理建议
    echo
    echo "停止后的操作建议："
    echo -e "  ${BLUE}查看容器状态:${NC}        docker-compose ps"
    echo -e "  ${BLUE}完全清理环境:${NC}        ./scripts/cleanup.sh"
    echo -e "  ${BLUE}重新启动服务:${NC}        ./scripts/start.sh"
    echo -e "  ${BLUE}查看服务日志:${NC}        ./scripts/logs.sh"
}

# 主函数
main() {
    # 默认参数
    FORCE_MODE="false"
    REMOVE_CONTAINERS="false"
    REMOVE_VOLUMES="false"
    REMOVE_NETWORKS="false"
    TIMEOUT=""
    SELECTED_SERVICES=()
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_usage
                exit 0
                ;;
            -f|--force)
                FORCE_MODE="true"
                shift
                ;;
            -r|--remove)
                REMOVE_CONTAINERS="true"
                shift
                ;;
            -v|--volumes)
                REMOVE_VOLUMES="true"
                shift
                ;;
            -n|--networks)
                REMOVE_NETWORKS="true"
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
    
    show_welcome
    load_config
    check_docker
    check_running_services
    stop_services
    cleanup_resources
    show_status
    
    echo
    log_success "分布式追踪演示服务停止完成！"
}

# 执行主函数
main "$@"