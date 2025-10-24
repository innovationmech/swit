#!/usr/bin/env bash

# Saga 示例应用停止和清理脚本
# 用于停止运行中的示例进程，清理测试数据和临时文件
set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 项目根目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"
EXAMPLES_DIR="$PROJECT_ROOT/pkg/saga/examples"

# 配置变量
CLEAN_ALL=false
CLEAN_COVERAGE=false
CLEAN_LOGS=false
CLEAN_PROCESSES=false
FORCE=false
DRY_RUN=false
VERBOSE=false

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
Saga 示例应用停止和清理脚本

用法:
    $0 [选项]

选项:
    -h, --help          显示此帮助信息
    -a, --all           清理所有内容（进程、文件、日志）
    -c, --coverage      仅清理覆盖率文件
    -l, --logs          仅清理日志文件
    -p, --processes     仅停止运行中的进程
    -f, --force         强制清理，不提示确认
    -n, --dry-run       试运行模式（显示将清理的内容但不执行）
    -v, --verbose       详细输出模式

清理类型:
    进程清理     停止所有运行中的示例进程
    覆盖率清理   删除 coverage_*.out 文件
    日志清理     删除测试日志和临时文件
    缓存清理     清理 Go 测试缓存

示例:
    $0                              # 交互式清理（会提示确认）
    $0 --all                        # 清理所有内容
    $0 --coverage                   # 仅清理覆盖率文件
    $0 --processes                  # 仅停止运行中的进程
    $0 --all --force                # 强制清理所有内容（不提示）
    $0 --dry-run --all              # 查看将清理的内容（不实际执行）

EOF
}

# 函数：确认操作
confirm_action() {
    local message=$1
    
    if [[ "$FORCE" == true ]]; then
        return 0
    fi
    
    echo -e "${YELLOW}[CONFIRM]${NC} $message"
    read -p "确认继续？(y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "操作已取消"
        return 1
    fi
    return 0
}

# 函数：停止运行中的进程
stop_processes() {
    log_step "检查运行中的示例进程..."
    
    # 查找相关进程
    local pids=$(pgrep -f "go.*test.*saga/examples" 2>/dev/null || true)
    
    if [[ -z "$pids" ]]; then
        log_info "没有发现运行中的示例进程"
        return 0
    fi
    
    log_warning "发现以下进程:"
    echo "$pids" | while read -r pid; do
        if ps -p "$pid" > /dev/null 2>&1; then
            local cmd=$(ps -p "$pid" -o args= 2>/dev/null || echo "未知")
            echo "  PID $pid: $cmd"
        fi
    done
    echo ""
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "试运行模式 - 将终止 $(echo "$pids" | wc -l) 个进程"
        return 0
    fi
    
    if ! confirm_action "将终止这些进程"; then
        return 0
    fi
    
    local killed=0
    echo "$pids" | while read -r pid; do
        if kill "$pid" 2>/dev/null; then
            log_info "已终止进程: $pid"
            killed=$((killed + 1))
        else
            log_warning "无法终止进程: $pid"
        fi
    done
    
    # 等待进程退出
    sleep 1
    
    # 强制终止仍在运行的进程
    local remaining=$(pgrep -f "go.*test.*saga/examples" 2>/dev/null || true)
    if [[ -n "$remaining" ]]; then
        log_warning "部分进程仍在运行，尝试强制终止..."
        echo "$remaining" | while read -r pid; do
            kill -9 "$pid" 2>/dev/null && log_info "已强制终止进程: $pid"
        done
    fi
    
    log_success "进程清理完成"
}

