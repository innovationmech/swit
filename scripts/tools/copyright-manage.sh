#!/usr/bin/env bash

# SWIT Copyright管理脚本
# 统一管理版权声明的检查、添加、更新等功能
set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置变量
PROJECT_NAME="swit"
BOILERPLATE_FILE="scripts/boilerplate.txt"

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
SWIT Copyright管理脚本

用法:
    $0 [选项]

选项:
    -h, --help              显示此帮助信息
    -c, --check             检查版权声明（不修改文件）
    -s, --setup             初始设置 - 为所有文件添加版权声明
    -a, --advanced OPERATION 高级操作 - 精确控制特定操作
    --dry-run              只显示将要执行的操作，不实际执行

模式:
    无参数                  标准版权管理（推荐）- 检查并自动修复
    --check                检查模式 - 只检查，不修改文件
    --setup                设置模式 - 为新项目添加版权声明
    --advanced             高级模式 - 执行特定操作

高级操作类型:
    force                  强制更新所有文件的版权声明
    debug                  调试版权检测过程（故障排除）
    files                  显示哪些文件被包含/排除
    validate               验证版权声明配置

示例:
    $0                      # 标准版权管理（检查+自动修复）
    $0 --check             # 只检查版权声明
    $0 --setup             # 为新项目设置版权声明
    $0 --advanced force    # 强制更新所有文件
    $0 --advanced debug    # 调试版权检测
    $0 --dry-run           # 查看将要执行的操作

环境变量:
    BOILERPLATE_FILE       版权模板文件路径 (默认: scripts/boilerplate.txt)
EOF
}

# 函数：检查是否在项目根目录
check_project_root() {
    if [ ! -f "go.mod" ]; then
        log_error "未找到 go.mod 文件，请在项目根目录运行此脚本"
        exit 1
    fi
    
    if [ ! -f "$BOILERPLATE_FILE" ]; then
        log_error "未找到版权模板文件: $BOILERPLATE_FILE"
        exit 1
    fi
}

# 函数：获取所有Go文件（排除生成的代码）
get_go_files() {
    find . -name '*.go' \
        -not -path './api/gen/*' \
        -not -path './_output/*' \
        -not -path './vendor/*' \
        -not -path './internal/*/docs/docs.go' \
        -not -path './docs/generated/*' \
        2>/dev/null || true
}

# 函数：获取标准版权声明哈希值
get_standard_copyright_hash() {
    awk '{if($0=="") print "//"; else print "// " $0}' "$BOILERPLATE_FILE" | shasum -a 256 | cut -d' ' -f1
}

# 函数：获取文件的版权声明哈希值
get_file_copyright_hash() {
    local file=$1
    awk '
    /^\/\/ Copyright/ { found=1 }
    found && /^$/ { exit }
    found && /^\/\// { print }
    found && /^[^\/]/ { exit }
    ' "$file" | shasum -a 256 | cut -d' ' -f1
}

# 函数：检查文件是否有版权声明
has_copyright() {
    local file=$1
    grep -q "Copyright" "$file" 2>/dev/null
}

# 函数：检查版权声明是否过期
is_copyright_outdated() {
    local file=$1
    local standard_hash=$(get_standard_copyright_hash)
    local file_hash=$(get_file_copyright_hash "$file")
    [ "$file_hash" != "$standard_hash" ]
}

