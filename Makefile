# MIFO — macOS system monitor
BINARY  := mifo
PKG     := ./cmd/mifo
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
DIST    := dist

# MIFO uses purego for IOKit/SMC access, so cgo is not required and
# cross-compiling between Apple Silicon and Intel works out of the box.
export CGO_ENABLED := 0

.PHONY: all build install run test race live vet fmt release release-arm64 release-amd64 universal clean

all: fmt vet test build

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) $(PKG)

install: build
	install -d /usr/local/bin 2>/dev/null || true
	install -m 755 $(BINARY) /usr/local/bin/$(BINARY) || { mkdir -p $(HOME)/.local/bin && install -m 755 $(BINARY) $(HOME)/.local/bin/$(BINARY); }

run:
	go run $(PKG)

test:
	go test ./...

# Race detector needs cgo.
race:
	CGO_ENABLED=1 go test -race ./...

# Exercise the real collectors against this Mac's hardware.
live:
	MIFO_LIVE=1 go test -run Live -v ./internal/metrics

vet:
	go vet ./...

fmt:
	gofmt -w .

release: clean release-arm64 release-amd64 universal
	cd $(DIST) && shasum -a 256 $(BINARY)-* > SHA256SUMS

release-arm64:
	mkdir -p $(DIST)
	GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-darwin-arm64 $(PKG)

release-amd64:
	mkdir -p $(DIST)
	GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-darwin-amd64 $(PKG)

universal: release-arm64 release-amd64
	lipo -create -output $(DIST)/$(BINARY)-darwin-universal $(DIST)/$(BINARY)-darwin-arm64 $(DIST)/$(BINARY)-darwin-amd64

clean:
	rm -rf $(DIST) $(BINARY)