# 函数：清理覆盖率文件
clean_coverage() {
    log_step "清理覆盖率文件..."
    
    cd "$EXAMPLES_DIR"
    
    # 查找覆盖率文件
    local coverage_files=$(find . -maxdepth 1 -name "coverage_*.out" -type f 2>/dev/null || true)
    
    if [[ -z "$coverage_files" ]]; then
        log_info "没有发现覆盖率文件"
        return 0
    fi
    
    local count=$(echo "$coverage_files" | wc -l)
    
    if [[ "$VERBOSE" == true ]]; then
        log_info "发现以下覆盖率文件:"
        echo "$coverage_files" | sed 's/^/  /'
        echo ""
    else
        log_info "发现 $count 个覆盖率文件"
    fi
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "试运行模式 - 将删除 $count 个文件"
        return 0
    fi
    
    if ! confirm_action "将删除这些覆盖率文件"; then
        return 0
    fi
    
    echo "$coverage_files" | while read -r file; do
        if rm -f "$file" 2>/dev/null; then
            [[ "$VERBOSE" == true ]] && log_info "已删除: $file"
        else
            log_warning "无法删除: $file"
        fi
    done
    
    log_success "覆盖率文件清理完成"
}

# 函数：清理日志文件
clean_logs() {
    log_step "清理日志文件..."
    
    cd "$EXAMPLES_DIR"
    
    # 查找日志文件
    local log_patterns=("*.log" "test_*.txt" "debug_*.txt" "trace_*.txt")
    local log_files=""
    
    for pattern in "${log_patterns[@]}"; do
        local files=$(find . -maxdepth 1 -name "$pattern" -type f 2>/dev/null || true)
        if [[ -n "$files" ]]; then
            log_files="$log_files$files"$'\n'
        fi
    done
    
    log_files=$(echo "$log_files" | sed '/^$/d')
    
    if [[ -z "$log_files" ]]; then
        log_info "没有发现日志文件"
        return 0
    fi
    
    local count=$(echo "$log_files" | wc -l)
    
    if [[ "$VERBOSE" == true ]]; then
        log_info "发现以下日志文件:"
        echo "$log_files" | sed 's/^/  /'
        echo ""
    else
        log_info "发现 $count 个日志文件"
    fi
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "试运行模式 - 将删除 $count 个文件"
        return 0
    fi
    
    if ! confirm_action "将删除这些日志文件"; then
        return 0
    fi
    
    echo "$log_files" | while read -r file; do
        if rm -f "$file" 2>/dev/null; then
            [[ "$VERBOSE" == true ]] && log_info "已删除: $file"
        else
            log_warning "无法删除: $file"
        fi
    done
    
    log_success "日志文件清理完成"
}

# 函数：清理测试缓存
clean_cache() {
    log_step "清理测试缓存..."
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "试运行模式 - 将清理 Go 测试缓存"
        return 0
    fi
    
    cd "$EXAMPLES_DIR"
    
    if go clean -testcache; then
        log_success "测试缓存清理完成"
    else
        log_warning "测试缓存清理失败"
    fi
}

# 函数：清理临时文件
clean_temp_files() {
    log_step "清理临时文件..."
    
    cd "$EXAMPLES_DIR"
    
    # 查找临时文件
    local temp_patterns=("*.tmp" "*.temp" "test_*")
    local temp_files=""
    
    for pattern in "${temp_patterns[@]}"; do
        local files=$(find . -maxdepth 1 -name "$pattern" -type f 2>/dev/null || true)
        if [[ -n "$files" ]]; then
            temp_files="$temp_files$files"$'\n'
        fi
    done
    
    temp_files=$(echo "$temp_files" | sed '/^$/d')
    
    if [[ -z "$temp_files" ]]; then
        log_info "没有发现临时文件"
        return 0
    fi
    
    local count=$(echo "$temp_files" | wc -l)
    
    if [[ "$VERBOSE" == true ]]; then
        log_info "发现以下临时文件:"
        echo "$temp_files" | sed 's/^/  /'
        echo ""
    else
        log_info "发现 $count 个临时文件"
    fi
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "试运行模式 - 将删除 $count 个文件"
        return 0
    fi
    
    if ! confirm_action "将删除这些临时文件"; then
        return 0
    fi
    
    echo "$temp_files" | while read -r file; do
        if rm -f "$file" 2>/dev/null; then
            [[ "$VERBOSE" == true ]] && log_info "已删除: $file"
        else
            log_warning "无法删除: $file"
        fi
    done
    
    log_success "临时文件清理完成"
}

# 函数：清理所有内容
clean_all() {
    log_step "执行完整清理..."
    
    echo ""
    stop_processes
    echo ""
    clean_coverage
    echo ""
    clean_logs
    echo ""
    clean_temp_files
    echo ""
    clean_cache
    echo ""
    
    log_success "完整清理完成"
}

