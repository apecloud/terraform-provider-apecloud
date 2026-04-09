# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif


export GO111MODULE = auto
export GONOPROXY = github.com/apecloud
export GOPRIVATE = github.com/apecloud

GOOS ?= $(shell go env GOOS)
ARCH ?= $(shell go env GOARCH)


.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

.PHONY: goimportstool
goimportstool: ## Install goimports tool.
ifeq (, $(shell which goimports))
	@{ \
	set -e ;\
	echo 'installing goimports' ;\
	go install golang.org/x/tools/cmd/goimports@latest;\
	echo 'Successfully installed' ;\
	}
GOIMPORTS=$(GOBIN)/goimports
else
GOIMPORTS=$(shell which goimports)
endif

.PHONY: golangci-lint
golangci-lint: module golangci ## Run golangci-lint against code.
	CGO_ENABLED=1 $(GOLANGCILINT) run -c .golangci.yaml ./...

.PHONY: module
module: ## Run go mod tidy->verify against go modules.
	go mod tidy
	go mod verify

.PHONY: lint
lint: module ## make build manifests
	@ echo -e "\033[1;32mgolangci-lint terraform-provider-kbcloud...\033[0m"; \
	$(MAKE) golangci-lint

.PHONY: goimports
goimports: goimportstool ## Run goimports against code.
	$(GOIMPORTS) -local github.com/apecloud/terraform-provider-kbcloud -w $$(git ls-files|grep "\.go$$")

.PHONY: build
generate build: ## Build the terraform provider executable.
	@echo "Building terraform-provider-kbcloud..."
	@mkdir -p cmd
	@go build -o cmd/terraform-provider-kbcloud .
	@echo "Build completed."

.PHONY: generate
generate: ## Run Python generator to create terraform models and schemas.
	@echo "Generating Terraform models and schemas from OpenAPI spec..."
	@cd .generator/src && PYTHONPATH=. python -m generator.cli ../specs/adminapi-bundle-tmp.yaml ../configuration.yaml
	@echo "Generation completed."
	@goimports -w ./internal/types/ ./internal/resource/ ./internal/datasource/

.PHONY: tfplugindocs
tfplugindocs: ## Install tfplugindocs into cmd directory.
	@if [ ! -f "cmd/tfplugindocs" ]; then \
		echo "Installing tfplugindocs into cmd/ ..."; \
		mkdir -p cmd; \
		GOBIN=$(shell pwd)/cmd GOPROXY=https://goproxy.cn,direct go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.19.4; \
	else \
		echo "tfplugindocs already exists in cmd/ , skipping installation."; \
	fi

.PHONY: docs
docs: tfplugindocs ## Generate Terraform Provider documentation using tfplugindocs.
	@echo "Generating Terraform Provider documentation..."
	@./cmd/tfplugindocs
	@echo "Documentation generation completed."
