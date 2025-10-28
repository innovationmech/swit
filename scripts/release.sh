#!/usr/bin/env bash
# Copyright 2025 Swit Project Authors
#
# Licensed under the MIT License

set -euo pipefail

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 显示帮助信息
show_help() {
    cat << EOF
Swit 发布脚本

用法: $0 [选项] <版本号>

选项:
    -h, --help          显示帮助信息
    -d, --dry-run       干跑模式，不实际执行发布
    -s, --skip-tests    跳过测试
    -b, --skip-build    跳过构建
    -t, --tag-only      仅创建 Git 标签
    -p, --push          推送到远程仓库

参数:
    <版本号>            发布版本号，例如 v0.9.0

示例:
    $0 v0.9.0                    # 完整发布流程（本地）
    $0 -p v0.9.0                 # 完整发布流程并推送
    $0 -d v0.9.0                 # 干跑模式
    $0 -t v0.9.0                 # 仅创建标签
    $0 -s -b v0.9.0              # 跳过测试和构建

EOF
}

# 参数解析
DRY_RUN=false
SKIP_TESTS=false
SKIP_BUILD=false
TAG_ONLY=false
PUSH=false
VERSION=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -d|--dry-run)
            DRY_RUN=true
            shift
            ;;
        -s|--skip-tests)
            SKIP_TESTS=true
            shift
            ;;
        -b|--skip-build)
            SKIP_BUILD=true
            shift
            ;;
        -t|--tag-only)
            TAG_ONLY=true
            shift
            ;;
        -p|--push)
            PUSH=true
            shift
            ;;
        *)
            if [[ -z "$VERSION" ]]; then
                VERSION=$1
            else
                log_error "未知参数: $1"
                show_help
                exit 1
            fi
            shift
            ;;
    esac
done

# 验证版本号
if [[ -z "$VERSION" ]]; then
    log_error "请指定版本号"
    show_help
    exit 1
fi

# 验证版本号格式
if ! [[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$ ]]; then
    log_error "版本号格式错误: $VERSION"
    log_error "正确格式: vX.Y.Z 或 vX.Y.Z-suffix (例如 v0.9.0 或 v1.0.0-rc.1)"
    exit 1
fi

log_info "===== Swit 发布流程 ====="
log_info "版本: $VERSION"
log_info "干跑模式: $DRY_RUN"
log_info "跳过测试: $SKIP_TESTS"
log_info "跳过构建: $SKIP_BUILD"
log_info "仅标签: $TAG_ONLY"
log_info "推送: $PUSH"
log_info ""

# 进入项目根目录
cd "$PROJECT_ROOT"

# 1. 检查工作目录状态
log_info "1. 检查工作目录状态..."
if [[ -n $(git status --porcelain) ]]; then
    log_warn "工作目录有未提交的更改:"
    git status --short
    if [[ "$DRY_RUN" == "false" ]]; then
        read -p "是否继续? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_error "用户取消"
            exit 1
        fi
    fi
fi

# 2. 检查是否在正确的分支
log_info "2. 检查分支..."
CURRENT_BRANCH=$(git branch --show-current)
if [[ "$CURRENT_BRANCH" != "master" ]] && [[ "$CURRENT_BRANCH" != "main" ]]; then
    log_warn "当前分支不是 master/main: $CURRENT_BRANCH"
    if [[ "$DRY_RUN" == "false" ]]; then
        read -p "是否继续? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_error "用户取消"
            exit 1
        fi
    fi
fi

# 3. 拉取最新代码
log_info "3. 拉取最新代码..."
if [[ "$DRY_RUN" == "false" ]]; then
    git pull origin "$CURRENT_BRANCH" || {
        log_error "拉取最新代码失败"
        exit 1
    }
fi

# 4. 检查标签是否已存在
log_info "4. 检查标签..."
if git rev-parse "$VERSION" >/dev/null 2>&1; then
    log_error "标签 $VERSION 已存在"
    exit 1
fi

# 如果只创建标签，跳过后续步骤
if [[ "$TAG_ONLY" == "true" ]]; then
    log_info "仅创建标签模式，跳过测试和构建"
    SKIP_TESTS=true
    SKIP_BUILD=true
