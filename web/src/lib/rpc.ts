// JSON-RPC 2.0 client over WebSocket.
// Replaces the REST API client (api.ts) with a unified WS-based protocol.

// --- Types (kept identical to api.ts for drop-in replacement) ---

export interface ProgressState {
  job_id?: string
  status: string
  phase: string
  project: string
  total_files: number
  done_files: number
  total_chunks: number
  done_chunks: number
  pid: number
  started_at: string
  updated_at: string
  error?: string
}

export interface ProjectStat {
  path: string
  chunks: number
  last_indexed_at: string
  model?: string
  extensions?: string[]
  include?: string[]
  exclude?: string[]
  max_file_size_mb?: number
}

export interface MemoryProviderStat {
  name: string
  enabled: boolean
  chunks: number
  files: number
}

export interface MemoryProgressState {
  status: string
  total_files: number
  done_files: number
  total_chunks: number
  done_chunks: number
  started_at: string
  updated_at: string
  error?: string
}

export interface MemoryFile {
  path: string
  name: string
  provider: string
  title: string
  source?: string
  date?: string
  size: number
  type: string
}

export interface CPUMetrics {
  usage_percent: number
  cores: number
}

export interface MemoryMetrics {
  total_bytes: number
  used_bytes: number
  usage_percent: number
}

export interface GPUMetrics {
  index: number
  name: string
  utilization_percent: number
  memory_used_bytes: number
  memory_total_bytes: number
  temperature_c: number
}

export interface SystemMetrics {
  cpu: CPUMetrics
  memory: MemoryMetrics
  gpus?: GPUMetrics[]
}

export interface Job {
  id: string
  type: string
  description: string
  status: 'pending' | 'running' | 'done' | 'failed'
  created_at: string
  started_at?: string
  finished_at?: string
  error?: string
}

export interface StatusResponse {
  projects: string[]
  total_chunks: number
  provider: string
  model: string
  symbols: number
  progress: ProgressState
  eta?: string
  eta_seconds?: number
  project_stats: Record<string, ProjectStat>
  memory_stats?: MemoryProviderStat[]
  memory_progress?: MemoryProgressState
  jobs?: Job[]
  sys_metrics?: SystemMetrics
}

export interface SearchResult {
  id: string
  project_id: string
  path: string
  language: string
  symbol: string
  signature: string
  content: string
  start_line: number
  end_line: number
  score: number
}

export interface SearchResponse {
  query: string
  results: SearchResult[]
  count: number
}

export interface JobResponse {
  job_id: string
}

export interface GraphNode {
  id: string
  path: string
  symbol: string
  kind: string
  language: string
  start_line: number
  end_line: number
  x: number
  y: number
  degree: number
}

export interface GraphEdge {
  from: string
  to: string
  line: number
}

export interface GraphResponse {
  nodes: GraphNode[]
  edges: GraphEdge[]
  total_nodes: number
  total_edges: number
  subsampled: boolean
  isolated_count: number
}

export interface PackageInfo {
  name: string
  files: number
}

export interface EntryPoint {
  symbol: string
  path: string
  kind: string
}

export interface Hotspot {
  symbol: string
  path: string
  incoming: number
  outgoing: number
}

export interface RecentlyChanged {
  path: string
  status: string
}

export interface Architecture {
  project: string
  total_files: number
  total_symbols: number
  total_edges: number
  languages: Record<string, number>
  packages: PackageInfo[]
  entry_points: EntryPoint[]
  hotspots: Hotspot[]
  symbols_by_kind: Record<string, number>
  recently_changed?: RecentlyChanged[]
}

export interface DeadCodeCandidate {
  symbol: string
  path: string
  kind: string
  start_line: number
  end_line: number
}

export interface DeadCodeResponse {
  candidates: DeadCodeCandidate[]
  count: number
}

export interface ChangeEntry {
  path: string
  status: string
}

export interface ChangesResponse {
  changes: ChangeEntry[]
  count: number
}

export interface AddProjectOptions {
  extensions?: string[]
  include?: string[]
  exclude?: string[]
  max_file_size_mb?: number
}

// --- JSON-RPC client ---

type PendingRequest = {
  resolve: (value: any) => void
  reject: (reason: any) => void
}

let ws: WebSocket | null = null
let nextId = 1
const pending = new Map<number, PendingRequest>()
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let statusListeners: ((state: StatusResponse) => void)[] = []
let connectPromise: Promise<void> | null = null

function wsUrl(): string {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${proto}//${location.host}/ws`
}

function connect(): Promise<void> {
  if (ws && ws.readyState === WebSocket.OPEN) return Promise.resolve()
  if (connectPromise) return connectPromise

  connectPromise = new Promise<void>((resolve, reject) => {
    const socket = new WebSocket(wsUrl())
    socket.onopen = () => {
      ws = socket
      connectPromise = null
      resolve()
    }
    socket.onerror = () => {
      connectPromise = null
      reject(new Error('WebSocket connection failed'))
    }
    socket.onclose = () => {
      ws = null
      connectPromise = null
      // Reject all pending requests
      for (const [, p] of pending) {
        p.reject(new Error('WebSocket closed'))
      }
      pending.clear()
      // Reconnect after 2 seconds
      if (reconnectTimer) clearTimeout(reconnectTimer)
      reconnectTimer = setTimeout(() => {
        reconnectTimer = null
        connect().catch(() => {})
      }, 2000)
    }
    socket.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data)
        // Check if this is a response to a pending request
        if (msg.id !== undefined && pending.has(msg.id)) {
          const p = pending.get(msg.id)!
          pending.delete(msg.id)
          if (msg.error) {
            p.reject(new Error(msg.error.message || 'RPC error'))
          } else {
            p.resolve(msg.result)
          }
          return
        }
        // Check if this is a notification
        if (msg.method === 'gnostis/status') {
          const state = msg.params as StatusResponse
          for (const listener of statusListeners) {
            listener(state)
          }
        }
      } catch {
        // ignore parse errors
      }
    }
  })

  return connectPromise
}

