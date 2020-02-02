.PHONY: release test

EXE_NAME=wf
GIT_TAG=$(shell (git describe --tags --always --dirty))
OS_ARCH_SUFFIX=$(shell go env GOOS)-$(shell go env GOARCH)
BUILD_TIME=$(shell date -u --iso-8601=seconds)

RELEASE_EXE_NAME=$(EXE_NAME)-$(GIT_TAG)-$(OS_ARCH_SUFFIX)

$(RELEASE_EXE_NAME): $(wildcard wait/*.go) $(wildcard cmd/*.go)
	go build -ldflags="-X github.com/bow/wf/cmd.version=$(GIT_TAG) -X github.com/bow/wf/cmd.buildTime=$(BUILD_TIME)" -o $(RELEASE_EXE_NAME)

release: $(RELEASE_EXE_NAME)

test:
	go test -race -coverprofile c.out -v ./...
