# Configuration

Gnostis is configured entirely through environment variables. No YAML config file is needed.

Data is stored in `~/.gnostis/data` and logs in `~/.gnostis/gnostis.log` by default.

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `GNOSTIS_LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `GNOSTIS_DATA_DIR` | `~/.gnostis/data` | Data directory (vector store, symbol index, etc.) |
| `GNOSTIS_PROJECTS_DIR` | `~/.gnostis/projects` | Directory containing per-project JSON files |
| `GNOSTIS_EMBEDDINGS_PROVIDER` | `ollama` | Embedding provider: `ollama` or `openai` |
| `GNOSTIS_EMBEDDINGS_URL` | `http://localhost:11434/v1` | Embedding API endpoint |
| `GNOSTIS_EMBEDDINGS_MODEL` | `nomic-embed-text` | Embedding model name |
| `GNOSTIS_EMBEDDINGS_API_KEY` | _(empty)_ | API key for `openai` provider |
| `GNOSTIS_EMBEDDINGS_BATCH_SIZE` | `32` | Max texts per embedding request |
| `GNOSTIS_WEB_ENABLED` | `true` | Enable the web dashboard |
| `GNOSTIS_WEB_PORT` | `7878` | Web dashboard port |
| `GNOSTIS_MEMORY_CASCADE_ENABLED` | `false` | Enable Cascade memory indexing |
| `GNOSTIS_MEMORY_CASCADE_SOURCE_DIRS` | _(auto-detected)_ | Comma-separated list of Cascade trajectory dirs |
| `GNOSTIS_MEMORY_CASCADE_MIN_MSG_LEN` | `10` | Minimum user message length to index |
| `GNOSTIS_MEMORY_CURSOR_ENABLED` | `false` | Enable Cursor memory indexing |
| `GNOSTIS_MEMORY_CURSOR_SOURCE_DIRS` | _(empty)_ | Comma-separated list of Cursor dirs |
| `GNOSTIS_MEMORY_CURSOR_MIN_MSG_LEN` | `10` | Minimum user message length to index |

## Example

```bash
export GNOSTIS_LOG_LEVEL=debug
export GNOSTIS_EMBEDDINGS_PROVIDER=ollama
export GNOSTIS_EMBEDDINGS_URL=http://localhost:11434/v1
export GNOSTIS_EMBEDDINGS_MODEL=nomic-embed-text
export GNOSTIS_WEB_PORT=7878
export GNOSTIS_MEMORY_CASCADE_ENABLED=true
export GNOSTIS_MEMORY_CASCADE_SOURCE_DIRS=$HOME/.codeium/windsurf/cascade
```

## Per-project files

Individual projects are stored as JSON files in the projects directory (`~/.gnostis/projects/` by default). Each file is named `<project-name>.json`:

```json
{
  "path": "/home/user/projects/myapp",
  "name": "myapp",
  "extensions": [],
  "include": [],
  "exclude": [],
  "max_file_size_mb": 5
}
```

- `path` (required): absolute directory path.
- `name`: project name; inferred from directory name if omitted.
- `extensions`: file extensions to index (e.g. `.go`, `.py`). Defaults to a standard set if empty.
- `include`: glob patterns; if set, only matching files are indexed.
- `exclude`: glob patterns to exclude from indexing (merged with built-in defaults like `.git/**`, `node_modules/**`, `vendor/**`).
- `max_file_size_mb`: skip files larger than this. Default: `5`.

The `add_project` MCP tool creates a JSON file **without** starting indexing. Call `rebuild_project` separately to index.

The `edit_project` MCP tool updates a project's indexing parameters (extensions, include/exclude patterns, max file size).

The `remove_project` MCP tool deletes the JSON file and removes indexed chunks.

Changes to the projects directory are detected at runtime and trigger an automatic reload.

## MCP server

The MCP server name is hardcoded to `gnostis`. The version is set at build time from the git commit hash.

Gnostis supports the `streamable-http` MCP transport at the `/mcp` endpoint, plus a stdio proxy mode.

## Memory

Opt-in indexing of chat/dialogue memory. Memory is stored separately from project code in `~/.gnostis/data/memory`.

Each provider has the same options:

- `enabled`: when `true`, Gnostis indexes provider files into the isolated memory store.
- `source_dirs`: comma-separated list of directories containing provider-specific files.
- `min_user_message_length`: shortest user message to keep. Default: `10`.

Supported providers:

- `cascade`: Windsurf/Cascade/Devin Desktop conversation trajectories.
- `cursor`: placeholder for future Cursor support.

## Filter precedence

1. `.gitignore`
2. `include`
3. `exclude`
4. `extensions`
5. `max_file_size_mb`

## Default extensions and excludes

When a project does not specify `extensions` or `exclude`, these built-in defaults apply:

**Extensions:** `.go`, `.py`, `.js`, `.ts`, `.jsx`, `.tsx`, `.rs`, `.md`

**Excludes:** `node_modules/**`, `.git/**`, `vendor/**`, `dist/**`, `build/**`, `__pycache__/**`
