.PHONY: build clean test lint install cover tidy

BINARY=vault
BINARY_DAEMON=vaultd
BUILD_DIR=./build
GO=go
GOFLAGS=-ldflags="-s -w"

build:
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/vault

build-all: build
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_DAEMON) ./cmd/vaultd

clean:
	rm -rf $(BUILD_DIR) cover.out coverage/

test:
	$(GO) test ./... -v -count=1 -race -coverprofile=cover.out

test-short:
	$(GO) test ./... -v -count=1 -short -race

cover: test
	$(GO) tool cover -html=cover.out -o coverage/index.html
	@echo "Coverage report: file://$(PWD)/coverage/index.html"

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	}
	golangci-lint run ./...

install:
	$(GO) install ./cmd/vault
	$(GO) install ./cmd/vaultd

tidy:
	$(GO) mod tidy
	$(GO) mod verify

dev: build
	$(BUILD_DIR)/$(BINARY)
