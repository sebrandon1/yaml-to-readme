# CLI Reference

## Usage

```
./readmebuilder [directory] [flags]
```

- `[directory]`: The root directory to recursively search for YAML files. Required.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--regenerate` | | `false` | Regenerate all summaries, even if they already exist in the output file. |
| `--localcache` | | `false` | Write individual summaries to a cache directory for each YAML file processed. |
| `--include-hidden-directories` | | `false` | Include hidden directories (starting with `.`) when searching for YAML files. By default, hidden directories like `.git`, `.vscode`, etc. are skipped. |
| `--provider` | | `ollama` | LLM provider: `ollama` (default) or `openai`. The `openai` provider works with any OpenAI-compatible API. Requires `OPENAI_API_KEY` env var; optionally set `OPENAI_BASE_URL` for custom endpoints. |
| `--model` | | `llama3.2:latest` | LLM model to use. Example: `--model mistral:latest` (Ollama) or `--model gpt-4o-mini` (OpenAI). |
| `--output` | `-o` | `yaml_details.md` | Output filename. |
| `--cache-dir` | | `.yaml_summary_cache` | Cache directory name for `--localcache`. |
| `--dry-run` | | `false` | Preview which YAML files would be processed without calling the LLM. Shows file counts and lists files that would be summarized. |
| `--concurrency` | `-j` | `1` | Number of concurrent workers for processing YAML files. |
| `--format` | | `markdown` | Output format: `markdown`, `json`, or `html`. |
| `--verbose` | `-v` | `false` | Enable verbose debug logging to stderr. Useful for troubleshooting file discovery and processing decisions. |

## Output Formats

- **Markdown** (default): Creates or updates a `yaml_details.md` file in the target directory, grouping summaries by subdirectory. Each entry includes a link to the YAML file and a concise, high-level summary.
- **JSON**: Outputs a structured JSON array with directory, file path, and summary fields.
- **HTML**: Generates a self-contained HTML page with styled summary cards grouped by directory.

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `OPENAI_API_KEY` | For OpenAI provider | API key for OpenAI or compatible endpoint. |
| `OPENAI_BASE_URL` | No | Custom base URL for OpenAI-compatible APIs (vLLM, llama.cpp, Azure OpenAI). |
| `OLLAMA_HOST` | No | Custom Ollama endpoint (useful for Docker setups). |

## Notes

- If the required model is not available in the configured provider, the tool will exit with an error.
- Summaries are strictly limited to two sentences, with no lists, markdown, or code in the output.
- Files that already have a summary in the output file are skipped unless `--regenerate` is used.
