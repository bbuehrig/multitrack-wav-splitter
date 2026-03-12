# Multitrack WAV Splitter – build for multiple OS/arch
# Output: build/mtwav-split-<os>-<arch>[.exe]

BINARY := mtwav-split
BUILD_DIR := build
MAIN := ./cmd/mtwav-split

# Build for host OS/arch only
.PHONY: build
build:
	go build -o $(BUILD_DIR)/$(BINARY) $(MAIN)

# Cross-compilation targets
OS_ARCHES := \
	windows-amd64 \
	windows-arm64 \
	linux-amd64 \
	linux-arm64 \
	darwin-amd64 \
	darwin-arm64

.PHONY: build-all $(OS_ARCHES)
build-all: $(OS_ARCHES)

define build_target
$(1):
	@mkdir -p $(BUILD_DIR)
	GOOS=$(word 1,$(subst -, ,$(1))) GOARCH=$(word 2,$(subst -, ,$(1))) go build -o $(BUILD_DIR)/$(BINARY)-$(1)$(if $(filter windows-%,$(1)),.exe,) $(MAIN)
	@echo "Built $(BUILD_DIR)/$(BINARY)-$(1)$(if $(filter windows-%,$(1)),.exe,)"
endef

$(foreach t,$(OS_ARCHES),$(eval $(call build_target,$(t))))

# Aliases for common targets
.PHONY: build-windows-amd64 build-windows-arm64 build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64
build-windows-amd64: windows-amd64
build-windows-arm64: windows-arm64
build-linux-amd64:   linux-amd64
build-linux-arm64:   linux-arm64
build-darwin-amd64:  darwin-amd64
build-darwin-arm64:  darwin-arm64

# WASM build for browser GUI
.PHONY: wasm wasm-exec web
WASM_OUT := web/public/mtwav-split.wasm
WASM_EXEC := web/public/wasm_exec.js

wasm: wasm-exec
	GOOS=js GOARCH=wasm go build -o $(WASM_OUT) ./cmd/wasm
	@echo "Built $(WASM_OUT)"

# Build full web app (WASM + React production build)
web: wasm
	cd web && npm run build
	@echo "Built web/dist/"

wasm-exec:
	@mkdir -p web/public
	@GOROOT="$$(go env GOROOT)"; \
	if [ -f "$$GOROOT/lib/wasm/wasm_exec.js" ]; then \
	  cp "$$GOROOT/lib/wasm/wasm_exec.js" $(WASM_EXEC); \
	elif [ -f "$$GOROOT/misc/wasm/wasm_exec.js" ]; then \
	  cp "$$GOROOT/misc/wasm/wasm_exec.js" $(WASM_EXEC); \
	else \
	  echo "Error: wasm_exec.js not found in GOROOT (try lib/wasm or misc/wasm)"; exit 1; \
	fi
	@echo "Copied wasm_exec.js to $(WASM_EXEC)"

# Tests and clean
.PHONY: test clean
test:
	go test ./...

clean:
	rm -rf $(BUILD_DIR)
