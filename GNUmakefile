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

default: cmd

.PHONY: cmd
cmd: gaeneco simulate

.PHONY: gaeneco
gaeneco:
	@echo "==> Building $@..."
	@GOBIN=$(BUILD_BIN_DIR) go install ./cmd/gaeneco

.PHONY: simulate
simulate:
	@echo "==> Building $@..."
	@GOBIN=$(BUILD_BIN_DIR) go install ./cmd/simulate

HELP_FORMAT="    \033[36m%-25s\033[0m %s\n"
.PHONY: help
help: ## Display this usage information
	@echo "Valid targets:"
	@grep -E '^[^ ]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		sort | \
		awk 'BEGIN {FS = ":.*?## "}; \
			{printf $(HELP_FORMAT), $$1, $$2}'
	@echo ""
