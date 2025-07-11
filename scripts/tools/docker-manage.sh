#!/bin/bash

# Docker统一管理脚本
# 支持多种使用模式和组件化管理

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

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

log_debug() {
    if [[ "${DEBUG:-false}" == "true" ]]; then
        echo -e "${PURPLE}[DEBUG]${NC} $1"
    fi
}

# Docker组件定义（配置化）
declare -A DOCKER_COMPONENTS=(
    [images]="Docker镜像管理"
    [compose]="Docker Compose环境管理"
    [registry]="Docker镜像仓库操作"
    [cleanup]="Docker清理操作"
)

# Docker组件操作映射
declare -A DOCKER_COMPONENT_ACTIONS=(
    [images]="build,tag,push,pull,list,remove"
    [compose]="up,down,restart,logs,status,clean"
    [registry]="login,logout,push,pull"
    [cleanup]="containers,images,volumes,networks,system"
)

# 环境验证
check_docker_environment() {
    log_info "验证Docker环境..."
    
    # 检查Docker是否安装
    if ! command -v docker &> /dev/null; then
        log_error "Docker未安装，请先安装Docker"
        return 1
    fi
    
    # 检查Docker服务是否运行
    if ! docker info &> /dev/null; then
        log_error "Docker服务未运行，请启动Docker服务"
        return 1
    fi
    
    # 检查docker-compose是否安装
    if ! command -v docker-compose &> /dev/null; then
        log_warning "docker-compose未安装，Compose功能将不可用"
    fi
    
    # 检查项目Dockerfile
    local dockerfiles=(
        "build/docker/swit-serve/Dockerfile"
        "build/docker/switauth/Dockerfile"
    )
    
    for dockerfile in "${dockerfiles[@]}"; do
        if [[ ! -f "$PROJECT_ROOT/$dockerfile" ]]; then
            log_warning "缺少Dockerfile: $dockerfile"
        fi
    done
    
    # 检查docker-compose配置
    if [[ ! -f "$PROJECT_ROOT/deployments/docker/docker-compose.yml" ]]; then
        log_warning "缺少docker-compose.yml配置文件"
    fi
    
    log_success "Docker环境验证完成"
}

# 标准Docker构建（推荐用于生产发布）
docker_build_standard() {
    log_info "执行标准Docker构建..."
    
    # 环境验证
    check_docker_environment || return 1
    
    # 切换到项目根目录
    cd "$PROJECT_ROOT"
    
    # 构建认证服务镜像
    log_info "构建swit-auth镜像..."
    docker build -t swit-auth:latest -f build/docker/switauth/Dockerfile .
    if [[ $? -eq 0 ]]; then
        log_success "swit-auth镜像构建完成"
    else
        log_error "swit-auth镜像构建失败"
        return 1
    fi
    
    # 构建主服务镜像
    log_info "构建swit-serve镜像..."
    docker build -t swit-serve:latest -f build/docker/swit-serve/Dockerfile .
    if [[ $? -eq 0 ]]; then
        log_success "swit-serve镜像构建完成"
    else
        log_error "swit-serve镜像构建失败"
        return 1
    fi
    
    # 镜像标记
    local git_tag=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "latest")
    docker tag swit-auth:latest swit-auth:$git_tag
    docker tag swit-serve:latest swit-serve:$git_tag
    
    log_success "标准Docker构建完成"
    log_info "镜像标签: latest, $git_tag"
}

# 快速Docker构建（开发时使用）
docker_build_quick() {
    log_info "执行快速Docker构建..."
    
    # 环境验证
    check_docker_environment || return 1
    
    # 切换到项目根目录
    cd "$PROJECT_ROOT"
    
    # 使用缓存快速构建
    log_info "快速构建swit-auth镜像（使用缓存）..."
    docker build --cache-from swit-auth:latest -t swit-auth:dev -f build/docker/switauth/Dockerfile .
    
    log_info "快速构建swit-serve镜像（使用缓存）..."
    docker build --cache-from swit-serve:latest -t swit-serve:dev -f build/docker/swit-serve/Dockerfile .
    
    log_success "快速Docker构建完成"
    log_info "镜像标签: dev"
}

# Docker开发环境设置
docker_setup_dev() {
    log_info "设置Docker开发环境..."
    
    # 环境验证
    check_docker_environment || return 1
    
    # 检查docker-compose
    if ! command -v docker-compose &> /dev/null; then
        log_error "docker-compose未安装，无法设置开发环境"
        return 1
    fi
    
    # 切换到Docker部署目录
    cd "$PROJECT_ROOT/deployments/docker"
    
    # 启动开发环境
    log_info "启动Docker Compose开发环境..."
    if [[ -f "start.sh" ]]; then
        chmod +x start.sh
        ./start.sh start
    else
        docker-compose up -d --build
    fi
    
    # 等待服务启动
    log_info "等待服务启动..."
    sleep 10
    
    # 检查服务状态
    log_info "检查服务状态..."
    docker-compose ps
    
    log_success "Docker开发环境设置完成"
    log_info "访问地址："
    log_info "  - Consul UI: http://localhost:8500"
    log_info "  - 认证服务: http://localhost:9001"
    log_info "  - 主要服务: http://localhost:9000"
    log_info "  - MySQL: localhost:3306 (root/root)"
}

