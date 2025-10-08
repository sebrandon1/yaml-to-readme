SHELL := /bin/bash
APP_NAME := readmebuilder
MODULE_NAME := github.com/sebrandon1/yaml-to-readme
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GO_VERSION := $(shell go version | awk '{print $$3}')

# Colors for output
CYAN := \033[36m
GREEN := \033[32m
YELLOW := \033[33m
RED := \033[31m
RESET := \033[0m

.PHONY: vet lint test test-coverage coverage-html version info

vet:
	@echo -e "$(CYAN)Running go vet...$(RESET)"
	go vet ./...

lint:
	@echo -e "$(CYAN)Running golangci-lint...$(RESET)"
	@if ! command -v golangci-lint &> /dev/null; then \
		echo -e "$(YELLOW)golangci-lint not found.$(RESET)"; \
		echo -e "$(YELLOW)Please install it (e.g., 'brew install golangci-lint' or see https://golangci-lint.run/) and re-run 'make lint'.$(RESET)"; \
		exit 0; \
	fi; \
	golangci-lint run ./...

test:
	@echo -e "$(CYAN)Running tests...$(RESET)"
	go test ./...

test-coverage:
	@echo -e "$(CYAN)Running tests with coverage...$(RESET)"
	@mkdir -p coverage
	go test -coverprofile=coverage/coverage.out ./...
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	go tool cover -func=coverage/coverage.out
	@echo -e "$(GREEN)Coverage report: coverage/coverage.html$(RESET)"

coverage-html:
	@echo -e "$(CYAN)Generating HTML coverage report...$(RESET)"
	@mkdir -p coverage
	go test -coverprofile=coverage/coverage.out ./...
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo -e "$(GREEN)HTML coverage report generated: coverage/coverage.html$(RESET)"
	@echo -e "$(CYAN)Opening coverage report in browser...$(RESET)"
	@open coverage/coverage.html

build:
	@echo -e "$(CYAN)Building $(APP_NAME)...$(RESET)"
	go build -o $(APP_NAME) -ldflags="-s -w" .
	@echo -e "$(GREEN)Built: $(APP_NAME)$(RESET)"

version:
	@echo -e "$(GREEN)App:        $(RESET)$(APP_NAME)"
	@echo -e "$(GREEN)Version:    $(RESET)$(VERSION)"
	@echo -e "$(GREEN)Build time: $(RESET)$(BUILD_TIME)"
	@echo -e "$(GREEN)Git commit: $(RESET)$(GIT_COMMIT)"
	@echo -e "$(GREEN)Go version: $(RESET)$(GO_VERSION)"

info:
	@echo -e "$(CYAN)Project Information$(RESET)"
	@echo -e "$(CYAN)===================$(RESET)"
	@echo -e "$(GREEN)Name:       $(RESET)$(APP_NAME)"
	@echo -e "$(GREEN)Module:     $(RESET)$(MODULE_NAME)"
	@echo -e "$(GREEN)Version:    $(RESET)$(VERSION)"
	@echo -e "$(GREEN)Go version: $(RESET)$(GO_VERSION)"
