.PHONY: all build test lint fmt vet tidy clean docker-build docker-run coverage

BINARY      := ironclaw-mcp
CMD_PATH    := ./cmd/ironclaw-mcp
DOCKER_IMG  := ironclaw-mcp
DOCKER_TAG  := latest
COVER_OUT   := coverage.out
COVER_HTML  := coverage.html
GO          := go
GOLANGCI    := golangci-lint

# Build
build:
	$(GO) build -ldflags "-s -w" -o bin/$(BINARY) $(CMD_PATH)

# Run all tests with race detector
test:
	$(GO) test -race -count=1 ./...

# Test with coverage output
coverage:
	$(GO) test -race -coverprofile=$(COVER_OUT) -covermode=atomic ./...
	$(GO) tool cover -html=$(COVER_OUT) -o $(COVER_HTML)
	$(GO) tool cover -func=$(COVER_OUT) | grep total

# Lint
lint:
	$(GOLANGCI) run ./...

# Format
fmt:
	$(GO) fmt ./...

gofumpt:
	gofumpt -l -w .

# Vet
vet:
	$(GO) vet ./...

# Tidy modules
tidy:
	$(GO) mod tidy

# Full pre-commit check
check: tidy fmt vet lint test

# Clean
clean:
	rm -rf bin/ $(COVER_OUT) $(COVER_HTML)

# Docker
docker-build:
	docker build -t $(DOCKER_IMG):$(DOCKER_TAG) .

docker-run:
	docker run --rm \
	  -e IRONCLAW_BASE_URL=$${IRONCLAW_BASE_URL:-http://host.docker.internal:3000} \
	  -e IRONCLAW_API_KEY=$${IRONCLAW_API_KEY:-} \
	  $(DOCKER_IMG):$(DOCKER_TAG)

# Alias
all: check build
