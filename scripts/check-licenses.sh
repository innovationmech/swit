#!/bin/bash

# 许可证验证脚本
# 快速检查项目依赖的许可证合规性

set -e

echo "🔍 检查项目许可证合规性..."
echo "=================================="

# 检查 go-licenses 是否安装
if ! command -v go-licenses &> /dev/null; then
    echo "📦 安装 go-licenses 工具..."
    go install github.com/google/go-licenses@latest
fi

# 加载许可证配置
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
LICENSES_CONFIG="$PROJECT_ROOT/.licenses"

if [ ! -f "$LICENSES_CONFIG" ]; then
    echo "❌ 错误: 许可证配置文件 .licenses 不存在"
    echo "请确保项目根目录有 .licenses 配置文件"
    exit 1
fi

echo "📋 加载许可证配置: $LICENSES_CONFIG"
source "$LICENSES_CONFIG"

# 验证配置是否正确加载
if [ -z "${FORBIDDEN_LICENSES+x}" ]; then
    echo "❌ 错误: 无法从配置文件加载 FORBIDDEN_LICENSES 数组"
    exit 1
fi

echo "🚫 已配置 ${#FORBIDDEN_LICENSES[@]} 个禁止的许可证"
echo "✅ 已配置 ${#ALLOWED_LICENSES[@]} 个允许的许可证"
if [ -n "${REVIEW_REQUIRED_LICENSES+x}" ]; then
    echo "⚠️  已配置 ${#REVIEW_REQUIRED_LICENSES[@]} 个需要审查的许可证"
fi

# 生成许可证报告
echo "📄 生成许可证报告..."
go-licenses report ./... > licenses_temp.txt 2>&1 || true

# 检查禁止的许可证
echo "🚫 检查禁止的许可证..."
FORBIDDEN_FOUND=false

for license in "${FORBIDDEN_LICENSES[@]}"; do
    if grep -q "$license" licenses_temp.txt; then
        echo "❌ 发现禁止的许可证: $license"
        FORBIDDEN_FOUND=true
    fi
done

# 检查需要审查的许可证
REVIEW_FOUND=false
if [ -n "${REVIEW_REQUIRED_LICENSES+x}" ] && [ ${#REVIEW_REQUIRED_LICENSES[@]} -gt 0 ]; then
    echo "⚠️  检查需要审查的许可证..."
    for license in "${REVIEW_REQUIRED_LICENSES[@]}"; do
        if grep -q "$license" licenses_temp.txt; then
            echo "⚠️  发现需要审查的许可证: $license"
            REVIEW_FOUND=true
        fi
    done
fi

if [ "$FORBIDDEN_FOUND" = "true" ]; then
    echo ""
    echo "❌ 许可证检查失败!"
    echo "项目包含不兼容的许可证，请检查以下报告:"
    echo ""
    cat licenses_temp.txt
    rm licenses_temp.txt
    exit 1
fi

# 显示许可证统计
echo "📊 许可证统计:"
echo "发现的许可证类型:"
grep -o '[A-Z][A-Z0-9-]*[0-9]\|MIT\|BSD.*\|Apache.*\|MPL.*' licenses_temp.txt | sort | uniq -c | sort -nr || true

echo ""
if [ "$REVIEW_FOUND" = "true" ]; then
    echo "⚠️  许可证检查通过，但有需要审查的许可证!"
    echo "所有依赖都使用兼容的许可证，但发现了需要法律审查的许可证。"
    echo "请联系法律团队确认这些许可证的使用是否合规。"
else
    echo "✅ 许可证检查完全通过!"
    echo "所有依赖都使用完全兼容的许可证。"
fi

# 显示总体统计
TOTAL_DEPS=$(wc -l < licenses_temp.txt)
echo "📈 依赖统计: $TOTAL_DEPS 个依赖包"

# 清理临时文件
rm licenses_temp.txt

echo ""
echo "🎉 许可证合规性检查完成!"
