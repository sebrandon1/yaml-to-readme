# Docker

## Build the Image

```bash
docker build -t readmebuilder .
```

## Run Against a Local Directory

The container requires access to an Ollama instance running on the host.

### Linux (host networking)

```bash
docker run --rm --network host \
  -v /path/to/yaml-repo:/data \
  readmebuilder /data
```

### macOS / Windows

Use `host.docker.internal` to reach the host Ollama instance:

```bash
docker run --rm \
  -e OLLAMA_HOST=http://host.docker.internal:11434 \
  -v /path/to/yaml-repo:/data \
  readmebuilder /data
```
