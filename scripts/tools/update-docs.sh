#!/bin/bash

# SWIT 项目文档更新脚本
# 用于生成和更新所有服务的API文档

set -e

echo "🔄 开始更新 SWIT 项目文档..."

# 生成 Swagger 文档
echo "📝 生成 Swagger API 文档..."
make swagger

# 创建统一文档链接
echo "🔗 创建统一文档访问链接..."
make swagger-copy

# 显示文档位置
echo ""
echo "✅ 文档更新完成！"
echo ""
echo "📊 访问方式："
echo "  项目文档首页:     docs/README.md"
echo "  API文档汇总:      docs/generated/README.md"
echo "  SwitServe API:    http://localhost:9000/swagger/index.html"
echo "  SwitAuth API:     http://localhost:8080/swagger/index.html"
echo ""
echo "📁 文档位置："
echo "  SwitServe 生成文档: internal/switserve/docs/"
echo "  SwitAuth 生成文档:  internal/switauth/docs/ (待添加)"
echo "  项目级文档:        docs/"
echo "  统一访问入口:      docs/generated/"
echo ""
echo "🛠  下次文档更新："
echo "  make swagger              # 生成所有服务文档"
echo "  make swagger-switserve    # 仅生成 SwitServe 文档"
echo "  make swagger-switauth     # 仅生成 SwitAuth 文档"
echo "  make swagger-copy         # 创建统一访问链接" 