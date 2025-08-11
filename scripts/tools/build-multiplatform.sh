#!/usr/bin/env bash

# SWIT 多平台构建脚本
# 支持交叉编译到多个操作系统和架构
set -e

# 检查 Bash 版本
if [ "${BASH_VERSION%%.*}" -lt 4 ]; then
    echo "错误: 此脚本需要 Bash 4.0 或更高版本"
    echo "当前版本: $BASH_VERSION"
    echo "在 macOS 上，可以通过 Homebrew 安装新版本: brew install bash"
    exit 1
fi

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置变量
PROJECT_NAME="swit"
VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 输出目录
OUTPUT_DIR="_output"
DIST_DIR="${OUTPUT_DIR}/dist"
BUILD_DIR="${OUTPUT_DIR}/build"

# 支持的平台和架构
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/arm64"
)

# 服务定义 (name:path 格式)
SERVICES=(
    "swit-serve:cmd/swit-serve/swit-serve.go"
    "swit-auth:cmd/swit-auth/swit-auth.go"
    "switctl:cmd/switctl/switctl.go"
)

# 构建标志
LDFLAGS="-w -s"
BUILD_FLAGS="-trimpath"

# 函数：获取服务的主文件路径
get_service_main_file() {
    local service_name=$1
    for service in "${SERVICES[@]}"; do
        local name="${service%%:*}"
        local path="${service##*:}"
        if [ "$name" = "$service_name" ]; then
            echo "$path"
            return 0
        fi
    done
    return 1
}

# 函数：获取所有服务名称
get_all_service_names() {
    for service in "${SERVICES[@]}"; do
        echo "${service%%:*}"
    done
}

# 函数：检查服务是否存在
service_exists() {
    local service_name=$1
    for service in "${SERVICES[@]}"; do
        local name="${service%%:*}"
        if [ "$name" = "$service_name" ]; then
            return 0
        fi
    done
    return 1
}

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
SWIT 多平台构建脚本

用法:
    $0 [选项] [服务名...]

选项:
    -h, --help              显示此帮助信息
    -v, --version VERSION   设置版本号 (默认: git描述或'dev')
    -p, --platform PLATFORM 指定构建平台 (格式: os/arch)
    -s, --service SERVICE   构建指定服务 (可多次使用)
    -o, --output DIR        输出目录 (默认: ${OUTPUT_DIR})
    --ldflags FLAGS         自定义链接标志
    --build-flags FLAGS     自定义构建标志
    --clean                 构建前清理输出目录
    --archive               构建完成后创建压缩包
    --checksum              生成校验和文件
    --parallel              并行构建 (实验性)
    --dry-run              只显示构建命令，不实际执行

支持的平台:
$(printf "    %s\n" "${PLATFORMS[@]}")

支持的服务:
$(get_all_service_names | sed 's/^/    /')

示例:
    $0                                    # 构建所有服务的所有平台
    $0 -s swit-serve -s swit-auth         # 只构建指定服务
    $0 -p linux/amd64 -p darwin/amd64     # 只构建指定平台
    $0 --clean --archive --checksum       # 清理、构建、打包并生成校验和
    $0 --dry-run                          # 查看将要执行的构建命令

环境变量:
    VERSION                 构建版本号
    GOOS                   目标操作系统
    GOARCH                 目标架构
    CGO_ENABLED            CGO设置 (默认: 0)
EOF
}

# 函数：检查依赖
check_dependencies() {
    log_info "检查构建依赖..."
    
    if ! command -v go &> /dev/null; then
        log_error "Go 未安装或不在 PATH 中"
        exit 1
    fi
    
    if [ ! -f "go.mod" ]; then
        log_error "未找到 go.mod 文件，请在项目根目录运行此脚本"
        exit 1
    fi
    
    # 检查 Go 版本
    GO_VERSION=$(go version | cut -d' ' -f3 | sed 's/go//')
    log_info "Go 版本: ${GO_VERSION}"
    
    log_success "依赖检查完成"
}

# 函数：准备构建
prepare_build() {
    log_info "准备构建环境..."
    
    # 创建输出目录
    mkdir -p "${BUILD_DIR}" "${DIST_DIR}"
    
    # 运行 go mod tidy
    log_info "运行 go mod tidy..."
    go mod tidy
    
    log_success "构建环境准备完成"
}

