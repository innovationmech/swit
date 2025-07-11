#!/bin/bash

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

# 检查 Docker 和 docker-compose 是否安装
check_requirements() {
    log_info "检查系统要求..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装，请先安装 Docker"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        log_error "docker-compose 未安装，请先安装 docker-compose"
        exit 1
    fi
    
    log_success "系统要求检查通过"
}

# 设置构建变量
setup_build_vars() {
    log_info "设置构建变量..."
    
    # 如果 .env 文件不存在，从示例创建
    if [ ! -f ".env" ]; then
        if [ -f ".env.example" ]; then
            cp .env.example .env
            log_info "已从 .env.example 创建 .env 文件"
        fi
    fi
    
    # 设置构建时变量
    export VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
    export BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    export GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    
    log_info "构建变量已设置:"
    echo "  VERSION: ${VERSION}"
    echo "  BUILD_TIME: ${BUILD_TIME}"
    echo "  GIT_COMMIT: ${GIT_COMMIT}"
}

# 启动服务
start_services() {
    log_info "启动 Swit 开发环境..."
    
    # 构建并启动所有服务
    docker-compose up -d --build
    
    log_success "所有服务已启动"
    log_info "等待服务健康检查..."
    
    # 等待服务启动
    sleep 10
    
    # 检查服务状态
    check_services_health
}

# 检查服务健康状态
check_services_health() {
    log_info "检查服务健康状态..."
    
    services=("swit-mysql" "swit-consul" "swit-auth" "swit-serve")
    
    for service in "${services[@]}"; do
        if docker-compose ps "$service" | grep -q "Up"; then
            log_success "$service 运行正常"
        else
            log_warning "$service 可能存在问题，请检查日志"
        fi
    done
}

# 显示服务信息
show_services_info() {
    echo ""
    log_info "========== Swit 开发环境信息 =========="
    echo -e "${GREEN}服务访问地址：${NC}"
    echo "  📊 Consul UI:      http://localhost:8500"
    echo "  🔐 认证服务:       http://localhost:9001"
    echo "  🚀 主要服务:       http://localhost:9000"
    echo "  🗄️ MySQL 数据库:   localhost:3306 (root/root)"
    echo ""
    echo -e "${GREEN}健康检查端点：${NC}"
    echo "  认证服务: http://localhost:9001/health"
    echo "  主要服务: http://localhost:9000/health"
    echo ""
    echo -e "${GREEN}常用命令：${NC}"
    echo "  查看日志: docker-compose logs -f [服务名]"
    echo "  停止服务: docker-compose down"
    echo "  重启服务: docker-compose restart [服务名]"
    echo "  查看状态: docker-compose ps"
    echo "=========================================="
}

# 停止服务
stop_services() {
    log_info "停止 Swit 开发环境..."
    docker-compose down
    log_success "所有服务已停止"
}

# 重启服务
restart_services() {
    log_info "重启 Swit 开发环境..."
    docker-compose restart
    log_success "所有服务已重启"
}

# 查看日志
show_logs() {
    if [ -n "$1" ]; then
        log_info "显示 $1 服务的日志..."
        docker-compose logs -f "$1"
    else
        log_info "显示所有服务的日志..."
        docker-compose logs -f
    fi
}

# 清理环境
clean_environment() {
    log_warning "这将删除所有容器、网络和数据卷（包括数据库数据）"
    read -p "确定要继续吗？(y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log_info "清理环境..."
        docker-compose down -v --remove-orphans
        docker system prune -f
        log_success "环境清理完成"
    else
        log_info "已取消清理操作"
    fi
}

# 帮助信息
show_help() {
    echo "Swit 开发环境管理脚本"
    echo ""
    echo "用法: $0 [命令]"
    echo ""
    echo "命令:"
    echo "  start     启动所有服务 (默认)"
    echo "  stop      停止所有服务"
    echo "  restart   重启所有服务"
    echo "  status    查看服务状态"
    echo "  logs      查看所有服务日志"
    echo "  logs <服务名>  查看指定服务日志"
    echo "  clean     清理环境（删除所有数据）"
    echo "  help      显示此帮助信息"
    echo ""
    echo "服务名: mysql, consul, swit-auth, swit-serve"
}

# 查看状态
show_status() {
    log_info "服务状态："
    docker-compose ps
}

# 主函数
main() {
    cd "$(dirname "$0")"
    
    case "${1:-start}" in
        "start")
            check_requirements
            setup_build_vars
            start_services
            show_services_info
            ;;
        "stop")
            stop_services
            ;;
        "restart")
            restart_services
            ;;
        "status")
            show_status
            ;;
        "logs")
            show_logs "$2"
            ;;
        "clean")
            clean_environment
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            log_error "未知命令: $1"
            show_help
            exit 1
            ;;
    esac
}

main "$@" 