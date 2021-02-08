.DEFAULT_GOAL := ci
BIN ?= docker-socket-firewall
PKG := github.com/linead/docker-socket-firewall

TAG_COMMIT := $(shell git rev-list --abbrev-commit --tags --max-count=1)
TAG := $(shell git describe --abbrev=0 --tags ${TAG_COMMIT} 2>/dev/null || true)
COMMIT := $(shell git rev-parse --short HEAD)
DATE := $(shell git log -1 --format=%cd --date=format:"%Y%m%d")
VERSION := $(TAG:v%=%)
ifneq ($(COMMIT), $(TAG_COMMIT))
	VERSION := $(VERSION)-next-$(COMMIT)-$(DATE)
endif
ifeq ($(VERSION,), "" )
	VERSION := $(COMMIT)-$(DATA)
endif
ifneq ($(shell git status --porcelain),)
	VERSION := $(VERSION)-dirty
endif

LDFLAGS := -ldflags "-X main.version=$(VERSION)"

local : ARCH ?= $(shell go env GOOS)-$(shell go env GOARCH)
ARCH ?= linux-amd64

CLI_PLATFORMS := linux-amd64 darwin-amd64 

platform_temp = $(subst -, ,$(ARCH))
GOOS = $(word 1, $(platform_temp))
GOARCH = $(word 2, $(platform_temp))
VERSION ?= master

local: build-dirs
	GOOS=$(GOOS) \
	GOARCH=$(GOARCH) \
	go build $(LDFLAGS)
	mv $(BIN) _output/bin/$(GOOS)/$(GOARCH)/

mac:
	GOOS=darwin \
	GOARCH=amd64 \
	go build $(LDFLAGS)
	mv $(BIN) _output/bin/darwin/amd64/

linux:
	GOOS=linux \
	GOARCH=amd64 \
	go build $(LDFLAGS)
	mv $(BIN) _output/bin/linux/amd64/

tests:
	go test -covermode=count ./...

ci: build-ci-dirs tests mac linux

build-ci-dirs:
	@mkdir -p _output/bin/linux/amd64 _output/bin/darwin/amd64
	@mkdir -p .go/src/$(PKG) .go/pkg .go/bin .go/std/linux/amd64 .go/std/darwin/amd64 .go/go-build

build-dirs:
	@mkdir -p _output/bin/$(GOOS)/$(GOARCH)
	@mkdir -p .go/src/$(PKG) .go/pkg .go/bin .go/std/$(GOOS)/$(GOARCH) .go/go-build


