#!/bin/bash

# 分布式追踪演示环境日志查看脚本
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
    echo -e "${PURPLE}[LOGS]${NC} $1"
}

# 显示欢迎信息
show_welcome() {
    echo "======================================================================"
    echo -e "${BLUE}📋 SWIT 框架分布式追踪演示环境日志查看器${NC}"
    echo "======================================================================"
    echo -e "${YELLOW}本工具提供以下日志查看功能：${NC}"
    echo "  🐳 Docker 容器日志查看"
    echo "  🔍 日志搜索和过滤"
    echo "  📊 实时日志跟踪"
    echo "  💾 日志导出和保存"
    echo "  🎯 错误日志筛选"
    echo ""
}

# 检查 Docker 服务
check_docker_services() {
    log_header "检查 Docker 服务状态..."
    
    cd "$PROJECT_ROOT"
    
    local containers=(
        "jaeger"
        "order-service"
        "payment-service"
        "inventory-service"
    )
    
    local running_containers=()
    local stopped_containers=()
    
    for container in "${containers[@]}"; do
        if docker ps --filter "name=$container" --format "{{.Names}}" | grep -q "^${container}$"; then
            running_containers+=("$container")
        else
            stopped_containers+=("$container")
        fi
    done
    
    if [ ${#running_containers[@]} -gt 0 ]; then
        log_success "运行中的容器: ${running_containers[*]}"
    fi
    
    if [ ${#stopped_containers[@]} -gt 0 ]; then
        log_warning "未运行的容器: ${stopped_containers[*]}"
    fi
    
    if [ ${#running_containers[@]} -eq 0 ]; then
        log_error "没有运行中的容器，请先启动服务: ./scripts/setup.sh"
        exit 1
    fi
}

# 显示服务列表
show_service_menu() {
    echo ""
    echo "可用的服务:"
    echo "  1. jaeger          - Jaeger 分布式追踪后端"
    echo "  2. order-service   - 订单服务"
    echo "  3. payment-service - 支付服务"
    echo "  4. inventory-service - 库存服务"
    echo "  5. all             - 所有服务"
    echo ""
}

# 查看指定服务的日志
view_service_logs() {
    local service=$1
    local follow=${2:-false}
    local tail_lines=${3:-100}
    local save_file=${4:-""}
    local filter_errors=${5:-false}
    
    log_header "查看 $service 服务日志..."
    
    cd "$PROJECT_ROOT"
    
    # 构建 docker-compose logs 命令
    local cmd_args=()
    
    if [ "$follow" = true ]; then
        cmd_args+=("-f")
    fi
    
    cmd_args+=("--tail=$tail_lines")
    cmd_args+=("$service")
    
    log_info "执行命令: docker-compose logs ${cmd_args[*]}"
    echo ""
    
    if [ -n "$save_file" ]; then
        # 保存日志到文件
        mkdir -p "$(dirname "$save_file")"
        
        if command -v docker-compose &> /dev/null; then
            docker-compose logs "${cmd_args[@]}" | tee "$save_file"
        else
            docker compose logs "${cmd_args[@]}" | tee "$save_file"
        fi
        
        log_success "日志已保存到: $save_file"
    else
        # 直接显示日志
        if [ "$filter_errors" = true ]; then
            # 过滤错误日志
            if command -v docker-compose &> /dev/null; then
                docker-compose logs "${cmd_args[@]}" | grep -i "error\|exception\|panic\|fatal"
            else
                docker compose logs "${cmd_args[@]}" | grep -i "error\|exception\|panic\|fatal"
            fi
        else
            # 显示所有日志
            if command -v docker-compose &> /dev/null; then
                docker-compose logs "${cmd_args[@]}"
            else
                docker compose logs "${cmd_args[@]}"
            fi
        fi
    fi
}

# 查看所有服务日志
view_all_logs() {
    local follow=${1:-false}
    local tail_lines=${2:-50}
    local save_file=${3:-""}
    local filter_errors=${4:-false}
    
    log_header "查看所有服务日志..."
    
    cd "$PROJECT_ROOT"
    
    local cmd_args=()
    
    if [ "$follow" = true ]; then
        cmd_args+=("-f")
    fi
    
    cmd_args+=("--tail=$tail_lines")
    
    log_info "执行命令: docker-compose logs ${cmd_args[*]}"
    echo ""
    
    if [ -n "$save_file" ]; then
        mkdir -p "$(dirname "$save_file")"
        
        if command -v docker-compose &> /dev/null; then
            docker-compose logs "${cmd_args[@]}" | tee "$save_file"
        else
            docker compose logs "${cmd_args[@]}" | tee "$save_file"
        fi
        
        log_success "日志已保存到: $save_file"
    else
        if [ "$filter_errors" = true ]; then
            if command -v docker-compose &> /dev/null; then
                docker-compose logs "${cmd_args[@]}" | grep -i "error\|exception\|panic\|fatal"
            else
                docker compose logs "${cmd_args[@]}" | grep -i "error\|exception\|panic\|fatal"
            fi
        else
            if command -v docker-compose &> /dev/null; then
                docker-compose logs "${cmd_args[@]}"
            else
                docker compose logs "${cmd_args[@]}"
            fi
        fi
    fi
}

# 搜索日志内容
search_logs() {
    local service=$1
    local search_term=$2
    local context_lines=${3:-3}
    
    log_header "在 $service 服务日志中搜索: $search_term"
    
    cd "$PROJECT_ROOT"
    
    if [ "$service" = "all" ]; then
        log_info "搜索所有服务日志..."
        if command -v docker-compose &> /dev/null; then
            docker-compose logs | grep -i "$search_term" -C "$context_lines"
        else
            docker compose logs | grep -i "$search_term" -C "$context_lines"
        fi
    else
        log_info "搜索 $service 服务日志..."
        if command -v docker-compose &> /dev/null; then
            docker-compose logs "$service" | grep -i "$search_term" -C "$context_lines"
        else
            docker compose logs "$service" | grep -i "$search_term" -C "$context_lines"
        fi
    fi
}

# 分析日志统计
analyze_logs() {
    local service=$1
    local hours=${2:-1}
    
    log_header "分析 $service 服务日志 (最近 $hours 小时)"
    
    cd "$PROJECT_ROOT"
    
    local temp_file=$(mktemp)
    
    # 获取日志
    if [ "$service" = "all" ]; then
        if command -v docker-compose &> /dev/null; then
            docker-compose logs --since="${hours}h" > "$temp_file"
        else
            docker compose logs --since="${hours}h" > "$temp_file"
        fi
    else
        if command -v docker-compose &> /dev/null; then
            docker-compose logs --since="${hours}h" "$service" > "$temp_file"
        else
            docker compose logs --since="${hours}h" "$service" > "$temp_file"
        fi
    fi
    
    echo ""
    echo "======================================================================"
    echo -e "${YELLOW}📊 日志统计分析 (最近 $hours 小时)${NC}"
    echo "======================================================================"
    
    # 总行数
    local total_lines=$(wc -l < "$temp_file")
    echo "📋 总日志行数: $total_lines"
    
    # 错误统计
    local error_count=$(grep -ci "error\|exception\|panic\|fatal" "$temp_file" || echo "0")
    echo "❌ 错误日志数: $error_count"
    
    # 警告统计
    local warning_count=$(grep -ci "warning\|warn" "$temp_file" || echo "0")
    echo "⚠️  警告日志数: $warning_count"
    
    # INFO 统计
    local info_count=$(grep -ci "info" "$temp_file" || echo "0")
    echo "ℹ️  信息日志数: $info_count"
    
    echo ""
    echo "🔍 最频繁的错误类型:"
    grep -i "error\|exception\|panic\|fatal" "$temp_file" | \
        sed 's/.*\(error\|exception\|panic\|fatal\)[[:space:]]*[:]*[[:space:]]*//' | \
        sort | uniq -c | sort -nr | head -5 || echo "  无错误日志"
    
    echo ""
    echo "📈 按小时的日志分布:"
    grep -o "[0-9]\{4\}-[0-9]\{2\}-[0-9]\{2\}T[0-9]\{2\}" "$temp_file" | \
        cut -c12-13 | sort | uniq -c | sort -k2n | \
        awk '{printf "  %02d:00 - %02d:59  %s\n", $2, $2, $1}' || echo "  无时间戳信息"
    
    rm -f "$temp_file"
}

# 监控实时错误日志
monitor_errors() {
    local service=${1:-"all"}
    
    log_header "监控 $service 服务的实时错误日志"
    log_info "按 Ctrl+C 停止监控"
    echo ""
    
    cd "$PROJECT_ROOT"
    
    if [ "$service" = "all" ]; then
        if command -v docker-compose &> /dev/null; then
            docker-compose logs -f | grep -i --line-buffered "error\|exception\|panic\|fatal\|warning\|warn"
        else
            docker compose logs -f | grep -i --line-buffered "error\|exception\|panic\|fatal\|warning\|warn"
        fi
    else
        if command -v docker-compose &> /dev/null; then
            docker-compose logs -f "$service" | grep -i --line-buffered "error\|exception\|panic\|fatal\|warning\|warn"
        else
            docker compose logs -f "$service" | grep -i --line-buffered "error\|exception\|panic\|fatal\|warning\|warn"
        fi
    fi
}

# 导出日志报告
export_log_report() {
    local service=${1:-"all"}
    local hours=${2:-24}
    local output_dir="$PROJECT_ROOT/data/logs/reports"
    local timestamp=$(date '+%Y%m%d-%H%M%S')
    local report_file="$output_dir/log-report-$service-$timestamp.html"
    
    log_header "生成 $service 服务日志报告"
    
    mkdir -p "$output_dir"
    
    cd "$PROJECT_ROOT"
    
    # 生成 HTML 报告
    cat > "$report_file" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>SWIT 分布式追踪日志报告 - $service</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .header { background: #f0f0f0; padding: 20px; border-radius: 5px; }
        .section { margin: 20px 0; }
        .error { color: red; }
        .warning { color: orange; }
        .info { color: blue; }
        .success { color: green; }
        pre { background: #f8f8f8; padding: 10px; overflow-x: auto; }
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <div class="header">
        <h1>🔍 SWIT 分布式追踪日志报告</h1>
        <p><strong>服务:</strong> $service</p>
        <p><strong>时间范围:</strong> 最近 $hours 小时</p>
        <p><strong>生成时间:</strong> $(date)</p>
    </div>

    <div class="section">
        <h2>📊 统计摘要</h2>
EOF
    
    # 获取日志并分析
    local temp_file=$(mktemp)
    if [ "$service" = "all" ]; then
        if command -v docker-compose &> /dev/null; then
            docker-compose logs --since="${hours}h" > "$temp_file"
        else
            docker compose logs --since="${hours}h" > "$temp_file"
        fi
    else
        if command -v docker-compose &> /dev/null; then
            docker-compose logs --since="${hours}h" "$service" > "$temp_file"
        else
            docker compose logs --since="${hours}h" "$service" > "$temp_file"
        fi
    fi
    
    # 添加统计信息到报告
    local total_lines=$(wc -l < "$temp_file")
    local error_count=$(grep -ci "error\|exception\|panic\|fatal" "$temp_file" || echo "0")
    local warning_count=$(grep -ci "warning\|warn" "$temp_file" || echo "0")
    local info_count=$(grep -ci "info" "$temp_file" || echo "0")
    
    cat >> "$report_file" << EOF
        <table>
            <tr><th>指标</th><th>数值</th></tr>
            <tr><td>总日志行数</td><td>$total_lines</td></tr>
            <tr><td class="error">错误日志数</td><td>$error_count</td></tr>
            <tr><td class="warning">警告日志数</td><td>$warning_count</td></tr>
            <tr><td class="info">信息日志数</td><td>$info_count</td></tr>
        </table>
    </div>

    <div class="section">
        <h2>❌ 最近错误日志</h2>
        <pre>
EOF
    
    # 添加错误日志
    grep -i "error\|exception\|panic\|fatal" "$temp_file" | tail -20 | \
        sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g' >> "$report_file" || echo "无错误日志" >> "$report_file"
    
    cat >> "$report_file" << EOF
        </pre>
    </div>

    <div class="section">
        <h2>⚠️ 最近警告日志</h2>
        <pre>
EOF
    
    # 添加警告日志
    grep -i "warning\|warn" "$temp_file" | tail -20 | \
        sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g' >> "$report_file" || echo "无警告日志" >> "$report_file"
    
    cat >> "$report_file" << EOF
        </pre>
    </div>

    <div class="section">
        <h2>🔍 完整日志 (最后100行)</h2>
        <pre>
EOF
    
    # 添加最后100行日志
    tail -100 "$temp_file" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g' >> "$report_file"
    
    cat >> "$report_file" << EOF
        </pre>
    </div>

    <div class="section">
        <p><em>报告生成时间: $(date)</em></p>
        <p><em>SWIT 框架分布式追踪演示环境</em></p>
    </div>
</body>
</html>
EOF
    
    rm -f "$temp_file"
    
    log_success "日志报告已生成: $report_file"
    
    # 尝试在浏览器中打开报告
    if command -v open &> /dev/null; then  # macOS
        open "$report_file"
    elif command -v xdg-open &> /dev/null; then  # Linux
        xdg-open "$report_file"
    else
        log_info "请在浏览器中打开: $report_file"
    fi
}

# 显示使用帮助
show_help() {
    echo "用法: $0 [服务名] [选项]"
    echo ""
    echo "服务名:"
    echo "  jaeger            Jaeger 追踪后端日志"
    echo "  order-service     订单服务日志"
    echo "  payment-service   支付服务日志"
    echo "  inventory-service 库存服务日志"
    echo "  all               所有服务日志 (默认)"
    echo ""
    echo "选项:"
    echo "  --follow, -f      实时跟踪日志输出"
    echo "  --tail, -n        显示最后N行 (默认: 100)"
    echo "  --save            保存日志到文件"
    echo "  --errors-only     仅显示错误和警告日志"
    echo "  --search          搜索日志内容"
    echo "  --analyze         分析日志统计"
    echo "  --monitor-errors  实时监控错误日志"
    echo "  --export-report   导出HTML格式日志报告"
    echo "  --help, -h        显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0                           # 查看所有服务日志"
    echo "  $0 order-service             # 查看订单服务日志"
    echo "  $0 all --follow              # 实时跟踪所有服务日志"
    echo "  $0 --errors-only             # 仅显示错误日志"
    echo "  $0 --save logs/today.log     # 保存日志到文件"
    echo "  $0 --search \"error 404\"      # 搜索特定内容"
    echo "  $0 --analyze order-service   # 分析订单服务日志"
    echo "  $0 --monitor-errors          # 监控实时错误"
    echo "  $0 --export-report           # 导出日志报告"
    echo ""
}

# 主函数
main() {
    local service="all"
    local follow=false
    local tail_lines=100
    local save_file=""
    local filter_errors=false
    local search_term=""
    local analyze=false
    local monitor_errors=false
    local export_report=false
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            jaeger|order-service|payment-service|inventory-service|all)
                service="$1"
                shift
                ;;
            --follow|-f)
                follow=true
                shift
                ;;
            --tail|-n)
                tail_lines="$2"
                shift 2
                ;;
            --save)
                save_file="$2"
                shift 2
                ;;
            --errors-only)
                filter_errors=true
                shift
                ;;
            --search)
                search_term="$2"
                shift 2
                ;;
            --analyze)
                analyze=true
                if [[ $2 =~ ^[0-9]+$ ]]; then
                    analyze_hours="$2"
                    shift 2
                else
                    analyze_hours=1
                    shift
                fi
                ;;
            --monitor-errors)
                monitor_errors=true
                shift
                ;;
            --export-report)
                export_report=true
                if [[ $2 =~ ^[0-9]+$ ]]; then
                    export_hours="$2"
                    shift 2
                else
                    export_hours=24
                    shift
                fi
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
    
    # 切换到项目根目录
    cd "$PROJECT_ROOT"
    
    # 检查 Docker 服务
    check_docker_services
    
    # 根据选项执行相应操作
    if [ "$search_term" != "" ]; then
        search_logs "$service" "$search_term"
    elif [ "$analyze" = true ]; then
        analyze_logs "$service" "${analyze_hours:-1}"
    elif [ "$monitor_errors" = true ]; then
        monitor_errors "$service"
    elif [ "$export_report" = true ]; then
        export_log_report "$service" "${export_hours:-24}"
    else
        # 标准日志查看
        if [ "$service" = "all" ]; then
            view_all_logs "$follow" "$tail_lines" "$save_file" "$filter_errors"
        else
            view_service_logs "$service" "$follow" "$tail_lines" "$save_file" "$filter_errors"
        fi
    fi
}

# 信号处理
cleanup_on_exit() {
    log_info "接收到中断信号，清理并退出..."
    exit 0
}

trap cleanup_on_exit SIGINT SIGTERM

# 执行主函数
main "$@"