BINARY    := nexdns
MODULE    := github.com/nexdns/cli
VERSION   ?= dev
COMMIT    := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE      := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS   := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.Date=$(DATE)

.PHONY: build install test lint fmt clean

build:
	go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY) ./cmd/nexdns

install:
	go install -ldflags="$(LDFLAGS)" ./cmd/nexdns

test:
	go test ./... -v

lint:
	golangci-lint run

fmt:
	go fmt ./...

clean:
	rm -rf bin/ dist/
