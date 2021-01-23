.DEFAULT_GOAL := bin

.PHONY: bin dev

# Repository name.
REPO_NAME := $(shell (head -n 1 go.mod | sed 's/^module //g'))

# Name of tool.
TOOL_NAME := wf

# Git tag, for versioning.
GIT_TAG := $(shell (git describe --tags --always --dirty 2>/dev/null || echo 'unknown'))

# Build time, for versioning.
ifeq ($(shell uname),Linux)
BUILD_TIME := $(shell date -u --iso-8601=seconds)
else
BUILD_TIME := $(shell gdate -u --iso-8601=seconds)
endif

# Go-related variables, for naming binaries.
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

# Build flags and variables..
BASE_LD_FLAGS := -X $(REPO_NAME)/cmd.Version=$(GIT_TAG)
BASE_LD_FLAGS += -X $(REPO_NAME)/cmd.BuildTime=$(BUILD_TIME)
# Allow for optional LD flags from env, appended to base flags, stripping trailing whitespace.
LD_FLAGS := $(strip $(BASE_LD_FLAGS) $(LD_FLAGS))
# Build flags that can not be overwritten.
BASE_BUILD_FLAGS := -ldflags="$(LD_FLAGS)"

BUILD_DIR ?= target
$(BUILD_DIR):
	mkdir -p $@

BIN_NAME := $(BUILD_DIR)/$(TOOL_NAME)
BUILD_FLAGS := $(strip $(BASE_BUILD_FLAGS) $(BUILD_FLAGS))

ifdef RELEASE
BIN_NAME := $(BIN_NAME)-$(GOOS)-$(GOARCH)-$(GIT_TAG)
BUILD_FLAGS += -trimpath
endif

###

bin: $(BIN_NAME)
$(BIN_NAME): $(wildcard wait/*.go) $(wildcard cmd/*.go) | $(BUILD_DIR)
	go build $(BUILD_FLAGS) -o $@

clean:
	rm -rf $(BUILD_DIR) cov.out cov.html

dev:
	go get -t -u github.com/kyoh86/richgo
	go get -t -u golang.org/x/lint/golint
	go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.35.2

lint:
	golangci-lint run

lint-strict:
	golangci-lint run $(addprefix -E ,lll gosec nestif prealloc unconvert gocritic)

test:
	richgo test -race -coverprofile cov.out -v ./...
	go tool cover -html=cov.out -o cov.html
