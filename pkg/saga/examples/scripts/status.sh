#!/usr/bin/env bash

# Saga 示例应用状态监控工具
# 用于查看示例应用的运行状态、资源使用情况和健康检查
set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color

# 项目根目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"
EXAMPLES_DIR="$PROJECT_ROOT/pkg/saga/examples"

# 配置变量
WATCH_MODE=false
REFRESH_INTERVAL=2
DETAILED=false
JSON_OUTPUT=false
QUIET=false

# 函数：打印日志
log_info() {
    [[ "$QUIET" == true ]] && return
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    [[ "$QUIET" == true ]] && return
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    [[ "$QUIET" == true ]] && return
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

log_step() {
    [[ "$QUIET" == true ]] && return
    echo -e "${CYAN}==>${NC} $1"
}

# 函数：显示帮助信息
show_help() {
    cat << EOF
Saga 示例应用状态监控工具

用法:
    $0 [选项]

选项:
    -h, --help              显示此帮助信息
    -w, --watch             监控模式（持续刷新）
    -i, --interval SECONDS  刷新间隔（默认: 2秒）
    -d, --detailed          显示详细信息
    -j, --json              以 JSON 格式输出
    -q, --quiet             安静模式（仅输出状态数据）

监控内容:
    - 运行中的进程
    - 测试执行状态
    - 文件系统状态
    - 覆盖率统计
    - 最近的错误

示例:
    $0                              # 显示当前状态
    $0 --watch                      # 持续监控（2秒刷新）
    $0 --watch --interval 5         # 持续监控（5秒刷新）
    $0 --detailed                   # 显示详细信息
    $0 --json                       # JSON 格式输出
    $0 --quiet                      # 仅输出关键状态

EOF
}

# 函数：检查进程状态
check_processes() {
    local pids=$(pgrep -f "go.*test.*saga/examples" 2>/dev/null || true)
    local count=0
    local status="stopped"
    
    if [[ -n "$pids" ]]; then
        count=$(echo "$pids" | wc -l | tr -d ' ')
        status="running"
    fi
    
    if [[ "$JSON_OUTPUT" == true ]]; then
        echo "\"processes\": {"
        echo "  \"status\": \"$status\","
        echo "  \"count\": $count,"
        echo "  \"pids\": ["
        if [[ -n "$pids" ]]; then
            echo "$pids" | awk '{printf "    %s%s", $0, (NR==FNR ? "" : ",")} END {print ""}'
        fi
        echo "  ]"
        echo "},"
        return
    fi
    
    if [[ "$count" -eq 0 ]]; then
        echo "  ${YELLOW}●${NC} 进程状态: 无运行中的进程"
    else
        echo "  ${GREEN}●${NC} 进程状态: $count 个进程正在运行"
        
        if [[ "$DETAILED" == true ]]; then
            echo "$pids" | while read -r pid; do
                if ps -p "$pid" > /dev/null 2>&1; then
                    local cpu=$(ps -p "$pid" -o %cpu= 2>/dev/null | tr -d ' ' || echo "N/A")
                    local mem=$(ps -p "$pid" -o %mem= 2>/dev/null | tr -d ' ' || echo "N/A")
                    local time=$(ps -p "$pid" -o etime= 2>/dev/null | tr -d ' ' || echo "N/A")
                    local cmd=$(ps -p "$pid" -o args= 2>/dev/null | cut -c1-60 || echo "未知")
                    echo "      PID $pid: CPU ${cpu}%, MEM ${mem}%, 运行时间 $time"
                    echo "      命令: $cmd"
                fi
            done
        fi
    fi
}

# 函数：检查文件系统状态
check_filesystem() {
    cd "$EXAMPLES_DIR"
    
    # 统计文件
    local test_files=$(find . -maxdepth 1 -name "*_test.go" -type f 2>/dev/null | wc -l | tr -d ' ')
    local example_files=$(find . -maxdepth 1 -name "*.go" ! -name "*_test.go" -type f 2>/dev/null | wc -l | tr -d ' ')
    local coverage_files=$(find . -maxdepth 1 -name "coverage_*.out" -type f 2>/dev/null | wc -l | tr -d ' ')
    local log_files=$(find . -maxdepth 1 \( -name "*.log" -o -name "test_*.txt" \) -type f 2>/dev/null | wc -l | tr -d ' ')
    
    # 计算总大小
    local total_size=$(du -sh . 2>/dev/null | cut -f1)
    
    if [[ "$JSON_OUTPUT" == true ]]; then
        echo "\"filesystem\": {"
        echo "  \"test_files\": $test_files,"
        echo "  \"example_files\": $example_files,"
        echo "  \"coverage_files\": $coverage_files,"
        echo "  \"log_files\": $log_files,"
        echo "  \"total_size\": \"$total_size\""
        echo "},"
        return
    fi
    
    echo "  ${BLUE}●${NC} 文件系统状态:"
    echo "      示例文件: $example_files"
    echo "      测试文件: $test_files"
    echo "      覆盖率文件: $coverage_files"
    echo "      日志文件: $log_files"
    echo "      总大小: $total_size"
}

# 函数：检查覆盖率状态
check_coverage() {
    cd "$EXAMPLES_DIR"
    
    # 查找最新的覆盖率文件
    local latest_coverage=$(find . -maxdepth 1 -name "coverage_*.out" -type f -exec stat -f "%m %N" {} \; 2>/dev/null | sort -rn | head -1 | cut -d' ' -f2- || true)
    
    if [[ -z "$latest_coverage" ]]; then
        if [[ "$JSON_OUTPUT" == true ]]; then
            echo "\"coverage\": {"
            echo "  \"available\": false"
            echo "},"
        else
            echo "  ${YELLOW}●${NC} 覆盖率: 无可用数据"
        fi
        return
    fi
    
    # 提取覆盖率百分比
    local coverage_pct=$(go tool cover -func="$latest_coverage" 2>/dev/null | tail -1 | awk '{print $3}' || echo "N/A")
    local file_name=$(basename "$latest_coverage")
    local file_time=$(stat -f "%Sm" -t "%Y-%m-%d %H:%M:%S" "$latest_coverage" 2>/dev/null || stat -c "%y" "$latest_coverage" 2>/dev/null | cut -d'.' -f1)
    
    if [[ "$JSON_OUTPUT" == true ]]; then
        echo "\"coverage\": {"
        echo "  \"available\": true,"
        echo "  \"percentage\": \"$coverage_pct\","
        echo "  \"file\": \"$file_name\","
        echo "  \"timestamp\": \"$file_time\""
        echo "},"
        return
    fi
    
    # 根据覆盖率显示不同颜色
    local color=$GREEN
    local pct_num=$(echo "$coverage_pct" | sed 's/%//')
    if [[ "$pct_num" =~ ^[0-9.]+$ ]]; then
        if (( $(echo "$pct_num < 50" | bc -l) )); then
            color=$RED
        elif (( $(echo "$pct_num < 80" | bc -l) )); then
            color=$YELLOW
        fi
    fi
    
    echo "  ${color}●${NC} 覆盖率: $coverage_pct"
    
    if [[ "$DETAILED" == true ]]; then
        echo "      文件: $file_name"
        echo "      时间: $file_time"
        echo ""
        echo "      前5个函数覆盖率:"
        go tool cover -func="$latest_coverage" 2>/dev/null | head -6 | tail -5 | while read -r line; do
            echo "        $line"
        done
    fi
}

# 函数：检查最近的测试结果
check_test_results() {
    cd "$EXAMPLES_DIR"
    
    # 运行快速测试检查
    local test_output=$(go test -v -timeout 5s 2>&1 || true)
    local test_status="unknown"
    local pass_count=0
    local fail_count=0
    
    if echo "$test_output" | grep -q "PASS"; then
        test_status="pass"
    elif echo "$test_output" | grep -q "FAIL"; then
        test_status="fail"
    fi
    
    pass_count=$(echo "$test_output" | grep -c "PASS:" || echo "0")
    fail_count=$(echo "$test_output" | grep -c "FAIL:" || echo "0")
    
    if [[ "$JSON_OUTPUT" == true ]]; then
        echo "\"test_results\": {"
        echo "  \"status\": \"$test_status\","
        echo "  \"pass_count\": $pass_count,"
        echo "  \"fail_count\": $fail_count"
        echo "},"
        return
    fi
    
    case $test_status in
        "pass")
            echo "  ${GREEN}●${NC} 测试状态: 通过"
            ;;
        "fail")
            echo "  ${RED}●${NC} 测试状态: 失败"
            ;;
        *)
            echo "  ${YELLOW}●${NC} 测试状态: 未知"
            ;;
    esac
    
    if [[ "$DETAILED" == true ]]; then
        echo "      通过: $pass_count"
        echo "      失败: $fail_count"
    fi
}

