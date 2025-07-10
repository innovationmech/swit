#!/usr/bin/env bash

# SWIT Swagger文档管理脚本
# 统一管理swagger文档生成、格式化、安装等功能
set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置变量
PROJECT_NAME="swit"
SWAG_VERSION=${SWAG_VERSION:-"v1.8.12"}

# 服务定义
SERVICES=(
    "switserve:cmd/swit-serve/swit-serve.go:docs/generated/switserve:internal/switauth"
    "switauth:cmd/swit-auth/swit-auth.go:docs/generated/switauth:internal/switserve"
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
SWIT Swagger文档管理脚本

用法:
    $0 [选项]

选项:
    -h, --help              显示此帮助信息
    -d, --dev               开发模式 - 跳过格式化，快速生成
    -s, --setup             环境设置 - 安装swag工具
    -a, --advanced OPERATION 高级操作 - 精确控制特定操作
    --dry-run              只显示将要执行的操作，不实际执行

模式:
    无参数                  标准swagger文档生成（推荐）- 格式化、生成、整理
    --dev                  开发模式 - 跳过格式化，最快速度
    --setup                环境设置模式 - 安装swag工具
    --advanced             高级模式 - 执行特定操作

高级操作类型:
    format                 格式化swagger注释
    switserve              只生成switserve文档
    switauth               只生成switauth文档
    copy                   复制文档到统一位置
    clean                  清理生成的文档
    validate               验证swagger配置

示例:
    $0                      # 标准swagger文档生成
    $0 --dev                # 快速开发模式生成
    $0 --setup              # 安装swag工具
    $0 --advanced format    # 只格式化注释
    $0 --advanced switserve # 只生成switserve文档
    $0 --dry-run            # 查看将要执行的操作

环境变量:
    SWAG_VERSION           Swag工具版本 (默认: v1.8.12)
EOF
}

# 函数：检查是否在项目根目录
check_project_root() {
    if [ ! -f "go.mod" ]; then
        log_error "未找到 go.mod 文件，请在项目根目录运行此脚本"
        exit 1
    fi
}

# 函数：检查和安装swag工具
install_swag() {
    log_info "检查swag工具..."
    
    if command -v swag &> /dev/null; then
        local current_version=$(swag --version 2>/dev/null | head -1 || echo "unknown")
        log_info "swag已安装: $current_version"
        return 0
    fi
    
    log_info "安装swag工具..."
    if [ "$DRY_RUN" = "true" ]; then
        echo "go install github.com/swaggo/swag/cmd/swag@${SWAG_VERSION}"
        return 0
    fi
    
    if go install "github.com/swaggo/swag/cmd/swag@${SWAG_VERSION}"; then
        log_success "✓ swag工具安装完成"
    else
        log_error "swag工具安装失败"
        exit 1
    fi
}

# 函数：格式化swagger注释
format_swagger() {
    log_info "格式化swagger注释..."
    
    if [ "$DRY_RUN" = "true" ]; then
        for service_info in "${SERVICES[@]}"; do
            local service_name="${service_info%%:*}"
            local main_file="${service_info#*:}"
            main_file="${main_file%%:*}"
            local cmd_dir=$(dirname "$main_file")
            echo "swag fmt -g $cmd_dir"
        done
        return 0
    fi
    
    for service_info in "${SERVICES[@]}"; do
        local service_name="${service_info%%:*}"
        local main_file="${service_info#*:}"
        main_file="${main_file%%:*}"
        local cmd_dir=$(dirname "$main_file")
        
        log_info "格式化 $service_name 的swagger注释..."
        if swag fmt -g "$cmd_dir"; then
            log_success "✓ $service_name swagger注释格式化完成"
        else
            log_error "$service_name swagger注释格式化失败"
            exit 1
        fi
    done
}

# 函数：生成特定服务的swagger文档
generate_service_swagger() {
    local service_name=$1
    local main_file=$2
    local output_dir=$3
    local exclude_dir=$4
    
    log_info "生成 $service_name 的swagger文档..."
    
    if [ "$DRY_RUN" = "true" ]; then
        echo "mkdir -p $output_dir"
        echo "swag init -g $main_file -o $output_dir --parseDependency --parseInternal --exclude $exclude_dir"
        return 0
    fi
    
    # 创建输出目录
    mkdir -p "$output_dir"
    
    # 生成swagger文档
    if swag init -g "$main_file" -o "$output_dir" --parseDependency --parseInternal --exclude "$exclude_dir"; then
        log_success "✓ $service_name swagger文档生成完成"
        log_info "  输出目录: $output_dir/"
    else
        log_error "$service_name swagger文档生成失败"
        exit 1
    fi
}

