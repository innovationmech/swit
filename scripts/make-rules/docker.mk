# Docker 相关规则

.PHONY: image-serve
image-serve:
	@echo "Building Docker image for swit-serve"
	@$(DOCKER) build -f build/docker/swit-serve/Dockerfile -t $(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG) .

.PHONY: image-auth
image-auth:
	@echo "Building Docker image for swit-auth"
	@$(DOCKER) build -f build/docker/switauth/Dockerfile -t $(DOCKER_AUTH_IMAGE_NAME):$(DOCKER_IMAGE_TAG) .

.PHONY: image-all
image-all: image-serve image-auth
	@echo "All Docker images built successfully" 