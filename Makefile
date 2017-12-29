GO	?= GO15VENDOREXPERIMENT=1 go
GOPATH	:= $(firstword $(subst :, ,$(shell $(GO) env GOPATH)))

CHROME		?= /usr/bin/google-chrome
GOPHERJS	?= $(GOPATH)/bin/gopherjs
pkgs		= $(shell $(GO) list ./... | grep -v /vendor/)

CHROME_EXTENSION_KEY=/tmp/chrome-ssh-agent.pem

PREFIX	?= $(shell pwd)
BIN_DIR	?= $(shell pwd)
MAKECRX	?= $(PREFIX)/release/makecrx.sh

all: format style vet test build crx

format:
	@echo ">> formatting code"
	@$(GO) fmt $(pkgs)

style:
	@echo ">> checking code style"
	@! gofmt -d $(shell find . -path ./vendor -prune -o -name '*.go' -print) | grep '^'

vet:
	@echo ">> vetting code"
	@$(GO) vet $(pkgs)

test:
	@echo ">> running tests"
	@$(GO) test -short $(pkgs)

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

.PHONY: all
