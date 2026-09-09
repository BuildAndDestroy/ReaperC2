# ReaperC2 container image: build linux/amd64 + linux/arm64 and push to registry.reaper-ut.com.
#
# Recommended on Mac (avoids go SIGSEGV under QEMU in buildx):
#   make build          # cross-compile on host, docker only packages the image
#
# Linux CI / full Docker compile (after `make vendor`):
#   make build-docker
#
# Requires: docker, docker buildx, aws CLI (for ECR-backed login), git. `make build` also needs Go on the host.

AWS_REGION     ?= us-east-1
AWS_CLI_PROFILE ?=
ECR_REGISTRY   ?= registry.reaper-ut.com
ECR_REPOSITORY ?= reaperc2
IMAGE_TAG      ?= latest
GIT_SHA        ?= $(shell git rev-parse --short HEAD 2>/dev/null)
SCYTHE_GIT_REF ?= main

IMAGE          := $(ECR_REGISTRY)/$(ECR_REPOSITORY):$(IMAGE_TAG)
IMAGE_AMD64    := $(ECR_REGISTRY)/$(ECR_REPOSITORY):$(IMAGE_TAG)-amd64
IMAGE_ARM64    := $(ECR_REGISTRY)/$(ECR_REPOSITORY):$(IMAGE_TAG)-arm64
BUILDER_NAME   ?= reaperc2-builder
AWS_CMD        := AWS_CLI_PROFILE=$(AWS_CLI_PROFILE) $(CURDIR)/scripts/aws-for-make.sh

DOCKER_BUILD_ARGS := \
	--build-arg SCYTHE_GIT_REF=$(SCYTHE_GIT_REF) \
	--provenance=false \
	--sbom=false

GO_BUILD_FLAGS := -trimpath -ldflags="-s -w" -mod=vendor

.PHONY: help submodule vendor build-binaries build build-docker build-amd64 build-arm64 build-local push \
	ecr-login ecr-create-repo setup-buildx ecr-alias-tags

help:
	@echo "ReaperC2 ECR image build"
	@echo ""
	@echo "  Daily cluster update:  deployments/k8s/reaperc2/ship.sh   (push :latest + restart pods)"
	@echo "  make build            Host cross-compile + docker package (best on Apple Silicon)"
	@echo "  make build-docker     Full compile in Docker (needs: make vendor first)"
	@echo "  make build-binaries   Only compile bin/linux-amd64 and bin/linux-arm64/ReaperC2"
	@echo "  make build-amd64        Push amd64 image only (:latest by default)"
	@echo "  make build-arm64        Push arm64 image only"
	@echo "  deployments/k8s/reaperc2/build-push-image.sh --arch amd64|arm64|both  (wrapper)"
	@echo "  make vendor             go mod vendor (for build-docker)"
	@echo ""
	@echo "AWS: export AWS_ACCESS_KEY_ID + AWS_SECRET_ACCESS_KEY, or AWS_CLI_PROFILE=my-sso"
	@echo "  IMAGE=$(IMAGE)  (override ECR_REGISTRY if not registry.reaper-ut.com)"

submodule:
	git submodule update --init --recursive

vendor:
	go mod vendor

build-binaries: submodule vendor
	@mkdir -p bin/linux-amd64 bin/linux-arm64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o bin/linux-amd64/ReaperC2 ./cmd
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(GO_BUILD_FLAGS) -o bin/linux-arm64/ReaperC2 ./cmd
	@echo "Built bin/linux-amd64/ReaperC2 and bin/linux-arm64/ReaperC2"

setup-buildx:
	@docker buildx version >/dev/null
	@if ! docker buildx inspect $(BUILDER_NAME) >/dev/null 2>&1; then \
		docker buildx create --name $(BUILDER_NAME) --driver docker-container --use; \
	else \
		docker buildx use $(BUILDER_NAME); \
	fi
	@docker buildx inspect --bootstrap >/dev/null

ecr-login: setup-buildx
	@$(AWS_CMD) ecr get-login-password --region $(AWS_REGION) | \
		docker login --username AWS --password-stdin $(ECR_REGISTRY)

