#!/bin/bash

# 分布式追踪数据分析脚本
# Copyright (c) 2024 SWIT Framework Authors

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CONFIG_FILE="$PROJECT_ROOT/scripts/config/demo-config.env"
OUTPUT_DIR="$PROJECT_ROOT/reports"

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
    echo -e "${PURPLE}[ANALYSIS]${NC} $1"
}

# 显示欢迎信息
show_welcome() {
    echo "======================================================================"
    echo -e "${BLUE}📊 SWIT 框架分布式追踪数据分析工具${NC}"
    echo "======================================================================"
    echo -e "${YELLOW}本工具将分析 Jaeger 中的追踪数据并生成详细报告：${NC}"
    echo "  📈 性能指标分析"
    echo "  🔍 错误模式识别"
    echo "  📊 服务依赖关系"
    echo "  📝 可视化报告生成"
    echo
}

# 显示使用说明
show_usage() {
    echo "Usage: $0 [options]"
    echo
    echo "Options:"
    echo "  -h, --help              显示帮助信息"
    echo "  -d, --duration=TIME     分析时间范围 (默认: 1h)"
    echo "                          格式: 1h, 2h, 30m, 1d 等"
    echo "  -s, --service=NAME      分析特定服务 (可多次使用)"
    echo "  -o, --output=DIR        输出目录 (默认: ./reports)"
    echo "  -f, --format=FORMAT     报告格式 (json|html|text|all, 默认: html)"
    echo "  -t, --top=N             显示 top N 个结果 (默认: 10)"
    echo "  --min-duration=MS       最小追踪持续时间过滤 (毫秒, 默认: 0)"
    echo "  --max-duration=MS       最大追踪持续时间过滤 (毫秒)"
    echo "  --include-errors        只分析包含错误的追踪"
    echo "  --exclude-health        排除健康检查相关的追踪"
    echo "  --jaeger-url=URL        Jaeger 查询 API 地址"
    echo "  --save-raw              保存原始追踪数据"
    echo "  --verbose               详细输出"
    echo
    echo "Time Duration Examples:"
    echo "  1h, 2h30m, 90m, 1d, 30s"
    echo
    echo "Service Names:"
    echo "  order-service, payment-service, inventory-service"
    echo
    echo "Examples:"
    echo "  $0                                  # 分析最近1小时的所有数据"
    echo "  $0 -d 2h -s order-service         # 分析最近2小时的订单服务数据"
    echo "  $0 -d 30m --include-errors        # 分析最近30分钟的错误追踪"
    echo "  $0 -f all -o ./my-reports          # 生成所有格式的报告到指定目录"
    echo "  $0 --min-duration=100 --top=20    # 分析持续时间>100ms的前20个追踪"
}

# 加载配置
load_config() {
    if [ -f "$CONFIG_FILE" ]; then
        log_info "正在加载配置文件: $CONFIG_FILE"
        source "$CONFIG_FILE"
    else
        log_warning "配置文件不存在: $CONFIG_FILE，使用默认配置"
    fi
    
    # 默认配置
    JAEGER_QUERY_URL=${JAEGER_QUERY_URL:-"http://localhost:16686"}
    JAEGER_API_URL="$JAEGER_QUERY_URL/api"
}

