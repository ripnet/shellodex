VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -ldflags "-X github.com/ripnet/shellodex/internal/version.Version=$(VERSION) \
                     -X github.com/ripnet/shellodex/internal/version.Commit=$(COMMIT) \
                     -X github.com/ripnet/shellodex/internal/version.BuildDate=$(DATE)"

.PHONY: build install test clean snapshot

build:
	go build $(LDFLAGS) -o shellodex ./cmd/shellodex

install:
	go install $(LDFLAGS) ./cmd/shellodex

test:
	go test ./...

clean:
	rm -f shellodex
	rm -rf dist/

# Build a local snapshot release (all platforms, no publish) via GoReleaser
snapshot:
	goreleaser release --snapshot --clean
