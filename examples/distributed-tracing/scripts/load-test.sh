#!/bin/bash

# 分布式追踪演示环境负载测试脚本
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
    echo -e "${PURPLE}[LOAD-TEST]${NC} $1"
}

# 默认配置
DEFAULT_REQUESTS=100
DEFAULT_CONCURRENCY=10
DEFAULT_DURATION=30
DEFAULT_TIMEOUT=30

# 加载配置
load_config() {
    if [ -f "$CONFIG_FILE" ]; then
        source "$CONFIG_FILE"
    fi
    
    # 使用配置文件的值或默认值
    REQUESTS=${LOAD_TEST_REQUESTS:-$DEFAULT_REQUESTS}
    CONCURRENCY=${LOAD_TEST_CONCURRENCY:-$DEFAULT_CONCURRENCY}
    DURATION=${LOAD_TEST_DURATION:-$DEFAULT_DURATION}
    TIMEOUT=${LOAD_TEST_TIMEOUT:-$DEFAULT_TIMEOUT}
}

# 显示欢迎信息
show_welcome() {
    echo "======================================================================"
    echo -e "${BLUE}🚀 SWIT 框架分布式追踪演示环境负载测试${NC}"
    echo "======================================================================"
    echo -e "${YELLOW}本脚本将对分布式追踪演示环境进行负载测试：${NC}"
    echo "  🎯 目标服务: 订单服务 (Order Service)"
    echo "  📊 生成追踪数据用于性能分析"
    echo "  ⚡ 测试系统在负载下的表现"
    echo "  📈 收集性能指标和响应时间"
    echo ""
}

