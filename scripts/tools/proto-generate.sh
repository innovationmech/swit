#!/usr/bin/env bash

# SWIT Proto代码生成脚本
# 统一管理protobuf代码生成、格式化、检查等功能
set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置变量
PROJECT_NAME="swit"
API_DIR="api"
# 解析BUF_VERSION，优先环境变量，否则从scripts/mk/variables.mk读取，最后回退到v1.28.1
if [ -z "${BUF_VERSION}" ] || [ "${BUF_VERSION}" = "latest" ]; then
    if [ -f "scripts/mk/variables.mk" ]; then
        BUF_VERSION=$(grep -E '^BUF_VERSION\s*:?=\s*' scripts/mk/variables.mk | awk '{print $3}' | tr -d '\r')
    fi
    if [ -z "${BUF_VERSION}" ]; then
        BUF_VERSION="v1.28.1"
    fi
fi

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
SWIT Proto代码生成脚本

用法:
    $0 [选项]

选项:
    -h, --help              显示此帮助信息
    -d, --dev               开发模式 - 跳过依赖下载，快速生成
    -f, --format            格式化proto文件
    -l, --lint              检查proto文件语法
    -b, --breaking          检查破坏性变更
    -c, --clean             清理生成的代码
    --docs                 生成OpenAPI文档
    --setup                设置开发环境（安装工具和依赖）
    --validate             验证proto配置
    --dry-run              只显示将要执行的命令，不实际执行

模式:
    无参数                  标准生成模式（推荐）- 格式化、生成、检查
    --dev                  开发模式 - 跳过依赖下载，最快速度
    --setup                环境设置模式 - 安装工具和下载依赖

示例:
    $0                      # 标准proto代码生成
    $0 --dev                # 快速开发模式生成  
    $0 --setup              # 设置开发环境
    $0 --format --lint      # 只格式化和检查
    $0 --clean              # 清理生成的代码
    $0 --dry-run            # 查看将要执行的命令

环境变量:
    BUF_VERSION            Buf工具版本 (默认: latest)
    API_DIR                API目录路径 (默认: api)
EOF
}

# 函数：检查是否在项目根目录
check_project_root() {
    if [ ! -f "go.mod" ]; then
        log_error "未找到 go.mod 文件，请在项目根目录运行此脚本"
        exit 1
    fi
    
    if [ ! -d "$API_DIR" ]; then
        log_error "未找到 API 目录: $API_DIR"
        exit 1
    fi
}

# 函数：检查和安装Buf工具
install_buf() {
    log_info "检查Buf工具..."
    
    local desired_version_no_v="${BUF_VERSION#v}"
    local need_install="true"
    if command -v buf &> /dev/null; then
        # buf --version 通常输出形如: 1.28.1
        local current_version=$(buf --version 2>/dev/null | head -1 | awk '{print $1}')
        if [ -n "$current_version" ]; then
            log_info "Buf已安装: $current_version"
            if [ "$desired_version_no_v" = "$current_version" ]; then
                need_install="false"
                log_info "Buf版本匹配，无需重新安装"
            fi
        fi
    fi
    
    if [ "$need_install" = "true" ]; then
        log_info "安装Buf CLI 版本: ${BUF_VERSION}"
        if [ "$DRY_RUN" = "true" ]; then
            echo "go install github.com/bufbuild/buf/cmd/buf@${BUF_VERSION}"
            return 0
        fi
        if go install "github.com/bufbuild/buf/cmd/buf@${BUF_VERSION}"; then
            log_success "✓ Buf工具安装完成"
        else
            log_error "Buf工具安装失败"
            exit 1
        fi
    fi
}

# 函数：下载proto依赖
download_dependencies() {
    log_info "下载protobuf依赖..."
    
    if [ "$DRY_RUN" = "true" ]; then
        echo "cd $API_DIR && buf mod update"
        return 0
    fi
    
    cd "$API_DIR"
    if buf mod update; then
        log_success "✓ protobuf依赖下载完成"
    else
        log_warning "protobuf依赖下载失败，尝试继续..."
    fi
    cd - > /dev/null
}

# 函数：格式化proto文件
format_proto() {
    log_info "格式化proto文件..."
    
    if [ "$DRY_RUN" = "true" ]; then
        echo "cd $API_DIR && buf format -w"
        return 0
    fi
    
    cd "$API_DIR"
    if buf format -w; then
        log_success "✓ proto文件格式化完成"
    else
        log_error "proto文件格式化失败"
        cd - > /dev/null
        exit 1
    fi
    cd - > /dev/null
}

