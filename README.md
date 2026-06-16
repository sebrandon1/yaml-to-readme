# yaml-to-readme

![Generate Markdown](https://github.com/sebrandon1/yaml-to-readme/actions/workflows/generate-markdown.yml/badge.svg)
![Test Incoming Changes](https://github.com/sebrandon1/yaml-to-readme/actions/workflows/pre-main.yml/badge.svg)
![Release binaries](https://github.com/sebrandon1/yaml-to-readme/actions/workflows/release-binaries.yaml/badge.svg)

A CLI tool to recursively summarize YAML files in a directory using a local LLM, outputting results to a structured markdown, JSON, or HTML file. Supports Ollama (default) and OpenAI-compatible APIs.

## Key Features

- Recursive YAML file discovery with progress bar
- Multiple LLM providers: Ollama (local) and OpenAI-compatible APIs
- Output formats: Markdown, JSON, and HTML
- Concurrent processing with configurable workers
- Smart caching to skip already-summarized files
- Dry-run mode for previewing file discovery

## Quick Start

### Prerequisites

- **Ollama** (default): [Install Ollama](https://ollama.com/) and pull a model (default: `llama3.2:latest`)
- **OpenAI** (optional): Set `OPENAI_API_KEY` environment variable
- Go 1.25+ to build from source

### Build and Run

```bash
make build
./readmebuilder ./my-yaml-repo
```

### Common Flags

```bash
./readmebuilder --model mistral:latest ./my-yaml-repo        # Different model
./readmebuilder --concurrency 4 ./my-yaml-repo               # Parallel processing
./readmebuilder --format json --output out.json ./my-yaml-repo # JSON output
./readmebuilder --dry-run ./my-yaml-repo                      # Preview only
```

## Guides

| Guide | Description |
|-------|-------------|
| [CLI Reference](docs/cli-reference.md) | All flags, output formats, and environment variables |
| [Examples](docs/examples.md) | Detailed usage examples for every feature |
| [Docker](docs/docker.md) | Container build and run instructions |

## Development

```bash
make build             # Build the binary
make test              # Run unit tests
make integration-test  # Run integration tests
make lint              # Run golangci-lint
make vet               # Run go vet
```

---

For questions or issues, please open an issue on GitHub.
