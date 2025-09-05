#!/bin/bash

# 分布式追踪演示场景脚本
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
CYAN='\033[0;36m'
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
    echo -e "${PURPLE}[DEMO]${NC} $1"
}

log_scenario() {
    echo -e "${CYAN}[SCENARIO]${NC} $1"
}

# 加载配置
load_config() {
    if [ -f "$CONFIG_FILE" ]; then
        source "$CONFIG_FILE"
    fi
    
    # 默认值
    DEMO_DELAY_SECONDS=${DEMO_DELAY_SECONDS:-2}
    ORDER_SERVICE_PORT=${ORDER_SERVICE_PORT:-8081}
}

# 显示欢迎信息
show_welcome() {
    echo "======================================================================"
    echo -e "${BLUE}🎬 SWIT 框架分布式追踪演示场景${NC}"
    echo "======================================================================"
    echo -e "${YELLOW}本脚本提供多个演示场景来展示分布式追踪功能：${NC}"
    echo "  🛒 场景1: 正常订单流程 - 展示完整的服务调用链"
    echo "  💳 场景2: 支付失败处理 - 展示错误传播和处理"
    echo "  📦 场景3: 库存不足场景 - 展示业务异常处理"
    echo "  🐌 场景4: 性能瓶颈模拟 - 展示慢查询追踪"
    echo "  🔄 场景5: 并发订单处理 - 展示高并发场景"
    echo "  🎯 场景6: 全链路演示 - 综合展示所有功能"
    echo ""
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

# 发送HTTP请求并显示结果
send_request() {
    local url=$1
    local method=$2
    local data=$3
    local description=$4
    local expect_success=${5:-true}
    
    log_info "发送请求: $description"
    log_info "URL: $method $url"
    
    if [ -n "$data" ]; then
        log_info "数据: $data"
    fi
    
    local response_file=$(mktemp)
    local http_code
    
    if [ -n "$data" ]; then
        http_code=$(curl -s -w "%{http_code}" -o "$response_file" \
            -X "$method" "$url" \
            -H "Content-Type: application/json" \
            -d "$data" \
            --max-time 30 || echo "000")
    else
        http_code=$(curl -s -w "%{http_code}" -o "$response_file" \
            -X "$method" "$url" \
            --max-time 30 || echo "000")
    fi
    
    local response_body=$(cat "$response_file")
    rm -f "$response_file"
    
    if [ "$expect_success" = true ]; then
        if [ "$http_code" = "200" ] || [ "$http_code" = "201" ]; then
            log_success "✅ 请求成功 (HTTP $http_code)"
        else
            log_warning "⚠️  请求返回 HTTP $http_code"
        fi
    else
        if [ "$http_code" = "400" ] || [ "$http_code" = "422" ] || [ "$http_code" = "500" ]; then
            log_success "✅ 预期的错误响应 (HTTP $http_code)"
        else
            log_warning "⚠️  非预期响应 (HTTP $http_code)"
        fi
    fi
    
    if command -v jq &> /dev/null && [[ "$response_body" =~ ^[[:space:]]*[\{\[] ]]; then
        log_info "响应: $(echo "$response_body" | jq -c .)"
    else
        log_info "响应: $response_body"
    fi
    
    echo "$response_body"
}

# 场景1: 正常订单流程
scenario_normal_flow() {
    log_scenario "🛒 场景1: 正常订单流程演示"
    echo "=========================================="
    echo "此场景模拟正常的订单创建流程："
    echo "1. 客户下单"
    echo "2. 检查库存"
    echo "3. 处理支付"
    echo "4. 创建订单"
    echo "5. 更新库存"
    echo ""
    
    local customers=(
        "customer-001|张三"
        "customer-002|李四"
        "customer-003|王五"
    )
    
    local products=(
        "product-001|iPhone 14|999.99"
        "product-002|MacBook Pro|2499.99"
        "product-003|iPad Air|599.99"
    )
    
    for customer_info in "${customers[@]}"; do
        IFS='|' read -r customer_id customer_name <<< "$customer_info"
        
        for product_info in "${products[@]}"; do
            IFS='|' read -r product_id product_name price <<< "$product_info"
            
            local quantity=$((RANDOM % 3 + 1))
            local total_amount=$(awk "BEGIN {printf \"%.2f\", $price * $quantity}")
            
            log_info "👤 客户: $customer_name ($customer_id)"
            log_info "🛍️  商品: $product_name ($product_id)"
            log_info "💰 金额: $price × $quantity = $total_amount"
            
            local order_data='{
                "customer_id": "'$customer_id'",
                "product_id": "'$product_id'",
                "quantity": '$quantity',
                "amount": '$total_amount'
            }'
            
            send_request \
                "http://localhost:$ORDER_SERVICE_PORT/api/v1/orders" \
                "POST" \
                "$order_data" \
                "创建订单: $customer_name 购买 $product_name"
            
            log_info "等待 ${DEMO_DELAY_SECONDS} 秒..."
            sleep "$DEMO_DELAY_SECONDS"
            echo ""
        done
    done
    
    log_success "✅ 正常订单流程演示完成"
}

# 场景2: 支付失败处理
scenario_payment_failure() {
    log_scenario "💳 场景2: 支付失败处理演示"
    echo "=========================================="
    echo "此场景模拟支付失败的情况："
    echo "1. 客户下单"
    echo "2. 库存检查通过"
    echo "3. 支付处理失败"
    echo "4. 触发补偿事务"
    echo "5. 订单创建失败"
    echo ""
    
    # 使用特殊的客户ID来触发支付失败
    local failing_customers=(
        "payment-fail-001|支付失败客户1"
        "payment-fail-002|支付失败客户2"
        "invalid-card-001|无效卡客户"
    )
    
    local products=(
        "product-001|测试商品A|99.99"
        "product-002|测试商品B|199.99"
    )
    
    for customer_info in "${failing_customers[@]}"; do
        IFS='|' read -r customer_id customer_name <<< "$customer_info"
        
        local product_info="${products[0]}"
        IFS='|' read -r product_id product_name price <<< "$product_info"
        
        log_info "👤 客户: $customer_name ($customer_id)"
        log_info "🛍️  商品: $product_name ($product_id)"
        log_info "💰 金额: $price"
        
        local order_data='{
            "customer_id": "'$customer_id'",
            "product_id": "'$product_id'",
            "quantity": 1,
            "amount": '$price'
        }'
        
        send_request \
            "http://localhost:$ORDER_SERVICE_PORT/api/v1/orders" \
            "POST" \
            "$order_data" \
            "创建订单 (预期支付失败): $customer_name" \
            false
        
        log_info "等待 ${DEMO_DELAY_SECONDS} 秒..."
        sleep "$DEMO_DELAY_SECONDS"
        echo ""
    done
    
    log_success "✅ 支付失败处理演示完成"
}

# 场景3: 库存不足场景
scenario_inventory_shortage() {
    log_scenario "📦 场景3: 库存不足场景演示"
    echo "=========================================="
    echo "此场景模拟库存不足的情况："
    echo "1. 客户下单大量商品"
    echo "2. 库存检查失败"
    echo "3. 订单创建失败"
    echo "4. 返回库存不足错误"
    echo ""
    
    local customers=(
        "customer-bulk-001|批量采购客户1"
        "customer-bulk-002|批量采购客户2"
    )
    
    local products=(
        "product-limited-001|限量商品A|99.99"
        "product-limited-002|限量商品B|199.99"
    )
    
    for customer_info in "${customers[@]}"; do
        IFS='|' read -r customer_id customer_name <<< "$customer_info"
        
        for product_info in "${products[@]}"; do
            IFS='|' read -r product_id product_name price <<< "$product_info"
            
            # 故意请求大量商品来触发库存不足
            local quantity=$((RANDOM % 100 + 50))
            local total_amount=$(awk "BEGIN {printf \"%.2f\", $price * $quantity}")
            
            log_info "👤 客户: $customer_name ($customer_id)"
            log_info "🛍️  商品: $product_name ($product_id)"
            log_info "💰 金额: $price × $quantity = $total_amount"
            log_warning "⚠️  请求数量较大，可能库存不足"
            
            local order_data='{
                "customer_id": "'$customer_id'",
                "product_id": "'$product_id'",
                "quantity": '$quantity',
                "amount": '$total_amount'
            }'
            
            send_request \
                "http://localhost:$ORDER_SERVICE_PORT/api/v1/orders" \
                "POST" \
                "$order_data" \
                "创建大量订单 (预期库存不足): $customer_name" \
                false
            
            log_info "等待 ${DEMO_DELAY_SECONDS} 秒..."
            sleep "$DEMO_DELAY_SECONDS"
            echo ""
        done
    done
    
    log_success "✅ 库存不足场景演示完成"
}

