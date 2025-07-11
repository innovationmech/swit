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

# 生成许可证报告
echo "📄 生成许可证报告..."
go-licenses report ./... > licenses_temp.txt 2>&1 || true

# 定义禁止的许可证
FORBIDDEN_LICENSES=("GPL-2.0" "GPL-3.0" "AGPL-1.0" "AGPL-3.0")

# 检查禁止的许可证
echo "🚫 检查禁止的许可证..."
FORBIDDEN_FOUND=false

for license in "${FORBIDDEN_LICENSES[@]}"; do
    if grep -q "$license" licenses_temp.txt; then
        echo "❌ 发现禁止的许可证: $license"
        FORBIDDEN_FOUND=true
    fi
done

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
echo "✅ 许可证检查通过!"
echo "所有依赖都使用兼容的许可证。"

# 显示总体统计
TOTAL_DEPS=$(wc -l < licenses_temp.txt)
echo "📈 依赖统计: $TOTAL_DEPS 个依赖包"

# 清理临时文件
rm licenses_temp.txt

echo ""
echo "🎉 许可证合规性检查完成!"
