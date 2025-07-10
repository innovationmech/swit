#!/usr/bin/env bash

# SWIT 开发环境管理脚本
# 统一管理开发环境设置、工具安装、Git hooks、CI流水线等功能
set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置变量
PROJECT_NAME="swit"
HOOKS_DIR=".git/hooks"
PRECOMMIT_HOOK="scripts/tools/pre-commit.sh"

# 相关脚本路径
SWAGGER_SCRIPT="scripts/tools/swagger-manage.sh"
PROTO_SCRIPT="scripts/tools/proto-generate.sh"
COPYRIGHT_SCRIPT="scripts/tools/copyright-manage.sh"
BUILD_SCRIPT="scripts/tools/build-multiplatform.sh"
TEST_SCRIPT="scripts/tools/test-manage.sh"
CLEAN_SCRIPT="scripts/tools/clean-manage.sh"

# 开发环境组件定义
DEV_COMPONENTS=(
    "hooks:Git钩子:install_git_hooks"
    "swagger:Swagger工具:setup_swagger_tools"
    "proto:Protobuf工具:setup_proto_tools"
    "copyright:版权管理:setup_copyright_tools"
    "build:构建工具:setup_build_tools"
    "test:测试环境:setup_test_env"
)

# CI流水线步骤定义
CI_STEPS=(
    "tidy:依赖整理:run_tidy"
    "copyright:版权检查:run_copyright"
    "quality:质量检查:run_quality"
    "proto:Proto生成:run_proto"
    "test:运行测试:run_test"
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
SWIT 开发环境管理脚本

用法:
    $0 [选项]

选项:
    -h, --help              显示此帮助信息
    -q, --quick             快速设置模式 (最小必要组件)
    -c, --ci                CI流水线模式 (自动化测试和检查)
    -a, --advanced COMPONENT 高级管理模式 (操作特定组件)
    -l, --list              列出可用的开发环境组件
    -n, --dry-run           试运行模式 (显示命令但不执行)
    -v, --verbose           详细输出模式
    -f, --force             强制重新安装所有组件

开发环境模式:
    默认模式      标准开发环境设置 - 完整的开发环境 (推荐)
    --quick       快速设置 - 最小必要组件，快速开始开发
    --ci          CI流水线 - 自动化测试和质量检查
    --advanced    高级管理 - 操作特定开发环境组件

高级管理组件:
    hooks         Git钩子管理
    swagger       Swagger工具设置
    proto         Protobuf工具设置
    copyright     版权管理工具
    build         构建工具设置
    test          测试环境设置
    clean         清理开发环境
    validate      验证开发环境

示例:
    $0                              # 标准开发环境设置
    $0 --quick                      # 快速设置
    $0 --ci                         # CI流水线
    $0 --advanced hooks             # Git钩子管理
    $0 --advanced swagger           # Swagger工具设置
    $0 --advanced validate          # 验证开发环境
    $0 --list                       # 列出可用组件
    $0 --dry-run                    # 试运行模式

EOF
}

# 函数：检查项目环境
check_project_env() {
    if [[ ! -f "go.mod" ]]; then
        log_error "未找到 go.mod 文件，请在项目根目录运行此脚本"
        exit 1
    fi
    
    if [[ ! -d ".git" ]]; then
        log_error "未找到 .git 目录，请确保在Git仓库中运行"
        exit 1
    fi
    
    if [[ "$verbose" == "true" ]]; then
        log_info "项目根目录: $(pwd)"
        log_info "Git仓库状态: $(git status --porcelain | wc -l) 个修改文件"
    fi
}

# 函数：检查Go环境
check_go_env() {
    if ! command -v go &> /dev/null; then
        log_error "Go 未安装或不在 PATH 中"
        exit 1
    fi
    
    local go_version
    go_version=$(go version 2>/dev/null | cut -d' ' -f3)
    
    if [[ "$verbose" == "true" ]]; then
        log_info "Go 版本: $go_version"
        log_info "GOPATH: $(go env GOPATH)"
        log_info "GOOS/GOARCH: $(go env GOOS)/$(go env GOARCH)"
    fi
}

