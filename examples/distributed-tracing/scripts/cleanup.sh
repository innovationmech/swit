#!/bin/bash

# 分布式追踪演示环境清理脚本
# Copyright (c) 2024 SWIT Framework Authors

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

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
    echo -e "${PURPLE}[CLEANUP]${NC} $1"
}

# 显示欢迎信息
show_welcome() {
    echo "======================================================================"
    echo -e "${BLUE}🧹 SWIT 框架分布式追踪演示环境清理脚本${NC}"
    echo "======================================================================"
    echo -e "${YELLOW}此脚本将清理以下资源：${NC}"
    echo "  🐳 Docker 容器 (jaeger, order-service, payment-service, inventory-service)"
    echo "  🌐 Docker 网络 (tracing-network)"
    echo "  💾 Docker 数据卷 (可选)"
    echo "  🖼️  Docker 镜像 (可选)"
    echo "  📁 临时数据文件"
    echo ""
}

# 确认清理操作
confirm_cleanup() {
    local cleanup_volumes=${1:-false}
    local cleanup_images=${2:-false}
    
    echo -e "${YELLOW}⚠️  警告：此操作将停止并删除所有相关的 Docker 资源！${NC}"
    
    if [ "$cleanup_volumes" = true ]; then
        echo -e "${RED}⚠️  包括数据卷 - 所有数据库数据将被永久删除！${NC}"
    fi
    
    if [ "$cleanup_images" = true ]; then
        echo -e "${RED}⚠️  包括镜像 - 下次启动需要重新构建！${NC}"
    fi
    
    echo ""
    read -p "确认继续清理操作？(y/N): " -n 1 -r
    echo
    
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "用户取消清理操作"
        exit 0
    fi
}

# 停止容器
stop_containers() {
    log_header "停止 Docker 容器..."
    
    cd "$PROJECT_ROOT"
    
    # 使用 docker-compose 停止服务
    if command -v docker-compose &> /dev/null; then
        if [ -f "docker-compose.yml" ]; then
            docker-compose stop
            log_success "使用 docker-compose 停止服务"
        fi
    elif docker compose version &> /dev/null; then
        if [ -f "docker-compose.yml" ]; then
            docker compose stop
            log_success "使用 docker compose 停止服务"
        fi
    fi
    
    # 手动停止可能遗留的容器
    local containers=(
        "jaeger"
        "order-service" 
        "payment-service"
        "inventory-service"
    )
    
    for container in "${containers[@]}"; do
        if docker ps -q -f name="$container" | grep -q .; then
            log_info "手动停止容器: $container"
            docker stop "$container" || log_warning "无法停止容器: $container"
        fi
    done
}

# 删除容器
remove_containers() {
    log_header "删除 Docker 容器..."
    
    cd "$PROJECT_ROOT"
    
    # 使用 docker-compose 删除容器
    if command -v docker-compose &> /dev/null; then
        if [ -f "docker-compose.yml" ]; then
            docker-compose rm -f
            log_success "使用 docker-compose 删除容器"
        fi
    elif docker compose version &> /dev/null; then
        if [ -f "docker-compose.yml" ]; then
            docker compose rm -f
            log_success "使用 docker compose 删除容器"
        fi
    fi
    
    # 删除可能遗留的容器
    local containers=(
        "jaeger"
        "order-service"
        "payment-service" 
        "inventory-service"
    )
    
    for container in "${containers[@]}"; do
        if docker ps -aq -f name="$container" | grep -q .; then
            log_info "手动删除容器: $container"
            docker rm -f "$container" 2>/dev/null || log_warning "无法删除容器: $container"
        fi
    done
    
    log_success "容器清理完成"
}

# 删除网络
remove_networks() {
    log_header "删除 Docker 网络..."
    
    local networks=(
        "distributed-tracing_tracing-network"
        "distributed-tracing_default"
    )
    
    for network in "${networks[@]}"; do
        if docker network inspect "$network" > /dev/null 2>&1; then
            log_info "删除网络: $network"
            docker network rm "$network" 2>/dev/null || log_warning "无法删除网络: $network"
        fi
    done
    
    log_success "网络清理完成"
}

