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

# 检查 kubectl 是否可用
check_kubectl() {
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl 未安装，请先安装 kubectl"
        exit 1
    fi
    
    if ! kubectl cluster-info &> /dev/null; then
        log_error "无法连接到 Kubernetes 集群，请检查配置"
        exit 1
    fi
    
    log_success "kubectl 已就绪"
}

# 检查必要的镜像
check_images() {
    log_info "检查必要的容器镜像..."
    
    local missing_images=()
    
    if ! docker images | grep -q "swit-auth.*latest"; then
        missing_images+=("swit-auth:latest")
    fi
    
    if ! docker images | grep -q "swit-serve.*latest"; then
        missing_images+=("swit-serve:latest")
    fi
    
    if [ ${#missing_images[@]} -ne 0 ]; then
        log_warning "以下镜像不存在："
        for img in "${missing_images[@]}"; do
            echo "  - $img"
        done
        log_info "正在构建缺失的镜像..."
        build_images
    else
        log_success "所有必要的镜像都已存在"
    fi
}

# 构建镜像
build_images() {
    log_info "构建应用镜像..."
    
    # 切换到项目根目录
    cd "$(dirname "$0")/../.."
    
    # 构建认证服务镜像
    log_info "构建 swit-auth 镜像..."
    if docker build -f build/docker/switauth/Dockerfile -t swit-auth:latest .; then
        log_success "swit-auth 镜像构建完成"
    else
        log_error "swit-auth 镜像构建失败"
        exit 1
    fi
    
    # 构建主要服务镜像
    log_info "构建 swit-serve 镜像..."
    if docker build -f build/docker/swit-serve/Dockerfile -t swit-serve:latest .; then
        log_success "swit-serve 镜像构建完成"
    else
        log_error "swit-serve 镜像构建失败"
        exit 1
    fi
    
    # 返回 k8s 目录
    cd deployments/k8s
}

# 部署应用
deploy() {
    log_info "开始部署 Swit 到 Kubernetes..."
    
    # 部署顺序：命名空间 -> 存储 -> 配置 -> 基础设施 -> 应用
    local deploy_order=(
        "namespace.yaml"
        "storage.yaml"
        "secret.yaml"
        "configmap.yaml"
        "mysql.yaml"
        "consul.yaml"
        "swit-auth.yaml"
        "swit-serve.yaml"
        "ingress.yaml"
    )
    
    for file in "${deploy_order[@]}"; do
        if [ -f "$file" ]; then
            log_info "部署 $file..."
            if kubectl apply -f "$file"; then
                log_success "$file 部署完成"
            else
                log_error "$file 部署失败"
                return 1
            fi
        else
            log_warning "$file 文件不存在，跳过"
        fi
    done
    
    log_success "所有组件部署完成"
}

# 等待服务就绪
wait_for_services() {
    log_info "等待服务启动..."
    
    # 等待 MySQL 就绪
    log_info "等待 MySQL 就绪..."
    kubectl wait --for=condition=ready pod -l app=mysql -n swit --timeout=300s
    
    # 等待 Consul 就绪
    log_info "等待 Consul 就绪..."
    kubectl wait --for=condition=ready pod -l app=consul -n swit --timeout=300s
    
    # 等待认证服务就绪
    log_info "等待认证服务就绪..."
    kubectl wait --for=condition=ready pod -l app=swit-auth -n swit --timeout=300s
    
    # 等待主要服务就绪
    log_info "等待主要服务就绪..."
    kubectl wait --for=condition=ready pod -l app=swit-serve -n swit --timeout=300s
    
    log_success "所有服务已就绪"
}

# 显示部署信息
show_deployment_info() {
    echo ""
    log_info "========== Swit Kubernetes 部署信息 =========="
    
    echo -e "${GREEN}命名空间：${NC}"
    echo "  swit"
    echo ""
    
    echo -e "${GREEN}服务状态：${NC}"
    kubectl get pods -n swit -o wide
    echo ""
    
    echo -e "${GREEN}服务地址：${NC}"
    kubectl get services -n swit
    echo ""
    
    echo -e "${GREEN}外部访问地址：${NC}"
    local node_ip=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="ExternalIP")].address}')
    if [ -z "$node_ip" ]; then
        node_ip=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}')
    fi
    
    echo "  🚀 主要服务:       http://$node_ip:30900"
    echo "  🔐 认证服务:       http://$node_ip:30901"
    echo "  📊 Consul UI:      http://$node_ip:30850"
    echo ""
    
    echo -e "${GREEN}常用命令：${NC}"
    echo "  查看 Pod 状态:     kubectl get pods -n swit"
    echo "  查看服务状态:      kubectl get services -n swit"
    echo "  查看 Pod 日志:     kubectl logs -f <pod-name> -n swit"
    echo "  进入 Pod:         kubectl exec -it <pod-name> -n swit -- sh"
    echo "  删除部署:         ./deploy.sh delete"
    echo "=========================================="
}

