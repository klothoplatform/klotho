GO_FILES := $(shell find . -type f -name '*.go') go.mod go.sum
ENGINE_TEMPLATES := $(shell find . -type f -path pkg/templates)
IAC_TEMPLATES := $(shell find . -type f -path pkg/infra/iac3/templates -and -not -path */node_modules/*)

engine: $(GO_FILES) $(ENGINE_TEMPLATES)
	CGO_ENABLED=1 \
	GOOS=linux \
	GOARCH=amd64 \
	CC="zig cc -target x86_64-linux-musl" \
	CXX="zig c++ -target x86_64-linux-musl" \
	go build --tags extended -o engine -ldflags="-s -w" ./cmd/engine

iac: $(GO_FILES) $(IAC_TEMPLATES)
	CGO_ENABLED=1 \
	GOOS=linux \
	GOARCH=amd64 \
	CC="zig cc -target x86_64-linux-musl" \
	CXX="zig c++ -target x86_64-linux-musl" \
	go build --tags extended -o iac -ldflags="-s -w" ./cmd/iac

.PHONY: clean_debug
clean_debug:
	find . \( -name '*.gv' -or -name '*.gv.svg' \) -delete
