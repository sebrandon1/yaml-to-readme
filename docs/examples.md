# Examples

## Basic Usage

```bash
# Summarize all YAML files in a directory (excludes hidden directories)
./readmebuilder ./my-yaml-repo
```

## Include Hidden Directories

```bash
./readmebuilder --include-hidden-directories ./my-yaml-repo
```

## Regenerate All Summaries

```bash
./readmebuilder --regenerate --include-hidden-directories ./my-yaml-repo
```

## Use a Different Ollama Model

```bash
./readmebuilder --model mistral:latest ./my-yaml-repo
```

## Custom Output Filename

```bash
./readmebuilder --output summary.md ./my-yaml-repo
```

## Custom Cache Directory

```bash
./readmebuilder --localcache --cache-dir .my_cache ./my-yaml-repo
```

## Dry Run (Preview Files)

```bash
./readmebuilder --dry-run ./my-yaml-repo
```

## Concurrent Processing

```bash
./readmebuilder --concurrency 4 ./my-yaml-repo
```

## JSON Output

```bash
./readmebuilder --format json --output summaries.json ./my-yaml-repo
```

## HTML Output

```bash
./readmebuilder --format html --output summaries.html ./my-yaml-repo
```

## OpenAI Provider

```bash
OPENAI_API_KEY=sk-... ./readmebuilder --provider openai --model gpt-4o-mini ./my-yaml-repo
```

## Custom OpenAI-Compatible Endpoint

Use with vLLM, llama.cpp server, or Azure OpenAI:

```bash
OPENAI_API_KEY=dummy OPENAI_BASE_URL=http://localhost:8000 \
  ./readmebuilder --provider openai --model my-local-model ./my-yaml-repo
```

## Provider Fallback Chain

Try OpenAI first; fall back to Ollama if it fails:

```bash
OPENAI_API_KEY=sk-... ./readmebuilder --provider openai,ollama ./my-yaml-repo
```

## Filter Files with Include / Exclude Globs

Process only files under `charts/` and skip anything in `testdata/`:

```bash
./readmebuilder --include 'charts/**' --exclude 'testdata/**' ./my-yaml-repo
```

## Custom LLM Request Timeout

Set a 2-minute timeout per LLM request (useful for slower local models):

```bash
./readmebuilder --timeout 2m ./my-yaml-repo
```

## Custom Prompt Template

Use a project-specific prompt file (supports `{filename}` and `{content}` placeholders):

```bash
./readmebuilder --prompt-template ./my-prompt.txt ./my-yaml-repo
```

## Config File

Instead of passing flags every time, create `.readmebuilder.yaml` in your repo root:

```yaml
provider: openai
model: gpt-4o-mini
format: markdown
concurrency: 4
```

Then just run:

```bash
./readmebuilder ./my-yaml-repo
```

## GitHub Actions Step Summary

Append a summary to the GitHub Actions job summary page (requires `$GITHUB_STEP_SUMMARY`, set automatically by Actions):

```bash
./readmebuilder --format github-summary ./my-yaml-repo
```