# 场景4: 性能瓶颈模拟
scenario_performance_bottleneck() {
    log_scenario "🐌 场景4: 性能瓶颈模拟演示"
    echo "=========================================="
    echo "此场景通过创建多个慢查询来模拟性能瓶颈："
    echo "1. 并发创建多个订单"
    echo "2. 触发数据库慢查询"
    echo "3. 展示长耗时的 span"
    echo "4. 分析性能瓶颈点"
    echo ""
    
    local concurrent_orders=5
    local pids=()
    
    log_info "并发创建 $concurrent_orders 个订单..."
    
    for ((i=1; i<=concurrent_orders; i++)); do
        (
            local customer_id="perf-customer-$(printf "%03d" $i)"
            local product_id="heavy-product-$(printf "%03d" $((i % 3 + 1)))"
            local quantity=$((RANDOM % 5 + 1))
            local amount=$(awk "BEGIN {printf \"%.2f\", $(((RANDOM % 50000 + 10000) / 100))}")
            
            local order_data='{
                "customer_id": "'$customer_id'",
                "product_id": "'$product_id'",
                "quantity": '$quantity',
                "amount": '$amount',
                "simulate_slow_query": true
            }'
            
            log_info "[$i] 创建性能测试订单: $customer_id"
            send_request \
                "http://localhost:$ORDER_SERVICE_PORT/api/v1/orders" \
                "POST" \
                "$order_data" \
                "并发订单 #$i (模拟慢查询)" \
                true > /dev/null
            
        ) &
        pids+=($!)
    done
    
    # 等待所有并发请求完成
    log_info "等待所有并发请求完成..."
    for pid in "${pids[@]}"; do
        wait "$pid"
    done
    
    log_success "✅ 性能瓶颈模拟演示完成"
}

