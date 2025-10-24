#!/usr/bin/env bash

# Saga 示例应用日志查看工具
# 用于查看和分析示例应用的运行日志、测试输出和调试信息
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
LOG_TYPE=""
FOLLOW=false
LINES=50
FILTER=""
VERBOSE=false
SHOW_ERRORS_ONLY=false
COLORIZE=true
TIMESTAMP=false

# 可用的日志类型
declare -A LOG_TYPES=(
    ["test"]="测试输出日志"
    ["coverage"]="覆盖率报告"
    ["error"]="错误日志"
    ["debug"]="调试日志"
    ["all"]="所有日志"
)

# 函数：打印日志
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

log_step() {
    echo -e "${CYAN}==>${NC} $1"
}

# 函数：显示帮助信息
show_help() {
    cat << EOF
Saga 示例应用日志查看工具

用法:
    $0 [选项] [日志类型]

日志类型:
    test        查看测试输出日志
    coverage    查看覆盖率报告
    error       仅查看错误日志
    debug       查看调试日志
    all         查看所有日志（默认）

选项:
    -h, --help              显示此帮助信息
    -f, --follow            持续跟踪日志（类似 tail -f）
    -n, --lines NUM         显示最后 NUM 行（默认: 50）
    --filter PATTERN        过滤包含指定模式的行
    -e, --errors-only       仅显示错误和警告
    -v, --verbose           显示详细信息（包括元数据）
    --no-color              禁用颜色输出
    -t, --timestamp         显示时间戳
    -l, --list              列出所有可用的日志文件

示例:
    $0                              # 查看所有日志（最后50行）
    $0 test                         # 查看测试输出
    $0 coverage                     # 查看覆盖率报告
    $0 --follow test                # 持续跟踪测试日志
    $0 --lines 100 error            # 查看最后100行错误日志
    $0 --filter "Saga"              # 过滤包含 "Saga" 的日志行
    $0 --errors-only                # 仅显示错误和警告
    $0 --list                       # 列出所有日志文件

EOF
}

# 函数：列出所有日志文件
list_log_files() {
    log_step "可用的日志文件"
    echo ""
    
    cd "$EXAMPLES_DIR"
    
    # 测试输出日志
    local test_logs=$(find . -maxdepth 1 -name "test_*.log" -type f 2>/dev/null || true)
    if [[ -n "$test_logs" ]]; then
        echo "${GREEN}测试输出日志:${NC}"
        echo "$test_logs" | while read -r file; do
            local size=$(du -h "$file" | cut -f1)
            local mtime=$(stat -f "%Sm" -t "%Y-%m-%d %H:%M:%S" "$file" 2>/dev/null || stat -c "%y" "$file" 2>/dev/null | cut -d'.' -f1)
            printf "  %-40s %8s  %s\n" "$file" "$size" "$mtime"
        done
        echo ""
    fi
    
    # 覆盖率文件
    local coverage_files=$(find . -maxdepth 1 -name "coverage_*.out" -type f 2>/dev/null || true)
    if [[ -n "$coverage_files" ]]; then
        echo "${GREEN}覆盖率报告:${NC}"
        echo "$coverage_files" | while read -r file; do
            local size=$(du -h "$file" | cut -f1)
            local mtime=$(stat -f "%Sm" -t "%Y-%m-%d %H:%M:%S" "$file" 2>/dev/null || stat -c "%y" "$file" 2>/dev/null | cut -d'.' -f1)
            printf "  %-40s %8s  %s\n" "$file" "$size" "$mtime"
        done
        echo ""
    fi
    
    # 错误日志
    local error_logs=$(find . -maxdepth 1 -name "*error*.log" -type f 2>/dev/null || true)
    if [[ -n "$error_logs" ]]; then
        echo "${RED}错误日志:${NC}"
        echo "$error_logs" | while read -r file; do
            local size=$(du -h "$file" | cut -f1)
            local mtime=$(stat -f "%Sm" -t "%Y-%m-%d %H:%M:%S" "$file" 2>/dev/null || stat -c "%y" "$file" 2>/dev/null | cut -d'.' -f1)
            printf "  %-40s %8s  %s\n" "$file" "$size" "$mtime"
        done
        echo ""
    fi
    
    # 调试日志
    local debug_logs=$(find . -maxdepth 1 -name "debug_*.log" -o -name "trace_*.log" -type f 2>/dev/null || true)
    if [[ -n "$debug_logs" ]]; then
        echo "${CYAN}调试日志:${NC}"
        echo "$debug_logs" | while read -r file; do
            local size=$(du -h "$file" | cut -f1)
            local mtime=$(stat -f "%Sm" -t "%Y-%m-%d %H:%M:%S" "$file" 2>/dev/null || stat -c "%y" "$file" 2>/dev/null | cut -d'.' -f1)
            printf "  %-40s %8s  %s\n" "$file" "$size" "$mtime"
        done
        echo ""
    fi
    
    # 检查是否有任何日志文件
    if [[ -z "$test_logs" && -z "$coverage_files" && -z "$error_logs" && -z "$debug_logs" ]]; then
        log_warning "没有发现任何日志文件"
        echo ""
        log_info "提示: 运行测试后会生成日志文件"
        echo "  示例: ./run.sh test order"
    fi
}