# 函数：检查最近的错误
check_recent_errors() {
    cd "$EXAMPLES_DIR"
    
    # 从最近的测试输出中提取错误
    local errors=$(go test -v 2>&1 | grep -iE "(error|fail)" | head -5 || true)
    local error_count=$(echo "$errors" | grep -c "." || echo "0")
    
    if [[ "$JSON_OUTPUT" == true ]]; then
        echo "\"recent_errors\": {"
        echo "  \"count\": $error_count,"
        echo "  \"errors\": ["
        if [[ -n "$errors" ]]; then
            echo "$errors" | jq -Rs 'split("\n")[:-1] | map({"message": .})' 2>/dev/null || echo "[]"
        else
            echo "  ]"
        fi
        echo "}"
        return
    fi
    
    if [[ $error_count -eq 0 ]]; then
        echo "  ${GREEN}●${NC} 最近错误: 无"
    else
        echo "  ${RED}●${NC} 最近错误: $error_count 个"
        
        if [[ "$DETAILED" == true ]]; then
            echo "$errors" | head -3 | while read -r line; do
                echo "      $(echo "$line" | cut -c1-80)"
            done
        fi
    fi
}

# 函数：检查系统资源
check_system_resources() {
    # CPU 使用率
    local cpu_usage=$(top -l 1 | grep "CPU usage" | awk '{print $3}' || echo "N/A")
    
    # 内存使用
    local mem_info=$(vm_stat 2>/dev/null || free -h 2>/dev/null | grep "Mem:" || echo "N/A")
    
    # 磁盘空间
    local disk_usage=$(df -h "$EXAMPLES_DIR" | tail -1 | awk '{print $5}' || echo "N/A")
    
    if [[ "$JSON_OUTPUT" == true ]]; then
        echo "\"system\": {"
        echo "  \"cpu_usage\": \"$cpu_usage\","
        echo "  \"disk_usage\": \"$disk_usage\""
        echo "}"
        return
    fi
    
    echo "  ${CYAN}●${NC} 系统资源:"
    echo "      CPU 使用率: $cpu_usage"
    echo "      磁盘使用: $disk_usage"
}

