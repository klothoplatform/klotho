GO_FILES := $(wildcard pkg/**/*.go) go.sum go.mod

engine: $(GO_FILES)
	CGO_ENABLED=1 \
	GOOS=linux \
	GOARCH=amd64 \
	CC="zig cc -target x86_64-linux-musl" \
	CXX="zig c++ -target x86_64-linux-musl" \
	go build --tags extended -o engine -ldflags="-s -w" ./cmd/engine

iac: $(GO_FILES)
	CGO_ENABLED=1 \
	GOOS=linux \
	GOARCH=amd64 \
	CC="zig cc -target x86_64-linux-musl" \
	CXX="zig c++ -target x86_64-linux-musl" \
	go build --tags extended -o iac -ldflags="-s -w" ./cmd/iac
