# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at https://mozilla.org/MPL/2.0/.

SHELL = bash
PROJECT_ROOT := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
THIS_OS := $(shell uname | cut -d- -f1)

BUILD_DIR := $(PROJECT_ROOT)/build
BUILD_BIN_DIR := $(BUILD_DIR)/bin

GIT_COMMIT := $(shell git rev-parse --short HEAD)
GIT_DIRTY := $(if $(shell git status --porcelain),+)

GO_LDFLAGS := "-X github.com/lachlanorr/rkcy/version.GitCommit=$(GIT_COMMIT)$(GIT_DIRTY)"

default: all

.PHONY: all
all: check proto examples ## build everything

.PHONY: check
check: ## check license headers in files
	@go run ./scripts/check_file_headers.go cmd internal pkg proto scratch scripts

.PHONY: clean
clean: ## remove all build artifacts
	@rm -rf ./build
	@rm -rf ./pkg/rkcy/pb/*.pb.go ./pkg/rkcy/pb/*_grpc.pb.go ./pkg/rkcy/pb/*.pb.gw.go
	@rm -rf ./examples/rpg/pb/*.pb.go ./examples/rpg/pb/*_grpc.pb.go ./examples/rpg/pb/*.pb.gw.go

.PHONY: proto
proto: ## generate protocol buffers
	@echo "==> Building $@..."
	@go generate pkg/rkcy/pb/gen.go
	@go generate examples/rpg/pb/gen.go

.PHONY: examples
examples: rpg rpg_edge rpg_sim ## compile all examples

.PHONY: rpg
rpg: ## compile rpg example
	@echo "==> Building $@..."
	@go build \
	-ldflags $(GO_LDFLAGS) \
	-o $(BUILD_BIN_DIR)/rpg \
	./examples/rpg/rpg
	@cp ./examples/rpg/rpg/platform.json $(BUILD_BIN_DIR)

.PHONY: rpg_edge
rpg_edge: ## compile rpg_edge example
	@echo "==> Building $@..."
	@go build \
	-ldflags $(GO_LDFLAGS) \
	-o $(BUILD_BIN_DIR)/rpg_edge \
    ./examples/rpg/rpg_edge

.PHONY: rpg_sim
rpg_sim: ## compile rpg_sim cmd
	@echo "==> Building $@..."
	@go build \
	-ldflags $(GO_LDFLAGS) \
	-o $(BUILD_BIN_DIR)/rpg_sim \
    ./examples/rpg/rpg_sim

HELP_FORMAT="    \033[36m%-25s\033[0m %s\n"
.PHONY: help
help: ## display this usage information
	@echo "Valid targets:"
	@grep -E '^[^ ]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		sort | \
		awk 'BEGIN {FS = ":.*?## "}; \
			{printf $(HELP_FORMAT), $$1, $$2}'
	@echo ""