# 函数：显示完整状态
show_status() {
    if [[ "$JSON_OUTPUT" == true ]]; then
        echo "{"
        echo "\"timestamp\": \"$(date -u '+%Y-%m-%dT%H:%M:%SZ')\","
        check_processes
        check_filesystem
        check_coverage
        check_test_results
        check_recent_errors
        check_system_resources
        echo "}"
        return
    fi
    
    # 显示横幅（仅在非 watch 模式下）
    if [[ "$WATCH_MODE" == false ]]; then
        echo ""
        echo "========================================"
        echo "  Saga 示例应用 - 状态监控"
        echo "========================================"
        echo ""
        echo "时间: $(date '+%Y-%m-%d %H:%M:%S')"
        echo ""
    fi
    
    log_step "运行状态"
    check_processes
    echo ""
    
    log_step "文件系统"
    check_filesystem
    echo ""
    
    log_step "测试覆盖率"
    check_coverage
    echo ""
    
    log_step "测试结果"
    check_test_results
    echo ""
    
    log_step "最近错误"
    check_recent_errors
    echo ""
    
    if [[ "$DETAILED" == true ]]; then
        log_step "系统资源"
        check_system_resources
        echo ""
    fi
}

# 函数：监控模式
watch_status() {
    log_info "监控模式启动 (每 ${REFRESH_INTERVAL}s 刷新)"
    log_info "按 Ctrl+C 退出"
    echo ""
    
    while true; do
        # 清屏
        clear
        
        # 显示标题
        echo "========================================"
        echo "  Saga 示例应用 - 实时监控"
        echo "========================================"
        echo ""
        echo "时间: $(date '+%Y-%m-%d %H:%M:%S')"
        echo "刷新间隔: ${REFRESH_INTERVAL}s"
        echo ""
        
        # 显示状态
        show_status
        
        # 等待
        sleep "$REFRESH_INTERVAL"
    done
}

# 函数：健康检查
health_check() {
    local healthy=true
    local issues=()
    
    # 检查目录是否存在
    if [[ ! -d "$EXAMPLES_DIR" ]]; then
        healthy=false
        issues+=("示例目录不存在")
    fi
    
    # 检查是否有测试文件
    cd "$EXAMPLES_DIR"
    local test_count=$(find . -maxdepth 1 -name "*_test.go" -type f 2>/dev/null | wc -l | tr -d ' ')
    if [[ $test_count -eq 0 ]]; then
        healthy=false
        issues+=("没有找到测试文件")
    fi
    
    # 检查 Go 环境
    if ! command -v go &> /dev/null; then
        healthy=false
        issues+=("Go 未安装")
    fi
    
    if [[ "$JSON_OUTPUT" == true ]]; then
        echo "{"
        echo "  \"healthy\": $healthy,"
        echo "  \"issues\": ["
        for issue in "${issues[@]}"; do
            echo "    \"$issue\","
        done | sed '$ s/,$//'
        echo "  ]"
        echo "}"
        return
    fi
    
    if [[ "$healthy" == true ]]; then
        log_success "健康检查: 通过"
        return 0
    else
        log_error "健康检查: 失败"
        echo ""
        log_error "发现以下问题:"
        for issue in "${issues[@]}"; do
            echo "  - $issue"
        done
        return 1
    fi
}

# 主函数
main() {
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -w|--watch)
                WATCH_MODE=true
                shift
                ;;
            -i|--interval)
                REFRESH_INTERVAL="$2"
                shift 2
                ;;
            -d|--detailed)
                DETAILED=true
                shift
                ;;
            -j|--json)
                JSON_OUTPUT=true
                shift
                ;;
            -q|--quiet)
                QUIET=true
                shift
                ;;
            --health)
                health_check
                exit $?
                ;;
            -*)
                log_error "未知选项: $1"
                echo ""
                show_help
                exit 1
                ;;
            *)
                log_error "不需要位置参数: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    # 检查示例目录
    if [[ ! -d "$EXAMPLES_DIR" ]]; then
        log_error "示例目录不存在: $EXAMPLES_DIR"
        exit 1
    fi
    
    # 执行监控
    if [[ "$WATCH_MODE" == true ]]; then
        watch_status
    else
        show_status
        
        if [[ "$QUIET" == false ]]; then
            log_success "完成！"
        fi
    fi
}

# 执行主函数
main "$@"