# 检查工具依赖
check_load_test_tools() {
    log_header "检查负载测试工具..."
    
    local available_tools=()
    local selected_tool=""
    
    # 检查 hey
    if command -v hey &> /dev/null; then
        available_tools+=("hey")
        log_success "✅ hey 负载测试工具可用"
    fi
    
    # 检查 ab (Apache Bench)
    if command -v ab &> /dev/null; then
        available_tools+=("ab")
        log_success "✅ ab (Apache Bench) 可用"
    fi
    
    # 检查 curl
    if command -v curl &> /dev/null; then
        available_tools+=("curl")
        log_success "✅ curl 可用"
    fi
    
    if [ ${#available_tools[@]} -eq 0 ]; then
        log_error "没有找到可用的负载测试工具"
        echo "请安装以下工具之一："
        echo "  hey:  go install github.com/rakyll/hey@latest"
        echo "  ab:   sudo apt-get install apache2-utils (Ubuntu/Debian)"
        echo "  curl: 通常系统自带"
        exit 1
    fi
    
    # 选择最优工具
    if [[ " ${available_tools[*]} " =~ " hey " ]]; then
        selected_tool="hey"
    elif [[ " ${available_tools[*]} " =~ " ab " ]]; then
        selected_tool="ab"
    else
        selected_tool="curl"
    fi
    
    log_success "选择负载测试工具: $selected_tool"
    echo "$selected_tool"
}

# 检查服务状态
check_services() {
    log_header "检查服务状态..."
    
    local services=(
        "http://localhost:8081/health|订单服务"
        "http://localhost:8083/health|库存服务"
        "http://localhost:16686|Jaeger UI"
    )
    
    for service_info in "${services[@]}"; do
        IFS='|' read -r url name <<< "$service_info"
        
        if curl -sf "$url" --max-time 5 > /dev/null 2>&1; then
            log_success "✅ $name 可用"
        else
            log_error "❌ $name 不可用: $url"
            echo "请先运行: ./scripts/setup.sh"
            exit 1
        fi
    done
}

# 生成测试数据
generate_test_data() {
    local count=$1
    local data_file="$PROJECT_ROOT/data/test-orders.json"
    
    log_header "生成测试数据 ($count 条)..."
    
    mkdir -p "$(dirname "$data_file")"
    
    cat > "$data_file" << 'EOF'
[
EOF
    
    for ((i=1; i<=count; i++)); do
        local customer_id="customer-$(printf "%04d" $((RANDOM % 1000 + 1)))"
        local product_id="product-$(printf "%03d" $((RANDOM % 100 + 1)))"
        local quantity=$((RANDOM % 5 + 1))
        local amount=$(awk "BEGIN {printf \"%.2f\", $(((RANDOM % 20000 + 1000) / 100))}")
        
        cat >> "$data_file" << EOF
  {
    "customer_id": "$customer_id",
    "product_id": "$product_id", 
    "quantity": $quantity,
    "amount": $amount
  }
EOF
        
        if [ $i -lt $count ]; then
            echo "," >> "$data_file"
        fi
    done
    
    cat >> "$data_file" << 'EOF'
]
EOF
    
    log_success "测试数据生成完成: $data_file"
    echo "$data_file"
}

# 运行 hey 负载测试
run_hey_test() {
    local requests=$1
    local concurrency=$2
    local data_file=$3
    local output_file=$4
    
    log_info "使用 hey 运行负载测试..."
    log_info "请求数: $requests, 并发: $concurrency"
    
    # 选择随机一条测试数据
    local test_data=$(jq -r '.[0]' "$data_file")
    
    hey -n "$requests" -c "$concurrency" \
        -m POST \
        -H "Content-Type: application/json" \
        -d "$test_data" \
        "http://localhost:8081/api/v1/orders" \
        | tee "$output_file"
}

# 运行 ab 负载测试
run_ab_test() {
    local requests=$1
    local concurrency=$2
    local data_file=$3
    local output_file=$4
    
    log_info "使用 ab 运行负载测试..."
    log_info "请求数: $requests, 并发: $concurrency"
    
    # 创建临时文件包含测试数据
    local temp_data=$(mktemp)
    jq -r '.[0]' "$data_file" > "$temp_data"
    
    ab -n "$requests" -c "$concurrency" \
        -p "$temp_data" \
        -T "application/json" \
        "http://localhost:8081/api/v1/orders" \
        | tee "$output_file"
    
    rm -f "$temp_data"
}

# 运行 curl 负载测试
run_curl_test() {
    local requests=$1
    local concurrency=$2
    local data_file=$3
    local output_file=$4
    
    log_info "使用 curl 运行负载测试..."
    log_info "请求数: $requests, 并发: $concurrency"
    
    local test_data=$(jq -r '.[0]' "$data_file")
    local start_time=$(date +%s)
    local success_count=0
    local error_count=0
    
    {
        echo "开始时间: $(date)"
        echo "请求总数: $requests"
        echo "并发数: $concurrency"
        echo "=========================================="
        
        # 使用循环模拟并发请求
        for ((i=1; i<=requests; i++)); do
            (
                local response_code
                response_code=$(curl -s -o /dev/null -w "%{http_code}" \
                    -X POST "http://localhost:8081/api/v1/orders" \
                    -H "Content-Type: application/json" \
                    -d "$test_data" \
                    --max-time "$TIMEOUT") || response_code="000"
                
                if [ "$response_code" = "200" ] || [ "$response_code" = "201" ]; then
                    echo "SUCCESS: Request $i"
                else
                    echo "ERROR: Request $i (HTTP $response_code)"
                fi
            ) &
            
            # 控制并发数
            if [ $((i % concurrency)) -eq 0 ]; then
                wait
            fi
        done
        
        wait
        
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        echo "=========================================="
        echo "结束时间: $(date)"
        echo "总耗时: ${duration}s"
        echo "QPS: $(awk "BEGIN {printf \"%.2f\", $requests / $duration}")"
        
    } | tee "$output_file"
}

# 运行负载测试
run_load_test() {
    local test_type=$1
    local requests=$2
    local concurrency=$3
    
    log_header "运行 $test_type 负载测试..."
    
    # 设置参数
    case $test_type in
        light)
            requests=50
            concurrency=5
            ;;
        medium)
            requests=200
            concurrency=20
            ;;
        heavy)
            requests=500
            concurrency=50
            ;;
        custom)
            # 使用传入的参数
            ;;
        *)
            requests=${requests:-$DEFAULT_REQUESTS}
            concurrency=${concurrency:-$DEFAULT_CONCURRENCY}
            ;;
    esac
    
    local timestamp=$(date '+%Y%m%d-%H%M%S')
    local output_dir="$PROJECT_ROOT/data/load-test-results"
    local output_file="$output_dir/load-test-$test_type-$timestamp.txt"
    
    mkdir -p "$output_dir"
    
    # 生成测试数据
    local data_file
    data_file=$(generate_test_data 10)
    
    # 获取可用的负载测试工具
    local tool
    tool=$(check_load_test_tools)
    
    log_info "开始负载测试: $requests 请求, $concurrency 并发"
    
    # 根据工具运行测试
    case $tool in
        hey)
            run_hey_test "$requests" "$concurrency" "$data_file" "$output_file"
            ;;
        ab)
            run_ab_test "$requests" "$concurrency" "$data_file" "$output_file"
            ;;
        curl)
            run_curl_test "$requests" "$concurrency" "$data_file" "$output_file"
            ;;
    esac
    
    log_success "负载测试完成，结果保存至: $output_file"
    echo "$output_file"
}

