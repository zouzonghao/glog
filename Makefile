# Makefile for the Glog project

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run

# --- Platform Configuration ---
# Determine the host OS and architecture. GOHOSTOS/GOHOSTARCH are not affected by GOOS/GOARCH env vars.
NATIVE_GOOS := $(shell go env GOHOSTOS)
NATIVE_GOARCH := $(shell go env GOHOSTARCH)

# Set target OS and architecture. Use GOOS/GOARCH from the environment if provided,
# otherwise default to the host's native platform.
TARGET_GOOS := $(or $(GOOS),$(NATIVE_GOOS))
TARGET_GOARCH := $(or $(GOARCH),$(NATIVE_GOARCH))

# --- Binary Naming ---
BINARY_NAME=glog
# This is now a template for build-platform
BINARY_FILENAME_TEMPLATE=$(BINARY_NAME)-$(1)-$(2)

BUILD_SCRIPT=build.go
BUILD_TOOL_NAME=build_tool

# --- Main Targets ---
default: run

# New target to build all release binaries
release-all: build-tool prepare
	@$(MAKE) build-platform GOOS=linux GOARCH=amd64
	@$(MAKE) build-platform GOOS=windows GOARCH=amd64
	@$(MAKE) build-platform GOOS=darwin GOARCH=arm64
	@$(MAKE) cleanup
	@echo "All release builds complete."

# Renamed 'release' to 'build-platform' to avoid confusion and handle single platform builds.
build-platform: build-tool prepare
	@$(MAKE) build
	@$(MAKE) cleanup

run:
	@echo "Running in development mode..."
	@$(GORUN) .

# --- Build Steps ---
build-tool:
	@echo "--> Building the build tool for $(NATIVE_GOOS)/$(NATIVE_GOARCH)..."
	@GOOS=$(NATIVE_GOOS) GOARCH=$(NATIVE_GOARCH) $(GOBUILD) -o $(BUILD_TOOL_NAME) $(BUILD_SCRIPT)
	@chmod +x $(BUILD_TOOL_NAME)

prepare:
	@echo "--> Preparing assets for release..."
	@./$(BUILD_TOOL_NAME) -release

build:
	@{ \
		BINARY_FILENAME=$(BINARY_NAME)-$(TARGET_GOOS)-$(TARGET_GOARCH); \
		if [ "$(TARGET_GOOS)" = "windows" ]; then \
			BINARY_FILENAME=$${BINARY_FILENAME}.exe; \
		fi; \
		echo "--> Building application for $(TARGET_GOOS)/$(TARGET_GOARCH)..."; \
		CGO_ENABLED=0 GOOS=$(TARGET_GOOS) GOARCH=$(TARGET_GOARCH) $(GOBUILD) -ldflags="-s -w" -o $$BINARY_FILENAME -tags release .; \
	}

cleanup:
	@echo "--> Cleaning up temporary assets..."
	@./$(BUILD_TOOL_NAME) -clean
	@rm -f $(BUILD_TOOL_NAME)

clean:
	@echo "Cleaning up project..."
	@if [ -f $(BUILD_TOOL_NAME) ]; then ./$(BUILD_TOOL_NAME) -clean; fi
	@rm -f $(BUILD_TOOL_NAME)
	@rm -f glog glog-*
	@echo "Project cleaned."

.PHONY: default release-all build-platform run build-tool prepare build cleanup clean