# 函数：着色输出
colorize_line() {
    local line=$1
    
    if [[ "$COLORIZE" == false ]]; then
        echo "$line"
        return
    fi
    
    # 错误着色
    if echo "$line" | grep -qiE "(error|fail|fatal|panic)"; then
        echo -e "${RED}$line${NC}"
    # 警告着色
    elif echo "$line" | grep -qiE "(warn|warning)"; then
        echo -e "${YELLOW}$line${NC}"
    # 成功着色
    elif echo "$line" | grep -qiE "(success|pass|ok|✓)"; then
        echo -e "${GREEN}$line${NC}"
    # 信息着色
    elif echo "$line" | grep -qiE "(info|step)"; then
        echo -e "${BLUE}$line${NC}"
    # Saga 相关着色
    elif echo "$line" | grep -qiE "saga"; then
        echo -e "${MAGENTA}$line${NC}"
    else
        echo "$line"
    fi
}

# 函数：查看测试日志
view_test_logs() {
    log_step "查看测试日志"
    
    cd "$EXAMPLES_DIR"
    
    # 运行测试并捕获输出
    log_info "运行测试以生成日志..."
    echo ""
    
    local test_cmd="go test -v"
    
    if [[ "$FOLLOW" == true ]]; then
        # 实时跟踪模式
        $test_cmd 2>&1 | while IFS= read -r line; do
            if [[ -n "$FILTER" ]] && ! echo "$line" | grep -q "$FILTER"; then
                continue
            fi
            
            if [[ "$SHOW_ERRORS_ONLY" == true ]] && ! echo "$line" | grep -qiE "(error|fail|fatal|warn)"; then
                continue
            fi
            
            if [[ "$TIMESTAMP" == true ]]; then
                echo "[$(date '+%Y-%m-%d %H:%M:%S')] $(colorize_line "$line")"
            else
                colorize_line "$line"
            fi
        done
    else
        # 查看最近的测试输出
        local output=$($test_cmd 2>&1 | tail -n "$LINES")
        
        echo "$output" | while IFS= read -r line; do
            if [[ -n "$FILTER" ]] && ! echo "$line" | grep -q "$FILTER"; then
                continue
            fi
            
            if [[ "$SHOW_ERRORS_ONLY" == true ]] && ! echo "$line" | grep -qiE "(error|fail|fatal|warn)"; then
                continue
            fi
            
            colorize_line "$line"
        done
    fi
}

# 函数：查看覆盖率报告
view_coverage() {
    log_step "查看覆盖率报告"
    
    cd "$EXAMPLES_DIR"
    
    # 查找最新的覆盖率文件
    local latest_coverage=$(find . -maxdepth 1 -name "coverage_*.out" -type f -exec stat -f "%m %N" {} \; 2>/dev/null | sort -rn | head -1 | cut -d' ' -f2- || true)
    
    if [[ -z "$latest_coverage" ]]; then
        log_warning "没有发现覆盖率文件"
        log_info "运行以下命令生成覆盖率报告:"
        echo "  ./run.sh --coverage order"
        return 1
    fi
    
    log_info "覆盖率文件: $latest_coverage"
    echo ""
    
    if [[ "$VERBOSE" == true ]]; then
        # 显示详细覆盖率
        log_step "详细覆盖率:"
        echo ""
        go tool cover -func="$latest_coverage" | while IFS= read -r line; do
            colorize_line "$line"
        done
    else
        # 显示摘要
        log_step "覆盖率摘要:"
        echo ""
        go tool cover -func="$latest_coverage" | tail -n 20 | while IFS= read -r line; do
            colorize_line "$line"
        done
    fi
    
    echo ""
    log_info "生成 HTML 报告:"
    echo "  go tool cover -html=$latest_coverage"
}

# 函数：查看错误日志
view_error_logs() {
    log_step "查看错误日志"
    
    cd "$EXAMPLES_DIR"
    
    # 从测试输出中提取错误
    log_info "提取测试错误..."
    echo ""
    
    go test -v 2>&1 | grep -iE "(error|fail|fatal|panic)" | tail -n "$LINES" | while IFS= read -r line; do
        if [[ -n "$FILTER" ]] && ! echo "$line" | grep -q "$FILTER"; then
            continue
        fi
        
        if [[ "$TIMESTAMP" == true ]]; then
            echo "[$(date '+%Y-%m-%d %H:%M:%S')] $(colorize_line "$line")"
        else
            colorize_line "$line"
        fi
    done
}