# 函数：显示清理摘要
show_summary() {
    log_step "清理摘要"
    
    cd "$EXAMPLES_DIR"
    
    echo ""
    echo "当前状态:"
    
    # 检查进程
    local pids=$(pgrep -f "go.*test.*saga/examples" 2>/dev/null || true)
    if [[ -z "$pids" ]]; then
        echo "  ${GREEN}✓${NC} 没有运行中的进程"
    else
        local count=$(echo "$pids" | wc -l)
        echo "  ${YELLOW}!${NC} 发现 $count 个运行中的进程"
    fi
    
    # 检查覆盖率文件
    local coverage_count=$(find . -maxdepth 1 -name "coverage_*.out" -type f 2>/dev/null | wc -l)
    if [[ $coverage_count -eq 0 ]]; then
        echo "  ${GREEN}✓${NC} 没有覆盖率文件"
    else
        echo "  ${YELLOW}!${NC} 发现 $coverage_count 个覆盖率文件"
    fi
    
    # 检查日志文件
    local log_count=$(find . -maxdepth 1 \( -name "*.log" -o -name "test_*.txt" \) -type f 2>/dev/null | wc -l)
    if [[ $log_count -eq 0 ]]; then
        echo "  ${GREEN}✓${NC} 没有日志文件"
    else
        echo "  ${YELLOW}!${NC} 发现 $log_count 个日志文件"
    fi
    
    echo ""
}

# 函数：交互式清理
interactive_clean() {
    log_step "交互式清理模式"
    echo ""
    
    log_info "请选择要执行的清理操作:"
    echo ""
    echo "  1) 停止运行中的进程"
    echo "  2) 清理覆盖率文件"
    echo "  3) 清理日志文件"
    echo "  4) 清理临时文件"
    echo "  5) 清理测试缓存"
    echo "  6) 执行完整清理（所有操作）"
    echo "  7) 显示当前状态"
    echo "  0) 退出"
    echo ""
    
    read -p "请输入选项 [0-7]: " -n 1 -r
    echo
    echo ""
    
    case $REPLY in
        1)
            stop_processes
            ;;
        2)
            clean_coverage
            ;;
        3)
            clean_logs
            ;;
        4)
            clean_temp_files
            ;;
        5)
            clean_cache
            ;;
        6)
            clean_all
            ;;
        7)
            show_summary
            ;;
        0)
            log_info "退出"
            exit 0
            ;;
        *)
            log_error "无效的选项: $REPLY"
            exit 1
            ;;
    esac
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
            -a|--all)
                CLEAN_ALL=true
                shift
                ;;
            -c|--coverage)
                CLEAN_COVERAGE=true
                shift
                ;;
            -l|--logs)
                CLEAN_LOGS=true
                shift
                ;;
            -p|--processes)
                CLEAN_PROCESSES=true
                shift
                ;;
            -f|--force)
                FORCE=true
                shift
                ;;
            -n|--dry-run)
                DRY_RUN=true
                shift
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
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
    
    # 显示横幅
    echo ""
    echo "========================================"
    echo "  Saga 示例应用 - 停止和清理"
    echo "========================================"
    echo ""
    
    # 检查示例目录
    if [[ ! -d "$EXAMPLES_DIR" ]]; then
        log_error "示例目录不存在: $EXAMPLES_DIR"
        exit 1
    fi
    
    # 执行清理操作
    if [[ "$CLEAN_ALL" == true ]]; then
        clean_all
    elif [[ "$CLEAN_PROCESSES" == true ]]; then
        stop_processes
    elif [[ "$CLEAN_COVERAGE" == true ]]; then
        clean_coverage
    elif [[ "$CLEAN_LOGS" == true ]]; then
        clean_logs
    else
        # 没有指定选项，进入交互模式
        interactive_clean
    fi
    
    echo ""
    
    # 显示摘要
    if [[ "$VERBOSE" == true || "$DRY_RUN" == true ]]; then
        show_summary
    fi
    
    log_success "完成！"
}

# 执行主函数
main "$@"

