# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A CLI tool to recursively summarize YAML files in a directory using a local LLM (Ollama or OpenAI-compatible), outputting results to a structured markdown, JSON, or HTML file.

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
- `--provider` - LLM provider: ollama (default) or openai
- `--model` - Specify LLM model (default: llama3.2:latest)
- `--regenerate` - Force regeneration of summaries
- `--localcache` - Use local cache for summaries
- `--include-hidden-directories` - Include hidden directories in scan
- `--format` - Output format: markdown (default), json, or html
- `--output` / `-o` - Output filename
- `--concurrency` / `-j` - Number of concurrent workers
- `--dry-run` - Preview files without calling the LLM
- `--verbose` / `-v` - Enable debug logging

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
  - `root.go` - Main CLI logic, YAML processing, output writers
  - `provider.go` - `LLMProvider` interface definition
  - `provider_ollama.go` - Ollama provider implementation
  - `provider_openai.go` - OpenAI-compatible provider implementation
  - `provider_mock.go` - Mock provider for testing
  - `ollama_client.go` - Low-level Ollama API client wrapper
  - `root_test.go` - Unit tests
  - `integration_test.go` - Integration tests

## Dependencies

- **Ollama** (default provider): [Ollama](https://ollama.com/) must be installed and running locally
- **OpenAI** (optional provider): Requires `OPENAI_API_KEY` env var; set `OPENAI_BASE_URL` for custom endpoints
- Default model: `llama3.2:latest` (can override with `--model` flag)
- Go 1.25+

## Code Style

- Follow standard Go conventions
- Use `go fmt` before committing
- Run `go vet` and `golangci-lint` for linting