# 函数：安装Git钩子
install_git_hooks() {
    log_info "📋 安装Git钩子..."
    
    if [[ ! -f "$PRECOMMIT_HOOK" ]]; then
        log_error "预提交钩子脚本不存在: $PRECOMMIT_HOOK"
        return 1
    fi
    
    if [[ "$dry_run" == "true" ]]; then
        echo "    [DRY-RUN] mkdir -p $HOOKS_DIR"
        echo "    [DRY-RUN] cp $PRECOMMIT_HOOK $HOOKS_DIR/pre-commit"
        echo "    [DRY-RUN] chmod +x $HOOKS_DIR/pre-commit"
        return 0
    fi
    
    # 创建hooks目录
    mkdir -p "$HOOKS_DIR"
    
    # 复制预提交钩子
    if cp "$PRECOMMIT_HOOK" "$HOOKS_DIR/pre-commit"; then
        chmod +x "$HOOKS_DIR/pre-commit"
        log_success "✅ Git钩子安装完成"
        
        if [[ "$verbose" == "true" ]]; then
            log_info "  预提交钩子: $HOOKS_DIR/pre-commit"
        fi
    else
        log_error "Git钩子安装失败"
        return 1
    fi
}

# 函数：设置Swagger工具
setup_swagger_tools() {
    log_info "📚 设置Swagger工具..."
    
    if [[ ! -f "$SWAGGER_SCRIPT" ]]; then
        log_warning "Swagger管理脚本不存在，跳过设置"
        return 0
    fi
    
    if [[ "$dry_run" == "true" ]]; then
        echo "    [DRY-RUN] bash $SWAGGER_SCRIPT --setup"
        return 0
    fi
    
    if bash "$SWAGGER_SCRIPT" --setup; then
        log_success "✅ Swagger工具设置完成"
    else
        log_warning "Swagger工具设置失败，但继续其他设置"
    fi
}

# 函数：设置Protobuf工具
setup_proto_tools() {
    log_info "🔧 设置Protobuf工具..."
    
    if [[ ! -f "$PROTO_SCRIPT" ]]; then
        log_warning "Proto管理脚本不存在，跳过设置"
        return 0
    fi
    
    if [[ "$dry_run" == "true" ]]; then
        echo "    [DRY-RUN] bash $PROTO_SCRIPT --setup"
        return 0
    fi
    
    if bash "$PROTO_SCRIPT" --setup; then
        log_success "✅ Protobuf工具设置完成"
    else
        log_warning "Protobuf工具设置失败，但继续其他设置"
    fi
}

# 函数：设置版权管理工具
setup_copyright_tools() {
    log_info "©️  设置版权管理工具..."
    
    if [[ ! -f "$COPYRIGHT_SCRIPT" ]]; then
        log_warning "版权管理脚本不存在，跳过设置"
        return 0
    fi
    
    if [[ "$dry_run" == "true" ]]; then
        echo "    [DRY-RUN] bash $COPYRIGHT_SCRIPT --setup"
        return 0
    fi
    
    if bash "$COPYRIGHT_SCRIPT" --setup 2>/dev/null; then
        log_success "✅ 版权管理工具设置完成"
    else
        log_warning "版权管理工具设置失败，但继续其他设置"
    fi
}