# 函数：查看调试日志
view_debug_logs() {
    log_step "查看调试日志"
    
    cd "$EXAMPLES_DIR"
    
    # 查找调试日志文件
    local debug_files=$(find . -maxdepth 1 \( -name "debug_*.log" -o -name "trace_*.log" \) -type f 2>/dev/null || true)
    
    if [[ -z "$debug_files" ]]; then
        log_warning "没有发现调试日志文件"
        log_info "设置 SAGA_LOG_LEVEL=debug 运行测试以生成调试日志"
        return 1
    fi
    
    # 选择最新的文件
    local latest_debug=$(echo "$debug_files" | xargs -I {} stat -f "%m %N" {} 2>/dev/null | sort -rn | head -1 | cut -d' ' -f2-)
    
    log_info "调试日志: $latest_debug"
    echo ""
    
    if [[ "$FOLLOW" == true ]]; then
        tail -f -n "$LINES" "$latest_debug" | while IFS= read -r line; do
            if [[ -n "$FILTER" ]] && ! echo "$line" | grep -q "$FILTER"; then
                continue
            fi
            colorize_line "$line"
        done
    else
        tail -n "$LINES" "$latest_debug" | while IFS= read -r line; do
            if [[ -n "$FILTER" ]] && ! echo "$line" | grep -q "$FILTER"; then
                continue
            fi
            colorize_line "$line"
        done
    fi
}

# 函数：查看所有日志
view_all_logs() {
    log_step "查看所有日志"
    echo ""
    
    # 测试日志
    echo "${GREEN}=== 测试日志 ===${NC}"
    view_test_logs
    
    echo ""
    echo ""
    
    # 覆盖率
    echo "${GREEN}=== 覆盖率报告 ===${NC}"
    view_coverage || true
    
    echo ""
    echo ""
    
    # 错误日志
    echo "${GREEN}=== 错误日志 ===${NC}"
    view_error_logs || true
}

# 函数：实时监控
live_monitor() {
    log_step "实时监控模式"
    log_info "按 Ctrl+C 退出"
    echo ""
    
    cd "$EXAMPLES_DIR"
    
    # 运行测试并实时显示输出
    go test -v 2>&1 | while IFS= read -r line; do
        if [[ -n "$FILTER" ]] && ! echo "$line" | grep -q "$FILTER"; then
            continue
        fi
        
        if [[ "$SHOW_ERRORS_ONLY" == true ]] && ! echo "$line" | grep -qiE "(error|fail|fatal|warn)"; then
            continue
        fi
        
        if [[ "$TIMESTAMP" == true ]]; then
            echo "[$(date '+%Y-%m-%d %H:%M:%S')] $(colorize_line "$line")"
        else
            colorize_line "$line"
        fi
    done
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
            -f|--follow)
                FOLLOW=true
                shift
                ;;
            -n|--lines)
                LINES="$2"
                shift 2
                ;;
            --filter)
                FILTER="$2"
                shift 2
                ;;
            -e|--errors-only)
                SHOW_ERRORS_ONLY=true
                shift
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            --no-color)
                COLORIZE=false
                shift
                ;;
            -t|--timestamp)
                TIMESTAMP=true
                shift
                ;;
            -l|--list)
                list_log_files
                exit 0
                ;;
            -*)
                log_error "未知选项: $1"
                echo ""
                show_help
                exit 1
                ;;
            *)
                LOG_TYPE="$1"
                shift
                ;;
        esac
    done
    
    # 检查示例目录
    if [[ ! -d "$EXAMPLES_DIR" ]]; then
        log_error "示例目录不存在: $EXAMPLES_DIR"
        exit 1
    fi
    
    # 如果没有指定日志类型，默认为 all
    if [[ -z "$LOG_TYPE" ]]; then
        LOG_TYPE="all"
    fi
    
    # 验证日志类型
    if [[ ! ${LOG_TYPES[$LOG_TYPE]+_} ]]; then
        log_error "未知的日志类型: $LOG_TYPE"
        echo ""
        log_info "可用的日志类型:"
        for key in "${!LOG_TYPES[@]}"; do
            printf "  ${GREEN}%-12s${NC} %s\n" "$key" "${LOG_TYPES[$key]}"
        done
        exit 1
    fi
    
    # 显示横幅
    echo ""
    echo "========================================"
    echo "  Saga 示例应用 - 日志查看"
    echo "========================================"
    echo ""
    
    # 显示配置
    if [[ "$VERBOSE" == true ]]; then
        log_info "配置:"
        echo "  日志类型: ${LOG_TYPES[$LOG_TYPE]}"
        echo "  显示行数: $LINES"
        [[ -n "$FILTER" ]] && echo "  过滤模式: $FILTER"
        [[ "$FOLLOW" == true ]] && echo "  跟踪模式: 启用"
        [[ "$SHOW_ERRORS_ONLY" == true ]] && echo "  仅错误: 启用"
        echo ""
    fi
    
    # 查看日志
    case $LOG_TYPE in
        "test")
            view_test_logs
            ;;
        "coverage")
            view_coverage
            ;;
        "error")
            view_error_logs
            ;;
        "debug")
            view_debug_logs
            ;;
        "all")
            if [[ "$FOLLOW" == true ]]; then
                live_monitor
            else
                view_all_logs
            fi
            ;;
    esac
    
    echo ""
    log_success "完成！"
}

# 执行主函数
main "$@"