// Ensure connection is established before sending requests
async function ensureConnected(): Promise<WebSocket> {
  await connect()
  if (!ws || ws.readyState !== WebSocket.OPEN) {
    throw new Error('WebSocket not connected')
  }
  return ws!
}

// Call a JSON-RPC method and return the result.
// MCP tool calls use the "tools/call" method with the tool name in params.
// Dashboard-specific methods are called directly by name.
async function call<T = any>(method: string, params?: any): Promise<T> {
  const socket = await ensureConnected()
  const id = nextId++

  return new Promise<T>((resolve, reject) => {
    pending.set(id, { resolve, reject })

    const msg: any = {
      jsonrpc: '2.0',
      id,
      method,
    }
    if (params !== undefined) {
      msg.params = params
    }

    socket.send(JSON.stringify(msg))

    // Timeout after 30 seconds
    setTimeout(() => {
      if (pending.has(id)) {
        pending.delete(id)
        reject(new Error('Request timeout'))
      }
    }, 30000)
  })
}

// Call an MCP tool via the standard tools/call JSON-RPC method.
// The MCP server returns tool results as { content: [{ type: 'text', text: '...' }] }.
// We extract and parse the text content.
async function callTool<T = any>(name: string, args?: any): Promise<T> {
  const result = await call<any>('tools/call', { name, arguments: args ?? {} })
  // MCP tool result format: { content: [{ type: 'text', text: '...' }] }
  if (result?.content?.[0]?.text) {
    const text = result.content[0].text
    try {
      return JSON.parse(text) as T
    } catch {
      return text as unknown as T
    }
  }
  return result as T
}

// --- Public API (same signatures as api.ts) ---

export async function getStatus(): Promise<StatusResponse> {
  return callTool<StatusResponse>('get_index_status')
}

export async function reindexProject(name: string): Promise<JobResponse> {
  return callTool<JobResponse>('rebuild_project', { project: name })
}

export async function reindexAll(): Promise<JobResponse> {
  return callTool<JobResponse>('rebuild_index')
}

export async function pickDirectory(): Promise<{ path: string }> {
  return call<{ path: string }>('pick_directory')
}

export async function addProject(path: string, name: string, opts?: AddProjectOptions): Promise<{ name: string }> {
  return callTool<{ name: string }>('add_project', { path, name, ...opts })
}

export async function editProject(name: string, opts: AddProjectOptions): Promise<void> {
  await callTool('edit_project', { name, ...opts })
}

export async function removeProject(name: string): Promise<void> {
  await callTool('remove_project', { name })
}

export async function openProject(name: string): Promise<void> {
  await call('open_project', { name })
}

export async function reindexFiles(paths: string[]): Promise<void> {
  await callTool('reindex_files', { paths })
}

export async function search(query: string, topK = 10, project?: string): Promise<SearchResponse> {
  const results = await callTool<SearchResult[]>('search_codebase', { query, top_k: topK, project })
  return {
    query,
    results: results || [],
    count: results?.length ?? 0,
  }
}

export async function getMemoryFiles(): Promise<MemoryFile[]> {
  return callTool<MemoryFile[]>('memory_files')
}

export async function openMemoryFile(path: string): Promise<void> {
  await call('open_memory_file', { path })
}

export async function getGraph(project?: string, opts?: { connected_only?: boolean; max_nodes?: number }): Promise<GraphResponse> {
  return callTool<GraphResponse>('graph_layout', {
    project: project ?? '',
    connected_only: opts?.connected_only ?? true,
    max_nodes: opts?.max_nodes ?? 800,
  })
}

export async function getArchitecture(project: string): Promise<Architecture> {
  return callTool<Architecture>('get_architecture', { project })
}

export async function getDeadCode(project: string, kind?: string, topK?: number): Promise<DeadCodeResponse> {
  const candidates = await callTool<DeadCodeCandidate[]>('dead_code', {
    project,
    kind: kind ?? 'both',
    top_k: topK ?? 50,
  })
  return {
    candidates: candidates || [],
    count: candidates?.length ?? 0,
  }
}

export async function getChanges(project: string): Promise<ChangesResponse> {
  const changes = await callTool<ChangeEntry[]>('detect_changes', { project })
  return {
    changes: changes || [],
    count: changes?.length ?? 0,
  }
}

// Subscribe to status notifications pushed by the server.
// Replaces the SSE-based subscribeToEvents.
export function subscribeToEvents(onMessage: (state: StatusResponse) => void): () => void {
  statusListeners.push(onMessage)

  // Ensure connection is open
  connect().catch(() => {})

  return () => {
    statusListeners = statusListeners.filter((l) => l !== onMessage)
  }
}
