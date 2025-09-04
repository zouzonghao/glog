# Makefile for the Glog project

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run

# --- Platform Configuration ---
HOST_GOOS := $(shell go env GOOS)
HOST_GOARCH := $(shell go env GOARCH)

TARGET_GOOS ?= $(HOST_GOOS)
TARGET_GOARCH ?= $(HOST_GOARCH)

# --- Binary Naming ---
BINARY_NAME=glog
ifeq ($(TARGET_GOOS)-$(TARGET_GOARCH), $(HOST_GOOS)-$(HOST_GOARCH))
	BINARY_FILENAME=$(BINARY_NAME)
else
	BINARY_FILENAME=$(BINARY_NAME)-$(TARGET_GOOS)-$(TARGET_GOARCH)
endif

BUILD_SCRIPT=build.go
BUILD_TOOL_NAME=build_tool

# --- CGO Cross-Compilation Setup using Zig ---
CGO_ARGS=
ZIG_TARGET=
# Check if we are cross-compiling.
ifeq ($(TARGET_GOOS)-$(TARGET_GOARCH), $(HOST_GOOS)-$(HOST_GOARCH))
	# Native build, no special CGO flags needed.
else
	# Cross-compilation, determine the correct Zig target triple.
	# For linux/amd64, it's x86_64-linux-gnu.
	# This can be expanded for other targets if needed.
ifeq ($(TARGET_GOOS)-$(TARGET_GOARCH), linux-amd64)
	ZIG_TARGET=x86_64-linux-gnu
endif
	# Add more targets here, e.g.:
	# ifeq ($(TARGET_GOOS)-$(TARGET_GOARCH), windows-amd64)
	# ZIG_TARGET=x86_64-windows-gnu
	# endif

	# Configure Go to use Zig as the C/C++ compiler.
	CGO_ARGS=CGO_ENABLED=1 \
	CC="zig cc -target $(ZIG_TARGET)" \
	CXX="zig c++ -target $(ZIG_TARGET)"
endif

# --- Main Targets ---
default: run

release: build-tool prepare build cleanup
	@echo "Release build complete. Binary is at ./$(BINARY_FILENAME)"

run:
	@echo "Running in development mode..."
	@$(GORUN) .

# --- Build Steps ---
build-tool:
	@echo "--> Building the build tool for $(HOST_GOOS)/$(HOST_GOARCH)..."
	@GOOS=$(HOST_GOOS) GOARCH=$(HOST_GOARCH) $(GOBUILD) -o $(BUILD_TOOL_NAME) $(BUILD_SCRIPT)
	@chmod +x $(BUILD_TOOL_NAME)

prepare:
	@echo "--> Preparing assets for release..."
	@./$(BUILD_TOOL_NAME) -release

build:
	@echo "--> Building application for $(TARGET_GOOS)/$(TARGET_GOARCH)..."
	@$(CGO_ARGS) GOOS=$(TARGET_GOOS) GOARCH=$(TARGET_GOARCH) $(GOBUILD) -ldflags="-s -w" -o $(BINARY_FILENAME) -tags release .

cleanup:
	@echo "--> Cleaning up temporary assets..."
	@./$(BUILD_TOOL_NAME) -clean
	@rm -f $(BUILD_TOOL_NAME)

clean:
	@echo "Cleaning up project..."
	@if [ -f $(BUILD_TOOL_NAME) ]; then ./$(BUILD_TOOL_NAME) -clean; fi
	@rm -f $(BUILD_TOOL_NAME)
	@rm -f $(BINARY_NAME) $(BINARY_NAME)-*
	@echo "Project cleaned."

.PHONY: default release run build-tool prepare build cleanup clean