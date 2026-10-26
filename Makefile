GO ?= go
UV ?= uv
GOEXE = $(shell $(GO) env GOEXE)
TARGETS := windows/386 windows/amd64 windows/arm64 linux/386 linux/amd64 linux/arm64 darwin/amd64 darwin/arm64
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null)
LDFLAGS := -s -w -X github.com/gzj/selfext/internal/version.gitVersion=$(VERSION)

.PHONY: all build gui gui-run gui-test stubs test release clean
all: build

# Compile the eight extractors once, with the developer's installed Go toolchain.
stubs:
	$(GO) run ./cmd/build-stubs

# A native packager containing ALL target extractors; no SDK is embedded.
build:
	mkdir -p target
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags '$(LDFLAGS)' -o target/ .

# PyInstaller bundles the native CLI with the GUI on the current platform.
gui: build
	$(UV) run --project gui --locked --group build python -m PyInstaller --onefile --windowed --noconfirm --noupx \
		--name selfextg --distpath target --workpath build/pyinstaller \
		--specpath build --add-binary "$(abspath target/selfext$(GOEXE)):." gui/main.py

# Launch and test the independent uv/PySide6 project.
gui-run: build
	$(UV) run --project gui --locked selfextg

gui-test: build
	$(UV) run --project gui --locked python -m unittest discover -s gui/tests -v

test:
	$(GO) test ./...

# Distribute a standalone packager for each supported host.
release:
	mkdir -p target release
	@set -eu; for target in $(TARGETS); do \
		os=$${target%/*}; arch=$${target#*/}; name=selfext_$${os}-$${arch}; \
		if [ "$$os" = windows ]; then name=$$name.exe; fi; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GO) build -trimpath -ldflags '$(LDFLAGS)' -o target/$$name .; \
		tar -czf release/$$name.tar.gz -C target $$name -C .. README.md LICENSE THIRD_PARTY_NOTICES.md; \
	done

clean:
	rm -rf target release build
