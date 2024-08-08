GO_VERSION := $(shell awk '/^go / {print $$2; exit}' go.mod)
GO_ARCH_EXT := tar.gz
GO_ARCH_EXT_WIN := zip

OS_WINDOWS := windows
OS_LINUX := linux
OS_DARWIN := darwin

ARCH_32 := 386
ARCH_64 := amd64
ARCH_ARM64 := arm64

COLOR_RESET=\033[0m
COLOR_GREEN=\033[32m

.PHONY: all download_go

# ---------------------------- build && release ------------------------------------------
all: target/$(OS_WINDOWS)-$(ARCH_32) \
     target/$(OS_WINDOWS)-$(ARCH_64) \
     target/$(OS_WINDOWS)-$(ARCH_ARM64) \
     target/$(OS_LINUX)-$(ARCH_32) \
     target/$(OS_LINUX)-$(ARCH_64) \
     target/$(OS_LINUX)-$(ARCH_ARM64) \
     target/$(OS_DARWIN)-$(ARCH_64) \
     target/$(OS_DARWIN)-$(ARCH_ARM64)

define target-go
target/$(1)-$(2):
	@echo "$(COLOR_GREEN)--------------------- Building for $(1)/$(2) ---------------------$(COLOR_RESET)"
	$$(MAKE) OS=$1 ARCH=$2 download_go
	$$(MAKE) generate_wrapper
	$$(MAKE) Go_Arch=go$$(GO_VERSION).$1-$2.$3 generate_assets 
	$$(MAKE) OS=$(1) ARCH=$(2) GoVersionEmbeded=go$$(GO_VERSION).$(1)-$(2) BIN=selfext_$(1)-$(2)$(if $(filter $(1),$(OS_WINDOWS)),.exe) go_build
endef

$(eval $(call target-go,$(OS_WINDOWS),$(ARCH_32),$(GO_ARCH_EXT_WIN)))
$(eval $(call target-go,$(OS_WINDOWS),$(ARCH_64),$(GO_ARCH_EXT_WIN)))
$(eval $(call target-go,$(OS_WINDOWS),$(ARCH_ARM64),$(GO_ARCH_EXT_WIN)))

$(eval $(call target-go,$(OS_LINUX),$(ARCH_32),$(GO_ARCH_EXT)))
$(eval $(call target-go,$(OS_LINUX),$(ARCH_64),$(GO_ARCH_EXT)))
$(eval $(call target-go,$(OS_LINUX),$(ARCH_ARM64),$(GO_ARCH_EXT)))

$(eval $(call target-go,$(OS_DARWIN),$(ARCH_64),$(GO_ARCH_EXT)))
$(eval $(call target-go,$(OS_DARWIN),$(ARCH_ARM64),$(GO_ARCH_EXT)))

release: all
	md release
	for file in *; do
	  if [ -f "$file" ]; then
		if [[ "$file" == *.exe ]]; then
		  zip "${file}.zip" "$file"
		else
		  tar -czvf "${file}.tar.gz" "$file"
		fi
	  fi
	done
	mv target/*.tar.gz release/
	mv target/*.zip release/

# ---------------------------- test ------------------------------------------
test:

# ---------------------------- clean ------------------------------------------
clean:
	rm -rf target
	rm -rf assets/go
	rm -f assets/assets.go
	rm -rf assets/wrapper/vendor
	rm -rf assets/wrapper/wrapper.go
	rm -rf assets/wrapper/wrapper.zip

# ---------------------------- download go sdk ------------------------------------------
download_go:
	mkdir -p assets/go	
	@echo "Fetching Go version $(GO_VERSION)..."
	@if [ "$(OS)" = "windows" ]; then \
		filename="go$(GO_VERSION).$${OS}-$${ARCH}.$(GO_ARCH_EXT_WIN)"; \
	else \
		filename="go$(GO_VERSION).$${OS}-$${ARCH}.$(GO_ARCH_EXT)"; \
	fi; \
	echo "Checking if file exists: assets/go/$${filename}"; \
	if [ ! -f "assets/go/$${filename}" ]; then \
		echo "Downloading Go: $${filename}"; \
		curl -Lo assets/go/$${filename} "https://go.dev/dl/$${filename}"; \
		echo "Download complete: $${filename}"; \
	else \
		echo "File already exists: assets/go/$${filename}"; \
	fi

# ---------------------------- go run generate wrapper.go ------------------------------------------
Wrapper_Arch := wrapper.zip
generate_wrapper:
	@echo "$(COLOR_GREEN)1. generate wrapper.go......$(COLOR_RESET)"
	cd assets/wrapper &&  \
	go run wrapper_generator.go && \
	go fmt wrapper.go && \
	trap 'mv go.mod go.mod_' INT TERM EXIT && \
	mv go.mod_ go.mod && \
	go mod tidy && \
	go mod vendor && \
	7z a $(Wrapper_Arch) go.mod go.sum wrapper_generator.go vendor -bso0 -bse0 && \
	mv go.mod go.mod_ 

# ---------------------------- go run generate assets.go ------------------------------------------
generate_assets:
	@echo "$(COLOR_GREEN)2. generate assets.go......$(COLOR_RESET)"
	cd assets &&  \
	go run assets_generator.go -go $${Go_Arch} -wrapper $(Wrapper_Arch)

# ---------------------------- go build selfext.go with flags ------------------------------------------
go_build:
	@echo "$(COLOR_GREEN)3. go build......$(COLOR_RESET)"
	GIT_VERSION=`git describe --tags --abbrev=0 | tr -d '\n' | sed 's/^v//'` && \
	GIT_COMMIT=`git rev-parse HEAD` && \
	GIT_BRANCH=`git rev-parse --abbrev-ref HEAD` && \
	BUILD_DATE=`date -u +%Y-%m-%dT%H:%M:%SZ` && \
	env GOOS=$${OS} GOARCH=$${ARCH} go build -ldflags "\
		-X 'github.com/gzj/selfext/internal/version.gitVersion=$${GIT_VERSION}' \
		-X 'github.com/gzj/selfext/internal/version.gitCommit=$${GIT_COMMIT}' \
		-X 'github.com/gzj/selfext/internal/version.gitBranch=$${GIT_BRANCH}' \
		-X 'github.com/gzj/selfext/internal/version.buildDate=$${BUILD_DATE}' \
		-X 'github.com/gzj/selfext/internal/version.goVersionEmbeded=$${GoVersionEmbeded}'" \
		-o ./target/$${BIN} selfext.go
