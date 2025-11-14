# 代码质量管理规则
# 统一管理依赖、格式化、代码检查等质量相关功能

# =============================================================================
# 基础质量操作
# =============================================================================

# 依赖管理 - 整理Go模块依赖
.PHONY: tidy
tidy: proto swagger
	@echo "🔧 整理Go模块依赖..."
	@$(GO) mod tidy
	@echo "✅ Go模块依赖整理完成"

# 代码格式化 - 使用gofmt格式化代码
.PHONY: format
format:
	@echo "🎨 格式化Go代码..."
	@$(GOFMT) -w .
	@echo "✅ 代码格式化完成"

# 代码检查（完整版）- 包含依赖生成
.PHONY: vet
vet: proto swagger
	@echo "🔍 运行代码检查（包含依赖生成）..."
	@$(GOVET) ./...
	@echo "✅ 代码检查完成"

# 代码检查（快速版）- 跳过依赖生成
.PHONY: vet-fast
vet-fast:
	@echo "🔍 运行快速代码检查..."
	@$(GOVET) ./...
	@echo "✅ 快速代码检查完成"

# 代码静态分析 - 使用golint进行代码规范检查
.PHONY: lint
lint:
	@echo "📝 运行代码规范检查..."
	@if command -v golint >/dev/null 2>&1; then \
		golint ./...; \
	else \
		echo "⚠️  golint未安装，跳过代码规范检查"; \
		echo "💡 安装方法: go install golang.org/x/lint/golint@latest"; \
	fi
	@echo "✅ 代码规范检查完成"

# 代码安全检查 - 使用gosec进行安全扫描
.PHONY: security
security:
	@echo "🔒 运行安全扫描..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "⚠️  gosec未安装，跳过安全扫描"; \
		echo "💡 安装方法: go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
	fi
	@echo "✅ 安全扫描完成"

# =============================================================================
# 核心质量目标 (用户主要使用)
# =============================================================================

# 标准质量检查（推荐用于CI/CD和发布前）
.PHONY: quality
quality: tidy format vet lint
	@echo "🎯 标准质量检查完成"
	@echo "✅ 包含: 依赖整理 + 代码格式化 + 完整检查 + 规范检查"

# 快速质量检查（开发时使用）
.PHONY: quality-dev
quality-dev: format vet-fast
	@echo "🚀 快速质量检查完成"
	@echo "✅ 包含: 代码格式化 + 快速检查"

# 质量环境设置（安装必要的质量检查工具）
.PHONY: quality-setup
quality-setup:
	@echo "🛠️  设置代码质量检查环境..."
	@echo "📦 检查并安装质量检查工具..."
	
	@echo "检查golint..."
	@if ! command -v golint >/dev/null 2>&1; then \
		echo "📥 安装golint..."; \
		go install golang.org/x/lint/golint@latest; \
	else \
		echo "✅ golint已安装"; \
	fi
	
	@echo "检查gosec..."
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "📥 安装gosec..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
	else \
		echo "✅ gosec已安装"; \
	fi
	
	@echo "检查goimports..."
	@if ! command -v goimports >/dev/null 2>&1; then \
		echo "📥 安装goimports..."; \
		go install golang.org/x/tools/cmd/goimports@latest; \
	else \
		echo "✅ goimports已安装"; \
	fi
	
	@echo "检查staticcheck..."
	@if ! command -v staticcheck >/dev/null 2>&1; then \
		echo "📥 安装staticcheck..."; \
		go install honnef.co/go/tools/cmd/staticcheck@latest; \
	else \
		echo "✅ staticcheck已安装"; \
	fi
	
	@echo "🎉 质量检查环境设置完成"