# 检查依赖
check_dependencies() {
    log_header "检查依赖工具"
    
    local missing_tools=()
    
    # 检查必需工具
    if ! command -v curl &> /dev/null; then
        missing_tools+=("curl")
    fi
    
    if ! command -v jq &> /dev/null; then
        missing_tools+=("jq")
    fi
    
    # 检查可选工具
    local optional_missing=()
    if ! command -v python3 &> /dev/null; then
        optional_missing+=("python3")
    fi
    
    if ! command -v node &> /dev/null; then
        optional_missing+=("node")
    fi
    
    if [ ${#missing_tools[@]} -gt 0 ]; then
        log_error "缺少必需工具: ${missing_tools[*]}"
        echo "安装方法："
        echo "  Ubuntu/Debian: sudo apt-get install curl jq"
        echo "  macOS: brew install curl jq"
        echo "  CentOS/RHEL: sudo yum install curl jq"
        exit 1
    fi
    
    if [ ${#optional_missing[@]} -gt 0 ]; then
        log_warning "缺少可选工具: ${optional_missing[*]}"
        log_info "这些工具用于高级报告生成，基础功能不受影响"
    fi
    
    log_success "依赖检查完成"
}

# 检查 Jaeger 连接
check_jaeger_connection() {
    log_header "检查 Jaeger 连接"
    
    local health_url="$JAEGER_API_URL/health"
    
    if ! curl -f -s --connect-timeout 10 "$health_url" > /dev/null 2>&1; then
        log_error "无法连接到 Jaeger: $JAEGER_QUERY_URL"
        log_info "请确保："
        log_info "  1. Jaeger 服务正在运行"
        log_info "  2. URL 地址正确: $JAEGER_QUERY_URL"
        log_info "  3. 网络连接正常"
        echo
        log_info "尝试启动服务: ./scripts/start.sh"
        exit 1
    fi
    
    log_success "Jaeger 连接正常: $JAEGER_QUERY_URL"
}

# 获取服务列表
get_services() {
    log_info "获取可用服务列表..."
    
    local services_url="$JAEGER_API_URL/services"
    local services_json
    
    if ! services_json=$(curl -f -s "$services_url" 2>/dev/null); then
        log_error "获取服务列表失败"
        return 1
    fi
    
    local services
    services=$(echo "$services_json" | jq -r '.data[]' 2>/dev/null | sort)
    
    if [ -z "$services" ]; then
        log_warning "未找到任何服务数据"
        return 1
    fi
    
    echo "$services"
}

# 时间转换为微秒时间戳
parse_duration_to_microseconds() {
    local duration="$1"
    local now_us=$(date +%s%6N)
    local duration_us=0
    
    # 解析时间格式 (1h, 30m, 90s, 1d 等)
    if [[ $duration =~ ^([0-9]+)([hmsd])$ ]]; then
        local value="${BASH_REMATCH[1]}"
        local unit="${BASH_REMATCH[2]}"
        
        case $unit in
            s) duration_us=$((value * 1000000)) ;;
            m) duration_us=$((value * 60 * 1000000)) ;;
            h) duration_us=$((value * 3600 * 1000000)) ;;
            d) duration_us=$((value * 86400 * 1000000)) ;;
        esac
    elif [[ $duration =~ ^([0-9]+)([hm])([0-9]+)([ms])$ ]]; then
        # 处理 2h30m 这样的格式
        local h_value="${BASH_REMATCH[1]}"
        local h_unit="${BASH_REMATCH[2]}"
        local m_value="${BASH_REMATCH[3]}"
        local m_unit="${BASH_REMATCH[4]}"
        
        local h_us=0
        local m_us=0
        
        if [ "$h_unit" = "h" ]; then
            h_us=$((h_value * 3600 * 1000000))
        fi
        
        if [ "$m_unit" = "m" ]; then
            m_us=$((m_value * 60 * 1000000))
        elif [ "$m_unit" = "s" ]; then
            m_us=$((m_value * 1000000))
        fi
        
        duration_us=$((h_us + m_us))
    else
        log_error "无法解析时间格式: $duration"
        log_info "支持的格式: 1h, 30m, 90s, 1d, 2h30m"
        return 1
    fi
    
    local start_time_us=$((now_us - duration_us))
    echo "$start_time_us $now_us"
}

