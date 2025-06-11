# yaml-to-readme

A CLI tool to recursively summarize YAML files in a directory using a local Ollama model, outputting results to a structured markdown file.

## How to Use

### Prerequisites
- [Ollama](https://ollama.com/) must be installed and running locally.
- The desired model (default: `llama3.2:latest`) must be available in your local Ollama instance.
- Go 1.24+ is required to build from source.

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

#### Example
```
./readmebuilder ./my-yaml-repo
```

This will:
- Recursively find all `.yaml` and `.yml` files under `./my-yaml-repo`.
- Summarize each file using the local Ollama model.
- Write a grouped summary to `yaml_details.md` in the root of the provided directory.
- Show a progress bar and timing information.
- Skip files that already have a summary in `yaml_details.md` (unless `--regenerate` is used).

### Flags

- `--regenerate`  
  Regenerate all summaries, even if they already exist in `yaml_details.md`.
- `--localcache`  
  Write individual summaries to `.yaml_summary_cache` in the repo root for each YAML file processed.

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