.DEFAULT_GOAL := ci
BIN ?= docker-socket-firewall
PKG := github.com/linead/docker-socket-firewall
GIT_DIRTY := $(shell test -n "$(git status -z .)" || echo -dirty)
GIT_INFO := $(shell git describe --tags --always)$(GIT_DIRTY)
LDFLAGS = "-X main.gitInfo=${GIT_INFO}"

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
	go build -ldflags $(LDFLAGS)
	mv $(BIN) _output/bin/$(GOOS)/$(GOARCH)/

mac:
	GOOS=darwin \
	GOARCH=amd64 \
	go build -ldflags $(LDFLAGS)
	mv $(BIN) _output/bin/darwin/amd64/

linux:
	GOOS=linux \
	GOARCH=amd64 \
	go build -ldflags $(LDFLAGS)
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


