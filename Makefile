SHELL := /bin/bash
APP_NAME := readmebuilder

.PHONY: vet lint test

vet:
	go vet ./...

lint:
	if ! command -v golangci-lint &> /dev/null; then \
		GO111MODULE=on go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	golangci-lint run ./...

test:
	go test ./...

build:
	go build -o $(APP_NAME) -ldflags="-s -w" .
