APP_NAME := azbutils
VERSION := $(shell git describe --tags --always --dirty)
BUILD_DIR := build
LDFLAGS := -X "main.version=$(VERSION)"

# Supported platforms for cross-compilation
PLATFORMS := \
	darwin/arm64 \
	darwin/amd64 \
	linux/amd64 \
	linux/arm64 \
	windows/amd64

all: clean build

# Build for current platform
build:
	@echo "Building $(APP_NAME) ($(VERSION))..."
	@go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) .

# Cross-compile for all target platforms
build-all:
	@echo "Cross-compiling..."
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*}; \
		GOARCH=$${platform#*/}; \
		output="$(BUILD_DIR)/$(APP_NAME)-$${GOOS}-$${GOARCH}"; \
		if [ "$${GOOS}" = "windows" ]; then output="$${output}.exe"; fi; \
		echo "Building for $${GOOS}/$${GOARCH}..."; \
		GOOS=$${GOOS} GOARCH=$${GOARCH} go build -ldflags "$(LDFLAGS)" -o "$${output}" . || exit 1; \
	done

# Install locally
install:
	@echo "Installing $(APP_NAME) to GOPATH/bin..."
	go install -ldflags "$(LDFLAGS)" .

# Cleanup build artifacts
clean:
	@echo "Cleaning build directory..."
	rm -rf $(BUILD_DIR)

.PHONY: all build build-all install clean