# 函数：检查proto文件语法
lint_proto() {
    log_info "检查proto文件语法..."
    
    if [ "$DRY_RUN" = "true" ]; then
        echo "cd $API_DIR && buf lint"
        return 0
    fi
    
    cd "$API_DIR"
    if buf lint; then
        log_success "✓ proto语法检查通过"
    else
        log_error "proto语法检查失败"
        cd - > /dev/null
        exit 1
    fi
    cd - > /dev/null
}

# 函数：检查破坏性变更
check_breaking() {
    log_info "检查破坏性变更..."
    
    if [ "$DRY_RUN" = "true" ]; then
        echo "cd $API_DIR && buf breaking --against '.git#branch=main'"
        return 0
    fi
    
    cd "$API_DIR"
    if buf breaking --against '.git#branch=main' 2>/dev/null; then
        log_success "✓ 无破坏性变更"
    else
        local exit_code=$?
        if [ $exit_code -eq 100 ]; then
            log_warning "发现破坏性变更，请仔细检查变更内容"
        else
            log_warning "破坏性变更检查失败（可能是因为没有main分支或其他git问题）"
        fi
    fi
    cd - > /dev/null
}

# 函数：生成proto代码
generate_proto() {
    log_info "生成protobuf代码..."
    
    if [ "$DRY_RUN" = "true" ]; then
        echo "cd $API_DIR && buf generate"
        return 0
    fi
    
    cd "$API_DIR"
    
    # 带重试逻辑的生成
    local max_attempts=3
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        log_info "🔄 尝试 $attempt/$max_attempts..."
        
        if buf generate 2>&1; then
            log_success "✅ protobuf代码生成完成"
            cd - > /dev/null
            return 0
        else
            if [ $attempt -lt $max_attempts ]; then
                log_warning "⚠️  生成失败，等待30秒后重试..."
                sleep 30
            else
                log_error "❌ protobuf代码生成失败（已尝试 $max_attempts 次）"
                log_error "💡 这可能是由于BSR速率限制或网络问题导致的"
                log_error "📖 参考: https://buf.build/docs/bsr/rate-limits/"
                log_error "🔧 请稍后再试，或重新运行: $0"
                cd - > /dev/null
                exit 1
            fi
        fi
        
        ((attempt++))
    done
}

# 函数：清理生成的代码
clean_generated() {
    log_info "清理生成的proto代码..."
    
    if [ "$DRY_RUN" = "true" ]; then
        echo "rm -rf ${API_DIR}/gen"
        return 0
    fi
    
    if [ -d "${API_DIR}/gen" ]; then
        rm -rf "${API_DIR}/gen"
        log_success "✓ 生成的proto代码已清理"
    else
        log_info "没有找到需要清理的代码"
    fi
}

# 函数：生成文档
generate_docs() {
    log_info "生成OpenAPI文档..."
    
    if [ -d "${API_DIR}/gen/openapiv2" ]; then
        log_success "✓ OpenAPI文档已生成"
        echo "  位置: ${API_DIR}/gen/openapiv2/"
        find "${API_DIR}/gen/openapiv2" -name "*.json" -o -name "*.yaml" | head -5 | sed 's/^/    /'
    else
        log_warning "未找到OpenAPI文档，请确保已生成proto代码"
    fi
}

# 函数：验证配置
validate_config() {
    log_info "验证protobuf配置..."
    
    if [ "$DRY_RUN" = "true" ]; then
        echo "cd $API_DIR && buf mod ls-lint-rules"
        echo "cd $API_DIR && buf mod ls-breaking-rules"
        return 0
    fi
    
    cd "$API_DIR"
    
    log_info "Lint规则:"
    buf mod ls-lint-rules | sed 's/^/  /'
    
    echo ""
    log_info "Breaking change规则:"
    buf mod ls-breaking-rules | sed 's/^/  /'
    
    log_success "✓ protobuf配置验证完成"
    cd - > /dev/null
}

# 函数：设置开发环境
setup_environment() {
    log_info "设置protobuf开发环境..."
    
    # 安装buf工具
    install_buf
    
    # 下载依赖
    download_dependencies
    
    log_success "✓ protobuf开发环境设置完成"
    echo ""
    log_info "现在可以使用以下命令:"
    echo "  $0                # 标准proto代码生成"
    echo "  $0 --dev          # 快速开发模式生成"
    echo "  $0 --format       # 格式化proto文件"
    echo "  $0 --lint         # 检查proto语法"
}

