# octonote — Build System
# Targets: tui, gui, all, cross, clean

BINARY_NAME   := octonote
GUI_BINARY    := octonote-gui
MODULE        := github.com/nottaker/octonote
VERSION       := 2.2.0
LDFLAGS       := -s -w -X main.version=$(VERSION)

# Directories
TUI_DIR       := ./tui
GUI_DIR       := ./gui
NPM_BIN_DIR   := ./npm/bin/binaries

# Cross-compile targets: OS/ARCH pairs
PLATFORMS := \
	darwin/amd64 \
	darwin/arm64 \
	linux/amd64  \
	linux/arm64  \
	windows/amd64 \
	windows/arm64

# Wails path helper
WAILS := $(shell which wails 2>/dev/null || echo $(shell go env GOPATH)/bin/wails)

.PHONY: all tui gui cross clean deps fmt vet test help

## Default: build both TUI and GUI
all: deps tui gui

## Build the TUI (Bubble Tea) binary for the current platform
tui: deps
	@echo "→ Building TUI…"
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) $(TUI_DIR)
	@echo "✓ $(BINARY_NAME) built"

## Build the Wails GUI desktop app
gui: deps
	@echo "→ Building Wails GUI…"
	cd $(GUI_DIR) && $(WAILS) build -clean -ldflags "$(LDFLAGS)"
	@if [ -d "$(GUI_DIR)/build/bin/octoNote.app" ]; then \
		echo "#!/bin/sh" > $(GUI_BINARY) ; \
		echo "DIR=\"\$$(cd \"\$$(dirname \"\$$0\")\" && pwd)\"" >> $(GUI_BINARY) ; \
		echo "exec \"\$$DIR/$(GUI_DIR)/build/bin/octoNote.app/Contents/MacOS/octonote\" \"\$$@\"" >> $(GUI_BINARY) ; \
		chmod +x $(GUI_BINARY) ; \
	elif [ -f "$(GUI_DIR)/build/bin/octonote" ]; then \
		cp "$(GUI_DIR)/build/bin/octonote" $(GUI_BINARY) ; \
		chmod +x $(GUI_BINARY) ; \
	fi
	@echo "✓ Wails build complete. Created ./$(GUI_BINARY) and $(GUI_DIR)/build/bin/"

## Run the TUI in development mode (live reload via air, if installed)
dev-tui:
	go run $(TUI_DIR)/main.go

## Run the Wails GUI in development mode (hot reload)
dev-gui:
	cd $(GUI_DIR) && $(WAILS) dev

## Cross-compile TUI for all release platforms and copy into npm/bin/binaries
cross: deps
	@echo "→ Cross-compiling for all platforms…"
	@mkdir -p $(NPM_BIN_DIR)
	$(foreach PLATFORM, $(PLATFORMS), \
		$(eval OS   := $(word 1, $(subst /, ,$(PLATFORM)))) \
		$(eval ARCH := $(word 2, $(subst /, ,$(PLATFORM)))) \
		$(eval EXT  := $(if $(filter windows,$(OS)),.exe,)) \
		$(eval OUT  := $(NPM_BIN_DIR)/$(BINARY_NAME)-$(OS)-$(if $(filter amd64,$(ARCH)),amd64,$(ARCH))$(EXT)) \
		GOOS=$(OS) GOARCH=$(ARCH) go build -trimpath -ldflags "$(LDFLAGS)" -o $(OUT) $(TUI_DIR) && \
		echo "  ✓ $(OUT)" ; \
	)
	@echo "✓ Cross-compilation complete"

## Download/update Go module dependencies
deps:
	go mod tidy
	go mod download

## Format all Go source files
fmt:
	gofmt -l -w .

## Run Go vet static analysis
vet:
	go vet ./...

## Run all tests
test:
	go test -race ./...

## Remove build artefacts
clean:
	rm -f $(BINARY_NAME) $(GUI_BINARY)
	rm -f $(NPM_BIN_DIR)/octonote-*
	cd $(GUI_DIR) && rm -rf build/
	@echo "✓ Clean"

## Show available targets
help:
	@grep -E '^##' $(MAKEFILE_LIST) | sed 's/## //'
