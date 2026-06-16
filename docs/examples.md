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