# 函数：设置构建工具
setup_build_tools() {
    log_info "🔨 设置构建工具..."
    
    # 检查构建依赖
    local missing_tools=()
    
    if ! command -v gofmt &> /dev/null; then
        missing_tools+=("gofmt")
    fi
    
    if ! command -v go &> /dev/null; then
        missing_tools+=("go")
    fi
    
    if [[ ${#missing_tools[@]} -gt 0 ]]; then
        log_warning "缺少构建工具: ${missing_tools[*]}"
        log_info "请安装Go工具链"
    else
        log_success "✅ 构建工具检查完成"
    fi
    
    if [[ "$verbose" == "true" ]]; then
        log_info "  可用构建工具: go, gofmt, go vet"
    fi
}

# 函数：设置测试环境
setup_test_env() {
    log_info "🧪 设置测试环境..."
    
    if [[ "$dry_run" == "true" ]]; then
        echo "    [DRY-RUN] go version"
        echo "    [DRY-RUN] 检查测试文件"
        return 0
    fi
    
    # 检查测试文件
    local test_files
    test_files=$(find . -name "*_test.go" 2>/dev/null | wc -l)
    
    if [[ "$test_files" -gt 0 ]]; then
        log_success "✅ 测试环境设置完成"
        if [[ "$verbose" == "true" ]]; then
            log_info "  发现 $test_files 个测试文件"
        fi
    else
        log_warning "未发现测试文件"
    fi
}

# 函数：验证开发环境
validate_environment() {
    log_info "🔍 验证开发环境..."
    
    local issues=0
    
    # 检查Git钩子
    if [[ -f "$HOOKS_DIR/pre-commit" ]]; then
        log_success "  ✅ Git预提交钩子已安装"
    else
        log_warning "  ⚠️  Git预提交钩子未安装"
        ((issues++))
    fi
    
    # 检查Go环境
    if command -v go &> /dev/null; then
        log_success "  ✅ Go环境可用"
    else
        log_error "  ❌ Go环境不可用"
        ((issues++))
    fi
    
    # 检查项目文件
    local required_files=("go.mod" "Makefile" ".git")
    for file in "${required_files[@]}"; do
        if [[ -e "$file" ]]; then
            log_success "  ✅ $file 存在"
        else
            log_error "  ❌ $file 缺失"
            ((issues++))
        fi
    done
    
    # 检查脚本文件
    local scripts=("$SWAGGER_SCRIPT" "$PROTO_SCRIPT" "$PRECOMMIT_HOOK")
    for script in "${scripts[@]}"; do
        if [[ -f "$script" ]]; then
            log_success "  ✅ $(basename "$script") 存在"
        else
            log_warning "  ⚠️  $(basename "$script") 缺失"
        fi
    done
    
    if [[ $issues -eq 0 ]]; then
        log_success "🎉 开发环境验证完成，一切正常！"
        return 0
    else
        log_warning "⚠️  发现 $issues 个问题，建议运行完整设置"
        return 1
    fi
}

# 函数：清理开发环境
clean_environment() {
    log_info "🧹 清理开发环境..."
    
    if [[ "$dry_run" == "true" ]]; then
        echo "    [DRY-RUN] 清理Git钩子"
        echo "    [DRY-RUN] 清理构建产物"
        echo "    [DRY-RUN] 清理临时文件"
        return 0
    fi
    
    # 清理Git钩子
    if [[ -f "$HOOKS_DIR/pre-commit" ]]; then
        rm -f "$HOOKS_DIR/pre-commit"
        log_info "  删除Git预提交钩子"
    fi
    
    # 调用清理脚本
    if [[ -f "$CLEAN_SCRIPT" ]]; then
        bash "$CLEAN_SCRIPT" --setup >/dev/null 2>&1 || true
        log_info "  调用清理脚本"
    fi
    
    log_success "✅ 开发环境清理完成"
}

# 函数：列出可用组件
list_components() {
    log_info "📋 可用的开发环境组件:"
    echo ""
    
    for component in "${DEV_COMPONENTS[@]}"; do
        IFS=":" read -r comp_name comp_desc comp_func <<< "$component"
        echo "  ${comp_name}$(printf '%*s' $((12-${#comp_name})) '') - ${comp_desc}"
    done
    
    echo ""
    echo "额外组件:"
    echo "  clean        - 清理开发环境"
    echo "  validate     - 验证开发环境"
    echo ""
    echo "CI流水线步骤:"
    for step in "${CI_STEPS[@]}"; do
        IFS=":" read -r step_name step_desc step_func <<< "$step"
        echo "  ${step_name}$(printf '%*s' $((12-${#step_name})) '') - ${step_desc}"
    done
}

# 函数：运行CI流水线步骤
run_tidy() {
    log_info "🔄 运行依赖整理..."
    if [[ "$dry_run" == "true" ]]; then
        echo "    [DRY-RUN] go mod tidy"
        return 0
    fi
    
    if go mod tidy; then
        log_success "✅ 依赖整理完成"
    else
        log_error "❌ 依赖整理失败"
        return 1
    fi
}

run_copyright() {
    log_info "©️  运行版权检查..."
    if [[ "$dry_run" == "true" ]]; then
        echo "    [DRY-RUN] bash $COPYRIGHT_SCRIPT"
        return 0
    fi
    
    if [[ -f "$COPYRIGHT_SCRIPT" ]] && bash "$COPYRIGHT_SCRIPT" >/dev/null 2>&1; then
        log_success "✅ 版权检查完成"
    else
        log_warning "⚠️  版权检查失败或脚本不存在"
    fi
}

run_quality() {
    log_info "🔍 运行质量检查..."
    if [[ "$dry_run" == "true" ]]; then
        echo "    [DRY-RUN] gofmt -w ."
        echo "    [DRY-RUN] go vet ./..."
        return 0
    fi
    
    # 格式化代码
    if gofmt -w . >/dev/null 2>&1; then
        log_info "  代码格式化完成"
    else
        log_warning "  代码格式化失败"
    fi
    
    # 运行vet检查
    if go vet ./... >/dev/null 2>&1; then
        log_success "✅ 质量检查完成"
    else
        log_error "❌ 质量检查失败"
        return 1
    fi
}

run_proto() {
    log_info "🔧 运行Proto生成..."
    if [[ "$dry_run" == "true" ]]; then
        echo "    [DRY-RUN] bash $PROTO_SCRIPT --dev"
        return 0
    fi
    
    if [[ -f "$PROTO_SCRIPT" ]] && bash "$PROTO_SCRIPT" --dev >/dev/null 2>&1; then
        log_success "✅ Proto生成完成"
    else
        log_warning "⚠️  Proto生成失败或脚本不存在"
    fi
}

run_test() {
    log_info "🧪 运行测试..."
    if [[ "$dry_run" == "true" ]]; then
        echo "    [DRY-RUN] bash $TEST_SCRIPT --dev"
        return 0
    fi
    
    if [[ -f "$TEST_SCRIPT" ]] && bash "$TEST_SCRIPT" --dev >/dev/null 2>&1; then
        log_success "✅ 测试运行完成"
    else
        log_error "❌ 测试运行失败"
        return 1
    fi
}

# 函数：标准开发环境设置
setup_standard() {
    log_info "🚀 开始标准开发环境设置"
    log_info "  设置完整的开发环境（推荐用于新环境）..."
    
    local failed_components=()
    
    # 执行所有组件设置
    for component in "${DEV_COMPONENTS[@]}"; do
        IFS=":" read -r comp_name comp_desc comp_func <<< "$component"
        
        if ! "$comp_func"; then
            failed_components+=("$comp_name")
        fi
    done
    
    # 显示结果
    if [[ ${#failed_components[@]} -eq 0 ]]; then
        log_success "🎉 标准开发环境设置完成！"
        echo ""
        log_info "💡 提示："
        log_info "  dev-quick    # 快速设置"
        log_info "  dev-ci       # CI流水线"
        log_info "  dev-advanced # 高级管理"
    else
        log_warning "⚠️  部分组件设置失败: ${failed_components[*]}"
        log_info "可以使用 --advanced 单独设置失败的组件"
    fi
}

# 函数：快速设置
setup_quick() {
    log_info "🔥 开始快速开发环境设置"
    log_info "  设置最小必要组件，快速开始开发..."
    
    # 只安装核心组件
    install_git_hooks
    setup_swagger_tools
    setup_proto_tools
    
    log_success "✅ 快速开发环境设置完成！"
    log_info "💡 提示: 如需完整环境，请运行不带参数的命令"
}

# 函数：CI流水线
run_ci_pipeline() {
    log_info "🔄 开始CI流水线"
    log_info "  运行自动化测试和质量检查..."
    
    local failed_steps=()
    
    # 执行所有CI步骤
    for step in "${CI_STEPS[@]}"; do
        IFS=":" read -r step_name step_desc step_func <<< "$step"
        
        if ! "$step_func"; then
            failed_steps+=("$step_name")
            # CI失败时立即退出
            if [[ "$step_name" == "test" || "$step_name" == "quality" ]]; then
                break
            fi
        fi
    done
    
    # 显示结果
    if [[ ${#failed_steps[@]} -eq 0 ]]; then
        log_success "🎉 CI流水线执行完成！"
    else
        log_error "❌ CI流水线失败，失败步骤: ${failed_steps[*]}"
        return 1
    fi
}

# 函数：高级管理
manage_advanced() {
    local component="$1"
    
    log_info "⚙️  高级管理模式 - 组件: ${component}"
    
    case "$component" in
        "hooks")
            install_git_hooks
            ;;
        "swagger")
            setup_swagger_tools
            ;;
        "proto")
            setup_proto_tools
            ;;
        "copyright")
            setup_copyright_tools
            ;;
        "build")
            setup_build_tools
            ;;
        "test")
            setup_test_env
            ;;
        "clean")
            clean_environment
            ;;
        "validate")
            validate_environment
            ;;
        *)
            log_error "未知的组件: $component"
            echo ""
            log_info "可用组件:"
            list_components
            exit 1
            ;;
    esac
}

