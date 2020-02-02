.PHONY: test

wf:
	go build -ldflags="-X github.com/bow/wf/cmd.version=$$(git describe 2>/dev/null || git rev-parse --short HEAD) -X github.com/bow/wf/cmd.buildTime=$$(date -u --iso-8601=seconds)" -o $@

test:
	go test -race -coverprofile c.out -v ./...
