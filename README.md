# yaml-to-readme

![Generate Markdown](https://github.com/sebrandon1/yaml-to-readme/actions/workflows/generate-markdown.yml/badge.svg)
![Test Incoming Changes](https://github.com/sebrandon1/yaml-to-readme/actions/workflows/pre-main.yml/badge.svg)
![Release binaries](https://github.com/sebrandon1/yaml-to-readme/actions/workflows/release-binaries.yaml/badge.svg)

A CLI tool to recursively summarize YAML files in a directory using a local Ollama model, outputting results to a structured markdown file.

## How to Use

### Prerequisites
- [Ollama](https://ollama.com/) must be installed and running locally.
- The desired model (default: `llama3.2:latest`) must be available in your local Ollama instance. You can override the model at runtime with `--model`.
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
- `--model`  
  Ollama model to use (default: `llama3.2:latest`). Example: `--model mistral:latest`.

### Output
- The tool creates or updates a `yaml_details.md` file in the target directory, grouping summaries by subdirectory.
- Each entry includes a link to the YAML file and a concise, high-level summary (max two sentences).

### Notes
- If the required Ollama model is not available, the tool will exit with an error.
- Summaries are strictly limited to two sentences, with no lists, markdown, or code in the output.

### Makefile Targets
- `make vet`   — Run `go vet` on the codebase.
- `make lint`  — Run `golangci-lint` (installs if missing).
- `make build` — Build the binary.

---

For questions or issues, please open an issue on GitHub.