# 函数：检查版权声明
check_copyright() {
    log_info "🔍 检查 Go 文件版权声明（排除生成代码）"
    
    local go_files=($(get_go_files))
    local missing_files=()
    local outdated_files=()
    local total_files=${#go_files[@]}
    
    if [ $total_files -eq 0 ]; then
        log_warning "未找到需要检查的Go文件"
        return 0
    fi
    
    log_info "检查 $total_files 个文件..."
    
    for file in "${go_files[@]}"; do
        if ! has_copyright "$file"; then
            missing_files+=("$file")
        elif is_copyright_outdated "$file"; then
            outdated_files+=("$file")
        fi
    done
    
    # 显示结果
    if [ ${#missing_files[@]} -gt 0 ]; then
        echo ""
        log_error "❌ 以下 ${#missing_files[@]} 个文件缺少版权声明:"
        printf "  %s\n" "${missing_files[@]}"
    fi
    
    if [ ${#outdated_files[@]} -gt 0 ]; then
        echo ""
        log_warning "⚠️  以下 ${#outdated_files[@]} 个文件版权声明过期:"
        printf "  %s\n" "${outdated_files[@]}"
    fi
    
    if [ ${#missing_files[@]} -eq 0 ] && [ ${#outdated_files[@]} -eq 0 ]; then
        echo ""
        log_success "✅ 所有 Go 文件都有最新的版权声明"
    fi
    
    echo ""
    log_info "📊 文件统计: $total_files 个文件已检查"
    
    # 返回状态
    if [ ${#missing_files[@]} -gt 0 ] || [ ${#outdated_files[@]} -gt 0 ]; then
        return 1
    fi
    return 0
}

# 函数：添加版权声明
add_copyright() {
    local file=$1
    
    if [ "$DRY_RUN" = "true" ]; then
        echo "添加版权声明到: $file"
        return 0
    fi
    
    log_info "添加版权声明到: $file"
    
    # 生成版权头到临时文件
    local temp_copyright=$(mktemp)
    awk '{if($0=="") print "//"; else print "// " $0}' "$BOILERPLATE_FILE" > "$temp_copyright"
    echo "" >> "$temp_copyright"
    
    # 如果文件顶部存在构建标签（//go:build 或 // +build），保持其在最顶部
    # 将版权头插入在构建标签块之后，否则直接头插
    if grep -qE '^(//go:build|// \+build)' "$file" 2>/dev/null; then
        local temp_output=$(mktemp)
        awk -v hdrfile="$temp_copyright" '
            BEGIN { inbuild=1; inserted=0 }
            # 顶部连续的构建标签
            NR==1 { }
            inbuild==1 && (/^\/\/go:build/ || /^\/\/ \+build/) { print; next }
            # 构建标签后的空行（必须保留）
            inbuild==1 && /^\s*$/ { print; inbuild=0; 
                while ((getline line < hdrfile) > 0) print line; close(hdrfile);
                inserted=1; next }
            # 一旦遇到非构建标签的第一行，先插入版权头再打印此行
            inbuild==1 {
                inbuild=0;
                while ((getline line < hdrfile) > 0) print line; close(hdrfile);
                inserted=1; print $0; next
            }
            { print }
        ' "$file" > "$temp_output"
        mv "$temp_output" "$file"
    else
        # 无构建标签：直接头插
        local temp_file=$(mktemp)
        cat "$temp_copyright" "$file" > "$temp_file"
        mv "$temp_file" "$file"
    fi
    
    rm -f "$temp_copyright"
}

# 函数：更新版权声明
update_copyright() {
    local file=$1
    
    if [ "$DRY_RUN" = "true" ]; then
        echo "更新版权声明: $file"
        return 0
    fi
    
    log_info "更新版权声明: $file"
    
    # 生成新的版权声明
    local temp_copyright=$(mktemp)
    awk '{if($0=="") print "//"; else print "// " $0}' "$BOILERPLATE_FILE" > "$temp_copyright"
    echo "" >> "$temp_copyright"
    
    # 在原位置替换旧版权块，保护 package 声明不被删除
    local temp_output=$(mktemp)
    awk -v hdrfile="$temp_copyright" '
        BEGIN { skipping=0; inserted=0 }
        # 开始检测到版权块
        /^\s*\/\/ Copyright/ && inserted==0 {
            skipping=1; next
        }
        # 仍在版权块中（跳过所有 // 注释行 和 空行）
        skipping==1 && (/^\s*\/\// || /^\s*$/) { next }
        # 版权块后的第一行：先输出新版权，再输出当前行
        skipping==1 {
            skipping=0; inserted=1;
            while ((getline line < hdrfile) > 0) print line
            close(hdrfile)
            print $0
            next
        }
        { print }
    ' "$file" > "$temp_output"
    
    if grep -q "^// Copyright" "$file" 2>/dev/null; then
        mv "$temp_output" "$file"
    else
        # 未找到旧版权块：按添加逻辑插入（照顾构建标签）
        rm -f "$temp_output"
        add_copyright "$file"
    fi
    
    rm -f "$temp_copyright"
}

# 函数：标准版权管理（检查+自动修复）
manage_copyright() {
    log_info "🚀 开始版权声明管理"
    
    local go_files=($(get_go_files))
    local missing_files=()
    local outdated_files=()
    
    # 检查文件状态
    for file in "${go_files[@]}"; do
        if ! has_copyright "$file"; then
            missing_files+=("$file")
        elif is_copyright_outdated "$file"; then
            outdated_files+=("$file")
        fi
    done
    
    # 显示当前状态
    echo ""
    check_copyright || true
    
    # 自动修复
    if [ ${#missing_files[@]} -gt 0 ]; then
        echo ""
        log_info "🔧 发现 ${#missing_files[@]} 个文件缺少版权声明，正在添加..."
        for file in "${missing_files[@]}"; do
            add_copyright "$file"
        done
        log_success "✅ 版权声明已添加"
    fi
    
    if [ ${#outdated_files[@]} -gt 0 ]; then
        echo ""
        log_info "🔄 发现 ${#outdated_files[@]} 个文件版权声明过期，正在更新..."
        for file in "${outdated_files[@]}"; do
            update_copyright "$file"
        done
        log_success "✅ 版权声明已更新"
    fi
    
    # 最终检查
    if [ ${#missing_files[@]} -eq 0 ] && [ ${#outdated_files[@]} -eq 0 ]; then
        echo ""
        log_success "🎉 所有文件的版权声明都是最新的！"
        return 0
    else
        echo ""
        log_info "🔍 重新检查结果..."
        check_copyright
        return $?
    fi
}

# 函数：初始设置（为所有文件添加版权声明）
setup_copyright() {
    log_info "🔧 初始版权声明设置"
    
    local go_files=($(get_go_files))
    local processed=0
    
    if [ ${#go_files[@]} -eq 0 ]; then
        log_warning "未找到需要处理的Go文件"
        return 0
    fi
    
    log_info "为 ${#go_files[@]} 个文件设置版权声明..."
    
    for file in "${go_files[@]}"; do
        if has_copyright "$file"; then
            if is_copyright_outdated "$file"; then
                update_copyright "$file"
                ((processed++))
            fi
        else
            add_copyright "$file"
            ((processed++))
        fi
    done
    
    echo ""
    if [ $processed -gt 0 ]; then
        log_success "✅ 已处理 $processed 个文件"
    else
        log_success "✅ 所有文件都已有最新的版权声明"
    fi
    
    # 显示最终状态
    echo ""
    check_copyright
}

# 函数：高级操作
advanced_operations() {
    local operation=$1
    
    case "$operation" in
        "force")
            force_update_all
            ;;
        "debug")
            debug_copyright
            ;;
        "files")
            show_file_info
            ;;
        "validate")
            validate_config
            ;;
        *)
            log_error "未知的高级操作: $operation"
            log_info "支持的操作: force, debug, files, validate"
            exit 1
            ;;
    esac
}

# 函数：强制更新所有文件
force_update_all() {
    log_info "⚡ 强制更新所有 Go 文件的版权声明"
    
    local go_files=($(get_go_files))
    
    if [ ${#go_files[@]} -eq 0 ]; then
        log_warning "未找到需要处理的Go文件"
        return 0
    fi
    
    log_warning "这将更新所有 ${#go_files[@]} 个文件的版权声明"
    
    if [ "$DRY_RUN" != "true" ]; then
        read -p "确认继续? [y/N]: " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "操作已取消"
            return 0
        fi
    fi
    
    for file in "${go_files[@]}"; do
        if has_copyright "$file"; then
            update_copyright "$file"
        else
            add_copyright "$file"
        fi
    done
    
    log_success "✅ 强制更新完成"
}

# 函数：调试版权检测
debug_copyright() {
    log_info "🔍 调试版权检测过程"
    
    local standard_hash=$(get_standard_copyright_hash)
    echo ""
    log_info "标准版权声明哈希值: $standard_hash"
    echo ""
    log_info "标准版权声明内容:"
    sed 's/^/\/\/ /' "$BOILERPLATE_FILE" | sed 's/^/  /'
    
    echo ""
    log_info "检查前3个文件的版权声明:"
    
    local go_files=($(get_go_files))
    local count=0
    
    for file in "${go_files[@]}"; do
        if [ $count -ge 3 ]; then
            break
        fi
        
        echo ""
        echo "文件: $file"
        
        if has_copyright "$file"; then
            local file_hash=$(get_file_copyright_hash "$file")
            echo "文件哈希值: $file_hash"
            echo "文件版权内容:"
            awk '/^\/\/ Copyright/{found=1} found && !/^\/\//{found=0; exit} found{print}' "$file" | sed 's/^/  /'
            
            if [ "$file_hash" = "$standard_hash" ]; then
                echo "匹配: ✅ YES"
            else
                echo "匹配: ❌ NO"
            fi
        else
            echo "状态: 无版权声明"
        fi
        
        ((count++))
    done
}

# 函数：显示文件信息
show_file_info() {
    log_info "📋 版权管理包含的文件信息"
    
    local go_files=($(get_go_files))
    local missing_count=0
    local outdated_count=0
    
    for file in "${go_files[@]}"; do
        if ! has_copyright "$file"; then
            ((missing_count++))
        elif is_copyright_outdated "$file"; then
            ((outdated_count++))
        fi
    done
    
    echo ""
    log_info "📊 统计信息:"
    echo "  总文件数:       ${#go_files[@]}"
    echo "  缺少版权:       $missing_count"
    echo "  过期版权:       $outdated_count"
    echo "  正常文件:       $((${#go_files[@]} - missing_count - outdated_count))"
    
    echo ""
    log_info "🚫 排除的目录:"
    echo "  - api/gen/*               (生成的 gRPC 代码)"
    echo "  - _output/*               (构建输出)"
    echo "  - vendor/*                (第三方依赖)"
    echo "  - internal/*/docs/docs.go (生成的 Swagger 文档)"
    
    echo ""
    log_info "📂 示例排除的文件:"
    find . -name '*.go' \( -path './api/gen/*' -o -path './_output/*' -o -path './vendor/*' -o -path './internal/*/docs/docs.go' \) 2>/dev/null | head -5 | sed 's/^/  /' || echo "  (暂无排除的文件)"
    
    if [ ${#go_files[@]} -gt 0 ]; then
        echo ""
        log_info "📄 包含的文件示例:"
        printf "  %s\n" "${go_files[@]}" | head -10
        if [ ${#go_files[@]} -gt 10 ]; then
            echo "  ... 还有 $((${#go_files[@]} - 10)) 个文件"
        fi
    fi
}

# 函数：验证配置
validate_config() {
    log_info "🔍 验证版权声明配置"
    
    echo ""
    log_info "检查版权模板文件:"
    if [ -f "$BOILERPLATE_FILE" ]; then
        log_success "✅ 版权模板文件存在: $BOILERPLATE_FILE"
        echo ""
        log_info "模板内容预览:"
        head -5 "$BOILERPLATE_FILE" | sed 's/^/  /'
        
        local line_count=$(wc -l < "$BOILERPLATE_FILE")
        echo "  ... 共 $line_count 行"
    else
        log_error "❌ 版权模板文件不存在: $BOILERPLATE_FILE"
        return 1
    fi
    
    echo ""
    log_info "检查项目结构:"
    if [ -f "go.mod" ]; then
        log_success "✅ Go模块文件存在"
    else
        log_error "❌ Go模块文件不存在"
        return 1
    fi
    
    local go_files=($(get_go_files))
    if [ ${#go_files[@]} -gt 0 ]; then
        log_success "✅ 找到 ${#go_files[@]} 个Go文件"
    else
        log_warning "⚠️  未找到Go文件"
    fi
    
    echo ""
    log_info "检查版权声明标准哈希值:"
    local standard_hash=$(get_standard_copyright_hash)
    echo "  哈希值: $standard_hash"
    
    echo ""
    log_success "✅ 配置验证完成"
}

# 函数：显示处理总结
show_summary() {
    if [ "$DRY_RUN" = "true" ]; then
        log_info "试运行模式 - 未实际修改文件"
        return
    fi
    
    log_info "版权声明管理总结:"
    echo ""
    
    local go_files=($(get_go_files))
    if [ ${#go_files[@]} -gt 0 ]; then
        log_info "处理的文件:"
        echo "  总文件数: ${#go_files[@]}"
        
        local with_copyright=0
        for file in "${go_files[@]}"; do
            if has_copyright "$file"; then
                with_copyright=$((with_copyright + 1))
            fi
        done
        
        echo "  有版权声明: $with_copyright"
        echo "  覆盖率: $(( with_copyright * 100 / ${#go_files[@]} ))%"
    fi
}

# 主函数
main() {
    # 默认值
    CHECK_ONLY=false
    SETUP_MODE=false
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
            -c|--check)
                CHECK_ONLY=true
                shift
                ;;
            -s|--setup)
                SETUP_MODE=true
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
    log_info "🚀 开始 SWIT Copyright管理"
    echo ""
    
    # 执行相应模式
    local start_time=$(date +%s)
    local exit_code=0
    
    if [ "$CHECK_ONLY" = "true" ]; then
        check_copyright
        exit_code=$?
    elif [ "$SETUP_MODE" = "true" ]; then
        setup_copyright
        exit_code=$?
    elif [ "$ADVANCED_MODE" = "true" ]; then
        if [ -z "$ADVANCED_OPERATION" ]; then
            log_error "高级模式需要指定操作类型"
            log_info "支持的操作: force, debug, files, validate"
            exit 1
        fi
        advanced_operations "$ADVANCED_OPERATION"
        exit_code=$?
    else
        # 默认模式：标准版权管理
        manage_copyright
        exit_code=$?
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo ""
    log_success "🎉 Copyright管理完成！用时 ${duration} 秒"
    
    # 显示总结
    echo ""
    show_summary
    
    # 返回适当的退出码
    exit $exit_code
}

# 运行主函数
main "$@" 