# 删除数据卷
remove_volumes() {
    log_header "删除 Docker 数据卷..."
    
    local volumes=(
        "distributed-tracing_order-data"
        "distributed-tracing_inventory-data"
    )
    
    local removed_count=0
    
    for volume in "${volumes[@]}"; do
        if docker volume inspect "$volume" > /dev/null 2>&1; then
            log_info "删除数据卷: $volume"
            if docker volume rm "$volume" 2>/dev/null; then
                removed_count=$((removed_count + 1))
            else
                log_warning "无法删除数据卷: $volume (可能正在被使用)"
            fi
        fi
    done
    
    if [ $removed_count -gt 0 ]; then
        log_success "成功删除 $removed_count 个数据卷"
    else
        log_info "没有需要删除的数据卷"
    fi
}

# 删除镜像
remove_images() {
    log_header "删除 Docker 镜像..."
    
    local image_patterns=(
        "distributed-tracing_order-service"
        "distributed-tracing_payment-service" 
        "distributed-tracing_inventory-service"
        "distributed-tracing-order-service"
        "distributed-tracing-payment-service"
        "distributed-tracing-inventory-service"
    )
    
    local removed_count=0
    
    for pattern in "${image_patterns[@]}"; do
        local images
        images=$(docker images -q "$pattern" 2>/dev/null || echo "")
        
        if [ -n "$images" ]; then
            log_info "删除镜像: $pattern"
            if echo "$images" | xargs docker rmi -f 2>/dev/null; then
                removed_count=$((removed_count + 1))
            else
                log_warning "无法删除镜像: $pattern"
            fi
        fi
    done
    
    # 清理悬挂的镜像
    local dangling_images
    dangling_images=$(docker images -f "dangling=true" -q 2>/dev/null || echo "")
    
    if [ -n "$dangling_images" ]; then
        log_info "清理悬挂的镜像..."
        echo "$dangling_images" | xargs docker rmi 2>/dev/null || log_warning "部分悬挂镜像无法删除"
    fi
    
    if [ $removed_count -gt 0 ]; then
        log_success "镜像清理完成"
    else
        log_info "没有需要删除的项目相关镜像"
    fi
}

# 清理临时文件
cleanup_temp_files() {
    log_header "清理临时文件..."
    
    local temp_paths=(
        "$PROJECT_ROOT/data/logs"
        "$PROJECT_ROOT/.docker-volumes"
        "/tmp/swit-tracing-*"
    )
    
    local cleaned_count=0
    
    for path in "${temp_paths[@]}"; do
        if [ -d "$path" ] || [ -f "$path" ]; then
            log_info "清理: $path"
            rm -rf "$path" 2>/dev/null && cleaned_count=$((cleaned_count + 1)) || log_warning "无法清理: $path"
        fi
    done
    
    # 清理可能的日志文件
    if [ -d "$PROJECT_ROOT/data" ]; then
        find "$PROJECT_ROOT/data" -name "*.log" -type f -delete 2>/dev/null || true
        find "$PROJECT_ROOT/data" -name "*.tmp" -type f -delete 2>/dev/null || true
    fi
    
    log_success "临时文件清理完成"
}

# 检查残留资源
check_remaining_resources() {
    log_header "检查残留资源..."
    
    # 检查容器
    local remaining_containers
    remaining_containers=$(docker ps -aq --filter "name=jaeger" --filter "name=order-service" --filter "name=payment-service" --filter "name=inventory-service" 2>/dev/null | wc -l)
    
    if [ "$remaining_containers" -gt 0 ]; then
        log_warning "发现 $remaining_containers 个残留容器"
    else
        log_success "✅ 没有残留容器"
    fi
    
    # 检查网络
    local remaining_networks
    remaining_networks=$(docker network ls --filter "name=tracing" --format "{{.Name}}" 2>/dev/null | wc -l)
    
    if [ "$remaining_networks" -gt 0 ]; then
        log_warning "发现 $remaining_networks 个残留网络"
    else
        log_success "✅ 没有残留网络"
    fi
    
    # 检查数据卷
    local remaining_volumes
    remaining_volumes=$(docker volume ls --filter "name=distributed-tracing" --format "{{.Name}}" 2>/dev/null | wc -l)
    
    if [ "$remaining_volumes" -gt 0 ]; then
        log_info "保留 $remaining_volumes 个数据卷"
    else
        log_success "✅ 没有数据卷"
    fi
}

