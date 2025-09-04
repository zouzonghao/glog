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
BINARY_FILENAME=$(BINARY_NAME)-$(TARGET_GOOS)-$(TARGET_GOARCH)
ifeq ($(TARGET_GOOS), windows)
	BINARY_FILENAME:=$(BINARY_FILENAME).exe
endif

BUILD_SCRIPT=build.go
BUILD_TOOL_NAME=build_tool

# --- Main Targets ---
default: run

release: build-tool prepare build cleanup
	@echo "Release build complete. Binary is at ./$(BINARY_FILENAME)"

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
	@echo "--> Building application for $(TARGET_GOOS)/$(TARGET_GOARCH)..."
	@GOOS=$(TARGET_GOOS) GOARCH=$(TARGET_GOARCH) $(GOBUILD) -ldflags="-s -w" -o $(BINARY_FILENAME) -tags release .

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