# 查询追踪数据
query_traces() {
    log_header "查询追踪数据"
    
    local time_range
    if ! time_range=$(parse_duration_to_microseconds "$DURATION"); then
        return 1
    fi
    
    local start_time end_time
    read -r start_time end_time <<< "$time_range"
    
    log_info "时间范围: $DURATION ($(date -d @$((start_time / 1000000)) '+%Y-%m-%d %H:%M:%S') - $(date -d @$((end_time / 1000000)) '+%Y-%m-%d %H:%M:%S'))"
    
    # 构建查询参数
    local query_params="start=$start_time&end=$end_time&limit=1000"
    
    # 添加服务过滤
    if [ ${#TARGET_SERVICES[@]} -gt 0 ]; then
        for service in "${TARGET_SERVICES[@]}"; do
            query_params="$query_params&service=$service"
        done
        log_info "分析服务: ${TARGET_SERVICES[*]}"
    fi
    
    # 添加持续时间过滤
    if [ -n "$MIN_DURATION" ]; then
        query_params="$query_params&minDuration=${MIN_DURATION}ms"
        log_info "最小持续时间: ${MIN_DURATION}ms"
    fi
    
    if [ -n "$MAX_DURATION" ]; then
        query_params="$query_params&maxDuration=${MAX_DURATION}ms"
        log_info "最大持续时间: ${MAX_DURATION}ms"
    fi
    
    # 错误过滤
    if [ "$INCLUDE_ERRORS_ONLY" = "true" ]; then
        query_params="$query_params&tags={\"error\":\"true\"}"
        log_info "只包含错误追踪"
    fi
    
    local traces_url="$JAEGER_API_URL/traces?$query_params"
    
    if [ "$VERBOSE" = "true" ]; then
        log_info "查询URL: $traces_url"
    fi
    
    log_info "正在获取追踪数据..."
    local traces_json
    if ! traces_json=$(curl -f -s "$traces_url" 2>/dev/null); then
        log_error "获取追踪数据失败"
        return 1
    fi
    
    # 保存原始数据
    if [ "$SAVE_RAW_DATA" = "true" ]; then
        local raw_file="$OUTPUT_DIR/raw_traces_$(date +%Y%m%d_%H%M%S).json"
        echo "$traces_json" > "$raw_file"
        log_info "原始数据已保存: $raw_file"
    fi
    
    local trace_count
    trace_count=$(echo "$traces_json" | jq '.data | length' 2>/dev/null)
    
    if [ "$trace_count" = "0" ] || [ -z "$trace_count" ]; then
        log_warning "未找到匹配的追踪数据"
        return 1
    fi
    
    log_success "找到 $trace_count 条追踪数据"
    echo "$traces_json"
}

# 分析性能指标
analyze_performance() {
    local traces_json="$1"
    log_header "分析性能指标"
    
    # 使用 jq 进行复杂的 JSON 分析
    local analysis_script=$(cat << 'EOF'
def calculate_duration($spans):
    ($spans | map(.duration) | add) // 0;

def extract_service_stats($spans):
    ($spans | group_by(.process.serviceName) | 
    map({
        service: .[0].process.serviceName,
        count: length,
        total_duration: map(.duration) | add,
        avg_duration: (map(.duration) | add) / length,
        min_duration: map(.duration) | min,
        max_duration: map(.duration) | max,
        errors: map(select(.tags[]? | select(.key == "error" and .value == "true"))) | length
    }));

def extract_operation_stats($spans):
    ($spans | group_by(.operationName) |
    map({
        operation: .[0].operationName,
        count: length,
        avg_duration: (map(.duration) | add) / length,
        errors: map(select(.tags[]? | select(.key == "error" and .value == "true"))) | length
    }) | sort_by(.avg_duration) | reverse);

{
    total_traces: (.data | length),
    total_spans: [.data[].spans[]] | length,
    avg_trace_duration: ([.data[] | [.spans[].duration] | add] | add) / (.data | length),
    service_stats: [.data[].spans[]] | extract_service_stats(.),
    operation_stats: [.data[].spans[]] | extract_operation_stats(.),
    error_traces: [.data[] | select(.spans[]?.tags[]? | select(.key == "error" and .value == "true"))] | length
}
EOF
    )
    
    local performance_stats
    performance_stats=$(echo "$traces_json" | jq "$analysis_script" 2>/dev/null)
    
    if [ -z "$performance_stats" ] || [ "$performance_stats" = "null" ]; then
        log_error "性能分析失败"
        return 1
    fi
    
    echo "$performance_stats"
}

# 生成文本报告
generate_text_report() {
    local stats="$1"
    local output_file="$2"
    
    log_info "生成文本报告..."
    
    cat > "$output_file" << EOF
# 分布式追踪性能分析报告
生成时间: $(date '+%Y-%m-%d %H:%M:%S')
分析持续时间: $DURATION
输出目录: $OUTPUT_DIR

## 总体统计
$(echo "$stats" | jq -r '
"- 总追踪数量: " + (.total_traces | tostring) + "\n" +
"- 总Span数量: " + (.total_spans | tostring) + "\n" +  
"- 平均追踪持续时间: " + ((.avg_trace_duration / 1000) | floor | tostring) + "ms\n" +
"- 错误追踪数量: " + (.error_traces | tostring)
')

## 服务性能统计
$(echo "$stats" | jq -r '
.service_stats[] | 
"### " + .service + "\n" +
"- 调用次数: " + (.count | tostring) + "\n" +
"- 平均响应时间: " + ((.avg_duration / 1000) | floor | tostring) + "ms\n" +
"- 最小响应时间: " + ((.min_duration / 1000) | floor | tostring) + "ms\n" +
"- 最大响应时间: " + ((.max_duration / 1000) | floor | tostring) + "ms\n" +
"- 错误次数: " + (.errors | tostring) + "\n" +
"- 错误率: " + (((.errors / .count) * 100) | floor | tostring) + "%\n"
')

## Top $(echo "$TOP_N") 最慢操作
$(echo "$stats" | jq -r --arg top "$TOP_N" '
.operation_stats[:($top | tonumber)] | 
.[] | 
"- " + .operation + ": " + ((.avg_duration / 1000) | floor | tostring) + "ms (调用" + (.count | tostring) + "次)"
')

EOF
    
    log_success "文本报告已生成: $output_file"
}

# 生成 JSON 报告
generate_json_report() {
    local stats="$1"
    local output_file="$2"
    
    log_info "生成 JSON 报告..."
    
    local report_data=$(cat << EOF
{
    "metadata": {
        "generated_at": "$(date -u '+%Y-%m-%dT%H:%M:%SZ')",
        "duration": "$DURATION",
        "jaeger_url": "$JAEGER_QUERY_URL",
        "analysis_params": {
            "services": $(printf '%s\n' "${TARGET_SERVICES[@]}" | jq -R . | jq -s .),
            "min_duration": "$MIN_DURATION",
            "max_duration": "$MAX_DURATION",
            "include_errors_only": $INCLUDE_ERRORS_ONLY,
            "exclude_health": $EXCLUDE_HEALTH
        }
    },
    "statistics": $stats
}
EOF
    )
    
    echo "$report_data" | jq . > "$output_file"
    log_success "JSON 报告已生成: $output_file"
}

# 生成 HTML 报告
generate_html_report() {
    local stats="$1"
    local output_file="$2"
    
    log_info "生成 HTML 报告..."
    
    # 提取统计数据用于图表
    local total_traces=$(echo "$stats" | jq -r '.total_traces')
    local total_spans=$(echo "$stats" | jq -r '.total_spans')
    local avg_duration=$(echo "$stats" | jq -r '(.avg_trace_duration / 1000) | floor')
    local error_traces=$(echo "$stats" | jq -r '.error_traces')
    
    # 生成服务性能数据
    local service_chart_data=$(echo "$stats" | jq -r '.service_stats | map({name: .service, avg_duration: (.avg_duration / 1000 | floor), count: .count, errors: .errors}) | @json')
    
    # 生成操作性能数据 
    local operation_chart_data=$(echo "$stats" | jq -r --arg top "$TOP_N" '.operation_stats[:($top | tonumber)] | map({name: .operation, avg_duration: (.avg_duration / 1000 | floor), count: .count}) | @json')
    
    cat > "$output_file" << EOF
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>分布式追踪性能分析报告</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; background-color: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; border-radius: 10px; margin-bottom: 30px; text-align: center; }
        .header h1 { font-size: 2.5em; margin-bottom: 10px; }
        .header p { font-size: 1.1em; opacity: 0.9; }
        .stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; margin-bottom: 30px; }
        .stat-card { background: white; padding: 20px; border-radius: 10px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); text-align: center; }
        .stat-card h3 { color: #666; font-size: 0.9em; text-transform: uppercase; letter-spacing: 1px; margin-bottom: 10px; }
        .stat-card .number { font-size: 2em; font-weight: bold; color: #4CAF50; }
        .stat-card .error { color: #f44336; }
        .chart-section { background: white; padding: 30px; border-radius: 10px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); margin-bottom: 30px; }
        .chart-section h2 { margin-bottom: 20px; color: #333; }
        .chart-container { position: relative; height: 400px; margin: 20px 0; }
        .service-table { background: white; border-radius: 10px; overflow: hidden; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .service-table table { width: 100%; border-collapse: collapse; }
        .service-table th, .service-table td { padding: 15px; text-align: left; border-bottom: 1px solid #eee; }
        .service-table th { background-color: #f8f9fa; font-weight: 600; color: #333; }
        .service-table tr:hover { background-color: #f8f9fa; }
        .footer { text-align: center; color: #666; margin-top: 40px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>📊 分布式追踪性能分析报告</h1>
            <p>生成时间: $(date '+%Y-%m-%d %H:%M:%S') | 分析时长: $DURATION</p>
        </div>

        <div class="stats-grid">
            <div class="stat-card">
                <h3>总追踪数量</h3>
                <div class="number">$total_traces</div>
            </div>
            <div class="stat-card">
                <h3>总Span数量</h3>
                <div class="number">$total_spans</div>
            </div>
            <div class="stat-card">
                <h3>平均响应时间</h3>
                <div class="number">${avg_duration}ms</div>
            </div>
            <div class="stat-card">
                <h3>错误追踪</h3>
                <div class="number error">$error_traces</div>
            </div>
        </div>

        <div class="chart-section">
            <h2>服务性能分析</h2>
            <div class="chart-container">
                <canvas id="serviceChart"></canvas>
            </div>
        </div>

        <div class="chart-section">
            <h2>Top $TOP_N 最慢操作</h2>
            <div class="chart-container">
                <canvas id="operationChart"></canvas>
            </div>
        </div>

        <div class="service-table">
            <table>
                <thead>
                    <tr>
                        <th>服务名称</th>
                        <th>调用次数</th>
                        <th>平均响应时间</th>
                        <th>最小响应时间</th>
                        <th>最大响应时间</th>
                        <th>错误次数</th>
                        <th>错误率</th>
                    </tr>
                </thead>
                <tbody>
$(echo "$stats" | jq -r '.service_stats[] | 
"                    <tr>
                        <td>" + .service + "</td>
                        <td>" + (.count | tostring) + "</td>
                        <td>" + ((.avg_duration / 1000) | floor | tostring) + "ms</td>
                        <td>" + ((.min_duration / 1000) | floor | tostring) + "ms</td>
                        <td>" + ((.max_duration / 1000) | floor | tostring) + "ms</td>
                        <td>" + (.errors | tostring) + "</td>
                        <td>" + (((.errors / .count) * 100) | floor | tostring) + "%</td>
                    </tr>"')
                </tbody>
            </table>
        </div>

        <div class="footer">
            <p>由 SWIT 框架分布式追踪分析工具生成</p>
        </div>
    </div>

    <script>
        // 服务性能图表
        const serviceData = $service_chart_data;
        const serviceCtx = document.getElementById('serviceChart').getContext('2d');
        new Chart(serviceCtx, {
            type: 'bar',
            data: {
                labels: serviceData.map(s => s.name),
                datasets: [{
                    label: '平均响应时间 (ms)',
                    data: serviceData.map(s => s.avg_duration),
                    backgroundColor: 'rgba(54, 162, 235, 0.6)',
                    borderColor: 'rgba(54, 162, 235, 1)',
                    borderWidth: 1
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    y: {
                        beginAtZero: true,
                        title: { display: true, text: '响应时间 (ms)' }
                    }
                }
            }
        });

        // 操作性能图表
        const operationData = $operation_chart_data;
        const operationCtx = document.getElementById('operationChart').getContext('2d');
        new Chart(operationCtx, {
            type: 'horizontalBar',
            data: {
                labels: operationData.map(o => o.name.length > 30 ? o.name.substring(0, 30) + '...' : o.name),
                datasets: [{
                    label: '平均响应时间 (ms)',
                    data: operationData.map(o => o.avg_duration),
                    backgroundColor: 'rgba(255, 99, 132, 0.6)',
                    borderColor: 'rgba(255, 99, 132, 1)',
                    borderWidth: 1
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    x: {
                        beginAtZero: true,
                        title: { display: true, text: '响应时间 (ms)' }
                    }
                }
            }
        });
    </script>
</body>
</html>
EOF
    
    log_success "HTML 报告已生成: $output_file"
    log_info "在浏览器中打开: file://$output_file"
}

# 生成报告
generate_reports() {
    local stats="$1"
    
    log_header "生成分析报告"
    
    # 创建输出目录
    mkdir -p "$OUTPUT_DIR"
    
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local base_name="trace_analysis_$timestamp"
    
    case "$OUTPUT_FORMAT" in
        "json")
            generate_json_report "$stats" "$OUTPUT_DIR/${base_name}.json"
            ;;
        "text")
            generate_text_report "$stats" "$OUTPUT_DIR/${base_name}.txt"
            ;;
        "html")
            generate_html_report "$stats" "$OUTPUT_DIR/${base_name}.html"
            ;;
        "all")
            generate_json_report "$stats" "$OUTPUT_DIR/${base_name}.json"
            generate_text_report "$stats" "$OUTPUT_DIR/${base_name}.txt"
            generate_html_report "$stats" "$OUTPUT_DIR/${base_name}.html"
            ;;
    esac
    
    log_success "所有报告已生成完成"
    log_info "报告目录: $OUTPUT_DIR"
}

# 主函数
main() {
    # 默认参数
    DURATION="1h"
    TARGET_SERVICES=()
    OUTPUT_FORMAT="html"
    TOP_N=10
    MIN_DURATION=""
    MAX_DURATION=""
    INCLUDE_ERRORS_ONLY="false"
    EXCLUDE_HEALTH="false"
    SAVE_RAW_DATA="false"
    VERBOSE="false"
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_usage
                exit 0
                ;;
            -d|--duration)
                DURATION="$2"
                shift 2
                ;;
            --duration=*)
                DURATION="${1#*=}"
                shift
                ;;
            -s|--service)
                TARGET_SERVICES+=("$2")
                shift 2
                ;;
            --service=*)
                TARGET_SERVICES+=("${1#*=}")
                shift
                ;;
            -o|--output)
                OUTPUT_DIR="$2"
                shift 2
                ;;
            --output=*)
                OUTPUT_DIR="${1#*=}"
                shift
                ;;
            -f|--format)
                OUTPUT_FORMAT="$2"
                shift 2
                ;;
            --format=*)
                OUTPUT_FORMAT="${1#*=}"
                shift
                ;;
            -t|--top)
                TOP_N="$2"
                shift 2
                ;;
            --top=*)
                TOP_N="${1#*=}"
                shift
                ;;
            --min-duration=*)
                MIN_DURATION="${1#*=}"
                shift
                ;;
            --max-duration=*)
                MAX_DURATION="${1#*=}"
                shift
                ;;
            --include-errors)
                INCLUDE_ERRORS_ONLY="true"
                shift
                ;;
            --exclude-health)
                EXCLUDE_HEALTH="true"
                shift
                ;;
            --jaeger-url=*)
                JAEGER_QUERY_URL="${1#*=}"
                shift
                ;;
            --save-raw)
                SAVE_RAW_DATA="true"
                shift
                ;;
            --verbose)
                VERBOSE="true"
                shift
                ;;
            *)
                log_error "未知选项: $1"
                show_usage
                exit 1
                ;;
        esac
    done
    
    # 验证参数
    if [[ ! "$OUTPUT_FORMAT" =~ ^(json|text|html|all)$ ]]; then
        log_error "无效的输出格式: $OUTPUT_FORMAT"
        log_info "支持的格式: json, text, html, all"
        exit 1
    fi
    
    # 确保输出目录是绝对路径
    OUTPUT_DIR=$(realpath "$OUTPUT_DIR")
    
    show_welcome
    load_config
    check_dependencies
    check_jaeger_connection
    
    # 获取并分析追踪数据
    local traces_json
    if ! traces_json=$(query_traces); then
        log_error "获取追踪数据失败"
        exit 1
    fi
    
    local performance_stats
    if ! performance_stats=$(analyze_performance "$traces_json"); then
        log_error "性能分析失败"
        exit 1
    fi
    
    generate_reports "$performance_stats"
    
    echo
    log_success "分布式追踪数据分析完成！"
}

# 执行主函数
main "$@"