# 函数：构建单个服务的单个平台
build_service_platform() {
    local service=$1
    local platform=$2
    local main_file=$(get_service_main_file "$service")
    
    if [ -z "$main_file" ]; then
        log_error "未知服务: $service"
        return 1
    fi
    
    local os_arch=(${platform//\// })
    local target_os=${os_arch[0]}
    local target_arch=${os_arch[1]}
    
    # 输出文件名
    local output_name="${service}"
    if [ "$target_os" = "windows" ]; then
        output_name="${service}.exe"
    fi
    
    # 创建带版本和平台标识的文件名用于发布
    local release_name="${service}-${VERSION}-${target_os}-${target_arch}"
    if [ "$target_os" = "windows" ]; then
        release_name="${service}-${VERSION}-${target_os}-${target_arch}.exe"
    fi
    
    local output_dir="${BUILD_DIR}/${service}/${platform}"
    local output_file="${output_dir}/${output_name}"
    local release_file="${BUILD_DIR}/${release_name}"
    
    mkdir -p "$output_dir"
    
    # 构建 ldflags
    local full_ldflags="${LDFLAGS}"
    full_ldflags="${full_ldflags} -X main.version=${VERSION}"
    full_ldflags="${full_ldflags} -X main.buildTime=${BUILD_TIME}"
    full_ldflags="${full_ldflags} -X main.gitCommit=${GIT_COMMIT}"
    
    log_info "构建 ${service} for ${platform}..."
    
    if [ "$DRY_RUN" = "true" ]; then
        echo "CGO_ENABLED=0 GOOS=${target_os} GOARCH=${target_arch} go build ${BUILD_FLAGS} -ldflags \"${full_ldflags}\" -o ${output_file} ${main_file}"
        return 0
    fi
    
    # 执行构建
    if CGO_ENABLED=0 GOOS=${target_os} GOARCH=${target_arch} go build ${BUILD_FLAGS} -ldflags "${full_ldflags}" -o "${output_file}" "${main_file}"; then
        log_success "✓ ${service} (${platform}) 构建完成"
        
        # 创建带版本和平台标识的副本用于发布
        cp "${output_file}" "${release_file}"
        
        # 显示文件信息
        if [ -f "$output_file" ]; then
            local file_size=$(du -h "$output_file" | cut -f1)
            log_info "  文件大小: ${file_size}"
            log_info "  发布文件: ${release_file}"
        fi
        
        return 0
    else
        log_error "✗ ${service} (${platform}) 构建失败"
        return 1
    fi
}

# 函数：并行构建
build_parallel() {
    local services=("$@")
    local pids=()
    local failures=0
    
    log_info "启动并行构建..."
    
    for service in "${services[@]}"; do
        for platform in "${SELECTED_PLATFORMS[@]}"; do
            (
                build_service_platform "$service" "$platform"
                echo $? > "${BUILD_DIR}/.${service}_${platform//\//_}.status"
            ) &
            pids+=($!)
        done
    done
    
    # 等待所有构建完成
    for pid in "${pids[@]}"; do
        wait "$pid"
    done
    
    # 检查构建结果
    for service in "${services[@]}"; do
        for platform in "${SELECTED_PLATFORMS[@]}"; do
            local status_file="${BUILD_DIR}/.${service}_${platform//\//_}.status"
            if [ -f "$status_file" ]; then
                local status=$(cat "$status_file")
                rm -f "$status_file"
                if [ "$status" != "0" ]; then
                    ((failures++))
                fi
            fi
        done
    done
    
    return $failures
}

# 函数：顺序构建
build_sequential() {
    local services=("$@")
    local failures=0
    
    for service in "${services[@]}"; do
        for platform in "${SELECTED_PLATFORMS[@]}"; do
            if ! build_service_platform "$service" "$platform"; then
                ((failures++))
            fi
        done
    done
    
    return $failures
}

# 函数：创建压缩包
create_archives() {
    log_info "创建发布压缩包..."
    
    for service in "${SELECTED_SERVICES[@]}"; do
        for platform in "${SELECTED_PLATFORMS[@]}"; do
            local os_arch=(${platform//\// })
            local target_os=${os_arch[0]}
            local target_arch=${os_arch[1]}
            
            local build_dir="${BUILD_DIR}/${service}/${platform}"
            local output_name="${service}"
            if [ "$target_os" = "windows" ]; then
                output_name="${service}.exe"
            fi
            
            if [ -f "${build_dir}/${output_name}" ]; then
                local archive_name="${service}-${VERSION}-${target_os}-${target_arch}"
                local archive_dir="${DIST_DIR}/${archive_name}"
                
                # 创建发布目录
                mkdir -p "$archive_dir"
                
                # 复制二进制文件
                cp "${build_dir}/${output_name}" "$archive_dir/"
                
                # 复制文档（如果存在）
                [ -f "README.md" ] && cp "README.md" "$archive_dir/"
                [ -f "README-CN.md" ] && cp "README-CN.md" "$archive_dir/"
                [ -f "LICENSE" ] && cp "LICENSE" "$archive_dir/"
                
                # 创建版本信息文件
                cat > "${archive_dir}/VERSION" << EOF
Service: ${service}
Version: ${VERSION}
Build Time: ${BUILD_TIME}
Git Commit: ${GIT_COMMIT}
Platform: ${target_os}/${target_arch}
EOF
                
                # 创建压缩包
                cd "${DIST_DIR}"
                if [ "$target_os" = "windows" ]; then
                    if command -v zip &> /dev/null; then
                        zip -r "${archive_name}.zip" "${archive_name}/" > /dev/null
                        log_success "✓ 创建 ${archive_name}.zip"
                    else
                        tar -czf "${archive_name}.tar.gz" "${archive_name}/"
                        log_success "✓ 创建 ${archive_name}.tar.gz (zip不可用)"
                    fi
                else
                    tar -czf "${archive_name}.tar.gz" "${archive_name}/"
                    log_success "✓ 创建 ${archive_name}.tar.gz"
                fi
                cd - > /dev/null
                
                # 删除临时目录
                rm -rf "$archive_dir"
            fi
        done
    done
}

# 函数：生成校验和
generate_checksums() {
    log_info "生成校验和文件..."
    
    local checksum_file="${DIST_DIR}/checksums.txt"
    
    if command -v sha256sum &> /dev/null; then
        cd "${DIST_DIR}"
        sha256sum *.tar.gz *.zip 2>/dev/null > checksums.txt || true
        cd - > /dev/null
    elif command -v shasum &> /dev/null; then
        cd "${DIST_DIR}"
        shasum -a 256 *.tar.gz *.zip 2>/dev/null > checksums.txt || true
        cd - > /dev/null
    else
        log_warning "未找到 sha256sum 或 shasum 命令，跳过校验和生成"
        return
    fi
    
    if [ -f "$checksum_file" ] && [ -s "$checksum_file" ]; then
        log_success "✓ 校验和文件已生成: $checksum_file"
    fi
}

# 函数：显示构建总结
show_summary() {
    log_info "构建总结:"
    echo ""
    
    log_info "构建配置:"
    echo "  版本: ${VERSION}"
    echo "  Git提交: ${GIT_COMMIT}"
    echo "  构建时间: ${BUILD_TIME}"
    echo "  输出目录: ${OUTPUT_DIR}"
    echo ""
    
    log_info "构建的服务:"
    printf "  %s\n" "${SELECTED_SERVICES[@]}"
    echo ""
    
    log_info "目标平台:"
    printf "  %s\n" "${SELECTED_PLATFORMS[@]}"
    echo ""
    
    if [ "$DRY_RUN" != "true" ]; then
        log_info "构建产物:"
        
        # 显示二进制文件
        if [ -d "${BUILD_DIR}" ]; then
            find "${BUILD_DIR}" -type f -name "*" ! -name ".*" | while read -r file; do
                local size=$(du -h "$file" | cut -f1)
                echo "  ${file} (${size})"
            done
        fi
        
        # 显示压缩包
        if [ -d "${DIST_DIR}" ] && [ "$(ls -A "${DIST_DIR}" 2>/dev/null)" ]; then
            echo ""
            log_info "发布包:"
            find "${DIST_DIR}" -type f \( -name "*.tar.gz" -o -name "*.zip" \) | while read -r file; do
                local size=$(du -h "$file" | cut -f1)
                echo "  ${file} (${size})"
            done
            
            if [ -f "${DIST_DIR}/checksums.txt" ]; then
                echo "  ${DIST_DIR}/checksums.txt"
            fi
        fi
    fi
}

# 主函数
main() {
    # 默认值
    SELECTED_PLATFORMS=("${PLATFORMS[@]}")
    SELECTED_SERVICES=($(get_all_service_names))
    CLEAN_BUILD=false
    CREATE_ARCHIVE=false
    GENERATE_CHECKSUM=false
    PARALLEL_BUILD=false
    DRY_RUN=false
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -v|--version)
                VERSION="$2"
                shift 2
                ;;
            -p|--platform)
                if [ -z "${CUSTOM_PLATFORMS:-}" ]; then
                    CUSTOM_PLATFORMS=()
                fi
                CUSTOM_PLATFORMS+=("$2")
                shift 2
                ;;
            -s|--service)
                if [ -z "${CUSTOM_SERVICES:-}" ]; then
                    CUSTOM_SERVICES=()
                fi
                CUSTOM_SERVICES+=("$2")
                shift 2
                ;;
            -o|--output)
                OUTPUT_DIR="$2"
                DIST_DIR="${OUTPUT_DIR}/dist"
                BUILD_DIR="${OUTPUT_DIR}/build"
                shift 2
                ;;
            --ldflags)
                LDFLAGS="$2"
                shift 2
                ;;
            --build-flags)
                BUILD_FLAGS="$2"
                shift 2
                ;;
            --clean)
                CLEAN_BUILD=true
                shift
                ;;
            --archive)
                CREATE_ARCHIVE=true
                shift
                ;;
            --checksum)
                GENERATE_CHECKSUM=true
                shift
                ;;
            --parallel)
                PARALLEL_BUILD=true
                shift
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            *)
                # 尝试作为服务名处理
                if service_exists "$1"; then
                    if [ -z "${CUSTOM_SERVICES:-}" ]; then
                        CUSTOM_SERVICES=()
                    fi
                    CUSTOM_SERVICES+=("$1")
                else
                    log_error "未知参数: $1"
                    show_help
                    exit 1
                fi
                shift
                ;;
        esac
    done
    
    # 应用自定义选择
    if [ -n "${CUSTOM_PLATFORMS:-}" ]; then
        SELECTED_PLATFORMS=("${CUSTOM_PLATFORMS[@]}")
    fi
    
    if [ -n "${CUSTOM_SERVICES:-}" ]; then
        SELECTED_SERVICES=("${CUSTOM_SERVICES[@]}")
    fi
    
    # 验证选择的平台
    for platform in "${SELECTED_PLATFORMS[@]}"; do
        local valid=false
        for supported_platform in "${PLATFORMS[@]}"; do
            if [ "$platform" = "$supported_platform" ]; then
                valid=true
                break
            fi
        done
        if [ "$valid" = false ]; then
            log_error "不支持的平台: $platform"
            log_info "支持的平台: ${PLATFORMS[*]}"
            exit 1
        fi
    done
    
    # 验证选择的服务
    for service in "${SELECTED_SERVICES[@]}"; do
        if ! service_exists "$service"; then
            log_error "未知服务: $service"
            log_info "支持的服务: $(get_all_service_names | tr '\n' ' ')"
            exit 1
        fi
    done
    
    # 开始构建流程
    echo ""
    log_info "🚀 开始 SWIT 多平台构建"
    echo ""
    
    # 检查依赖
    check_dependencies
    
    # 清理构建目录
    if [ "$CLEAN_BUILD" = true ]; then
        log_info "清理构建目录..."
        rm -rf "${OUTPUT_DIR}"
        log_success "构建目录已清理"
    fi
    
    # 准备构建
    prepare_build
    
    # 显示构建计划
    if [ "$DRY_RUN" = true ]; then
        log_info "构建计划 (试运行模式):"
    else
        log_info "开始构建:"
    fi
    echo "  服务: ${SELECTED_SERVICES[*]}"
    echo "  平台: ${SELECTED_PLATFORMS[*]}"
    echo ""
    
    # 执行构建
    local start_time=$(date +%s)
    local failures=0
    
    if [ "$PARALLEL_BUILD" = true ]; then
        build_parallel "${SELECTED_SERVICES[@]}"
        failures=$?
    else
        build_sequential "${SELECTED_SERVICES[@]}"
        failures=$?
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo ""
    if [ $failures -eq 0 ]; then
        log_success "🎉 所有构建任务完成！用时 ${duration} 秒"
    else
        log_error "❌ ${failures} 个构建任务失败"
    fi
    
    # 后处理
    if [ "$DRY_RUN" != "true" ] && [ $failures -eq 0 ]; then
        if [ "$CREATE_ARCHIVE" = true ]; then
            echo ""
            create_archives
        fi
        
        if [ "$GENERATE_CHECKSUM" = true ]; then
            echo ""
            generate_checksums
        fi
    fi
    
    # 显示总结
    echo ""
    show_summary
    
    exit $failures
}

# 运行主函数
main "$@" 