ecr-create-repo:
	@case "$(ECR_REGISTRY)" in \
	  *.dkr.ecr.*.amazonaws.com) \
	    $(AWS_CMD) ecr describe-repositories --repository-names $(ECR_REPOSITORY) --region $(AWS_REGION) >/dev/null 2>&1 || \
	    $(AWS_CMD) ecr create-repository --repository-name $(ECR_REPOSITORY) --region $(AWS_REGION) ;; \
	  *) echo "==> Skip ecr-create-repo (registry is $(ECR_REGISTRY))" ;; \
	esac

# Host compile + docker pack (no go mod download / go build inside QEMU).
build: build-binaries ecr-login ecr-create-repo
	@echo "==> Packaging linux/amd64 -> $(IMAGE_AMD64)"
	docker buildx build -f Dockerfile.pack \
		--platform linux/amd64 \
		--build-arg TARGETARCH=amd64 \
		-t $(IMAGE_AMD64) \
		$(DOCKER_BUILD_ARGS) \
		--push \
		.
	@echo "==> Packaging linux/arm64 -> $(IMAGE_ARM64)"
	docker buildx build -f Dockerfile.pack \
		--platform linux/arm64 \
		--build-arg TARGETARCH=arm64 \
		-t $(IMAGE_ARM64) \
		$(DOCKER_BUILD_ARGS) \
		--push \
		.
	@echo "==> Publishing multi-arch manifest $(IMAGE)"
	docker buildx imagetools create -t $(IMAGE) $(IMAGE_AMD64) $(IMAGE_ARM64)
	@$(MAKE) ecr-alias-tags

push: build

# Full Docker build using vendored modules (no network in container).
build-docker: submodule vendor ecr-login ecr-create-repo
	@echo "==> Docker build linux/amd64 -> $(IMAGE_AMD64)"
	docker buildx build \
		--platform linux/amd64 \
		-t $(IMAGE_AMD64) \
		$(DOCKER_BUILD_ARGS) \
		--push \
		.
	@echo "==> Docker build linux/arm64 -> $(IMAGE_ARM64)"
	docker buildx build \
		--platform linux/arm64 \
		-t $(IMAGE_ARM64) \
		$(DOCKER_BUILD_ARGS) \
		--push \
		.
	docker buildx imagetools create -t $(IMAGE) $(IMAGE_AMD64) $(IMAGE_ARM64)
	@$(MAKE) ecr-alias-tags

# Extra ECR tags: git SHA (rollback) and :latest (cluster always pulls this).
ecr-alias-tags:
	@if [ -n "$(GIT_SHA)" ] && [ "$(IMAGE_TAG)" != "$(GIT_SHA)" ]; then \
		echo "==> Also tagging $(ECR_REGISTRY)/$(ECR_REPOSITORY):$(GIT_SHA)"; \
		docker buildx imagetools create -t $(ECR_REGISTRY)/$(ECR_REPOSITORY):$(GIT_SHA) $(IMAGE); \
	fi
	@if [ "$(IMAGE_TAG)" != "latest" ]; then \
		echo "==> Also tagging $(ECR_REGISTRY)/$(ECR_REPOSITORY):latest"; \
		docker buildx imagetools create -t $(ECR_REGISTRY)/$(ECR_REPOSITORY):latest $(IMAGE); \
	fi

build-amd64: build-binaries ecr-login ecr-create-repo
	docker buildx build -f Dockerfile.pack \
		--platform linux/amd64 \
		--build-arg TARGETARCH=amd64 \
		-t $(IMAGE_AMD64) -t $(IMAGE) \
		$(DOCKER_BUILD_ARGS) \
		--push \
		.
	@$(MAKE) ecr-alias-tags

build-arm64: build-binaries ecr-login ecr-create-repo
	docker buildx build -f Dockerfile.pack \
		--platform linux/arm64 \
		--build-arg TARGETARCH=arm64 \
		-t $(IMAGE_ARM64) -t $(IMAGE) \
		$(DOCKER_BUILD_ARGS) \
		--push \
		.
	@$(MAKE) ecr-alias-tags

build-local: submodule vendor setup-buildx
	docker buildx build \
		--load \
		-t $(ECR_REPOSITORY):local \
		$(DOCKER_BUILD_ARGS) \
		.
