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

# 测试 HTTP 端点
test_http_endpoint() {
    local url=$1
    local name=$2
    local timeout=${3:-10}
    
    log_info "测试 $name ($url)..."
    
    if curl -f -s --connect-timeout $timeout "$url" > /dev/null; then
        log_success "$name 可访问"
        return 0
    else
        log_error "$name 无法访问"
        return 1
    fi
}

# 测试 TCP 端口
test_tcp_port() {
    local host=$1
    local port=$2
    local name=$3
    local timeout=${4:-5}
    
    log_info "测试 $name ($host:$port)..."
    
    if timeout $timeout bash -c "</dev/tcp/$host/$port"; then
        log_success "$name 端口可访问"
        return 0
    else
        log_error "$name 端口无法访问"
        return 1
    fi
}

# 主测试函数
main() {
    cd "$(dirname "$0")"
    
    echo "============================================"
    echo "         Swit 开发环境测试脚本"
    echo "============================================"
    echo ""
    
    # 检查 Docker 服务状态
    log_info "检查 Docker 服务状态..."
    if ! docker-compose ps | grep -q "Up"; then
        log_error "没有运行中的 Docker 服务，请先运行 ./start.sh"
        exit 1
    fi
    
    # 计数器
    success_count=0
    total_tests=0
    
    # 测试基础设施服务
    log_info ""
    log_info "=== 测试基础设施服务 ==="
    
    # 测试 MySQL
    ((total_tests++))
    if test_tcp_port "localhost" "3306" "MySQL 数据库"; then
        ((success_count++))
    fi
    
    # 测试 Consul
    ((total_tests++))
    if test_tcp_port "localhost" "8500" "Consul 服务发现"; then
        ((success_count++))
    fi
    
    # 测试 Consul UI
    ((total_tests++))
    if test_http_endpoint "http://localhost:8500/ui/" "Consul UI"; then
        ((success_count++))
    fi
    
    # 测试应用服务
    log_info ""
    log_info "=== 测试应用服务 ==="
    
    # 等待服务启动
    log_info "等待应用服务启动完成..."
    sleep 5
    
    # 测试认证服务
    ((total_tests++))
    if test_tcp_port "localhost" "9001" "认证服务端口"; then
        ((success_count++))
    fi
    
    ((total_tests++))
    if test_http_endpoint "http://localhost:9001/health" "认证服务健康检查"; then
        ((success_count++))
    fi
    
    # 测试主要服务
    ((total_tests++))
    if test_tcp_port "localhost" "9000" "主要服务端口"; then
        ((success_count++))
    fi
    
    ((total_tests++))
    if test_http_endpoint "http://localhost:9000/health" "主要服务健康检查"; then
        ((success_count++))
    fi
    
    # 测试服务发现注册
    log_info ""
    log_info "=== 测试服务发现注册 ==="
    
    ((total_tests++))
    if curl -s "http://localhost:8500/v1/catalog/services" | grep -q "swit"; then
        log_success "服务已在 Consul 中注册"
        ((success_count++))
    else
        log_warning "服务可能未在 Consul 中注册（这可能是正常的）"
    fi
    
    # 显示测试结果
    echo ""
    echo "============================================"
    echo "           测试结果汇总"
    echo "============================================"
    
    if [ $success_count -eq $total_tests ]; then
        log_success "所有测试通过！($success_count/$total_tests)"
        echo ""
        echo -e "${GREEN}🎉 开发环境部署成功！${NC}"
        echo ""
        echo "访问地址："
        echo "  📊 Consul UI:      http://localhost:8500"
        echo "  🔐 认证服务:       http://localhost:9001"
        echo "  🚀 主要服务:       http://localhost:9000"
        echo "  🗄️ MySQL 数据库:   localhost:3306 (root/root)"
        exit 0
    else
        log_error "部分测试失败 ($success_count/$total_tests)"
        echo ""
        echo "故障排除建议："
        echo "1. 检查服务日志: docker-compose logs -f"
        echo "2. 查看服务状态: docker-compose ps"
        echo "3. 重启服务: ./start.sh restart"
        echo "4. 清理重新部署: ./start.sh clean && ./start.sh start"
        exit 1
    fi
}

main "$@" 