# 函数：显示开发环境摘要
show_summary() {
    local mode="$1"
    local result="$2"
    
    if [[ "$result" == "0" ]]; then
        log_success "✅ ${mode}完成！"
    else
        log_error "❌ ${mode}失败！"
    fi
    
    echo ""
    log_info "📊 开发环境状态："
    
    # 检查关键组件状态
    local hook_status="❌"
    local go_status="❌"
    local git_status="❌"
    
    [[ -f "$HOOKS_DIR/pre-commit" ]] && hook_status="✅"
    command -v go &> /dev/null && go_status="✅"
    [[ -d ".git" ]] && git_status="✅"
    
    echo "  Git钩子: $hook_status"
    echo "  Go环境: $go_status"
    echo "  Git仓库: $git_status"
    echo ""
    
    if [[ "$result" == "0" ]]; then
        log_info "🎯 下一步："
        case "$mode" in
            "标准开发环境设置")
                log_info "  make test-dev    # 快速测试"
                log_info "  make build-dev   # 快速构建"
                ;;
            "快速开发环境设置")
                log_info "  make test-dev    # 快速测试"
                log_info "  dev-manage.sh    # 完整设置"
                ;;
            "CI流水线")
                log_info "  git commit       # 提交代码"
                log_info "  make build       # 构建发布"
                ;;
        esac
    fi
}

