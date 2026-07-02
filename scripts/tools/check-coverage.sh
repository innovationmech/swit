#!/usr/bin/env bash

# SWIT 覆盖率阈值检查脚本
# 基于 coverage.out 按核心包计算语句覆盖率，并与阈值比较。
# 用法: check-coverage.sh [coverage.out 路径]
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

COVERAGE_FILE="${1:-coverage.out}"

# 核心包覆盖率阈值（百分比，基于 2026-07 基线设定，留 2-3 个百分点缓冲）
# 基线: pkg/server 80.2% / pkg/transport 75.4% / pkg/messaging 61.7% / pkg/saga 73.9%
declare -a CORE_PACKAGES=(
    "pkg/server:78"
    "pkg/transport:72"
    "pkg/messaging:58"
    "pkg/saga:71"
)

if [ ! -f "${COVERAGE_FILE}" ]; then
    echo -e "${RED}❌ 覆盖率文件不存在: ${COVERAGE_FILE}${NC}"
    echo "   请先运行: make test-coverage"
    exit 1
fi

echo "📊 核心包覆盖率阈值检查 (${COVERAGE_FILE})"
echo ""

FAILED=0

for entry in "${CORE_PACKAGES[@]}"; do
    pkg="${entry%%:*}"
    threshold="${entry##*:}"

    # 从 coverage profile 中按语句数聚合计算包覆盖率
    actual=$(awk -v pkg="${pkg}/" 'NR>1 && index($1, pkg) > 0 {
        n = $(NF-1); hit = $NF;
        total += n;
        if (hit > 0) covered += n;
    } END {
        if (total > 0) printf "%.1f", 100 * covered / total;
        else print "-1";
    }' "${COVERAGE_FILE}")

    if [ "${actual}" = "-1" ]; then
        echo -e "${YELLOW}⚠️  ${pkg}: 覆盖率数据缺失（跳过）${NC}"
        continue
    fi

    if awk -v a="${actual}" -v t="${threshold}" 'BEGIN { exit !(a >= t) }'; then
        echo -e "${GREEN}✅ ${pkg}: ${actual}% (阈值 ${threshold}%)${NC}"
    else
        echo -e "${RED}❌ ${pkg}: ${actual}% 低于阈值 ${threshold}%${NC}"
        FAILED=1
    fi
done

echo ""
if [ "${FAILED}" -ne 0 ]; then
    echo -e "${RED}❌ 核心包覆盖率未达标，请补充测试${NC}"
    exit 1
fi
echo -e "${GREEN}✅ 所有核心包覆盖率达标${NC}"
