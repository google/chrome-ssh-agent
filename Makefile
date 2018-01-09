GO	?= GO15VENDOREXPERIMENT=1 go
GOPATH	:= $(firstword $(subst :, ,$(shell $(GO) env GOPATH)))

GOLINT		?= $(GOPATH)/bin/golint
GOPHERJS	?= $(GOPATH)/bin/gopherjs
pkgs		= $(shell $(GO) list ./... | grep -v /vendor/)

CHROME_EXTENSION_KEY=/tmp/chrome-ssh-agent.pem

PREFIX	?= $(shell pwd)
BIN_DIR	?= $(shell pwd)
MAKECRX	?= $(PREFIX)/release/makecrx.sh

NODE_PATH = $(shell $(PREFIX)/install-node.sh)
NPM = $(NODE_PATH)/npm

NODE_MODULES = $(PREFIX)/node_modules
NODE_SOURCE_MAP_SUPPORT=$(NODE_MODULES)/source-map-support
NODE_JSDOM=$(NODE_MODULES)/jsdom
NODE_SYSCALL=$(NODE_MODULES)/syscall.node

PATH := $(NODE_PATH):$(shell echo $$PATH)

all: format style vet lint test build crx

$(NODE_SOURCE_MAP_SUPPORT):
	@$(NPM) install source-map-support

$(NODE_JSDOM):
	@$(NPM) install jsdom

$(NODE_SYSCALL):
	@$(NPM) install node-gyp
	@cd $(GOPATH)/src/github.com/gopherjs/gopherjs/node-syscall && $(NODE_MODULES)/node-gyp/bin/node-gyp.js rebuild
	@ln $(GOPATH)/src/github.com/gopherjs/gopherjs/node-syscall/build/Release/syscall.node $(NODE_MODULES)/syscall.node

format:
	@echo ">> formatting code"
	@$(GO) fmt $(pkgs)

style:
	@echo ">> checking code style"
	@! gofmt -d $(shell find . -path ./vendor -prune -o -name '*.go' -print) | grep '^'

vet:
	@echo ">> vetting code"
	@$(GO) vet $(pkgs)

lint: $(GOLINT)
	@echo ">> linting code"
	@$(GOLINT) $(pkgs)

test: krpretty $(GOPHERJS) $(NODE_SOURCE_MAP_SUPPORT) $(NODE_JSDOM) $(NODE_SYSCALL)
	@echo ">> running tests"
	@$(GOPHERJS) test $(pkgs)

go-options: $(GOPHERJS)
	@echo ">> building options"
	@cd go/options && $(GOPHERJS) build

go-background: $(GOPHERJS)
	@echo ">> building background"
	@cd go/background && $(GOPHERJS) build

build: go-options go-background

crx: $(MAKECRX) build
	@echo ">> building Chrome extension"
	@$(MAKECRX) $(CHROME_EXTENSION_KEY)

$(GOPHERJS):
	@GOOS= GOARCH= $(GO) get -u github.com/gopherjs/gopherjs

$(GOLINT):
	@GOOS= GOARCH= $(GO) get -u github.com/golang/lint/golint

krpretty:
	@GOOS= GOARCH= $(GO) get -u github.com/kr/pretty

.PHONY: all
