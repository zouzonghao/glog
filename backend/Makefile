# Makefile for the Glog project

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run

# --- Platform Configuration ---
TARGET_GOOS := $(or $(GOOS),$(shell go env GOHOSTOS))
TARGET_GOARCH := $(or $(GOARCH),$(shell go env GOHOSTARCH))

# --- Binary Naming ---
BINARY_NAME=glog
BINARY_FILENAME=$(BINARY_NAME)-$(TARGET_GOOS)-$(TARGET_GOARCH)

# --- Main Targets ---
.PHONY: default run build release clean

default: run

# Run the application in development mode
run:
	@echo "--> Running in development mode..."
	@$(GORUN) .

# Build the application for the target platform
build:
	@{ \
		FILENAME=$(BINARY_FILENAME); \
		if [ "$(TARGET_GOOS)" = "windows" ]; then \
			FILENAME=$${FILENAME}.exe; \
		fi; \
		echo "--> Building application for $(TARGET_GOOS)/$(TARGET_GOARCH)..."; \
		CGO_ENABLED=1 GOOS=$(TARGET_GOOS) GOARCH=$(TARGET_GOARCH) $(GOBUILD) -ldflags="-s -w" -o $$FILENAME .; \
	}

# Build the release binary for linux/amd64
release:
	@$(MAKE) build GOOS=linux GOARCH=amd64
	@echo "--> Linux release build complete."

# Clean up build artifacts
clean:
	@echo "--> Cleaning up project..."
	@rm -f $(BINARY_NAME) $(BINARY_NAME)-*
	@echo "--> Project cleaned."
