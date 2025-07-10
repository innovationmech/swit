#!/usr/bin/env bash

# SWIT 测试管理脚本
# 统一管理项目的测试功能，包括单元测试、覆盖率测试、竞态检测等
set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置变量
PROJECT_NAME="swit"
GO_CMD="go"
PROTO_SCRIPT="scripts/tools/proto-generate.sh"
SWAGGER_SCRIPT="scripts/tools/swagger-manage.sh"

# 测试目标包定义
TEST_PACKAGES=(
    "all:所有包:./internal/... ./pkg/..."
    "internal:内部包:./internal/..."
    "pkg:公共包:./pkg/..."
)

# 测试类型定义
TEST_TYPES=(
    "unit:单元测试:-v"
    "race:竞态检测:-v -race"
    "bench:性能测试:-v -bench=."
    "short:快速测试:-v -short"
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

# 函数：显示帮助信息
show_help() {
    cat << EOF
SWIT 测试管理脚本

用法:
    $0 [选项]

选项:
    -h, --help              显示此帮助信息
    -d, --dev               快速开发模式 (跳过依赖生成)
    -c, --coverage          覆盖率测试模式 (生成覆盖率报告)
    -a, --advanced TYPE     高级测试模式 (指定测试类型)
    -p, --package PKG       指定测试包 (all, internal, pkg)
    -t, --type TYPE         指定测试类型 (unit, race, bench, short)
    -n, --dry-run           试运行模式 (显示命令但不执行)
    -v, --verbose           详细输出模式
    -s, --skip-deps         跳过依赖生成 (proto和swagger)

测试模式:
    默认模式      标准测试 - 运行所有测试（生成依赖+测试）
    --dev         快速开发 - 跳过依赖生成，直接测试
    --coverage    覆盖率测试 - 生成详细的覆盖率报告
    --advanced    高级测试 - 精确控制测试类型和包范围

高级测试选项:
    --advanced TYPE --package PKG
    
    TYPE:
        unit      标准单元测试
        race      竞态检测测试
        bench     性能基准测试
        short     快速测试（跳过耗时测试）
    
    PKG:
        all       所有包 (internal + pkg)
        internal  仅内部包
        pkg       仅公共包

示例:
    $0                                    # 标准测试
    $0 --dev                             # 快速开发测试
    $0 --coverage                        # 覆盖率测试
    $0 --advanced race --package all     # 所有包的竞态检测
    $0 --advanced unit --package internal # 仅内部包测试
    $0 --advanced bench --package pkg    # 仅公共包性能测试
    $0 --dry-run                        # 试运行模式

EOF
}

# 函数：检查Go环境
check_go_env() {
    if ! command -v go &> /dev/null; then
        log_error "Go 未安装或不在 PATH 中"
        exit 1
    fi
    
    if [[ "$verbose" == "true" ]]; then
        log_info "Go 版本: $(go version)"
    fi
}

# 函数：生成依赖
generate_deps() {
    local skip_deps="$1"
    
    if [[ "$skip_deps" == "true" ]]; then
        log_warning "⚠️  跳过依赖生成"
        return 0
    fi
    
    log_info "🔄 生成测试依赖..."
    
    # 生成proto代码
    if [[ -f "$PROTO_SCRIPT" ]]; then
        log_info "  生成proto代码..."
        if ! bash "$PROTO_SCRIPT" --dev >/dev/null 2>&1; then
            log_warning "Proto代码生成失败，继续测试..."
        fi
    fi
    
    # 生成swagger文档
    if [[ -f "$SWAGGER_SCRIPT" ]]; then
        log_info "  生成swagger文档..."
        if ! bash "$SWAGGER_SCRIPT" --dev >/dev/null 2>&1; then
            log_warning "Swagger文档生成失败，继续测试..."
        fi
    fi
}

# 函数：获取包路径
get_package_paths() {
    local package="$1"
    
    for pkg_def in "${TEST_PACKAGES[@]}"; do
        IFS=":" read -r pkg_name pkg_desc pkg_paths <<< "$pkg_def"
        if [[ "$pkg_name" == "$package" ]]; then
            echo "$pkg_paths"
            return 0
        fi
    done
    
    log_error "未知的包类型: $package"
    echo "支持的包类型: all, internal, pkg"
    exit 1
}

# 函数：获取测试选项
get_test_options() {
    local test_type="$1"
    
    for type_def in "${TEST_TYPES[@]}"; do
        IFS=":" read -r type_name type_desc type_options <<< "$type_def"
        if [[ "$type_name" == "$test_type" ]]; then
            echo "$type_options"
            return 0
        fi
    done
    
    log_error "未知的测试类型: $test_type"
    echo "支持的测试类型: unit, race, bench, short"
    exit 1
}

# 函数：运行测试
run_tests() {
    local test_type="$1"
    local package="$2"
    local dry_run="$3"
    local verbose="$4"
    
    local test_options
    test_options=$(get_test_options "$test_type")
    
    local package_paths
    package_paths=$(get_package_paths "$package")
    
    local test_cmd="$GO_CMD test $test_options $package_paths"
    
    log_info "🧪 运行测试 - 类型: ${test_type}, 包: ${package}"
    
    if [[ "$verbose" == "true" ]]; then
        log_info "  命令: $test_cmd"
    fi
    
    if [[ "$dry_run" == "true" ]]; then
        echo "    [DRY-RUN] $test_cmd"
        return 0
    fi
    
    # 执行测试
    if eval "$test_cmd"; then
        log_success "✅ 测试通过"
        return 0
    else
        log_error "❌ 测试失败"
        return 1
    fi
}

# 函数：运行覆盖率测试
run_coverage_tests() {
    local package_paths="$1"
    local dry_run="$2"
    local verbose="$3"
    
    local coverage_file="coverage.out"
    local coverage_html="coverage.html"
    
    local coverage_cmd="$GO_CMD test -v -coverprofile=$coverage_file $package_paths"
    local html_cmd="$GO_CMD tool cover -html=$coverage_file -o $coverage_html"
    
    log_info "📊 运行覆盖率测试..."
    
    if [[ "$verbose" == "true" ]]; then
        log_info "  覆盖率命令: $coverage_cmd"
        log_info "  报告命令: $html_cmd"
    fi
    
    if [[ "$dry_run" == "true" ]]; then
        echo "    [DRY-RUN] $coverage_cmd"
        echo "    [DRY-RUN] $html_cmd"
        return 0
    fi
    
    # 执行覆盖率测试
    if eval "$coverage_cmd"; then
        log_success "✅ 覆盖率测试完成"
        
        # 生成HTML报告
        if eval "$html_cmd"; then
            log_success "📋 覆盖率报告生成: $coverage_html"
            
            # 显示覆盖率统计
            if command -v go &> /dev/null; then
                local coverage_percent
                coverage_percent=$(go tool cover -func="$coverage_file" | grep "total:" | awk '{print $3}')
                log_info "📈 总覆盖率: $coverage_percent"
            fi
        else
            log_warning "覆盖率报告生成失败"
        fi
        
        return 0
    else
        log_error "❌ 覆盖率测试失败"
        return 1
    fi
}

# 函数：显示测试统计
show_summary() {
    local mode="$1"
    local result="$2"
    
    if [[ "$result" == "0" ]]; then
        log_success "✅ ${mode}测试完成！"
    else
        log_error "❌ ${mode}测试失败！"
    fi
    
    echo ""
    log_info "📊 测试统计："
    
    # 检查测试相关文件
    local coverage_files=0
    local test_files=0
    
    coverage_files=$(find . -name "coverage.*" 2>/dev/null | wc -l || echo 0)
    test_files=$(find . -name "*_test.go" 2>/dev/null | wc -l || echo 0)
    
    echo "  测试文件: ${test_files} 个"
    echo "  覆盖率文件: ${coverage_files} 个"
    echo ""
    
    if [[ "$result" == "0" ]]; then
        log_info "💡 提示："
        log_info "  test-dev      # 快速开发测试"
        log_info "  test-coverage # 覆盖率测试"
        log_info "  test-advanced # 高级测试控制"
    fi
}

# 主函数
main() {
    local mode=""
    local test_type="unit"
    local package="all"
    local dry_run="false"
    local verbose="false"
    local skip_deps="false"
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -d|--dev)
                mode="dev"
                skip_deps="true"
                shift
                ;;
            -c|--coverage)
                mode="coverage"
                shift
                ;;
            -a|--advanced)
                mode="advanced"
                shift
                ;;
            -p|--package)
                package="$2"
                shift 2
                ;;
            -t|--type)
                test_type="$2"
                shift 2
                ;;
            -n|--dry-run)
                dry_run="true"
                shift
                ;;
            -v|--verbose)
                verbose="true"
                shift
                ;;
            -s|--skip-deps)
                skip_deps="true"
                shift
                ;;
            *)
                log_error "未知选项: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    # 显示开始信息
    log_info "🚀 SWIT 测试管理脚本启动"
    
    if [[ "$dry_run" == "true" ]]; then
        log_warning "⚠️  试运行模式 - 仅显示命令，不执行实际操作"
    fi
    
    echo ""
    
    # 检查环境
    check_go_env
    
    local result=0
    
    # 根据模式执行测试
    case "$mode" in
        "dev")
            log_info "🔥 快速开发测试模式"
            generate_deps "$skip_deps"
            run_tests "$test_type" "$package" "$dry_run" "$verbose" || result=$?
            show_summary "快速开发" "$result"
            ;;
        "coverage")
            log_info "📊 覆盖率测试模式"
            generate_deps "$skip_deps"
            local package_paths
            package_paths=$(get_package_paths "$package")
            run_coverage_tests "$package_paths" "$dry_run" "$verbose" || result=$?
            show_summary "覆盖率" "$result"
            ;;
        "advanced")
            log_info "⚙️  高级测试模式 - 类型: ${test_type}, 包: ${package}"
            generate_deps "$skip_deps"
            run_tests "$test_type" "$package" "$dry_run" "$verbose" || result=$?
            show_summary "高级" "$result"
            ;;
        *)
            # 默认标准测试模式
            log_info "🧪 标准测试模式"
            log_info "  运行所有测试（包含依赖生成）..."
            generate_deps "$skip_deps"
            run_tests "$test_type" "$package" "$dry_run" "$verbose" || result=$?
            show_summary "标准" "$result"
            ;;
    esac
    
    exit $result
}

# 脚本入口
main "$@" 