# 场景5: 并发订单处理
scenario_concurrent_orders() {
    log_scenario "🔄 场景5: 并发订单处理演示"
    echo "=========================================="
    echo "此场景创建大量并发订单来测试系统并发处理能力："
    echo "1. 同时创建多个订单"
    echo "2. 测试并发安全性"
    echo "3. 观察资源竞争"
    echo "4. 分析并发性能"
    echo ""
    
    local concurrent_count=10
    local pids=()
    
    log_info "创建 $concurrent_count 个并发订单..."
    
    for ((i=1; i<=concurrent_count; i++)); do
        (
            local customer_id="concurrent-customer-$(printf "%03d" $i)"
            local product_id="concurrent-product-$(printf "%02d" $((i % 5 + 1)))"
            local quantity=$((RANDOM % 3 + 1))
            local amount=$(awk "BEGIN {printf \"%.2f\", $(((RANDOM % 20000 + 5000) / 100))}")
            
            local order_data='{
                "customer_id": "'$customer_id'",
                "product_id": "'$product_id'",
                "quantity": '$quantity',
                "amount": '$amount'
            }'
            
            send_request \
                "http://localhost:$ORDER_SERVICE_PORT/api/v1/orders" \
                "POST" \
                "$order_data" \
                "并发订单 #$i" \
                true > /dev/null
            
            echo "[$i] 完成"
        ) &
        pids+=($!)
        
        # 控制启动间隔，避免过于密集
        sleep 0.1
    done
    
    # 等待所有请求完成
    log_info "等待所有并发请求完成..."
    for pid in "${pids[@]}"; do
        wait "$pid"
    done
    
    log_success "✅ 并发订单处理演示完成"
}

# 场景6: 全链路演示
scenario_full_demo() {
    log_scenario "🎯 场景6: 全链路综合演示"
    echo "=========================================="
    echo "此场景综合展示所有功能，包括："
    echo "1. 正常订单处理"
    echo "2. 异常情况处理"
    echo "3. 并发场景"
    echo "4. 性能分析"
    echo ""
    
    log_info "开始全链路综合演示..."
    
    # 1. 创建一些正常订单
    log_info "Step 1: 创建正常订单..."
    local normal_customers=("customer-demo-001" "customer-demo-002")
    local products=("product-demo-001" "product-demo-002")
    
    for customer_id in "${normal_customers[@]}"; do
        for product_id in "${products[@]}"; do
            local quantity=$((RANDOM % 3 + 1))
            local amount=$(awk "BEGIN {printf \"%.2f\", $(((RANDOM % 30000 + 10000) / 100))}")
            
            local order_data='{
                "customer_id": "'$customer_id'",
                "product_id": "'$product_id'",
                "quantity": '$quantity',
                "amount": '$amount'
            }'
            
            send_request \
                "http://localhost:$ORDER_SERVICE_PORT/api/v1/orders" \
                "POST" \
                "$order_data" \
                "正常订单: $customer_id -> $product_id" \
                true > /dev/null
        done
    done
    
    sleep 2
    
    # 2. 创建一些异常订单
    log_info "Step 2: 创建异常订单..."
    local error_scenarios=(
        "payment-fail-demo-001"
        "inventory-shortage-demo-001"
    )
    
    for customer_id in "${error_scenarios[@]}"; do
        local order_data='{
            "customer_id": "'$customer_id'",
            "product_id": "error-product-001",
            "quantity": 1,
            "amount": 99.99
        }'
        
        send_request \
            "http://localhost:$ORDER_SERVICE_PORT/api/v1/orders" \
            "POST" \
            "$order_data" \
            "异常订单: $customer_id" \
            false > /dev/null
    done
    
    sleep 2
    
    # 3. 并发订单测试
    log_info "Step 3: 并发订单测试..."
    local pids=()
    for ((i=1; i<=5; i++)); do
        (
            local order_data='{
                "customer_id": "concurrent-demo-'$i'",
                "product_id": "concurrent-product-001",
                "quantity": 1,
                "amount": 199.99
            }'
            
            send_request \
                "http://localhost:$ORDER_SERVICE_PORT/api/v1/orders" \
                "POST" \
                "$order_data" \
                "并发测试 #$i" \
                true > /dev/null
        ) &
        pids+=($!)
    done
    
    for pid in "${pids[@]}"; do
        wait "$pid"
    done
    
    log_success "✅ 全链路综合演示完成"
}

