MODULE := github.com/bianoble/agent-sync
BINARY := agent-sync
BUILD_DIR := bin

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -s -w \
	-X '$(MODULE)/cmd/agent-sync/cmd.version=$(VERSION)' \
	-X '$(MODULE)/cmd/agent-sync/cmd.commit=$(COMMIT)' \
	-X '$(MODULE)/cmd/agent-sync/cmd.date=$(DATE)'

.PHONY: build test lint fmt vet clean

build:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) ./cmd/agent-sync

test:
	go test -race -count=1 ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -s -w .
	goimports -w .

vet:
	go vet ./...

clean:
	rm -rf $(BUILD_DIR)
	go clean -testcache
