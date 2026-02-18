# yaml-to-readme

![Generate Markdown](https://github.com/sebrandon1/yaml-to-readme/actions/workflows/generate-markdown.yml/badge.svg)
![Test Incoming Changes](https://github.com/sebrandon1/yaml-to-readme/actions/workflows/pre-main.yml/badge.svg)
![Release binaries](https://github.com/sebrandon1/yaml-to-readme/actions/workflows/release-binaries.yaml/badge.svg)

A CLI tool to recursively summarize YAML files in a directory using a local LLM, outputting results to a structured markdown file. Supports Ollama (default) and OpenAI-compatible APIs.

## How to Use

### Prerequisites
- **Ollama** (default): [Ollama](https://ollama.com/) must be installed and running locally. The desired model (default: `llama3.2:latest`) must be available.
- **OpenAI** (optional): Set `OPENAI_API_KEY` environment variable. Optionally set `OPENAI_BASE_URL` for compatible endpoints (vLLM, llama.cpp server, Azure OpenAI).
- Go 1.25+ is required to build from source.

### Build

```
make build
```
This will produce a binary named `readmebuilder` in the project directory.

### Usage

```
./readmebuilder [directory] [flags]
```
- `[directory]`: The root directory to recursively search for YAML files. Required.

#### Examples
```bash
# Basic usage - excludes hidden directories
./readmebuilder ./my-yaml-repo

# Include hidden directories in the search
./readmebuilder --include-hidden-directories ./my-yaml-repo

# Regenerate all summaries and include hidden directories
./readmebuilder --regenerate --include-hidden-directories ./my-yaml-repo

# Specify a different Ollama model without recompiling
./readmebuilder --model mistral:latest ./my-yaml-repo

# Write output to a custom filename
./readmebuilder --output summary.md ./my-yaml-repo

# Use a custom cache directory with localcache
./readmebuilder --localcache --cache-dir .my_cache ./my-yaml-repo

# Preview which files would be processed without calling the LLM
./readmebuilder --dry-run ./my-yaml-repo

# Process files with 4 concurrent workers
./readmebuilder --concurrency 4 ./my-yaml-repo

# Output summaries as JSON
./readmebuilder --format json --output summaries.json ./my-yaml-repo

# Output summaries as HTML
./readmebuilder --format html --output summaries.html ./my-yaml-repo

# Use OpenAI API instead of Ollama
OPENAI_API_KEY=sk-... ./readmebuilder --provider openai --model gpt-4o-mini ./my-yaml-repo

# Use a custom OpenAI-compatible endpoint (e.g., vLLM, llama.cpp)
OPENAI_API_KEY=dummy OPENAI_BASE_URL=http://localhost:8000 \
  ./readmebuilder --provider openai --model my-local-model ./my-yaml-repo
```

This will:
- Recursively find all `.yaml` and `.yml` files under `./my-yaml-repo` (excluding hidden directories by default).
- Summarize each file using the local Ollama model.
- Write a grouped summary to `yaml_details.md` in the root of the provided directory.
- Show a progress bar and timing information.
- Skip files that already have a summary in `yaml_details.md` (unless `--regenerate` is used).

### Flags

- `--regenerate`  
  Regenerate all summaries, even if they already exist in `yaml_details.md`.
- `--localcache`  
  Write individual summaries to `.yaml_summary_cache` in the repo root for each YAML file processed.
- `--include-hidden-directories`  
  Include hidden directories (starting with `.`) when searching for YAML files. By default, hidden directories like `.git`, `.vscode`, etc. are skipped for performance.
- `--provider`
  LLM provider: `ollama` (default) or `openai`. The `openai` provider works with any OpenAI-compatible API. Requires `OPENAI_API_KEY` env var; optionally set `OPENAI_BASE_URL` for custom endpoints.
- `--model`
  LLM model to use (default: `llama3.2:latest`). Example: `--model mistral:latest` (Ollama) or `--model gpt-4o-mini` (OpenAI).
- `--output`, `-o`
  Output markdown filename (default: `yaml_details.md`). Example: `--output summary.md`.
- `--cache-dir`
  Cache directory name for `--localcache` (default: `.yaml_summary_cache`). Example: `--cache-dir .my_cache`.
- `--dry-run`
  Preview which YAML files would be processed without calling the LLM. Shows file counts and lists files that would be summarized.
- `--concurrency`, `-j`
  Number of concurrent workers for processing YAML files (default: `1`). Example: `--concurrency 4`.
- `--format`
  Output format: `markdown` (default), `json`, or `html`. JSON outputs a structured array of file entries. HTML generates a styled, self-contained page. Example: `--format json`.
- `--verbose`, `-v`
  Enable verbose debug logging to stderr. Useful for troubleshooting file discovery and processing decisions.

### Output
- **Markdown** (default): Creates or updates a `yaml_details.md` file in the target directory, grouping summaries by subdirectory. Each entry includes a link to the YAML file and a concise, high-level summary.
- **JSON**: Outputs a structured JSON array with directory, file path, and summary fields.
- **HTML**: Generates a self-contained HTML page with styled summary cards grouped by directory.

### Notes
- If the required model is not available in the configured provider, the tool will exit with an error.
- Summaries are strictly limited to two sentences, with no lists, markdown, or code in the output.

### Docker

Build the container image:

```bash
docker build -t readmebuilder .
```

Run against a local directory (requires Ollama running on the host):

```bash
# On Linux (host networking)
docker run --rm --network host \
  -v /path/to/yaml-repo:/data \
  readmebuilder /data

# On macOS/Windows (use host.docker.internal)
docker run --rm \
  -e OLLAMA_HOST=http://host.docker.internal:11434 \
  -v /path/to/yaml-repo:/data \
  readmebuilder /data
```

### Makefile Targets
- `make vet`   — Run `go vet` on the codebase.
- `make lint`  — Run `golangci-lint` (installs if missing).
- `make test`  — Run unit tests.
- `make integration-test` — Run integration tests (uses mocked Ollama client).
- `make build` — Build the binary.

---

For questions or issues, please open an issue on GitHub.