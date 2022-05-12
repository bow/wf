# Common development tasks.

# Cross-platform adjustments.
SYS := $(shell uname 2> /dev/null)
ifeq ($(SYS),Linux)
DATE_EXE := date
GREP_EXE := grep
SED_EXE  := sed
else ifeq ($(SYS),Darwin)
DATE_EXE := gdate
GREP_EXE := ggrep
SED_EXE  := gsed
else
$(error Unsupported development platform)
endif

APP_NAME   := wf
GO_VERSION := $(shell (head -n 3 go.mod | $(SED_EXE) 's/^go//g' | tail -n 1))
REPO_NAME  := $(shell (head -n 1 go.mod | $(SED_EXE) 's/^module //g'))

GIT_TAG    := $(shell git describe --tags --always --dirty 2> /dev/null || echo "untagged")
GIT_COMMIT := $(shell git rev-parse --quiet --verify HEAD || echo "?")
GIT_DIRTY  := $(shell test -n "$(shell git status --porcelain)" && echo "-dirty" || true)
BUILD_TIME := $(shell $(DATE_EXE) -u '+%Y-%m-%dT%H:%M:%SZ')
IS_RELEASE := $(shell ((echo "${GIT_TAG}" | $(GREP_EXE) -qE "^v?[0-9]+\.[0-9]+\.[0-9]+$$") && echo '1') || true)

BIN_DIR  ?= $(CURDIR)/bin
BIN_NAME ?= $(APP_NAME)

# Linker flags for go-build
# BASE_LD_FLAGS are linker flags that can not be overwritten.
BASE_LD_FLAGS := -X ${REPO_NAME}/cmd.version=$(GIT_TAG)
BASE_LD_FLAGS += -X ${REPO_NAME}/cmd.buildTime=$(BUILD_TIME)
BASE_LD_FLAGS += -X ${REPO_NAME}/cmd.gitCommit=$(GIT_COMMIT)$(GIT_DIRTY)

# Allow for optional LD flags from env, appended to base flags, stripping trailing whitespaces.
LD_FLAGS := $(strip $(BASE_LD_FLAGS) $(LD_FLAGS))


all: help


.PHONY: bin
bin: $(BIN_DIR)/$(BIN_NAME)  ## Compile an executable binary.

$(BIN_DIR)/$(BIN_NAME): $(shell find . -type f -name '*.go' -print) go.mod
	go mod tidy && go build -trimpath -ldflags '$(LD_FLAGS)' -o $@


.PHONY: clean
clean:  ## Remove all build artifacts.
	@rm -f bin/* coverage.html .coverage.out .junit.xml


.PHONY: install-dev
install-dev:  ## Install dependencies for local development.
	@if command -v asdf 1>/dev/null 2>&1; then \
		if [ ! -f .tool-versions ]; then \
			(asdf plugin add golang 2> /dev/null || true) \
				&& asdf install golang $(GO_VERSION) \
				&& asdf local golang $(GO_VERSION) > .tool-versions; \
		fi; \
		asdf reshim; \
	fi
	go install gotest.tools/gotestsum@v1.8.0 \
		&& go install github.com/boumenot/gocover-cobertura@latest \
		&& go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.46.0
	@if command -v asdf 1>/dev/null 2>&1; then \
		asdf reshim; \
	fi


.PHONY: help
help:  ## Show this help.
	$(eval PADLEN=$(shell $(GREP_EXE) -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| cut -d':' -f1 \
		| awk '{cur = length($$0); lengths[cur] = lengths[cur] $$0 ORS; max=(cur > max ? cur : max)} END {printf "%s", max}' \
		|| (true && echo 0)))
	@($(GREP_EXE) --version > /dev/null 2>&1 || (>&2 "error: GNU grep not installed"; exit 1)) \
		&& printf "\033[36m◉ %s dev console\033[0m\n" "$(APP_NAME)" >&2 \
		&& $(GREP_EXE) -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
			| awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m» \033[33m%*-s\033[0m \033[36m· \033[0m%s\n", $(PADLEN), $$1, $$2}' \
			| sort


.PHONY: lint
lint:  ## Lint the code.
	golangci-lint run


.PHONY: test .coverage.out
test: .coverage.out  ## Run the test suite.
.coverage.out:
	gotestsum --format dots-v2 --junitfile .junit.xml -- ./... -parallel=$(shell nproc) -coverprofile=$@ -covermode=atomic \
		&& go tool cover -func=$@


.PHONY: test-cov-xml
test-cov-xml: .coverage.out  ## Run the test suite and output coverage to XML.
	gocover-cobertura < $< > .coverage.xml


.PHONY: test-cov-html
test-cov-html: .coverage.out  ## Run the test suite and output coverage to HTML.
	go tool cover -html=.coverage.out -o coverage.html
