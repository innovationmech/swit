# Swagger 相关规则

.PHONY: swagger-install
swagger-install:
	@echo "Installing swag tool"
	@$(GO) install github.com/swaggo/swag/cmd/swag@v1.8.12
	@echo "swag tool installed"

# Generate Swagger documentation for all services
.PHONY: swagger
swagger: swagger-fmt swagger-switserve swagger-switauth swagger-copy
	@echo "All Swagger documentation generated and organized"

# Generate Swagger documentation for switserve
.PHONY: swagger-switserve
swagger-switserve:
	@echo "Generating Swagger documentation for switserve"
	@$(SWAG) init -g cmd/swit-serve/swit-serve.go -o internal/switserve/docs --parseDependency --parseInternal --exclude internal/switauth
	@echo "SwitServe Swagger documentation generated at: internal/switserve/docs/"

# Generate Swagger documentation for switauth
.PHONY: swagger-switauth
swagger-switauth:
	@echo "Generating Swagger documentation for switauth"
	@$(SWAG) init -g cmd/swit-auth/swit-auth.go -o internal/switauth/docs --parseDependency --parseInternal --exclude internal/switserve
	@echo "SwitAuth Swagger documentation generated at: internal/switauth/docs/"

# Format Swagger annotations for all services
.PHONY: swagger-fmt
swagger-fmt: swagger-fmt-switserve swagger-fmt-switauth
	@echo "All Swagger annotations formatted"

.PHONY: swagger-fmt-switserve
swagger-fmt-switserve:
	@echo "Formatting Swagger annotations for switserve"
	@$(SWAG) fmt -g cmd/swit-serve
	@echo "SwitServe Swagger annotations formatted"

.PHONY: swagger-fmt-switauth
swagger-fmt-switauth:
	@echo "Formatting Swagger annotations for switauth"
	@$(SWAG) fmt -g cmd/swit-auth
	@echo "SwitAuth Swagger annotations formatted"

# Copy generated docs to unified location for easy access
.PHONY: swagger-copy
swagger-copy: swagger-switserve swagger-switauth
	@echo "Creating unified documentation links"
	@mkdir -p docs/generated/switserve
	@mkdir -p docs/generated/switauth
	@echo "# SwitServe API Documentation" > docs/generated/switserve/README.md
	@echo "" >> docs/generated/switserve/README.md
	@echo "**Generated API Documentation**: [internal/switserve/docs/](../../../internal/switserve/docs/)" >> docs/generated/switserve/README.md
	@echo "**Swagger UI**: http://localhost:8080/swagger/index.html" >> docs/generated/switserve/README.md
	@echo "**API Base URL**: http://localhost:8080" >> docs/generated/switserve/README.md
	@echo "" >> docs/generated/switserve/README.md
	@echo "文档文件位置：" >> docs/generated/switserve/README.md
	@echo "- JSON: [swagger.json](../../../internal/switserve/docs/swagger.json)" >> docs/generated/switserve/README.md
	@echo "- YAML: [swagger.yaml](../../../internal/switserve/docs/swagger.yaml)" >> docs/generated/switserve/README.md
	@echo "- Go Code: [docs.go](../../../internal/switserve/docs/docs.go)" >> docs/generated/switserve/README.md
	@echo "# SwitAuth API Documentation" > docs/generated/switauth/README.md
	@echo "" >> docs/generated/switauth/README.md
	@echo "**Generated API Documentation**: [internal/switauth/docs/](../../../internal/switauth/docs/)" >> docs/generated/switauth/README.md
	@echo "**Swagger UI**: http://localhost:8090/swagger/index.html" >> docs/generated/switauth/README.md
	@echo "**API Base URL**: http://localhost:8090" >> docs/generated/switauth/README.md
	@echo "" >> docs/generated/switauth/README.md
	@echo "文档文件位置：" >> docs/generated/switauth/README.md
	@echo "- JSON: [swagger.json](../../../internal/switauth/docs/swagger.json)" >> docs/generated/switauth/README.md
	@echo "- YAML: [swagger.yaml](../../../internal/switauth/docs/swagger.yaml)" >> docs/generated/switauth/README.md
	@echo "- Go Code: [docs.go](../../../internal/switauth/docs/docs.go)" >> docs/generated/switauth/README.md
	@echo "Unified documentation access created" 