# 函数：生成所有服务的swagger文档
generate_all_swagger() {
    log_info "生成所有服务的swagger文档..."
    
    for service_info in "${SERVICES[@]}"; do
        local service_name="${service_info%%:*}"
        local rest="${service_info#*:}"
        local main_file="${rest%%:*}"
        rest="${rest#*:}"
        local output_dir="${rest%%:*}"
        local exclude_dir="${rest##*:}"
        
        generate_service_swagger "$service_name" "$main_file" "$output_dir" "$exclude_dir"
    done
}

# 函数：复制文档到统一位置
copy_docs() {
    log_info "整理swagger文档..."
    
    if [ "$DRY_RUN" = "true" ]; then
        echo "文档已在统一位置，无需额外复制"
        return 0
    fi
    
    # 检查是否有生成的文档
    local has_docs=false
    for service_info in "${SERVICES[@]}"; do
        local service_name="${service_info%%:*}"
        local rest="${service_info#*:}"
        rest="${rest#*:}"
        local output_dir="${rest%%:*}"
        
        if [ -d "$output_dir" ] && [ "$(ls -A "$output_dir" 2>/dev/null)" ]; then
            has_docs=true
            break
        fi
    done
    
    if [ "$has_docs" = true ]; then
        log_success "✓ swagger文档已生成并整理完毕"
        log_info "  文档位置: docs/generated/"
    else
        log_warning "未找到生成的swagger文档"
    fi
}

# 函数：清理生成的文档
clean_docs() {
    log_info "清理swagger生成的文档..."
    
    if [ "$DRY_RUN" = "true" ]; then
        echo "rm -rf docs/generated/"
        echo "find internal -path '*/docs/docs.go' -type f -delete"
        return 0
    fi
    
    # 清理统一文档目录
    if [ -d "docs/generated" ]; then
        rm -rf docs/generated/
        log_success "✓ 统一文档目录已清理"
    fi
    
    # 清理各服务内部的docs.go文件
    local cleaned_files=$(find internal -path "*/docs/docs.go" -type f -delete -print 2>/dev/null || true)
    if [ -n "$cleaned_files" ]; then
        log_success "✓ 服务内部文档文件已清理"
        echo "$cleaned_files" | sed 's/^/  /'
    fi
    
    log_success "✓ swagger文档清理完成"
}

# 函数：验证swagger配置
validate_config() {
    log_info "验证swagger配置..."
    
    log_info "检查服务配置:"
    for service_info in "${SERVICES[@]}"; do
        local service_name="${service_info%%:*}"
        local rest="${service_info#*:}"
        local main_file="${rest%%:*}"
        rest="${rest#*:}"
        local output_dir="${rest%%:*}"
        local exclude_dir="${rest##*:}"
        
        echo "  服务: $service_name"
        echo "    主文件: $main_file"
        echo "    输出目录: $output_dir"
        echo "    排除目录: $exclude_dir"
        
        if [ -f "$main_file" ]; then
            log_success "    ✅ 主文件存在"
        else
            log_error "    ❌ 主文件不存在: $main_file"
        fi
        echo ""
    done
    
    log_info "检查项目结构:"
    if [ -f "go.mod" ]; then
        log_success "✅ Go模块文件存在"
    else
        log_error "❌ Go模块文件不存在"
    fi
    
    if command -v swag &> /dev/null; then
        local swag_version=$(swag --version 2>/dev/null | head -1 || echo "unknown")
        log_success "✅ swag工具已安装: $swag_version"
    else
        log_warning "⚠️  swag工具未安装"
        log_info "可运行: $0 --setup"
    fi
    
    log_success "✅ swagger配置验证完成"
}

# 函数：标准swagger文档生成
generate_standard() {
    log_info "🚀 开始标准swagger文档生成"
    
    # 确保swag工具可用
    install_swag
    
    # 格式化swagger注释
    format_swagger
    
    # 生成所有服务文档
    generate_all_swagger
    
    # 整理文档
    copy_docs
    
    log_success "🎉 标准swagger文档生成完成！"
}

# 函数：快速开发模式
generate_dev() {
    log_info "🚀 开始快速swagger文档生成（开发模式）"
    
    # 确保swag工具可用
    install_swag
    
    # 跳过格式化，直接生成
    generate_all_swagger
    
    # 整理文档
    copy_docs
    
    log_success "🎉 快速swagger文档生成完成！"
}

# 函数：环境设置
setup_environment() {
    log_info "🔧 设置swagger开发环境"
    
    # 安装swag工具
    install_swag
    
    log_success "✓ swagger开发环境设置完成"
    echo ""
    log_info "现在可以使用以下命令:"
    echo "  $0                # 标准swagger文档生成"
    echo "  $0 --dev          # 快速开发模式生成"
    echo "  $0 --advanced format    # 只格式化注释"
    echo "  $0 --advanced switserve # 只生成特定服务"
}

# 函数：高级操作
advanced_operations() {
    local operation=$1
    
    case "$operation" in
        "format")
            install_swag
            format_swagger
            ;;
        "switserve")
            install_swag
            generate_service_swagger "switserve" "cmd/swit-serve/swit-serve.go" "docs/generated/switserve" "internal/switauth"
            ;;
        "switauth")
            install_swag
            generate_service_swagger "switauth" "cmd/swit-auth/swit-auth.go" "docs/generated/switauth" "internal/switserve"
            ;;
        "copy")
            copy_docs
            ;;
        "clean")
            clean_docs
            ;;
        "validate")
            validate_config
            ;;
        *)
            log_error "未知的高级操作: $operation"
            log_info "支持的操作: format, switserve, switauth, copy, clean, validate"
            exit 1
            ;;
    esac
}

# 函数：显示生成总结
show_summary() {
    log_info "Swagger文档生成总结:"
    echo ""
    
    if [ "$DRY_RUN" = "true" ]; then
        log_info "试运行模式 - 未实际执行命令"
        return
    fi
    
    if [ -d "docs/generated" ]; then
        log_info "生成的文档:"
        
        for service_info in "${SERVICES[@]}"; do
            local service_name="${service_info%%:*}"
            local rest="${service_info#*:}"
            rest="${rest#*:}"
            local output_dir="${rest%%:*}"
            
            if [ -d "$output_dir" ]; then
                local file_count=$(find "$output_dir" -type f | wc -l)
                if [ "$file_count" -gt 0 ]; then
                    echo "  $service_name: $file_count 个文件"
                    echo "    位置: $output_dir/"
                    # 显示主要文件
                    find "$output_dir" -name "*.json" -o -name "*.yaml" -o -name "*.go" | head -3 | sed 's/^/      /'
                fi
            fi
        done
        
        echo ""
        log_info "文档访问:"
        echo "  统一入口: docs/generated/"
        echo "  在线访问: 启动服务后访问 /swagger/index.html"
    else
        log_warning "未找到生成的文档目录"
    fi
}

# 主函数
main() {
    # 默认值
    DEV_MODE=false
    SETUP_ONLY=false
    ADVANCED_MODE=false
    ADVANCED_OPERATION=""
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
            -s|--setup)
                SETUP_ONLY=true
                shift
                ;;
            -a|--advanced)
                ADVANCED_MODE=true
                ADVANCED_OPERATION="$2"
                shift 2
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
    log_info "🚀 开始 SWIT Swagger文档管理"
    echo ""
    
    # 执行相应模式
    local start_time=$(date +%s)
    
    if [ "$SETUP_ONLY" = "true" ]; then
        setup_environment
    elif [ "$ADVANCED_MODE" = "true" ]; then
        if [ -z "$ADVANCED_OPERATION" ]; then
            log_error "高级模式需要指定操作类型"
            log_info "支持的操作: format, switserve, switauth, copy, clean, validate"
            exit 1
        fi
        advanced_operations "$ADVANCED_OPERATION"
    elif [ "$DEV_MODE" = "true" ]; then
        generate_dev
    else
        # 默认模式：标准swagger文档生成
        generate_standard
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo ""
    log_success "🎉 Swagger文档管理完成！用时 ${duration} 秒"
    
    # 显示总结
    echo ""
    show_summary
}

# 运行主函数
main "$@" 