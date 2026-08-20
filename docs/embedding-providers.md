# Embedding Providers

Gnostis uses a provider interface so you can switch between local and remote embedding models without changing the core code.

## Interface

```go
type Provider interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    BatchSize() int
    ModelName() string
}
```

## Ollama

Recommended for local-first usage. Any model exposed through Ollama's OpenAI-compatible endpoint works.

```bash
export GNOSTIS_EMBEDDINGS_PROVIDER=ollama
export GNOSTIS_EMBEDDINGS_URL=http://localhost:11434/v1
export GNOSTIS_EMBEDDINGS_MODEL=nomic-embed-text
```

Install and run:

```bash
ollama pull nomic-embed-text
ollama serve
```

## OpenAI-compatible

Works with OpenAI, OpenRouter, LocalAI, LM Studio, and any other `/v1/embeddings` endpoint.

```bash
export GNOSTIS_EMBEDDINGS_PROVIDER=openai
export GNOSTIS_EMBEDDINGS_URL=https://api.openai.com/v1
export GNOSTIS_EMBEDDINGS_MODEL=text-embedding-3-small
export GNOSTIS_EMBEDDINGS_API_KEY=$OPENAI_API_KEY
```

## Model recommendations

| Model                           | Provider | Russian   | Code | Speed  |
| ------------------------------- | -------- | --------- | ---- | ------ |
| `nomic-embed-text`              | Ollama   | good      | good | fast   |
| `intfloat/multilingual-e5-base` | Ollama   | excellent | good | medium |
| `text-embedding-3-small`        | OpenAI   | excellent | good | fast   |