# 主函数
main() {
    local mode=""
    local component=""
    local dry_run="false"
    local verbose="false"
    local force="false"
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -q|--quick)
                mode="quick"
                shift
                ;;
            -c|--ci)
                mode="ci"
                shift
                ;;
            -a|--advanced)
                mode="advanced"
                component="$2"
                shift 2
                ;;
            -l|--list)
                list_components
                exit 0
                ;;
            -n|--dry-run)
                dry_run="true"
                shift
                ;;
            -v|--verbose)
                verbose="true"
                shift
                ;;
            -f|--force)
                force="true"
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
    log_info "🚀 SWIT 开发环境管理脚本启动"
    
    if [[ "$dry_run" == "true" ]]; then
        log_warning "⚠️  试运行模式 - 仅显示命令，不执行实际操作"
    fi
    
    echo ""
    
    # 检查环境
    check_project_env
    check_go_env
    
    local result=0
    
    # 根据模式执行操作
    case "$mode" in
        "quick")
            setup_quick || result=$?
            show_summary "快速开发环境设置" "$result"
            ;;
        "ci")
            run_ci_pipeline || result=$?
            show_summary "CI流水线" "$result"
            ;;
        "advanced")
            if [[ -z "$component" ]]; then
                log_error "高级模式需要指定组件"
                echo ""
                list_components
                exit 1
            fi
            manage_advanced "$component" || result=$?
            show_summary "高级管理($component)" "$result"
            ;;
        *)
            # 默认标准设置模式
            setup_standard || result=$?
            show_summary "标准开发环境设置" "$result"
            ;;
    esac
    
    exit $result
}

# 脚本入口
main "$@" 