# 显示清理报告
show_cleanup_report() {
    local cleanup_volumes=$1
    local cleanup_images=$2
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    echo ""
    echo "======================================================================"
    echo -e "${GREEN}🎉 分布式追踪演示环境清理完成！${NC}"
    echo "======================================================================"
    echo "清理时间: $timestamp"
    echo ""
    
    echo -e "${YELLOW}✅ 已清理的资源：${NC}"
    echo "  🐳 Docker 容器 (已停止并删除)"
    echo "  🌐 Docker 网络 (已删除)"
    
    if [ "$cleanup_volumes" = true ]; then
        echo "  💾 Docker 数据卷 (已删除)"
    else
        echo "  💾 Docker 数据卷 (已保留)"
    fi
    
    if [ "$cleanup_images" = true ]; then
        echo "  🖼️  Docker 镜像 (已删除)"
    else
        echo "  🖼️  Docker 镜像 (已保留)"
    fi
    
    echo "  📁 临时文件 (已清理)"
    echo ""
    
    echo -e "${YELLOW}🚀 重新启动环境：${NC}"
    echo "  ./scripts/setup.sh"
    echo ""
    
    if [ "$cleanup_volumes" != true ]; then
        echo -e "${BLUE}💡 提示：${NC}"
        echo "  数据卷已保留，重启环境后数据将恢复"
        echo "  如需完全清理数据，请使用: $0 --volumes"
        echo ""
    fi
}

# 显示使用帮助
show_help() {
    echo "用法: $0 [选项]"
    echo ""
    echo "选项："
    echo "  --volumes         删除数据卷（将丢失所有数据）"
    echo "  --images          删除构建的镜像"  
    echo "  --all             删除所有资源（容器、网络、数据卷、镜像）"
    echo "  --force, -f       跳过确认提示"
    echo "  --help, -h        显示此帮助信息"
    echo ""
    echo "示例："
    echo "  $0                # 基本清理（保留数据卷和镜像）"
    echo "  $0 --volumes      # 清理包括数据卷"
    echo "  $0 --all          # 清理所有资源"
    echo "  $0 --force        # 强制清理（无确认提示）"
    echo ""
}

# 主函数
main() {
    local cleanup_volumes=false
    local cleanup_images=false
    local force_cleanup=false
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            --volumes)
                cleanup_volumes=true
                shift
                ;;
            --images)
                cleanup_images=true
                shift
                ;;
            --all)
                cleanup_volumes=true
                cleanup_images=true
                shift
                ;;
            --force|-f)
                force_cleanup=true
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                log_error "未知选项: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    show_welcome
    
    # 确认清理操作
    if [ "$force_cleanup" != true ]; then
        confirm_cleanup "$cleanup_volumes" "$cleanup_images"
    fi
    
    # 切换到项目根目录
    cd "$PROJECT_ROOT"
    
    # 执行清理步骤
    stop_containers
    remove_containers
    remove_networks
    
    if [ "$cleanup_volumes" = true ]; then
        remove_volumes
    fi
    
    if [ "$cleanup_images" = true ]; then
        remove_images
    fi
    
    cleanup_temp_files
    check_remaining_resources
    show_cleanup_report "$cleanup_volumes" "$cleanup_images"
}

# 信号处理
cleanup_on_exit() {
    log_info "接收到中断信号，清理并退出..."
    exit 1
}

trap cleanup_on_exit SIGINT SIGTERM

# 执行主函数
main "$@"