# 高级质量管理（精确控制特定操作）
.PHONY: quality-advanced
quality-advanced:
	@if [ -z "$(OPERATION)" ]; then \
		echo "🔧 高级质量管理"; \
		echo ""; \
		echo "用法: make quality-advanced OPERATION=<操作> [TARGET=<目标>]"; \
		echo ""; \
		echo "📝 支持的操作:"; \
		echo "  tidy        - 整理Go模块依赖"; \
		echo "  format      - 格式化代码"; \
		echo "  vet         - 代码检查"; \
		echo "  lint        - 代码规范检查"; \
		echo "  security    - 安全扫描"; \
		echo "  imports     - 整理导入语句"; \
		echo "  static      - 静态代码分析"; \
		echo "  all         - 运行所有检查"; \
		echo ""; \
		echo "📖 示例:"; \
		echo "  make quality-advanced OPERATION=tidy"; \
		echo "  make quality-advanced OPERATION=lint TARGET=./internal/..."; \
		echo "  make quality-advanced OPERATION=all"; \
		exit 1; \
	fi
	@case "$(OPERATION)" in \
		tidy) \
			$(MAKE) tidy ;; \
		format) \
			$(MAKE) format ;; \
		vet) \
			$(MAKE) vet ;; \
		lint) \
			$(MAKE) quality-advanced-lint ;; \
		security) \
			$(MAKE) security ;; \
		imports) \
			$(MAKE) quality-advanced-imports ;; \
		static) \
			$(MAKE) quality-advanced-static ;; \
		all) \
			$(MAKE) quality && $(MAKE) security && $(MAKE) quality-advanced-imports && $(MAKE) quality-advanced-static ;; \
		*) \
			echo "❌ 不支持的操作: $(OPERATION)"; \
			$(MAKE) quality-advanced; \
			exit 1 ;; \
	esac

# =============================================================================
# 高级质量操作的具体实现
# =============================================================================

# 高级代码规范检查 - 支持指定目标
.PHONY: quality-advanced-lint
quality-advanced-lint:
	@echo "📝 运行高级代码规范检查..."
	@TARGET=$${TARGET:-./...}; \
	if command -v golint >/dev/null 2>&1; then \
		echo "🔍 检查目标: $$TARGET"; \
		golint $$TARGET; \
	else \
		echo "❌ golint未安装"; \
		echo "💡 请先运行: make quality-setup"; \
		exit 1; \
	fi

# 导入语句整理 - 使用goimports
.PHONY: quality-advanced-imports
quality-advanced-imports:
	@echo "📦 整理导入语句..."
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
		echo "✅ 导入语句整理完成"; \
	else \
		echo "❌ goimports未安装"; \
		echo "💡 请先运行: make quality-setup"; \
		exit 1; \
	fi

# 静态代码分析 - 使用staticcheck
.PHONY: quality-advanced-static
quality-advanced-static:
	@echo "🔬 运行静态代码分析..."
	@TARGET=$${TARGET:-./...}; \
	if command -v staticcheck >/dev/null 2>&1; then \
		echo "🔍 分析目标: $$TARGET"; \
		staticcheck $$TARGET; \
		echo "✅ 静态代码分析完成"; \
	else \
		echo "❌ staticcheck未安装"; \
		echo "💡 请先运行: make quality-setup"; \
		exit 1; \
	fi

# =============================================================================
# 安全扫描目标
# =============================================================================

# 安全扫描 - 运行所有配置的安全扫描器
.PHONY: security-scan
security-scan:
	@echo "🔒 运行安全扫描..."
	@./scripts/security-scan.sh
	@echo "✅ 安全扫描完成"

# gosec 安全扫描 - 静态代码安全分析
.PHONY: security-scan-gosec
security-scan-gosec:
	@echo "🔍 运行gosec安全扫描..."
	@if command -v gosec >/dev/null 2>&1; then \
		mkdir -p _output/security; \
		gosec -fmt=json -out=_output/security/gosec-report.json -no-fail ./...; \
		echo "✅ gosec扫描完成，报告保存到: _output/security/gosec-report.json"; \
	else \
		echo "⚠️  gosec未安装"; \
		echo "💡 安装方法: go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
		exit 1; \
	fi