# 分析测试结果
analyze_results() {
    local result_file=$1
    
    log_header "分析测试结果..."
    
    if [ ! -f "$result_file" ]; then
        log_error "结果文件不存在: $result_file"
        return 1
    fi
    
    echo ""
    echo "======================================================================"
    echo -e "${YELLOW}📊 负载测试结果摘要${NC}"
    echo "======================================================================"
    
    # 提取关键指标
    if grep -q "Requests/sec" "$result_file"; then
        # hey 工具输出格式
        local rps=$(grep "Requests/sec" "$result_file" | awk '{print $2}')
        local avg_latency=$(grep "Average" "$result_file" | head -1 | awk '{print $2}')
        local total_time=$(grep "Total:" "$result_file" | awk '{print $2}')
        
        echo "  🚀 每秒请求数 (RPS): $rps"
        echo "  ⏱️  平均响应时间: $avg_latency"
        echo "  🕐 总耗时: $total_time"
        
    elif grep -q "Requests per second" "$result_file"; then
        # ab 工具输出格式
        local rps=$(grep "Requests per second" "$result_file" | awk '{print $4}')
        local avg_time=$(grep "Time per request" "$result_file" | head -1 | awk '{print $4}')
        local total_time=$(grep "Time taken for tests" "$result_file" | awk '{print $5}')
        
        echo "  🚀 每秒请求数 (RPS): $rps"
        echo "  ⏱️  平均响应时间: ${avg_time}ms"
        echo "  🕐 总耗时: ${total_time}s"
        
    else
        # curl 或其他格式
        echo "  📄 详细结果请查看: $result_file"
    fi
    
    # 检查错误情况
    if grep -q "ERROR" "$result_file" || grep -q "error" "$result_file"; then
        local error_count=$(grep -c "ERROR\|error" "$result_file" || echo "0")
        log_warning "⚠️  发现 $error_count 个错误"
    else
        log_success "✅ 没有发现错误"
    fi
}

# 检查追踪数据
check_tracing_data() {
    log_header "检查生成的追踪数据..."
    
    sleep 5  # 等待追踪数据上传到 Jaeger
    
    local services_response
    services_response=$(curl -s "http://localhost:16686/api/services" --max-time 10 || echo "")
    
    if [ -n "$services_response" ] && command -v jq &> /dev/null; then
        local service_count
        service_count=$(echo "$services_response" | jq '.data | length' 2>/dev/null || echo "0")
        
        if [ "$service_count" -gt 0 ]; then
            log_success "✅ Jaeger 中发现 $service_count 个服务的追踪数据"
            
            echo "服务列表:"
            echo "$services_response" | jq -r '.data[]' 2>/dev/null | while read -r service; do
                echo "  - $service"
            done
            
            echo ""
            log_info "🔍 查看追踪数据: http://localhost:16686"
            
        else
            log_warning "⚠️  Jaeger 中暂未发现追踪数据"
        fi
    else
        log_warning "⚠️  无法获取 Jaeger 数据或 jq 未安装"
    fi
}

# 显示完成信息
show_completion() {
    local result_file=$1
    
    echo ""
    echo "======================================================================"
    echo -e "${GREEN}🎉 负载测试完成！${NC}"
    echo "======================================================================"
    echo ""
    echo -e "${YELLOW}📋 后续建议：${NC}"
    echo "  1. 查看详细结果: cat $result_file"
    echo "  2. 分析追踪数据: http://localhost:16686"
    echo "  3. 查看服务日志: ./scripts/logs.sh"
    echo "  4. 运行健康检查: ./scripts/health-check.sh"
    echo ""
    echo -e "${BLUE}💡 性能优化建议：${NC}"
    echo "  - 分析慢查询和瓶颈span"
    echo "  - 检查服务间调用耗时"
    echo "  - 监控资源使用情况"
    echo "  - 优化数据库查询"
    echo ""
}

# 显示使用帮助
show_help() {
    echo "用法: $0 [测试类型] [选项]"
    echo ""
    echo "测试类型:"
    echo "  light             轻量测试 (50 请求, 5 并发)"
    echo "  medium            中等测试 (200 请求, 20 并发)"
    echo "  heavy             重量测试 (500 请求, 50 并发)"
    echo "  custom            自定义测试参数"
    echo ""
    echo "自定义测试选项:"
    echo "  --requests, -n    请求总数 (默认: $DEFAULT_REQUESTS)"
    echo "  --concurrency, -c 并发数 (默认: $DEFAULT_CONCURRENCY)"
    echo "  --timeout, -t     超时时间 (默认: $DEFAULT_TIMEOUT)"
    echo ""
    echo "其他选项:"
    echo "  --help, -h        显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0 light          # 轻量测试"
    echo "  $0 medium         # 中等测试"
    echo "  $0 custom -n 1000 -c 100  # 自定义测试"
    echo ""
}

# 主函数
main() {
    local test_type="medium"
    local requests=""
    local concurrency=""
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            light|medium|heavy|custom)
                test_type="$1"
                shift
                ;;
            --requests|-n)
                requests="$2"
                shift 2
                ;;
            --concurrency|-c)
                concurrency="$2"
                shift 2
                ;;
            --timeout|-t)
                TIMEOUT="$2"
                shift 2
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
    load_config
    
    # 切换到项目根目录
    cd "$PROJECT_ROOT"
    
    # 检查服务状态
    check_services
    
    # 运行负载测试
    local result_file
    result_file=$(run_load_test "$test_type" "$requests" "$concurrency")
    
    # 分析结果
    analyze_results "$result_file"
    
    # 检查追踪数据
    check_tracing_data
    
    # 显示完成信息
    show_completion "$result_file"
}

# 信号处理
cleanup_on_exit() {
    log_info "接收到中断信号，清理并退出..."
    exit 1
}

trap cleanup_on_exit SIGINT SIGTERM

# 执行主函数
main "$@"