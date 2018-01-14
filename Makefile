GO		?= GO15VENDOREXPERIMENT=1 go
GOPATH		:= $(firstword $(subst :, ,$(shell $(GO) env GOPATH)))

GOLINT		?= $(GOPATH)/bin/golint
GOPHERJS	?= $(GOPATH)/bin/gopherjs
pkgs		= $(shell $(GO) list ./... | grep -v /vendor/)

PREFIX		?= $(shell pwd)
BIN_DIR		?= $(PREFIX)/bin

# These are read by deploy-webstore.py, so must be exported.
export EXTENSION_ID	= eechpbnaifiimgajnomdipfaamobdfha
export EXTENSION_ZIP	= $(BIN_DIR)/chrome-ssh-agent.zip
export PUBLISH_TARGET	= default

# Finding node-gyp requires going up one level and then querying. We do not want
# to find our own node_modules directory.
NODE_GYP	= $(shell npm bin)/node-gyp
NODE_SYSCALL	= node_modules/syscall.node


all: format style vet lint test build zip

$(NODE_SYSCALL):
	# See https://github.com/gopherjs/gopherjs/blob/master/doc/syscalls.md
	@cd $(GOPATH)/src/github.com/gopherjs/gopherjs/node-syscall && $(NODE_GYP) rebuild
	@mkdir -p $(shell dirname $(NODE_SYSCALL))
	@ln $(GOPATH)/src/github.com/gopherjs/gopherjs/node-syscall/build/Release/syscall.node $(NODE_SYSCALL)

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

test: $(GOPHERJS) $(NODE_SYSCALL)
	@echo ">> running tests"
	@$(GOPHERJS) test $(pkgs)

build: $(GOPHERJS)
	@echo ">> building"
	@cd go/options && $(GOPHERJS) build
	@cd go/background && $(GOPHERJS) build

$(EXTENSION_ZIP): build
	@echo ">> building Chrome extension"
	@mkdir -p $(shell dirname $(EXTENSION_ZIP))
	@zip -qr -9 -X "${EXTENSION_ZIP}" . --include \
		manifest.json \
		\*.css \
		\*.html \
		\*.js \
		\*.png \
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
