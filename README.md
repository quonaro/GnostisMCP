# Gnostis

A local "second brain" for developers. Gnostis indexes your projects with tree-sitter-aware chunking, stores embeddings locally, and exposes semantic search tools to AI agents through the Model Context Protocol (MCP).

## What it does

- Watches configured directories and incrementally indexes changed files.
- Splits code into symbol-level chunks (functions, types, methods, classes).
- Stores embeddings locally using `chromem-go`.
- Supports Ollama and OpenAI-compatible APIs for embeddings.
- Maintains a dedicated symbol index for fast exact symbol lookups.
- Exposes MCP tools: `search_codebase`, `find_symbol`, `get_file_context`, `list_projects`, `grep`, `list_files`, `directory_tree`, `get_recent_changes`, `query_documentation`, `add_project`, `edit_project`, `remove_project`, `rebuild_project`, `rebuild_index`, `reindex_files`, `get_index_status`, `get_index_job`.
- Includes a web dashboard for monitoring indexing progress and managing projects.

## Quick links

- [Architecture](docs/architecture.md)
- [Configuration](docs/config.md)
- [Embedding providers](docs/embedding-providers.md)
- [Indexing](docs/indexing.md)
- [MCP tools](docs/mcp-tools.md)

---

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick start](#quick-start)
  - [Option 1: `go run` (no install needed)](#option-1-go-run-no-install-needed)
  - [Option 2: Pre-built binary](#option-2-pre-built-binary)
  - [Option 3: Build from source](#option-3-build-from-source)
- [Configuration](#configuration)
  - [Environment variables](#environment-variables)
  - [Example `.env` file](#example-env-file)
- [Adding projects](#adding-projects)
- [Connecting to your editor](#connecting-to-your-editor)
  - [Cursor](#cursor)
  - [Windsurf](#windsurf)
  - [Claude Desktop](#claude-desktop)
- [Web dashboard](#web-dashboard)
- [Running as a systemd service](#running-as-a-systemd-service)
- [How it works](#how-it-works)
- [Logs](#logs)
- [Development](#development)

---

## Prerequisites

### 1. Go (for building from source)

- Go 1.23+ with CGO enabled (`CGO_ENABLED=1`).
- A C compiler (`gcc` on Linux, Xcode CLT on macOS).

### 2. Ollama (recommended for local embeddings)

```bash
# Install Ollama: https://ollama.com
ollama pull nomic-embed-text
ollama serve
```

Or use any OpenAI-compatible embedding API (OpenAI, OpenRouter, LocalAI, LM Studio).

### 3. Node.js (only for building the web dashboard from source)

- Node.js 18+ and npm.
- Only needed if building the frontend. Pre-built binaries include the dashboard.

---

## Quick start

Gnostis runs as a **stdio MCP server** — the MCP client (Cursor, Windsurf, Claude Desktop) manages the process lifecycle.

### Option 1: `go run` (no install needed)

Add to your MCP client config:

```json
{
  "mcpServers": {
    "gnostis": {
      "command": "go",
      "args": ["run", "github.com/quonaro/gnostis/cmd/gnostis@latest"]
    }
  }
}
```

### Option 2: Pre-built binary

```bash
go install github.com/quonaro/gnostis/cmd/gnostis@v0.0.18
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

### Option 3: Build from source

```bash
git clone https://github.com/quonaro/GnostisMCP.git
cd GnostisMCP

# Build frontend + Go binary
cd web && npm install && npm run build && cd ..
cp -r web/dist internal/web/dist

# Build with version stamp
COMMIT=$(git rev-parse --short HEAD)
go build \
  -ldflags="-s -w -X main.version=$COMMIT -X github.com/quonaro/gnostis/internal/mcp.version=$COMMIT" \
  -o gnostis ./cmd/gnostis

# Move to PATH
sudo mv gnostis /usr/local/bin/
# or: mv gnostis ~/.local/bin/
```

Then add to your MCP client config (use the full path if not in `PATH`):

```json
{
  "mcpServers": {
    "gnostis": {
      "command": "gnostis"
    }
  }
}
```

---

## Configuration

Gnostis is configured entirely through **environment variables**. No YAML config file is needed.

On first run, Gnostis creates:
- `~/.gnostis/data/` — vector store, symbol index, embeddings cache.
- `~/.gnostis/projects/` — per-project JSON files.
- `~/.gnostis/gnostis.log` — log file.

### Environment variables

| Variable | Default | Description |
|---|---|---|
| `GNOSTIS_LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `GNOSTIS_DATA_DIR` | `~/.gnostis/data` | Data directory (vector store, symbol index) |
| `GNOSTIS_PROJECTS_DIR` | `~/.gnostis/projects` | Directory containing per-project JSON files |
| `GNOSTIS_EMBEDDINGS_PROVIDER` | `ollama` | Embedding provider: `ollama` or `openai` |
| `GNOSTIS_EMBEDDINGS_URL` | `http://localhost:11434/v1` | Embedding API endpoint |
| `GNOSTIS_EMBEDDINGS_MODEL` | `nomic-embed-text` | Embedding model name |
| `GNOSTIS_EMBEDDINGS_API_KEY` | _(empty)_ | API key for `openai` provider |
| `GNOSTIS_EMBEDDINGS_BATCH_SIZE` | `32` | Max texts per embedding request |
| `GNOSTIS_WEB_ENABLED` | `true` | Enable the web dashboard |
| `GNOSTIS_WEB_PORT` | `7878` | Web dashboard port |
| `GNOSTIS_MEMORY_CASCADE_ENABLED` | `false` | Enable Cascade memory indexing |
| `GNOSTIS_MEMORY_CASCADE_SOURCE_DIRS` | _(auto-detected)_ | Comma-separated Cascade trajectory dirs |
| `GNOSTIS_MEMORY_CASCADE_MIN_MSG_LEN` | `10` | Minimum user message length to index |
| `GNOSTIS_MEMORY_CURSOR_ENABLED` | `false` | Enable Cursor memory indexing |
| `GNOSTIS_MEMORY_CURSOR_SOURCE_DIRS` | _(empty)_ | Comma-separated Cursor dirs |
| `GNOSTIS_MEMORY_CURSOR_MIN_MSG_LEN` | `10` | Minimum user message length to index |

See [docs/config.md](docs/config.md) for full details.

### Example `.env` file

Create a `.env` file and source it before starting Gnostis, or pass variables inline:

```bash
# .env
GNOSTIS_LOG_LEVEL=info
GNOSTIS_EMBEDDINGS_PROVIDER=ollama
GNOSTIS_EMBEDDINGS_URL=http://localhost:11434/v1
GNOSTIS_EMBEDDINGS_MODEL=nomic-embed-text
GNOSTIS_EMBEDDINGS_BATCH_SIZE=32
GNOSTIS_WEB_ENABLED=true
GNOSTIS_WEB_PORT=7878

# Optional: enable Cascade memory indexing
# GNOSTIS_MEMORY_CASCADE_ENABLED=true
# GNOSTIS_MEMORY_CASCADE_SOURCE_DIRS=/home/user/.codeium/windsurf/cascade
```

```bash
source .env
gnostis
```

Or pass variables inline in your MCP client config (see [Connecting to your editor](#connecting-to-your-editor)).

---

## Adding projects

Projects are managed as individual JSON files in `~/.gnostis/projects/`. Each file is named `<project-name>.json`:

```json
{
  "path": "/home/user/projects/myapp",
  "name": "myapp",
  "extensions": [".go", ".md"],
  "exclude": ["vendor/**", "**/*_test.go"],
  "max_file_size_mb": 5
}
```

You can add projects in three ways:

**1. Via MCP tool** (from your AI editor):

```
add_project(path="/home/user/projects/myapp", name="myapp")
rebuild_project(project="myapp")
```

**2. Via web dashboard** — open `http://localhost:7878` and click "Add Project".

**3. Manually** — create a JSON file in `~/.gnostis/projects/myapp.json` and Gnostis will detect it automatically.

To edit a project's settings:

```
edit_project(name="myapp", exclude=["vendor/**", "**/*_test.go", "docs/legacy/**"])
rebuild_project(project="myapp")
```

To remove:

```
remove_project(name="myapp")
```

---

## Connecting to your editor

### Cursor

Add to `~/.cursor/mcp.json` (or `.cursor/mcp.json` in your project root):

```json
{
  "mcpServers": {
    "gnostis": {
      "command": "gnostis",
      "env": {
        "GNOSTIS_EMBEDDINGS_PROVIDER": "ollama",
        "GNOSTIS_EMBEDDINGS_URL": "http://localhost:11434/v1",
        "GNOSTIS_EMBEDDINGS_MODEL": "nomic-embed-text"
      }
    }
  }
}
```

### Windsurf

Add to your Windsurf MCP config (`~/.codeium/windsurf/mcp_config.json`):

```json
{
  "mcpServers": {
    "gnostis": {
      "command": "gnostis",
      "env": {
        "GNOSTIS_EMBEDDINGS_PROVIDER": "ollama",
        "GNOSTIS_EMBEDDINGS_URL": "http://localhost:11434/v1",
        "GNOSTIS_EMBEDDINGS_MODEL": "nomic-embed-text"
      }
    }
  }
}
```

### Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `~/.config/claude/claude_desktop_config.json` (Linux):

```json
{
  "mcpServers": {
    "gnostis": {
      "command": "/usr/local/bin/gnostis",
      "env": {
        "GNOSTIS_EMBEDDINGS_PROVIDER": "ollama",
        "GNOSTIS_EMBEDDINGS_URL": "http://localhost:11434/v1",
        "GNOSTIS_EMBEDDINGS_MODEL": "nomic-embed-text"
      }
    }
  }
}
```

> **Tip:** If using `go run` instead of a pre-built binary, replace `"command": "gnostis"` with `"command": "go"` and add `"args": ["run", "github.com/quonaro/gnostis/cmd/gnostis@latest"]`.

> **Multiple editor windows:** When a second Gnostis process starts, it detects the primary instance and automatically runs as a stdio proxy, forwarding all MCP messages over HTTP. No extra configuration needed.

---

## Web dashboard

Gnostis includes a built-in web dashboard at `http://localhost:7878` (default port). It shows:

- Indexing progress (files, chunks, ETA).
- Project list with chunk counts and last-indexed timestamps.
- Add, edit, and remove projects.
- Trigger rebuilds.

Disable it with `GNOSTIS_WEB_ENABLED=false` or change the port with `GNOSTIS_WEB_PORT=9000`.

---

## Running as a systemd service

For always-on deployment (Linux, user-level service):

```bash
# Build and install
go build -o gnostis ./cmd/gnostis
mkdir -p ~/.local/bin
mv gnostis ~/.local/bin/

# Install systemd service
mkdir -p ~/.config/systemd/user
cp systemd/gnostis.service ~/.config/systemd/user/

# Set environment variables for the service
mkdir -p ~/.gnostis
cat > ~/.gnostis/env << 'EOF'
GNOSTIS_EMBEDDINGS_PROVIDER=ollama
GNOSTIS_EMBEDDINGS_URL=http://localhost:11434/v1
GNOSTIS_EMBEDDINGS_MODEL=nomic-embed-text
EOF
```

Edit `~/.config/systemd/user/gnostis.service` to add `EnvironmentFile=%h/.gnostis/env`:

```ini
[Service]
Type=simple
EnvironmentFile=%h/.gnostis/env
WorkingDirectory=%h/.gnostis
ExecStart=%h/.local/bin/gnostis run
Restart=unless-stopped
StandardOutput=journal
StandardError=journal
```

```bash
systemctl --user daemon-reload
systemctl --user enable gnostis
systemctl --user start gnostis

# Check status
systemctl --user status gnostis

# View logs
journalctl --user -u gnostis -f
```

---

## How it works

1. The MCP client spawns `gnostis` as a stdio subprocess.
2. Gnostis loads config from environment variables.
3. It opens the persistent embedding store from `~/.gnostis/data/`.
4. Background goroutines index configured projects and watch for file changes.
5. MCP tools serve semantic search, symbol lookup, and file context queries over stdin/stdout.
6. The web dashboard serves at `http://localhost:7878` with SSE for real-time progress.
7. When the client closes stdin, Gnostis shuts down gracefully.

### Multiple editor windows

When a second `gnostis` process starts (e.g. opening a second Cursor window), it detects the primary instance via a file lock on the data directory. The second process becomes a **stdio proxy** — it forwards all MCP JSON-RPC messages to the primary instance over HTTP. This means all editor windows share a single index and embedding store.

---

## Logs

Logs are written to:
- `~/.gnostis/gnostis.log` (file)
- stderr (console)

Set `GNOSTIS_LOG_LEVEL=debug` for detailed embedding request logs and model activity.

---

## Development

### Prerequisites

- Go 1.23+
- Node.js 18+
- [Air](https://github.com/air-verse/air) for live reload (`go install github.com/air-verse/air@latest`)

### Development workflow

```bash
# Install frontend dependencies
cd web && npm install && cd ..

# Run backend + frontend with live reload
# Requires Air and the lota task runner
lota dev

# Or run separately:
# Terminal 1: backend with live reload
air

# Terminal 2: frontend dev server
cd web && npm run dev
```

### Build

```bash
# Build frontend
cd web && npm run build && cd ..
cp -r web/dist internal/web/dist

# Build binary
COMMIT=$(git rev-parse --short HEAD)
go build \
  -ldflags="-s -w -X main.version=$COMMIT -X github.com/quonaro/gnostis/internal/mcp.version=$COMMIT" \
  -o gnostis ./cmd/gnostis
```

### Test & lint

```bash
go test ./...
golangci-lint run
gofmt -l .
```

### Project structure

```
cmd/gnostis/          Entry points (main.go, native.go, proxy.go)
internal/
  app/                Application orchestration, indexing, projects
  config/             Environment-based config loading
  chunker/            Tree-sitter symbol extraction
  directory/          Per-directory indexing rules
  embeddings/         Provider interface (Ollama, OpenAI)
  indexer/            File walking and filtering
  mcp/                MCP server and tool handlers
  search/             Search orchestration and reranking
  store/              chromem-go persistence
  web/                HTTP dashboard + embedded Svelte SPA
  watcher/            fsnotify debounced watcher
web/                  Svelte frontend source
docs/                 Documentation
systemd/              systemd service file
```
