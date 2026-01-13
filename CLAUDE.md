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

### Test
```bash
go test ./...
```

### Lint
```bash
make lint
```

## Architecture

- **`cmd/`** - CLI command implementations using Cobra
- **`main.go`** - Application entry point

## Dependencies

- [Ollama](https://ollama.com/) must be installed and running locally
- Default model: `llama3.2:latest` (can override with `--model` flag)
- Go 1.25+

## Code Style

- Follow standard Go conventions
- Use `go fmt` before committing
- Run `go vet` and `golangci-lint` for linting
