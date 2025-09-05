#!/bin/bash

# 分布式追踪演示环境健康检查脚本
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
    echo -e "${PURPLE}[HEALTH]${NC} $1"
}

# 加载配置
load_config() {
    if [ -f "$CONFIG_FILE" ]; then
        source "$CONFIG_FILE"
    fi
}

# 显示欢迎信息
show_welcome() {
    echo "======================================================================"
    echo -e "${BLUE}🔍 SWIT 框架分布式追踪演示环境健康检查${NC}"
    echo "======================================================================"
    echo ""
}

# 检查 Docker 容器状态
check_containers() {
    log_header "检查 Docker 容器状态..."
    
    local containers=(
        "jaeger|Jaeger 追踪后端"
        "order-service|订单服务"
        "payment-service|支付服务"
        "inventory-service|库存服务"
    )
    
    local failed_containers=()
    local total_containers=${#containers[@]}
    local running_containers=0
    
    for container_info in "${containers[@]}"; do
        IFS='|' read -r container_name display_name <<< "$container_info"
        
        local status=$(docker ps --filter "name=$container_name" --format "{{.Status}}" | head -1)
        
        if [ -n "$status" ]; then
            if [[ "$status" == *"Up"* ]]; then
                log_success "✅ $display_name ($container_name): 运行中"
                running_containers=$((running_containers + 1))
            else
                log_warning "⚠️  $display_name ($container_name): $status"
            fi
        else
            log_error "❌ $display_name ($container_name): 未运行或不存在"
            failed_containers+=("$display_name")
        fi
    done
    
    echo ""
    log_info "容器状态总结: $running_containers/$total_containers 运行中"
    
    if [ ${#failed_containers[@]} -gt 0 ]; then
        log_error "以下容器存在问题: ${failed_containers[*]}"
        return 1
    fi
    
    return 0
}

# 检查网络连通性
check_network() {
    log_header "检查网络连通性..."
    
    local network_name="distributed-tracing_tracing-network"
    
    if docker network inspect "$network_name" > /dev/null 2>&1; then
        log_success "✅ Docker 网络 ($network_name) 存在"
        
        # 检查网络中的容器
        local containers_in_network=$(docker network inspect "$network_name" --format '{{range .Containers}}{{.Name}} {{end}}')
        if [ -n "$containers_in_network" ]; then
            log_success "✅ 网络中的容器: $containers_in_network"
        else
            log_warning "⚠️  网络中没有发现容器"
        fi
    else
        log_error "❌ Docker 网络不存在"
        return 1
    fi
    
    return 0
}

# 检查服务健康端点
check_service_health() {
    log_header "检查服务健康端点..."
    
    local services=(
        "http://localhost:16686|Jaeger UI|GET"
        "http://localhost:16686/api/services|Jaeger API|GET"
        "http://localhost:8081/health|订单服务健康检查|GET"
        "http://localhost:8083/health|库存服务健康检查|GET"
    )
    
    local failed_services=()
    local total_services=${#services[@]}
    local healthy_services=0
    
    for service_info in "${services[@]}"; do
        IFS='|' read -r url display_name method <<< "$service_info"
        
        log_info "检查: $display_name"
        
        local response_code
        if [ "$method" = "GET" ]; then
            response_code=$(curl -s -o /dev/null -w "%{http_code}" "$url" --max-time 10 || echo "000")
        else
            response_code="000"
        fi
        
        if [ "$response_code" = "200" ]; then
            log_success "✅ $display_name: 健康 (HTTP $response_code)"
            healthy_services=$((healthy_services + 1))
        elif [ "$response_code" = "000" ]; then
            log_error "❌ $display_name: 连接失败"
            failed_services+=("$display_name")
        else
            log_warning "⚠️  $display_name: HTTP $response_code"
            failed_services+=("$display_name")
        fi
    done
    
    echo ""
    log_info "服务健康状态总结: $healthy_services/$total_services 健康"
    
    if [ ${#failed_services[@]} -gt 0 ]; then
        log_error "以下服务存在问题: ${failed_services[*]}"
        return 1
    fi
    
    return 0
}

# 检查 gRPC 服务
check_grpc_services() {
    log_header "检查 gRPC 服务..."
    
    local grpc_services=(
        "localhost:9081|订单服务 gRPC"
        "localhost:9082|支付服务 gRPC"
        "localhost:9083|库存服务 gRPC"
    )
    
    local failed_grpc=()
    local total_grpc=${#grpc_services[@]}
    local healthy_grpc=0
    
    # 检查是否有 grpc_health_probe 工具
    local has_grpc_probe=false
    if command -v grpc_health_probe &> /dev/null; then
        has_grpc_probe=true
    fi
    
    for service_info in "${grpc_services[@]}"; do
        IFS='|' read -r address display_name <<< "$service_info"
        
        log_info "检查: $display_name"
        
        if [ "$has_grpc_probe" = true ]; then
            if grpc_health_probe -addr="$address" &> /dev/null; then
                log_success "✅ $display_name: 健康"
                healthy_grpc=$((healthy_grpc + 1))
            else
                log_error "❌ $display_name: 不健康或无法连接"
                failed_grpc+=("$display_name")
            fi
        else
            # 使用 netcat 或 telnet 检查端口
            if nc -z ${address/:/ } 2>/dev/null || timeout 3 bash -c "cat < /dev/null > /dev/tcp/${address/:/ }" 2>/dev/null; then
                log_success "✅ $display_name: 端口可访问"
                healthy_grpc=$((healthy_grpc + 1))
            else
                log_error "❌ $display_name: 端口不可访问"
                failed_grpc+=("$display_name")
            fi
        fi
    done
    
    if [ "$has_grpc_probe" = false ]; then
        log_warning "⚠️  未安装 grpc_health_probe，使用端口检查代替"
        log_info "安装命令: go install github.com/grpc-ecosystem/grpc-health-probe@latest"
    fi
    
    echo ""
    log_info "gRPC 服务状态总结: $healthy_grpc/$total_grpc 健康"
    
    if [ ${#failed_grpc[@]} -gt 0 ]; then
        log_error "以下 gRPC 服务存在问题: ${failed_grpc[*]}"
        return 1
    fi
    
    return 0
}

# 检查服务间通信
check_service_communication() {
    log_header "检查服务间通信..."
    
    log_info "测试订单服务到支付服务的连通性..."
    
    # 创建一个简单的订单来测试服务间通信
    local test_order='{
        "customer_id": "health-check-001",
        "product_id": "test-product",
        "quantity": 1,
        "amount": 10.00
    }'
    
    local response_code
    response_code=$(curl -s -o /dev/null -w "%{http_code}" \
        -X POST "http://localhost:8081/api/v1/orders" \
        -H "Content-Type: application/json" \
        -d "$test_order" \
        --max-time 15 || echo "000")
    
    if [ "$response_code" = "200" ] || [ "$response_code" = "201" ]; then
        log_success "✅ 服务间通信正常"
        return 0
    elif [ "$response_code" = "000" ]; then
        log_error "❌ 服务间通信测试连接失败"
        return 1
    else
        log_warning "⚠️  服务间通信测试返回 HTTP $response_code"
        log_info "这可能是预期的业务逻辑响应"
        return 0
    fi
}

# 检查追踪数据
check_tracing_data() {
    log_header "检查追踪数据收集..."
    
    log_info "查询 Jaeger 服务列表..."
    
    local services_response
    services_response=$(curl -s "http://localhost:16686/api/services" --max-time 10 || echo "")
    
    if [ -n "$services_response" ]; then
        if command -v jq &> /dev/null; then
            local service_count
            service_count=$(echo "$services_response" | jq '.data | length' 2>/dev/null || echo "0")
            
            if [ "$service_count" -gt 0 ]; then
                log_success "✅ 发现 $service_count 个追踪服务"
                
                # 显示服务列表
                echo "$services_response" | jq -r '.data[]' 2>/dev/null | while read -r service; do
                    log_info "  - $service"
                done
            else
                log_warning "⚠️  未发现追踪数据，可能需要生成一些流量"
            fi
        else
            log_success "✅ Jaeger API 可访问"
            log_info "安装 jq 以获得更详细的追踪数据分析"
        fi
    else
        log_error "❌ 无法获取 Jaeger 服务数据"
        return 1
    fi
    
    return 0
}

# 检查数据持久化
check_data_persistence() {
    log_header "检查数据持久化..."
    
    # 检查 Docker volumes
    local volumes=(
        "distributed-tracing_order-data"
        "distributed-tracing_inventory-data"
    )
    
    local missing_volumes=()
    
    for volume in "${volumes[@]}"; do
        if docker volume inspect "$volume" > /dev/null 2>&1; then
            log_success "✅ 数据卷存在: $volume"
        else
            log_error "❌ 数据卷缺失: $volume"
            missing_volumes+=("$volume")
        fi
    done
    
    if [ ${#missing_volumes[@]} -gt 0 ]; then
        log_error "以下数据卷缺失: ${missing_volumes[*]}"
        return 1
    fi
    
    return 0
}

# 生成健康报告
generate_health_report() {
    local status=$1
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    local report_file="$PROJECT_ROOT/data/logs/health-report-$(date '+%Y%m%d-%H%M%S').txt"
    
    # 确保目录存在
    mkdir -p "$(dirname "$report_file")"
    
    {
        echo "SWIT 分布式追踪演示环境健康检查报告"
        echo "========================================"
        echo "检查时间: $timestamp"
        echo "总体状态: $status"
        echo ""
        
        echo "Docker 容器状态:"
        docker ps --filter "name=jaeger|order-service|payment-service|inventory-service" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
        echo ""
        
        echo "Docker 网络状态:"
        docker network ls --filter "name=tracing" --format "table {{.Name}}\t{{.Driver}}\t{{.Scope}}"
        echo ""
        
        echo "Docker 数据卷状态:"
        docker volume ls --filter "name=distributed-tracing" --format "table {{.Name}}\t{{.Driver}}"
        echo ""
        
        echo "服务端口监听状态:"
        netstat -tlnp 2>/dev/null | grep -E ':16686|:8081|:8083|:9081|:9082|:9083' || echo "无法获取端口信息"
        
    } > "$report_file"
    
    log_info "健康检查报告已保存至: $report_file"
}

# 显示修复建议
show_fix_suggestions() {
    echo ""
    echo "======================================================================"
    echo -e "${YELLOW}🔧 常见问题修复建议${NC}"
    echo "======================================================================"
    echo ""
    echo "1. 如果容器未运行："
    echo "   ./scripts/setup.sh"
    echo ""
    echo "2. 如果服务无响应："
    echo "   docker-compose restart [service-name]"
    echo ""
    echo "3. 查看详细日志："
    echo "   ./scripts/logs.sh"
    echo "   docker-compose logs [service-name]"
    echo ""
    echo "4. 完全重新部署："
    echo "   ./scripts/cleanup.sh"
    echo "   ./scripts/setup.sh"
    echo ""
    echo "5. 检查端口占用："
    echo "   lsof -i :16686"
    echo "   lsof -i :8081"
    echo ""
}

# 显示成功信息
show_success_summary() {
    echo ""
    echo "======================================================================"
    echo -e "${GREEN}✅ 分布式追踪演示环境健康检查通过！${NC}"
    echo "======================================================================"
    echo ""
    echo -e "${YELLOW}📊 服务访问地址：${NC}"
    echo "  🔍 Jaeger UI:    http://localhost:16686"
    echo "  🛍️  订单服务:     http://localhost:8081"
    echo "  📦 库存服务:     http://localhost:8083"
    echo "  💳 支付服务:     gRPC :9082"
    echo ""
    echo -e "${YELLOW}🚀 建议的下一步：${NC}"
    echo "  ./scripts/demo-scenarios.sh   # 运行演示场景"
    echo "  ./scripts/load-test.sh        # 运行负载测试"
    echo ""
}

# 主函数
main() {
    show_welcome
    load_config
    
    cd "$PROJECT_ROOT"
    
    local overall_status="HEALTHY"
    local check_results=()
    
    # 执行各项检查
    if check_containers; then
        check_results+=("✅ 容器状态检查")
    else
        check_results+=("❌ 容器状态检查")
        overall_status="UNHEALTHY"
    fi
    
    if check_network; then
        check_results+=("✅ 网络连通性检查")
    else
        check_results+=("❌ 网络连通性检查")
        overall_status="UNHEALTHY"
    fi
    
    if check_service_health; then
        check_results+=("✅ 服务健康检查")
    else
        check_results+=("❌ 服务健康检查")
        overall_status="UNHEALTHY"
    fi
    
    if check_grpc_services; then
        check_results+=("✅ gRPC 服务检查")
    else
        check_results+=("❌ gRPC 服务检查")
        overall_status="UNHEALTHY"
    fi
    
    if check_service_communication; then
        check_results+=("✅ 服务间通信检查")
    else
        check_results+=("❌ 服务间通信检查")
        overall_status="UNHEALTHY"
    fi
    
    if check_tracing_data; then
        check_results+=("✅ 追踪数据检查")
    else
        check_results+=("❌ 追踪数据检查")
        overall_status="UNHEALTHY"
    fi
    
    if check_data_persistence; then
        check_results+=("✅ 数据持久化检查")
    else
        check_results+=("❌ 数据持久化检查")
        overall_status="UNHEALTHY"
    fi
    
    # 显示检查结果总结
    echo ""
    log_header "健康检查结果总结:"
    for result in "${check_results[@]}"; do
        echo "  $result"
    done
    
    # 生成报告
    generate_health_report "$overall_status"
    
    # 根据结果显示相应信息
    if [ "$overall_status" = "HEALTHY" ]; then
        show_success_summary
        exit 0
    else
        show_fix_suggestions
        exit 1
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