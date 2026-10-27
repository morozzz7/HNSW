# Makefile for the HNSW coursework backend.
# Module: hnsw-coursework (Go). Server listens on :8080 by default.

GO ?= go
PORT ?= 8080

.PHONY: run test test-race bench vet lint tidy deps

run: tidy
	cd backend && $(GO) run ./cmd/server

test:
	cd backend && $(GO) test ./...

test-race:
	cd backend && $(GO) test -race ./...

bench:
	cd backend && $(GO) test -bench=. -benchmem ./...

vet:
	cd backend && $(GO) vet ./...

lint: 
	cd backend && $(GO) fmt ./... && $(GO) vet ./...
	@if command -v golangci-lint >/dev/null 2>&1; then \
		cd backend && golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed; ran gofmt + go vet only"; \
	fi

tidy:
	cd backend && $(GO) mod tidy

deps:
	cd backend && $(GO) mod download