fi

# 5. 运行测试
if [[ "$SKIP_TESTS" == "false" ]]; then
    log_info "5. 运行测试..."
    if [[ "$DRY_RUN" == "false" ]]; then
        make test || {
            log_error "测试失败"
            exit 1
        }
    else
        log_info "   [干跑] 跳过实际测试"
    fi
else
    log_info "5. 跳过测试 (--skip-tests)"
fi

# 6. 运行质量检查
if [[ "$SKIP_TESTS" == "false" ]]; then
    log_info "6. 运行质量检查..."
    if [[ "$DRY_RUN" == "false" ]]; then
        make quality || {
            log_error "质量检查失败"
            exit 1
        }
    else
        log_info "   [干跑] 跳过实际质量检查"
    fi
else
    log_info "6. 跳过质量检查 (--skip-tests)"
fi

# 7. 构建发布包
if [[ "$SKIP_BUILD" == "false" ]]; then
    log_info "7. 构建发布包..."
    if [[ "$DRY_RUN" == "false" ]]; then
        make build || {
            log_error "构建失败"
            exit 1
        }
    else
        log_info "   [干跑] 跳过实际构建"
    fi
else
    log_info "7. 跳过构建 (--skip-build)"
fi

# 8. 生成文档
if [[ "$SKIP_BUILD" == "false" ]]; then
    log_info "8. 生成文档..."
    if [[ "$DRY_RUN" == "false" ]]; then
        if command -v make swagger &> /dev/null; then
            make swagger || log_warn "文档生成失败，继续..."
        else
            log_warn "未找到 swagger 工具，跳过文档生成"
        fi
    else
        log_info "   [干跑] 跳过文档生成"
    fi
else
    log_info "8. 跳过文档生成 (--skip-build)"
fi

# 9. 创建 Git 标签
log_info "9. 创建 Git 标签..."
TAG_MESSAGE="Release $VERSION

查看完整发布说明:
https://github.com/innovationmech/swit/releases/tag/$VERSION

主要变更:
- Saga 分布式事务系统正式发布
- 完整的文档体系
- 企业级安全特性
- 监控与可观测性集成
- Dashboard 管理后台
- 丰富的示例应用

详情请查看 CHANGELOG.md 和 RELEASE_NOTES.md"

if [[ "$DRY_RUN" == "false" ]]; then
    git tag -a "$VERSION" -m "$TAG_MESSAGE" || {
        log_error "创建标签失败"
        exit 1
    }
    log_info "   标签创建成功: $VERSION"
else
    log_info "   [干跑] 将创建标签: $VERSION"
fi

# 10. 推送到远程
if [[ "$PUSH" == "true" ]]; then
    log_info "10. 推送到远程仓库..."
    if [[ "$DRY_RUN" == "false" ]]; then
        git push origin "$CURRENT_BRANCH" || {
            log_error "推送分支失败"
            exit 1
        }
        git push origin "$VERSION" || {
            log_error "推送标签失败"
            exit 1
        }
        log_info "    推送成功"
    else
        log_info "    [干跑] 将推送分支和标签"
    fi
else
    log_info "10. 跳过推送 (使用 -p 参数以推送)"
    log_warn "    记得手动推送: git push origin $CURRENT_BRANCH && git push origin $VERSION"
fi

# 11. 显示发布信息
log_info ""
log_info "===== 发布完成 ====="
log_info "版本: $VERSION"
log_info ""
log_info "后续步骤:"
if [[ "$PUSH" == "false" ]]; then
    log_info "  1. 推送到远程: git push origin $CURRENT_BRANCH && git push origin $VERSION"
fi
log_info "  2. 在 GitHub 上创建 Release:"
log_info "     https://github.com/innovationmech/swit/releases/new?tag=$VERSION"
log_info "  3. 上传 RELEASE_NOTES.md 的内容作为 Release 描述"
log_info "  4. 上传构建产物（如果需要）"
log_info "  5. 发布 Release 公告"
log_info ""

if [[ "$DRY_RUN" == "true" ]]; then
    log_warn "这是干跑模式，未执行实际操作"
fi

log_info "完成! 🎉"