# 删除部署
delete_deployment() {
    log_warning "这将删除整个 Swit Kubernetes 部署"
    read -p "确定要继续吗？(y/N): " -n 1 -r
    echo
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log_info "删除 Swit 部署..."
        
        # 删除应用服务
        kubectl delete -f swit-serve.yaml --ignore-not-found=true
        kubectl delete -f swit-auth.yaml --ignore-not-found=true
        kubectl delete -f ingress.yaml --ignore-not-found=true
        
        # 删除基础设施
        kubectl delete -f consul.yaml --ignore-not-found=true
        kubectl delete -f mysql.yaml --ignore-not-found=true
        
        # 删除配置
        kubectl delete -f configmap.yaml --ignore-not-found=true
        kubectl delete -f secret.yaml --ignore-not-found=true
        kubectl delete -f storage.yaml --ignore-not-found=true
        
        # 删除命名空间（这会删除所有剩余资源）
        kubectl delete namespace swit --ignore-not-found=true
        
        log_success "部署删除完成"
    else
        log_info "已取消删除操作"
    fi
}

# 查看状态
show_status() {
    log_info "Swit Kubernetes 部署状态："
    echo ""
    
    if kubectl get namespace swit &> /dev/null; then
        echo -e "${GREEN}命名空间状态：${NC}"
        kubectl get namespace swit
        echo ""
        
        echo -e "${GREEN}Pod 状态：${NC}"
        kubectl get pods -n swit -o wide
        echo ""
        
        echo -e "${GREEN}服务状态：${NC}"
        kubectl get services -n swit
        echo ""
        
        echo -e "${GREEN}存储状态：${NC}"
        kubectl get pvc -n swit
        echo ""
        
        echo -e "${GREEN}Ingress 状态：${NC}"
        kubectl get ingress -n swit
    else
        log_warning "Swit 部署不存在"
    fi
}

# 查看日志
show_logs() {
    local service=$1
    
    if [ -z "$service" ]; then
        log_info "可用的服务："
        echo "  mysql"
        echo "  consul"
        echo "  swit-auth"
        echo "  swit-serve"
        echo ""
        echo "用法: $0 logs <服务名>"
        return 1
    fi
    
    log_info "显示 $service 服务的日志..."
    kubectl logs -f -l app=$service -n swit --tail=100
}

# 重启服务
restart_service() {
    local service=$1
    
    if [ -z "$service" ]; then
        log_info "可用的服务："
        echo "  mysql"
        echo "  consul"
        echo "  swit-auth"
        echo "  swit-serve"
        echo ""
        echo "用法: $0 restart <服务名>"
        return 1
    fi
    
    log_info "重启 $service 服务..."
    kubectl rollout restart deployment/$service -n swit
    kubectl rollout status deployment/$service -n swit
    log_success "$service 服务重启完成"
}

# 帮助信息
show_help() {
    echo "Swit Kubernetes 部署管理脚本"
    echo ""
    echo "用法: $0 [命令]"
    echo ""
    echo "命令:"
    echo "  deploy    部署所有服务 (默认)"
    echo "  delete    删除所有部署"
    echo "  status    查看部署状态"
    echo "  logs      查看服务日志"
    echo "  restart   重启服务"
    echo "  build     构建应用镜像"
    echo "  help      显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0 deploy           # 部署整个应用"
    echo "  $0 logs swit-auth   # 查看认证服务日志"
    echo "  $0 restart mysql    # 重启 MySQL 服务"
    echo "  $0 status           # 查看部署状态"
}

# 主函数
main() {
    cd "$(dirname "$0")"
    
    case "${1:-deploy}" in
        "deploy")
            check_kubectl
            check_images
            deploy
            wait_for_services
            show_deployment_info
            ;;
        "delete")
            check_kubectl
            delete_deployment
            ;;
        "status")
            check_kubectl
            show_status
            ;;
        "logs")
            check_kubectl
            show_logs "$2"
            ;;
        "restart")
            check_kubectl
            restart_service "$2"
            ;;
        "build")
            build_images
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