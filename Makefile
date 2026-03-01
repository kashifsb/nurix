VERSION := 1.0.0
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS := -X 'github.com/kashifsb/nurix/internal/cli.Version=$(VERSION)' \
           -X 'github.com/kashifsb/nurix/internal/cli.BuildDate=$(BUILD_DATE)' \
           -X 'github.com/kashifsb/nurix/internal/cli.Commit=$(COMMIT)'

.PHONY: build install clean test test-verbose test-coverage lint release-local

build:
	go build -ldflags "$(LDFLAGS)" -o nurix ./cmd/main.go

install: build
	sudo mv nurix /usr/local/bin/nurix
	@echo ""
	@echo "✅ nurix v$(VERSION) installed to /usr/local/bin/nurix"
	@echo ""
	@echo "If this is a new version, run:"
	@echo "  nurix run db-migration"

clean:
	rm -f nurix nurix-linux-* nurix-darwin-* coverage.out coverage.html

test:
	go test -race -count=1 ./...

test-verbose:
	go test -v -race -count=1 ./...

test-coverage:
	go test -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out
	@echo ""
	@echo "To view HTML report:"
	@echo "  go tool cover -html=coverage.out -o coverage.html"

lint:
	go vet ./...
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "❌ Unformatted files:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi
	@echo "✅ All checks passed"

release-local:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o nurix-linux-amd64 ./cmd/main.go
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o nurix-linux-arm64 ./cmd/main.go
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o nurix-darwin-amd64 ./cmd/main.go
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o nurix-darwin-arm64 ./cmd/main.go
	@echo ""
	@echo "✅ Built binaries for all platforms"
