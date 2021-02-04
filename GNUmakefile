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

GO_LDFLAGS := "-X github.com/lachlanorr/rocketcycle/version.GitCommit=$(GIT_COMMIT)$(GIT_DIRTY)"

default: all

.PHONY: all
all: check proto cmd ## build everything

.PHONY: check
check: ## check license headers in files
	@go run ./scripts/check_file_headers.go cmd internal pkg proto scratch scripts

.PHONY: clean
clean: ## remove all build artifacts
	@rm -rf ./build

.PHONY: proto
proto: ## generate protocol buffers
	@echo "==> Building $@..."
	@mkdir -p $(BUILD_DIR)
	@protoc \
     -I . \
     -I ./third_party/proto/googleapis \
     --go_out $(BUILD_DIR) \
     --go_opt paths=source_relative \
     proto/admin/platform.proto \
     proto/process/apecs.proto \
     proto/storage/rpg.proto
	@protoc \
     -I . \
     -I ./third_party/proto/googleapis \
     -I ./third_party/proto/grpc-gateway \
     --go_out $(BUILD_DIR) \
     --go_opt paths=source_relative \
     --go-grpc_out $(BUILD_DIR) \
     --go-grpc_opt paths=source_relative \
     --grpc-gateway_out $(BUILD_DIR) \
     --grpc-gateway_opt logtostderr=true \
     --grpc-gateway_opt paths=source_relative \
     --grpc-gateway_opt generate_unbound_methods=true \
     --openapiv2_out $(BUILD_DIR) \
	 --openapiv2_opt logtostderr=true \
	 --openapiv2_opt fqn_for_openapi_name=true \
     proto/admin/api.proto \
     proto/edge/api.proto

.PHONY: cmd
cmd: rpg rcedge rcsim ## compile all cmds

.PHONY: rpg
rpg: ## compile rpg cmd
	@echo "==> Building $@..."
	@rm -rf ./pkg/rkcy/__static/admin/docs
	@mkdir -p ./pkg/rkcy/__static/admin/docs
	@cp -rf ./third_party/swagger-ui/* ./pkg/rkcy/__static/admin/docs
	@cp -f $(BUILD_DIR)/proto/admin/api.swagger.json ./pkg/rkcy/__static/admin/docs/swagger.json
	@go build \
	-ldflags $(GO_LDFLAGS) \
	-o $(BUILD_BIN_DIR)/rpg \
	./cmd/rpg
	@cp ./cmd/rpg/platform.json $(BUILD_BIN_DIR)

.PHONY: rcedge
rcedge: ## compile rcedge cmd
	@echo "==> Building $@..."
	@rm -rf ./cmd/rcedge/__static/
	@mkdir -p ./cmd/rcedge/__static/docs
	@cp -rf ./third_party/swagger-ui/* ./cmd/rcedge/__static/docs
	@cp -f $(BUILD_DIR)/proto/edge/api.swagger.json ./cmd/rcedge/__static/docs/swagger.json
	@go build \
	-ldflags $(GO_LDFLAGS) \
	-o $(BUILD_BIN_DIR)/rcedge \
    ./cmd/rcedge

.PHONY: rcsim
rcsim: ## compile rcsim cmd
	@echo "==> Building $@..."
	@go build \
	-ldflags $(GO_LDFLAGS) \
	-o $(BUILD_BIN_DIR)/rcsim \
    ./cmd/rcsim

HELP_FORMAT="    \033[36m%-25s\033[0m %s\n"
.PHONY: help
help: ## display this usage information
	@echo "Valid targets:"
	@grep -E '^[^ ]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		sort | \
		awk 'BEGIN {FS = ":.*?## "}; \
			{printf $(HELP_FORMAT), $$1, $$2}'
	@echo ""
