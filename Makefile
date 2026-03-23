# Retail Edge - SRE Infrastructure & Service Makefile
.DEFAULT_GOAL := help

# Configuration
BINARY_NAME=retail-syncworker
IMAGE_NAME=retail-edge-service
DOCKER_REPO=europe-west3-docker.pkg.dev/retail-backbone-gcp/retail-edge-repo
VERSION=$(shell git rev-parse --short HEAD)

.PHONY: build test lint docker-build docker-push infra-up help

help: ## Show this help menu
	@echo "Retail Edge - SRE Operations Control"
	@echo "------------------------------------"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

## --- Local Development (Go) ---

build: ## Compile the Go binary (used by CI)
	@echo "SRE Operations: Compiling binary..."
	@mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/$(BINARY_NAME) ./cmd/server/main.go

test: ## Run unit tests with race detection
	@echo "SRE Operations: Running race detection tests..."
	go test -v -race ./...

lint: ## Run static code analysis
	@echo "SRE Operations: Linting code..."
	go vet ./...

## --- Container & Registry ---

docker-build: ## Build multi-platform images (AMD64 & ARM64)
	@echo "SRE Operations: Building multi-arch images..."
	docker buildx build --platform linux/amd64,linux/arm64 \
		-t $(IMAGE_NAME):latest \
		-t $(IMAGE_NAME):$(VERSION) .

docker-push: ## Push images to Google Artifact Registry
	@echo "SRE Operations: Pushing to GCP..."
	docker tag $(IMAGE_NAME):latest $(DOCKER_REPO)/$(IMAGE_NAME):latest
	docker push $(DOCKER_REPO)/$(IMAGE_NAME):latest

## --- Infrastructure (GCP) ---

infra-up: ## Deploy GCP Backbone via Terraform
	@echo "SRE Operations: Provisioning GCP resources..."
	cd terraform/backbone && terraform init && terraform apply -auto-approve

clean: ## Clean up build artifacts
	rm -rf bin/
	@echo "✅ Cleanup complete."