# 高级Docker管理
docker_advanced() {
    local operation=${1:-help}
    local component=${2:-images}
    local service=${3:-all}
    
    log_info "执行高级Docker操作: $operation ($component)"
    
    case "$operation" in
        "build")
            case "$component" in
                "images")
                    if [[ "$service" == "all" ]]; then
                        docker_build_standard
                    else
                        docker_build_service "$service"
                    fi
                    ;;
                *)
                    log_error "不支持的组件: $component"
                    return 1
                    ;;
            esac
            ;;
        "start")
            case "$component" in
                "compose")
                    docker_compose_operation "up" "$service"
                    ;;
                *)
                    log_error "不支持的组件: $component"
                    return 1
                    ;;
            esac
            ;;
        "stop")
            case "$component" in
                "compose")
                    docker_compose_operation "down" "$service"
                    ;;
                *)
                    log_error "不支持的组件: $component"
                    return 1
                    ;;
            esac
            ;;
        "clean")
            docker_cleanup_operation "$component"
            ;;
        "help")
            show_advanced_help
            ;;
        *)
            log_error "不支持的操作: $operation"
            show_advanced_help
            return 1
            ;;
    esac
}

# 构建特定服务
docker_build_service() {
    local service="$1"
    
    cd "$PROJECT_ROOT"
    
    case "$service" in
        "auth"|"swit-auth")
            log_info "构建swit-auth镜像..."
            docker build -t swit-auth:latest -f build/docker/switauth/Dockerfile .
            ;;
        "serve"|"swit-serve")
            log_info "构建swit-serve镜像..."
            docker build -t swit-serve:latest -f build/docker/swit-serve/Dockerfile .
            ;;
        *)
            log_error "不支持的服务: $service"
            return 1
            ;;
    esac
}

# Docker Compose操作
docker_compose_operation() {
    local operation="$1"
    local service="$2"
    
    cd "$PROJECT_ROOT/deployments/docker"
    
    case "$operation" in
        "up")
            if [[ "$service" == "all" ]]; then
                docker-compose up -d --build
            else
                docker-compose up -d --build "$service"
            fi
            ;;
        "down")
            if [[ "$service" == "all" ]]; then
                docker-compose down
            else
                docker-compose stop "$service"
            fi
            ;;
        "restart")
            if [[ "$service" == "all" ]]; then
                docker-compose restart
            else
                docker-compose restart "$service"
            fi
            ;;
        "logs")
            if [[ "$service" == "all" ]]; then
                docker-compose logs -f
            else
                docker-compose logs -f "$service"
            fi
            ;;
        *)
            log_error "不支持的Compose操作: $operation"
            return 1
            ;;
    esac
}

# Docker清理操作
docker_cleanup_operation() {
    local component="$1"
    
    case "$component" in
        "containers")
            log_info "清理停止的容器..."
            docker container prune -f
            ;;
        "images")
            log_info "清理未使用的镜像..."
            docker image prune -f
            ;;
        "volumes")
            log_info "清理未使用的数据卷..."
            docker volume prune -f
            ;;
        "networks")
            log_info "清理未使用的网络..."
            docker network prune -f
            ;;
        "system")
            log_info "系统级清理..."
            docker system prune -f
            ;;
        "all")
            log_warning "这将删除所有未使用的Docker资源"
            read -p "确定要继续吗？(y/N): " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                docker system prune -a -f --volumes
                log_success "Docker系统清理完成"
            else
                log_info "已取消清理操作"
            fi
            ;;
        *)
            log_error "不支持的清理组件: $component"
            return 1
            ;;
    esac
}

# 显示高级操作帮助
show_advanced_help() {
    echo ""
    log_info "高级Docker操作帮助："
    echo ""
    echo "用法: docker-manage.sh advanced <操作> <组件> [服务]"
    echo ""
    echo "操作:"
    echo "  build    构建镜像"
    echo "  start    启动服务"
    echo "  stop     停止服务"
    echo "  clean    清理资源"
    echo ""
    echo "组件:"
    echo "  images   镜像管理"
    echo "  compose  Compose环境"
    echo ""
    echo "服务:"
    echo "  all      所有服务（默认）"
    echo "  auth     认证服务"
    echo "  serve    主要服务"
    echo ""
    echo "示例:"
    echo "  docker-manage.sh advanced build images auth"
    echo "  docker-manage.sh advanced start compose all"
    echo "  docker-manage.sh advanced clean containers"
}

# 显示使用帮助
show_help() {
    echo ""
    echo "Docker统一管理脚本"
    echo ""
    echo "用法: $0 <模式> [参数...]"
    echo ""
    echo "模式:"
    echo "  standard    标准构建 - 生产级镜像构建（推荐用于发布）"
    echo "  quick       快速构建 - 开发时快速构建（使用缓存）"
    echo "  setup       开发设置 - 启动完整的开发环境"
    echo "  advanced    高级管理 - 精确控制特定操作"
    echo ""
    echo "高级管理用法:"
    echo "  $0 advanced <操作> <组件> [服务]"
    echo ""
    echo "示例:"
    echo "  $0 standard                    # 标准镜像构建"
    echo "  $0 quick                       # 快速开发构建"
    echo "  $0 setup                       # 启动开发环境"
    echo "  $0 advanced build images all   # 构建所有镜像"
    echo "  $0 advanced start compose      # 启动Compose环境"
    echo "  $0 advanced clean system       # 系统清理"
    echo ""
}

# 主函数
main() {
    case "${1:-help}" in
        "standard")
            docker_build_standard
            ;;
        "quick")
            docker_build_quick
            ;;
        "setup")
            docker_setup_dev
            ;;
        "advanced")
            shift
            docker_advanced "$@"
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            log_error "未知模式: $1"
            show_help
            exit 1
            ;;
    esac
}

# 如果脚本被直接执行
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi 