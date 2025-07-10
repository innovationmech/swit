#!/usr/bin/env bash

# SWIT 清理管理脚本
# 统一管理项目的清理功能，包括构建输出、生成代码、测试文件等
set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置变量
PROJECT_NAME="swit"
OUTPUTDIR="_output"
API_GEN_DIR="api/gen"
DOCS_GEN_DIR="docs/generated"

# 清理目标类型定义
CLEAN_TARGETS=(
    "build:构建输出:${OUTPUTDIR}"
    "proto:Proto生成代码:${API_GEN_DIR}"
    "swagger:Swagger文档:${DOCS_GEN_DIR},internal/*/docs/docs.go"
    "test:测试文件:coverage.out,coverage.html,*.log,test.log"
    "temp:临时文件:.DS_Store,*.tmp"
    "cache:缓存文件:.go-cache,vendor"
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
SWIT 清理管理脚本

用法:
    $0 [选项]

选项:
    -h, --help              显示此帮助信息
    -d, --dev               快速清理模式 (仅构建输出)
    -s, --setup             深度清理模式 (包括缓存和依赖)
    -a, --advanced TYPE     高级清理模式 (指定清理类型)
    -l, --list              列出所有清理目标
    -n, --dry-run           试运行模式 (显示命令但不执行)
    -v, --verbose           详细输出模式

清理模式:
    默认模式      标准清理 - 删除所有生成的代码和构建产物
    --dev         快速清理 - 仅删除构建输出 (开发时常用)
    --setup       深度清理 - 删除所有内容包括缓存 (重置环境)
    --advanced    高级清理 - 精确控制特定类型清理

高级清理类型:
    build         仅清理构建输出
    proto         仅清理proto生成代码
    swagger       仅清理swagger文档
    test          仅清理测试文件
    temp          仅清理临时文件
    cache         仅清理缓存文件
    all           清理所有类型

示例:
    $0                                    # 标准清理
    $0 --dev                             # 快速清理（开发模式）
    $0 --setup                           # 深度清理（重置环境）
    $0 --advanced build                  # 仅清理构建输出
    $0 --advanced swagger                # 仅清理swagger文档
    $0 --list                           # 列出所有清理目标
    $0 --dry-run                        # 试运行模式

EOF
}

# 函数：列出所有清理目标
list_targets() {
    log_info "📋 可清理的目标类型："
    echo ""
    for target in "${CLEAN_TARGETS[@]}"; do
        IFS=":" read -r type desc paths <<< "$target"
        echo "  🎯 ${type}"
        echo "     描述: ${desc}"
        echo "     路径: ${paths}"
        echo ""
    done
}

# 函数：检查路径是否存在
path_exists() {
    local path="$1"
    [[ -e "$path" ]] || [[ -d "$path" ]] || [[ -f "$path" ]]
}

# 函数：清理指定类型
clean_type() {
    local type="$1"
    local dry_run="$2"
    local verbose="$3"
    
    log_info "🧹 清理 ${type} 类型..."
    
    for target in "${CLEAN_TARGETS[@]}"; do
        IFS=":" read -r target_type desc paths <<< "$target"
        
        if [[ "$target_type" == "$type" ]] || [[ "$type" == "all" ]]; then
            log_info "  清理 ${desc}..."
            
            IFS="," read -ra path_array <<< "$paths"
            for path in "${path_array[@]}"; do
                # 移除前后空格
                path=$(echo "$path" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
                
                if [[ "$path" == *"*"* ]]; then
                    # 处理通配符路径
                    if [[ "$dry_run" == "true" ]]; then
                        echo "    [DRY-RUN] find . -name \"$path\" -type f -delete"
                    else
                        if [[ "$verbose" == "true" ]]; then
                            log_info "    删除匹配 $path 的文件..."
                        fi
                        find . -name "$path" -type f -delete 2>/dev/null || true
                    fi
                else
                    # 处理具体路径
                    if path_exists "$path"; then
                        if [[ "$dry_run" == "true" ]]; then
                            echo "    [DRY-RUN] rm -rf $path"
                        else
                            if [[ "$verbose" == "true" ]]; then
                                log_info "    删除 $path"
                            fi
                            rm -rf "$path"
                        fi
                    else
                        if [[ "$verbose" == "true" ]]; then
                            log_warning "    $path 不存在，跳过"
                        fi
                    fi
                fi
            done
            
            if [[ "$type" != "all" ]]; then
                break
            fi
        fi
    done
}

# 函数：统计清理结果
show_summary() {
    local mode="$1"
    
    log_success "✅ ${mode}清理完成！"
    echo ""
    log_info "📊 清理统计："
    
    # 统计剩余文件
    local build_files=0
    local proto_files=0
    local swagger_files=0
    local test_files=0
    
    [[ -d "$OUTPUTDIR" ]] && build_files=$(find "$OUTPUTDIR" -type f 2>/dev/null | wc -l || echo 0)
    [[ -d "$API_GEN_DIR" ]] && proto_files=$(find "$API_GEN_DIR" -type f 2>/dev/null | wc -l || echo 0)
    [[ -d "$DOCS_GEN_DIR" ]] && swagger_files=$(find "$DOCS_GEN_DIR" -type f 2>/dev/null | wc -l || echo 0)
    test_files=$(find . -name "coverage.*" -o -name "*.log" -o -name "test.log" 2>/dev/null | wc -l || echo 0)
    
    echo "  构建文件: ${build_files} 个"
    echo "  Proto文件: ${proto_files} 个"
    echo "  Swagger文件: ${swagger_files} 个"
    echo "  测试文件: ${test_files} 个"
    echo ""
    
    if [[ $((build_files + proto_files + swagger_files + test_files)) -eq 0 ]]; then
        log_success "🎉 所有目标文件已清理完成！"
    else
        log_warning "⚠️  仍有文件残留，可能需要手动清理"
    fi
}

# 主函数
main() {
    local mode=""
    local advanced_type=""
    local dry_run="false"
    local verbose="false"
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -d|--dev)
                mode="dev"
                shift
                ;;
            -s|--setup)
                mode="setup"
                shift
                ;;
            -a|--advanced)
                mode="advanced"
                advanced_type="$2"
                shift 2
                ;;
            -l|--list)
                list_targets
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
            *)
                log_error "未知选项: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    # 显示开始信息
    log_info "🚀 SWIT 清理管理脚本启动"
    
    if [[ "$dry_run" == "true" ]]; then
        log_warning "⚠️  试运行模式 - 仅显示命令，不执行实际操作"
    fi
    
    echo ""
    
    # 根据模式执行清理
    case "$mode" in
        "dev")
            log_info "🔥 快速清理模式（开发用）"
            clean_type "build" "$dry_run" "$verbose"
            show_summary "快速"
            ;;
        "setup")
            log_info "🔄 深度清理模式（重置环境）"
            clean_type "all" "$dry_run" "$verbose"
            show_summary "深度"
            ;;
        "advanced")
            if [[ -z "$advanced_type" ]]; then
                log_error "高级模式需要指定清理类型"
                echo "支持的类型: build, proto, swagger, test, temp, cache, all"
                exit 1
            fi
            log_info "⚙️  高级清理模式 - 类型: ${advanced_type}"
            clean_type "$advanced_type" "$dry_run" "$verbose"
            show_summary "高级"
            ;;
        *)
            # 默认标准清理模式
            log_info "🧹 标准清理模式"
            log_info "  清理所有生成的代码和构建产物..."
            clean_type "build" "$dry_run" "$verbose"
            clean_type "proto" "$dry_run" "$verbose"
            clean_type "swagger" "$dry_run" "$verbose"
            clean_type "test" "$dry_run" "$verbose"
            clean_type "temp" "$dry_run" "$verbose"
            show_summary "标准"
            ;;
    esac
    
    echo ""
    log_info "💡 提示："
    log_info "  clean-dev     # 快速清理（仅构建输出）"
    log_info "  clean-setup   # 深度清理（重置环境）"
    log_info "  clean-advanced # 高级清理（精确控制）"
}

# 脚本入口
main "$@" 