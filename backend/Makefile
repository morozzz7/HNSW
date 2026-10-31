GO ?= go
PORT ?= 8080

.PHONY: run test test-race bench vet lint tidy deps

run:
	$(GO) run ./cmd/server

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

bench:
	$(GO) test -bench=. -benchmem ./...

vet:
	$(GO) vet ./...

lint:
	$(GO) fmt ./... && $(GO) vet ./...
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed; ran gofmt + go vet only"; \
	fi

tidy:
	$(GO) mod tidy

deps:
	$(GO) mod download
