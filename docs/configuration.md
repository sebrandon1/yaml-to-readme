# Configuration

`readmebuilder` supports a config file that lets you set persistent defaults without repeating flags on every run.

## Config File Location

By default the tool looks for `.readmebuilder.yaml` (or `.readmebuilder.yml`) in the current working directory. You can override the path with `--config`:

```bash
./readmebuilder --config /path/to/my-config.yaml ./my-yaml-repo
```

## Supported Keys

All keys are optional. Explicit CLI flags always take precedence over config-file values.

```yaml
provider: ollama          # LLM provider: ollama, openai, or fallback chain (e.g. openai,ollama)
model: llama3.2:latest    # LLM model name
format: markdown          # Output format: markdown, json, html, or github-summary
output: yaml_details.md   # Output filename
cache-dir: .yaml_summary_cache  # Cache directory for --localcache
concurrency: 1            # Number of concurrent workers
regenerate: false         # Re-summarize files that already have a summary
localcache: false         # Write per-file summaries to cache-dir
verbose: false            # Enable debug logging to stderr
dry-run: false            # Preview files without calling the LLM
```

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `OPENAI_API_KEY` | For OpenAI provider | API key for OpenAI or a compatible endpoint. |
| `OPENAI_BASE_URL` | No | Custom base URL for OpenAI-compatible APIs (vLLM, llama.cpp, Azure OpenAI). |
| `OLLAMA_HOST` | No | Custom Ollama endpoint (useful for Docker setups). |
| `GITHUB_STEP_SUMMARY` | For `github-summary` format | Path to the GitHub Actions step summary file. Set automatically by GitHub Actions. |

## Priority Order

Highest to lowest:

1. Explicit CLI flag
2. Config file (`.readmebuilder.yaml`)
3. Built-in default
