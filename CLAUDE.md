# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A CLI tool to recursively summarize YAML files in a directory using a local Ollama model, outputting results to a structured markdown file.

## Common Commands

### Build
```bash
make build
```

This produces a binary named `readmebuilder` in the project directory.

### Run
```bash
./readmebuilder [directory] [flags]
```

Available flags:
- `--model` - Specify Ollama model (default: llama3.2:latest)
- `--regenerate` - Force regeneration of summaries
- `--localcache` - Use local cache for summaries
- `--include-hidden-directories` - Include hidden directories in scan

### Test
```bash
make test              # Run short tests
make integration-test  # Run integration tests
make test-coverage     # Run tests with coverage
make coverage-html     # Generate HTML coverage report
```

### Lint and Vet
```bash
make lint
make vet
```

### Other Commands
```bash
make version           # Show version info
make info              # Show build info
```

## Architecture

- **`main.go`** - Application entry point
- **`cmd/`** - CLI command implementations using Cobra
  - `root.go` - Main CLI logic and YAML processing
  - `ollama_client.go` - Ollama API client interface
  - `mock_ollama_client.go` - Mock client for testing
  - `root_test.go` - Unit tests
  - `integration_test.go` - Integration tests

## Dependencies

- [Ollama](https://ollama.com/) must be installed and running locally
- Default model: `llama3.2:latest` (can override with `--model` flag)
- Go 1.25+

## Code Style

- Follow standard Go conventions
- Use `go fmt` before committing
- Run `go vet` and `golangci-lint` for linting
