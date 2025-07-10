#!/bin/bash

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 默认配置
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
CHART_PATH="${SCRIPT_DIR}"
RELEASE_NAME="swit"
NAMESPACE="swit"
VALUES_FILE="${CHART_PATH}/values.yaml"
TIMEOUT="600s"

# 显示帮助信息
show_help() {
    echo -e "${CYAN}Swit 微服务平台 Helm 部署脚本${NC}"
    echo ""
    echo "用法: $0 [选项] <命令>"
    echo ""
    echo "命令:"
    echo "  install      - 安装 Swit 平台"
    echo "  upgrade      - 升级 Swit 平台"
    echo "  uninstall    - 卸载 Swit 平台"
    echo "  status       - 查看部署状态"
    echo "  logs         - 查看服务日志"
    echo "  restart      - 重启服务"
    echo "  build        - 构建 Docker 镜像"
    echo "  lint         - 检查 Chart 语法"
    echo "  template     - 生成 Kubernetes 清单"
    echo ""
    echo "选项:"
    echo "  -n, --namespace NAME    指定命名空间 (默认: swit)"
    echo "  -r, --release NAME      指定发布名称 (默认: swit)"
    echo "  -f, --values FILE       指定 values 文件路径"
    echo "  -t, --timeout DURATION  指定超时时间 (默认: 600s)"
    echo "  --dry-run              仅输出将要执行的操作"
    echo "  --debug                启用调试模式"
    echo "  -h, --help             显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0 install                    # 使用默认配置安装"
    echo "  $0 -n production install     # 安装到 production 命名空间"
    echo "  $0 -f custom.yaml upgrade     # 使用自定义配置升级"
    echo "  $0 logs swit-auth             # 查看认证服务日志"
    echo "  $0 restart swit-serve         # 重启主要服务"
}

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
    if [[ "${DEBUG}" == "true" ]]; then
        echo -e "${PURPLE}[DEBUG]${NC} $1"
    fi
}

# 检查命令是否存在
check_command() {
    if ! command -v $1 &> /dev/null; then
        log_error "$1 未安装，请先安装 $1"
        exit 1
    fi
}

# 检查必要的工具
check_prerequisites() {
    log_info "检查必要工具..."
    check_command "helm"
    check_command "kubectl"
    check_command "docker"
    log_success "所有必要工具已安装"
}

# 创建命名空间
create_namespace() {
    log_info "检查命名空间 ${NAMESPACE}..."
    if ! kubectl get namespace "${NAMESPACE}" &> /dev/null; then
        log_info "创建命名空间 ${NAMESPACE}..."
        kubectl create namespace "${NAMESPACE}"
        log_success "命名空间 ${NAMESPACE} 已创建"
    else
        log_info "命名空间 ${NAMESPACE} 已存在"
    fi
}

# 构建 Docker 镜像
build_images() {
    log_info "构建 Docker 镜像..."
    
    cd "${PROJECT_ROOT}"
    
    # 构建认证服务镜像
    log_info "构建 swit-auth 镜像..."
    if docker build -t swit-auth:latest -f build/docker/switauth/Dockerfile .; then
        log_success "swit-auth 镜像构建成功"
    else
        log_error "swit-auth 镜像构建失败"
        return 1
    fi
    
    # 构建主要服务镜像
    log_info "构建 swit-serve 镜像..."
    if docker build -t swit-serve:latest -f build/docker/swit-serve/Dockerfile .; then
        log_success "swit-serve 镜像构建成功"
    else
        log_error "swit-serve 镜像构建失败"
        return 1
    fi
    
    log_success "所有镜像构建完成"
}

# 检查 Chart 语法
lint_chart() {
    log_info "检查 Helm Chart 语法..."
    cd "${CHART_PATH}"
    if helm lint .; then
        log_success "Chart 语法检查通过"
    else
        log_error "Chart 语法检查失败"
        exit 1
    fi
}

# 生成 Kubernetes 清单
template_chart() {
    log_info "生成 Kubernetes 清单..."
    cd "${CHART_PATH}"
    
    OUTPUT_DIR="${PROJECT_ROOT}/_output/helm"
    mkdir -p "${OUTPUT_DIR}"
    
    helm template "${RELEASE_NAME}" . \
        --namespace "${NAMESPACE}" \
        --values "${VALUES_FILE}" \
        --output-dir "${OUTPUT_DIR}"
    
    log_success "Kubernetes 清单已生成到 ${OUTPUT_DIR}"
}

# 安装 Swit 平台
install_swit() {
    log_info "安装 Swit 微服务平台..."
    
    create_namespace
    
    cd "${CHART_PATH}"
    
    helm install "${RELEASE_NAME}" . \
        --namespace "${NAMESPACE}" \
        --values "${VALUES_FILE}" \
        --timeout "${TIMEOUT}" \
        --wait \
        ${DRY_RUN:+--dry-run} \
        ${DEBUG:+--debug}
    
    if [[ "${DRY_RUN}" != "true" ]]; then
        log_success "Swit 平台安装成功"
        show_deployment_info
    fi
}

# 升级 Swit 平台
upgrade_swit() {
    log_info "升级 Swit 微服务平台..."
    
    cd "${CHART_PATH}"
    
    helm upgrade "${RELEASE_NAME}" . \
        --namespace "${NAMESPACE}" \
        --values "${VALUES_FILE}" \
        --timeout "${TIMEOUT}" \
        --wait \
        ${DRY_RUN:+--dry-run} \
        ${DEBUG:+--debug}
    
    if [[ "${DRY_RUN}" != "true" ]]; then
        log_success "Swit 平台升级成功"
        show_deployment_info
    fi
}

