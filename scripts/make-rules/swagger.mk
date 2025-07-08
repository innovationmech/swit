# Swagger 相关规则

# Clean old swagger documentation
.PHONY: swagger-clean
swagger-clean: clean-swagger

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
	@mkdir -p docs/generated/switserve
	@$(SWAG) init -g cmd/swit-serve/swit-serve.go -o docs/generated/switserve --parseDependency --parseInternal --exclude internal/switauth
	@echo "SwitServe Swagger documentation generated"
	@echo "All files: docs/generated/switserve/"

# Generate Swagger documentation for switauth
.PHONY: swagger-switauth
swagger-switauth:
	@echo "Generating Swagger documentation for switauth"
	@mkdir -p docs/generated/switauth
	@$(SWAG) init -g cmd/swit-auth/swit-auth.go -o docs/generated/switauth --parseDependency --parseInternal --exclude internal/switserve
	@echo "SwitAuth Swagger documentation generated"
	@echo "All files: docs/generated/switauth/"

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
	@echo "All Swagger documentation generated and organized" 