BIN ?= docker-socket-firewall
PKG := github.com/linead/docker-socket-firewall

local : ARCH ?= $(shell go env GOOS)-$(shell go env GOARCH)
ARCH ?= linux-amd64

SRC_DIRS := cmd pkg # directories which hold app source (not vendored)

CLI_PLATFORMS := linux-amd64 darwin-amd64 

platform_temp = $(subst -, ,$(ARCH))
GOOS = $(word 1, $(platform_temp))
GOARCH = $(word 2, $(platform_temp))
VERSION ?= master

local: build-dirs
	GOOS=$(GOOS) \
	GOARCH=$(GOARCH) \
	VERSION=$(VERSION) \
	PKG=$(PKG) \
	BIN=$(BIN) \
	OUTPUT_DIR=$$(pwd)/_output/bin/$(GOOS)/$(GOARCH) \
	./hack/build.sh

build: _output/bin/$(GOOS)/$(GOARCH)/$(BIN)

_output/bin/$(GOOS)/$(GOARCH)/$(BIN): build-dirs
	@echo "building: $@"
	$(MAKE) shell CMD="-c '\
    GOOS=$(GOOS) \
    GOARCH=$(GOARCH) \
    VERSION=$(VERSION) \
    PKG=$(PKG) \
    BIN=$(BIN) \
    OUTPUT_DIR=/output/$(GOOS)/$(GOARCH) \
    ./hack/build.sh'"

build-dirs:
	@mkdir -p _output/bin/$(GOOS)/$(GOARCH)
	@mkdir -p .go/src/$(PKG) .go/pkg .go/bin .go/std/$(GOOS)/$(GOARCH) .go/go-build