# 显示追踪数据建议
show_tracing_tips() {
    echo ""
    echo "======================================================================"
    echo -e "${YELLOW}🔍 追踪数据分析建议${NC}"
    echo "======================================================================"
    echo ""
    echo -e "${BLUE}1. 访问 Jaeger UI:${NC}"
    echo "   http://localhost:16686"
    echo ""
    echo -e "${BLUE}2. 查看服务列表:${NC}"
    echo "   - order-service (订单服务)"
    echo "   - payment-service (支付服务)"  
    echo "   - inventory-service (库存服务)"
    echo ""
    echo -e "${BLUE}3. 分析关键指标:${NC}"
    echo "   - 请求耗时分布"
    echo "   - 错误率统计"
    echo "   - 服务依赖关系"
    echo "   - 慢查询识别"
    echo ""
    echo -e "${BLUE}4. 查找问题:${NC}"
    echo "   - 搜索错误 tag: error=true"
    echo "   - 查看慢请求: duration > 1s"
    echo "   - 分析失败订单的完整调用链"
    echo ""
}

# 显示完成信息
show_completion() {
    local scenario=$1
    
    echo ""
    echo "======================================================================"
    echo -e "${GREEN}🎉 演示场景完成！${NC}"
    echo "======================================================================"
    echo ""
    echo "已完成场景: $scenario"
    echo ""
    show_tracing_tips
    
    echo -e "${YELLOW}🛠️  相关工具:${NC}"
    echo "  ./scripts/health-check.sh   # 检查系统状态"
    echo "  ./scripts/load-test.sh       # 运行负载测试"
    echo "  ./scripts/logs.sh            # 查看服务日志"
    echo ""
}

# 显示使用帮助
show_help() {
    echo "用法: $0 [场景] [选项]"
    echo ""
    echo "可用场景:"
    echo "  normal-flow           正常订单流程演示"
    echo "  payment-failure       支付失败处理演示"
    echo "  inventory-shortage    库存不足场景演示"
    echo "  performance-bottleneck 性能瓶颈模拟演示"
    echo "  concurrent-orders     并发订单处理演示"
    echo "  full-demo            全链路综合演示"
    echo "  all                  运行所有场景"
    echo ""
    echo "选项:"
    echo "  --delay, -d          场景间延迟时间 (秒, 默认: 2)"
    echo "  --help, -h           显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0 normal-flow       # 运行正常订单流程"
    echo "  $0 all               # 运行所有场景"
    echo "  $0 full-demo -d 5    # 运行全链路演示，延迟5秒"
    echo ""
}

# 主函数
main() {
    local scenario="normal-flow"
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            normal-flow|payment-failure|inventory-shortage|performance-bottleneck|concurrent-orders|full-demo|all)
                scenario="$1"
                shift
                ;;
            --delay|-d)
                DEMO_DELAY_SECONDS="$2"
                shift 2
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                log_error "未知选项或场景: $1"
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
    
    echo ""
    log_header "开始运行演示场景: $scenario"
    echo ""
    
    # 根据场景运行相应函数
    case $scenario in
        normal-flow)
            scenario_normal_flow
            ;;
        payment-failure)
            scenario_payment_failure
            ;;
        inventory-shortage)
            scenario_inventory_shortage
            ;;
        performance-bottleneck)
            scenario_performance_bottleneck
            ;;
        concurrent-orders)
            scenario_concurrent_orders
            ;;
        full-demo)
            scenario_full_demo
            ;;
        all)
            log_info "运行所有演示场景..."
            scenario_normal_flow
            sleep 3
            scenario_payment_failure
            sleep 3
            scenario_inventory_shortage
            sleep 3
            scenario_performance_bottleneck
            sleep 3
            scenario_concurrent_orders
            sleep 3
            scenario_full_demo
            scenario="所有场景"
            ;;
    esac
    
    show_completion "$scenario"
}

# 信号处理
cleanup_on_exit() {
    log_info "接收到中断信号，清理并退出..."
    exit 1
}

trap cleanup_on_exit SIGINT SIGTERM

# 执行主函数
main "$@"