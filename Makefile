BIN        := hagane
PKG        := ./cmd/hagane
CGO_ENABLED := 0

VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE       := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS    := -s -w \
	-X github.com/tamnd/hagane/cli.Version=$(VERSION) \
	-X github.com/tamnd/hagane/cli.Commit=$(COMMIT) \
	-X github.com/tamnd/hagane/cli.Date=$(DATE)

.PHONY: build install test test-short vet tidy clean run emit-hello

build:
	CGO_ENABLED=$(CGO_ENABLED) go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BIN) $(PKG)

install:
	CGO_ENABLED=$(CGO_ENABLED) go install -trimpath -ldflags "$(LDFLAGS)" $(PKG)

test:
	go test -race ./...

test-short:
	go test -short ./...

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf bin/ dist/

run:
	CGO_ENABLED=$(CGO_ENABLED) go run $(PKG)

emit-hello:
	go run $(PKG) emit ./testdata/hello
