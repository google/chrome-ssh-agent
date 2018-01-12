GO		?= GO15VENDOREXPERIMENT=1 go
GOPATH		:= $(firstword $(subst :, ,$(shell $(GO) env GOPATH)))

GOLINT		?= $(GOPATH)/bin/golint
GOPHERJS	?= $(GOPATH)/bin/gopherjs
pkgs		= $(shell $(GO) list ./... | grep -v /vendor/)

PREFIX		?= $(shell pwd)
BIN_DIR		?= $(PREFIX)/bin

EXTENSION_ID	= eechpbnaifiimgajnomdipfaamobdfha
EXTENSION_ZIP	= $(BIN_DIR)/chrome-ssh-agent.zip
PUBLISH_TARGET	= trustedTesters

NODE_PATH	= $(shell $(PREFIX)/install-node.sh)
NPM		= $(NODE_PATH)/npm

NODE_MODULES	= $(PREFIX)/node_modules
NODE_SOURCE_MAP_SUPPORT	= $(NODE_MODULES)/source-map-support
NODE_JSDOM	= $(NODE_MODULES)/jsdom
NODE_SYSCALL	= $(NODE_MODULES)/syscall.node

PATH := $(NODE_PATH):$(shell echo $$PATH)


all: format style vet lint test build zip

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

test: $(GOPHERJS) $(NODE_SOURCE_MAP_SUPPORT) $(NODE_JSDOM) $(NODE_SYSCALL)
	@echo ">> running tests"
	@$(GOPHERJS) test $(pkgs)

build: $(GOPHERJS)
	@echo ">> building"
	@cd go/options && $(GOPHERJS) build
	@cd go/background && $(GOPHERJS) build

$(EXTENSION_ZIP): build
	@echo ">> building Chrome extension"
	@zip -qr -9 -X "${EXTENSION_ZIP}" . --include \
		manifest.json \
		\*.css \
		\*.html \
		\*.js \
		\*CONTRIBUTING* \
		\*README* \
		\*LICENCE*

zip: $(EXTENSION_ZIP)

deploy-webstore: $(EXTENSION_ZIP)
	@echo ">> deploying to Chrome Web Store"
	@./deploy-webstore.py

$(GOPHERJS):
	@GOOS= GOARCH= $(GO) get -u github.com/gopherjs/gopherjs

$(GOLINT):
	@GOOS= GOARCH= $(GO) get -u github.com/golang/lint/golint

.PHONY: all
