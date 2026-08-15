# Gnostis

A local "second brain" for developers. Gnostis indexes your projects with tree-sitter-aware chunking, stores embeddings locally, and exposes semantic search tools to AI agents through the Model Context Protocol (MCP).

## What it does

- Watches configured directories and incrementally indexes changed files.
- Splits code into symbol-level chunks (functions, types, methods, classes).
- Stores embeddings locally using `chromem-go`.
- Supports Ollama and OpenAI-compatible APIs for embeddings.
- Maintains a dedicated symbol index for fast exact symbol lookups.
- Exposes MCP tools: `search_codebase`, `find_symbol`, `get_file_context`, `list_projects`, `grep`, `list_files`, `directory_tree`, `get_recent_changes`, `query_documentation`, `add_project`, `remove_project`, `rebuild_index`, `get_index_status`.

## Quick links

- [Architecture](docs/architecture.md)
- [Configuration](docs/config.md)
- [Embedding providers](docs/embedding-providers.md)
- [Indexing](docs/indexing.md)
- [MCP tools](docs/mcp-tools.md)

## Quick start

Gnostis runs as a **stdio MCP server** — the MCP client (Cursor, Windsurf, Devin) manages the process lifecycle.

### Option 1: `go run` (no install needed)

Add to your MCP client config:

```json
{
  "mcpServers": {
    "gnostis": {
      "command": "go",
      "args": ["run", "github.com/quonaro/GnostisMCP/cmd/gnostis@latest"]
    }
  }
}
```

### Option 2: Pre-built binary

```bash
go install github.com/quonaro/GnostisMCP/cmd/gnostis@latest
```

Then add to your MCP client config:

```json
{
  "mcpServers": {
    "gnostis": {
      "command": "gnostis"
    }
  }
}
```

## Configuration

On first run, Gnostis creates a default config at `~/.gnostis/config.yaml`:

```yaml
embeddings:
  provider: ollama
  url: http://localhost:11434/v1
  model: nomic-embed-text
  batch_size: 32

directories: []
```

Add directories to index by editing the config or using the `add_project` MCP tool:

```yaml
directories:
  - path: /home/user/projects/myapp
    name: myapp
```

See [docs/config.md](docs/config.md) for all options.

## How it works

1. The MCP client spawns `gnostis` as a stdio subprocess.
2. Gnostis opens the persistent embedding store from `~/.gnostis/data/`.
3. Background goroutines index configured directories and watch for file changes.
4. MCP tools serve semantic search, symbol lookup, and file context queries over stdin/stdout.
5. When the client closes stdin, Gnostis shuts down gracefully.

## Logs

Logs are written to `~/.gnostis/gnostis.log` and stderr.