# 函数：显示生成总结
show_summary() {
    log_info "Proto代码生成总结:"
    echo ""
    
    if [ "$DRY_RUN" = "true" ]; then
        log_info "试运行模式 - 未实际执行命令"
        return
    fi
    
    if [ -d "${API_DIR}/gen" ]; then
        log_info "生成的文件:"
        
        # 显示Go代码
        local go_files=$(find "${API_DIR}/gen/go" -name "*.go" 2>/dev/null | wc -l)
        if [ "$go_files" -gt 0 ]; then
            echo "  Go代码: $go_files 个文件"
            find "${API_DIR}/gen/go" -name "*.go" | head -3 | sed 's/^/    /'
            [ "$go_files" -gt 3 ] && echo "    ... 还有 $((go_files - 3)) 个文件"
        fi
        
        # 显示OpenAPI文档
        local api_files=$(find "${API_DIR}/gen/openapiv2" -name "*.json" -o -name "*.yaml" 2>/dev/null | wc -l)
        if [ "$api_files" -gt 0 ]; then
            echo "  OpenAPI文档: $api_files 个文件"
            find "${API_DIR}/gen/openapiv2" -name "*.json" -o -name "*.yaml" | head -2 | sed 's/^/    /'
        fi
        
        echo ""
        log_info "目录大小:"
        local dir_size=$(du -sh "${API_DIR}/gen" 2>/dev/null | cut -f1)
        echo "  ${API_DIR}/gen: $dir_size"
    else
        log_warning "未找到生成的代码目录"
    fi
}

# 主函数
main() {
    # 默认值
    FORMAT_ONLY=false
    LINT_ONLY=false
    BREAKING_ONLY=false
    CLEAN_ONLY=false
    DOCS_ONLY=false
    SETUP_ONLY=false
    VALIDATE_ONLY=false
    DEV_MODE=false
    DRY_RUN=false
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -d|--dev)
                DEV_MODE=true
                shift
                ;;
            -f|--format)
                FORMAT_ONLY=true
                shift
                ;;
            -l|--lint)
                LINT_ONLY=true
                shift
                ;;
            -b|--breaking)
                BREAKING_ONLY=true
                shift
                ;;
            -c|--clean)
                CLEAN_ONLY=true
                shift
                ;;
            --docs)
                DOCS_ONLY=true
                shift
                ;;
            --setup)
                SETUP_ONLY=true
                shift
                ;;
            --validate)
                VALIDATE_ONLY=true
                shift
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            *)
                log_error "未知参数: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    # 检查项目环境
    check_project_root
    
    echo ""
    log_info "🚀 开始 SWIT Proto代码处理"
    echo ""
    
    # 处理特殊模式
    if [ "$SETUP_ONLY" = "true" ]; then
        setup_environment
        exit 0
    fi
    
    if [ "$CLEAN_ONLY" = "true" ]; then
        clean_generated
        exit 0
    fi
    
    if [ "$VALIDATE_ONLY" = "true" ]; then
        install_buf
        validate_config
        exit 0
    fi
    
    if [ "$DOCS_ONLY" = "true" ]; then
        generate_docs
        exit 0
    fi
    
    # 确保buf工具可用
    install_buf
    
    # 处理单独的操作
    if [ "$FORMAT_ONLY" = "true" ] || [ "$LINT_ONLY" = "true" ] || [ "$BREAKING_ONLY" = "true" ]; then
        [ "$FORMAT_ONLY" = "true" ] && format_proto
        [ "$LINT_ONLY" = "true" ] && lint_proto
        [ "$BREAKING_ONLY" = "true" ] && check_breaking
        exit 0
    fi
    
    # 标准工作流程
    local start_time=$(date +%s)
    
    if [ "$DEV_MODE" = "true" ]; then
        log_info "运行开发模式流程（跳过依赖下载和lint检查）..."
        format_proto
        generate_proto
    else
        log_info "运行标准流程..."
        download_dependencies
        format_proto
        generate_proto
        lint_proto
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo ""
    log_success "🎉 Proto代码处理完成！用时 ${duration} 秒"
    
    # 显示总结
    echo ""
    show_summary
}

# 运行主函数
main "$@" 