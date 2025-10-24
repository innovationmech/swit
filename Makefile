.DEFAULT_GOAL := all

# 确定项目根目录
MAKEFILE_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

# 导入所有子规则文件
include $(MAKEFILE_DIR)scripts/mk/variables.mk
include $(MAKEFILE_DIR)scripts/mk/quality.mk
include $(MAKEFILE_DIR)scripts/mk/build.mk
include $(MAKEFILE_DIR)scripts/mk/test.mk
include $(MAKEFILE_DIR)scripts/mk/docker.mk
include $(MAKEFILE_DIR)scripts/mk/swagger.mk
include $(MAKEFILE_DIR)scripts/mk/proto.mk
include $(MAKEFILE_DIR)scripts/mk/dev.mk
include $(MAKEFILE_DIR)scripts/mk/copyright.mk
include $(MAKEFILE_DIR)scripts/mk/clean.mk
include $(MAKEFILE_DIR)scripts/mk/saga-examples.mk

# 主要组合目标（使用多平台构建系统）
.PHONY: all
all: proto swagger tidy copyright build

define USAGE_OPTIONS

【开发环境与CI】
  SETUP-DEV        开发环境设置 - 完整的开发环境 (推荐)
  SETUP-QUICK      快速开发设置 - 最小必要组件，快速开始
  DEV-ADVANCED     高级开发环境管理 - 精确控制特定组件 (COMPONENT=组件类型)
  CI               CI流水线 - 自动化测试和质量检查

【构建与清理】
  BUILD            构建项目 (开发模式) - 构建当前平台的所有服务
  BUILD-DEV        快速构建 - 跳过质量检查，加速开发迭代
  BUILD-RELEASE    发布构建 - 构建所有平台的发布版本
  BUILD-ADVANCED   高级构建 - 精确控制服务和平台 (需要 SERVICE 和 PLATFORM 参数)
  CLEAN            标准清理 - 删除所有生成的代码和构建产物
  CLEAN-DEV        快速清理 - 仅删除构建输出 (开发时常用)
  CLEAN-SETUP      深度清理 - 重置环境包括缓存和依赖
  CLEAN-ADVANCED   高级清理 - 精确控制特定类型 (TYPE=清理类型)

【测试相关】
  TEST             运行测试 - 所有测试包含依赖生成 (推荐)
  TEST-DEV         快速测试 - 跳过依赖生成，加速开发迭代
  TEST-COVERAGE    覆盖率测试 - 生成详细的覆盖率报告
  TEST-ADVANCED    高级测试 - 精确控制测试类型和包范围 (TYPE=测试类型)

【代码质量相关】
  TIDY             整理Go模块依赖 - 清理和更新go.mod文件
  FORMAT           代码格式化 - 使用gofmt统一代码格式
  QUALITY          代码质量检查 - 标准质量检查 (推荐用于CI/CD)
  QUALITY-DEV      快速质量检查 - 开发时使用，跳过部分检查
  QUALITY-SETUP    质量环境设置 - 安装必要的质量检查工具
  QUALITY-ADVANCED 高级质量管理 - 精确控制特定操作 (OPERATION=操作类型)

【API文档/Proto/版权】
  SWAGGER          生成swagger文档 (推荐用于开发和发布)
  SWAGGER-DEV      快速生成 - 跳过格式化，加速开发迭代
  SWAGGER-SETUP    环境设置 - 首次使用时安装swag工具
  SWAGGER-ADVANCED 高级操作 - 精确控制特定服务和操作 (OPERATION=操作类型)
  PROTO            生成proto代码 (推荐用于开发和发布)
  PROTO-DEV        快速生成 - 跳过依赖下载，加速开发迭代
  PROTO-SETUP      环境设置 - 首次使用时安装工具和下载依赖
  PROTO-ADVANCED   高级操作 - 精确控制格式化、检查等 (OPERATION=操作类型)
  COPYRIGHT        版权声明管理 - 检查并自动修复版权声明
  COPYRIGHT-CHECK  检查版权声明 - 只检查，不修改文件
  COPYRIGHT-SETUP  初始版权设置 - 为新项目添加版权声明
  COPYRIGHT-ADVANCED 高级版权操作 - 精确控制特定操作 (OPERATION=操作类型)

【Docker相关】
  DOCKER           Docker构建 - 标准镜像构建 (推荐用于生产发布)
  DOCKER-DEV       快速Docker构建 - 使用缓存，加速开发迭代
  DOCKER-SETUP     Docker开发环境 - 启动完整的开发环境
  DOCKER-ADVANCED  高级Docker管理 - 精确控制特定操作 (OPERATION=操作类型)

【Saga 示例】
  SAGA-EXAMPLES-RUN        运行指定示例 (EXAMPLE=示例名称)
  SAGA-EXAMPLES-ORDER      运行订单处理示例
  SAGA-EXAMPLES-PAYMENT    运行支付处理示例
  SAGA-EXAMPLES-INVENTORY  运行库存管理示例
  SAGA-EXAMPLES-USER       运行用户注册示例
  SAGA-EXAMPLES-E2E        运行端到端测试
  SAGA-EXAMPLES-ALL        运行所有示例
  SAGA-EXAMPLES-STATUS     查看示例状态
  SAGA-EXAMPLES-LOGS       查看示例日志
  SAGA-EXAMPLES-STOP       停止并清理所有内容
  SAGA-EXAMPLES-HELP       显示 Saga 示例帮助信息
endef
export USAGE_OPTIONS

.PHONY: help
help:
	@echo "$$USAGE_OPTIONS"

