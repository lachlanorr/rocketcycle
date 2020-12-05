# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at https://mozilla.org/MPL/2.0/.

SHELL = bash
PROJECT_ROOT := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
THIS_OS := $(shell uname | cut -d- -f1)

BUILD_DIR := $(PROJECT_ROOT)/build/bin
BUILD_BIN_DIR := $(PROJECT_ROOT)/build/bin

GIT_COMMIT := $(shell git rev-parse HEAD)
GIT_DIRTY := $(if $(shell git status --porcelain),+CHANGES)

GO_LDFLAGS := "-X github.com/lachlanorr/gaeneco/version.GitCommit=$(GIT_COMMIT)$(GIT_DIRTY)"

default: all

.PHONY: all
all: proto cmd ## build everything

.PHONY: clean ## remove all build artifacts
clean:
	@rm -rf ./build
	@rm -rf ./pb/*.pb.go

.PHONY: proto
proto: ## generate protocol buffers
	@echo "==> Building $@..."
	@protoc \
     --go_out=. \
     --go_opt=paths=source_relative \
     pb/txn.proto \
     pb/metadata.proto

.PHONY: cmd
cmd: gaeneco simulate ## compile all cmds

.PHONY: gaeneco
gaeneco: ## compile gaeneco cmd
	@echo "==> Building $@..."
	@go build \
	-ldflags $(GO_LDFLAGS) \
	-o $(BUILD_BIN_DIR)/gaeneco \
	./cmd/gaeneco

.PHONY: simulate
simulate: ## compile simulate cmd
	@echo "==> Building $@..."
	@go build \
	-ldflags $(GO_LDFLAGS) \
	-o $(BUILD_BIN_DIR)/simulate \
    ./cmd/simulate

HELP_FORMAT="    \033[36m%-25s\033[0m %s\n"
.PHONY: help
help: ## display this usage information
	@echo "Valid targets:"
	@grep -E '^[^ ]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		sort | \
		awk 'BEGIN {FS = ":.*?## "}; \
			{printf $(HELP_FORMAT), $$1, $$2}'
	@echo ""