# 卸载 Swit 平台
uninstall_swit() {
    log_warning "确定要卸载 Swit 平台吗? (y/N)"
    read -r confirm
    if [[ "${confirm,,}" == "y" ]]; then
        log_info "卸载 Swit 平台..."
        helm uninstall "${RELEASE_NAME}" --namespace "${NAMESPACE}"
        log_success "Swit 平台已卸载"
        
        log_info "是否删除命名空间 ${NAMESPACE}? (y/N)"
        read -r confirm_ns
        if [[ "${confirm_ns,,}" == "y" ]]; then
            kubectl delete namespace "${NAMESPACE}" --ignore-not-found=true
            log_success "命名空间 ${NAMESPACE} 已删除"
        fi
    else
        log_info "取消卸载操作"
    fi
}

# 查看部署状态
show_status() {
    log_info "查看 Swit 平台状态..."
    
    echo ""
    echo -e "${CYAN}=== Helm 发布状态 ===${NC}"
    helm status "${RELEASE_NAME}" --namespace "${NAMESPACE}" || true
    
    echo ""
    echo -e "${CYAN}=== Pod 状态 ===${NC}"
    kubectl get pods -n "${NAMESPACE}" -o wide || true
    
    echo ""
    echo -e "${CYAN}=== 服务状态 ===${NC}"
    kubectl get services -n "${NAMESPACE}" || true
    
    echo ""
    echo -e "${CYAN}=== PVC 状态 ===${NC}"
    kubectl get pvc -n "${NAMESPACE}" || true
    
    echo ""
    echo -e "${CYAN}=== Ingress 状态 ===${NC}"
    kubectl get ingress -n "${NAMESPACE}" || true
}

# 查看服务日志
show_logs() {
    local service=$1
    local lines=${2:-100}
    
    if [[ -z "${service}" ]]; then
        log_error "请指定服务名称: swit-auth, swit-serve, mysql, consul"
        return 1
    fi
    
    case "${service}" in
        "swit-auth"|"auth")
            selector="app.kubernetes.io/component=authentication"
            ;;
        "swit-serve"|"serve")
            selector="app.kubernetes.io/component=api-server"
            ;;
        "mysql"|"db"|"database")
            selector="app.kubernetes.io/component=database"
            ;;
        "consul"|"discovery")
            selector="app.kubernetes.io/component=service-discovery"
            ;;
        *)
            log_error "未知服务: ${service}"
            return 1
            ;;
    esac
    
    log_info "查看 ${service} 服务日志 (最近 ${lines} 行)..."
    kubectl logs -n "${NAMESPACE}" -l "${selector}" --tail="${lines}" -f
}

# 重启服务
restart_service() {
    local service=$1
    
    if [[ -z "${service}" ]]; then
        log_error "请指定服务名称: swit-auth, swit-serve, mysql, consul"
        return 1
    fi
    
    case "${service}" in
        "swit-auth"|"auth")
            deployment="${RELEASE_NAME}-auth"
            ;;
        "swit-serve"|"serve")
            deployment="${RELEASE_NAME}-serve"
            ;;
        "mysql"|"db"|"database")
            deployment="${RELEASE_NAME}-mysql"
            ;;
        "consul"|"discovery")
            deployment="${RELEASE_NAME}-consul"
            ;;
        *)
            log_error "未知服务: ${service}"
            return 1
            ;;
    esac
    
    log_info "重启 ${service} 服务..."
    kubectl rollout restart deployment "${deployment}" -n "${NAMESPACE}"
    kubectl rollout status deployment "${deployment}" -n "${NAMESPACE}" --timeout="${TIMEOUT}"
    log_success "${service} 服务重启完成"
}

# 显示部署信息
show_deployment_info() {
    echo ""
    echo -e "${GREEN}=== 部署完成 ===${NC}"
    echo ""
    echo "查看状态: $0 status"
    echo "查看日志: $0 logs <service-name>"
    echo "重启服务: $0 restart <service-name>"
    echo ""
    echo "端口转发示例:"
    echo "  kubectl port-forward -n ${NAMESPACE} svc/${RELEASE_NAME}-serve-service 9000:9000"
    echo "  kubectl port-forward -n ${NAMESPACE} svc/${RELEASE_NAME}-auth-service 9001:9001"
    echo "  kubectl port-forward -n ${NAMESPACE} svc/${RELEASE_NAME}-consul-service 8500:8500"
    echo ""
}

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case $1 in
        -n|--namespace)
            NAMESPACE="$2"
            shift 2
            ;;
        -r|--release)
            RELEASE_NAME="$2"
            shift 2
            ;;
        -f|--values)
            VALUES_FILE="$2"
            shift 2
            ;;
        -t|--timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN="true"
            shift
            ;;
        --debug)
            DEBUG="true"
            shift
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            break
            ;;
    esac
done

# 检查必要工具
check_prerequisites

# 执行命令
case "$1" in
    "install")
        lint_chart
        install_swit
        ;;
    "upgrade")
        lint_chart
        upgrade_swit
        ;;
    "uninstall")
        uninstall_swit
        ;;
    "status")
        show_status
        ;;
    "logs")
        show_logs "$2" "$3"
        ;;
    "restart")
        restart_service "$2"
        ;;
    "build")
        build_images
        ;;
    "lint")
        lint_chart
        ;;
    "template")
        lint_chart
        template_chart
        ;;
    *)
        log_error "未知命令: $1"
        echo ""
        show_help
        exit 1
        ;;
esac 