# govulncheck 漏洞扫描 - Go依赖漏洞检查
.PHONY: security-scan-vulncheck
security-scan-vulncheck:
	@echo "🔍 运行govulncheck漏洞扫描..."
	@if command -v govulncheck >/dev/null 2>&1; then \
		mkdir -p _output/security; \
		govulncheck -json ./... > _output/security/govulncheck-report.json || true; \
		echo "✅ govulncheck扫描完成，报告保存到: _output/security/govulncheck-report.json"; \
	else \
		echo "⚠️  govulncheck未安装"; \
		echo "💡 安装方法: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
		exit 1; \
	fi

# trivy 安全扫描 - 容器和文件系统扫描（可选）
.PHONY: security-scan-trivy
security-scan-trivy:
	@echo "🔍 运行trivy安全扫描..."
	@if command -v trivy >/dev/null 2>&1; then \
		mkdir -p _output/security; \
		trivy fs --format json --output _output/security/trivy-report.json --scanners vuln,misconfig,secret .; \
		echo "✅ trivy扫描完成，报告保存到: _output/security/trivy-report.json"; \
	else \
		echo "⚠️  trivy未安装"; \
		echo "💡 安装方法（macOS）: brew install aquasecurity/trivy/trivy"; \
		echo "💡 安装方法（Linux）: curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin"; \
		exit 1; \
	fi

# 安全扫描环境设置 - 安装所有安全扫描工具
.PHONY: security-scan-setup
security-scan-setup:
	@echo "🛠️  设置安全扫描环境..."
	
	@echo "检查gosec..."
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "📥 安装gosec..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
	else \
		echo "✅ gosec已安装"; \
	fi
	
	@echo "检查govulncheck..."
	@if ! command -v govulncheck >/dev/null 2>&1; then \
		echo "📥 安装govulncheck..."; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
	else \
		echo "✅ govulncheck已安装"; \
	fi
	
	@echo "检查trivy..."
	@if ! command -v trivy >/dev/null 2>&1; then \
		echo "⚠️  trivy未安装（可选工具）"; \
		echo "💡 手动安装方法（macOS）: brew install aquasecurity/trivy/trivy"; \
		echo "💡 手动安装方法（Linux）: curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin"; \
	else \
		echo "✅ trivy已安装"; \
	fi
	
	@echo "🎉 安全扫描环境设置完成"

# 安全扫描高级操作
.PHONY: security-scan-advanced
security-scan-advanced:
	@if [ -z "$(OPERATION)" ]; then \
		echo "🔧 高级安全扫描管理"; \
		echo ""; \
		echo "用法: make security-scan-advanced OPERATION=<操作> [OPTIONS]"; \
		echo ""; \
		echo "📝 支持的操作:"; \
		echo "  gosec       - 运行gosec扫描"; \
		echo "  vulncheck   - 运行govulncheck扫描"; \
		echo "  trivy       - 运行trivy扫描"; \
		echo "  all         - 运行所有扫描器"; \
		echo "  report      - 生成综合报告"; \
		echo ""; \
		echo "📝 可选参数:"; \
		echo "  TOOLS       - 指定工具列表（逗号分隔，如：gosec,govulncheck）"; \
		echo "  FORMAT      - 报告格式（json,html,sarif,text）"; \
		echo "  TARGET      - 扫描目标（默认：./...）"; \
		echo ""; \
		echo "📖 示例:"; \
		echo "  make security-scan-advanced OPERATION=gosec"; \
		echo "  make security-scan-advanced OPERATION=all FORMAT=html"; \
		echo "  make security-scan-advanced OPERATION=report FORMAT=html,json"; \
		exit 1; \
	fi
	@case "$(OPERATION)" in \
		gosec) \
			$(MAKE) security-scan-gosec ;; \
		vulncheck) \
			$(MAKE) security-scan-vulncheck ;; \
		trivy) \
			$(MAKE) security-scan-trivy ;; \
		all) \
			$(MAKE) security-scan ;; \
		report) \
			echo "🔍 生成综合安全报告..."; \
			./scripts/security-scan.sh --format $(FORMAT) ;; \
		*) \
			echo "❌ 不支持的操作: $(OPERATION)"; \
			$(MAKE) security-scan-advanced; \
			exit 1 ;; \
	esac 