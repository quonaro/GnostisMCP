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

const API_BASE = '/api'

async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const resp = await fetch(url, init)
  if (!resp.ok) {
    const body = await resp.json().catch(() => ({ error: resp.statusText }))
    throw new Error(body.error || resp.statusText)
  }
  return resp.json()
}

export async function getStatus(): Promise<StatusResponse> {
  return fetchJSON<StatusResponse>(`${API_BASE}/status`)
}

export async function rebuildProject(name: string): Promise<JobResponse> {
  return fetchJSON<JobResponse>(`${API_BASE}/rebuild/project`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  })
}

export async function rebuildIndex(): Promise<JobResponse> {
  return fetchJSON<JobResponse>(`${API_BASE}/rebuild/index`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: '{}',
  })
}

export async function pickDirectory(): Promise<{ path: string }> {
  return fetchJSON(`${API_BASE}/projects/pick-directory`)
}

export interface AddProjectOptions {
  extensions?: string[]
  include?: string[]
  exclude?: string[]
  max_file_size_mb?: number
}

export async function addProject(path: string, name: string, opts?: AddProjectOptions): Promise<{ name: string }> {
  return fetchJSON(`${API_BASE}/projects/add`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ path, name, ...opts }),
  })
}

export async function editProject(name: string, opts: AddProjectOptions): Promise<void> {
  await fetchJSON(`${API_BASE}/projects/edit`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, ...opts }),
  })
}

export async function removeProject(name: string): Promise<void> {
  await fetchJSON(`${API_BASE}/projects/remove`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  })
}

export async function openProject(name: string): Promise<void> {
  await fetchJSON(`${API_BASE}/projects/open`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  })
}

export async function reindexFiles(paths: string[]): Promise<void> {
  await fetchJSON(`${API_BASE}/reindex`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ paths }),
  })
}

export async function search(query: string, topK = 10, project?: string): Promise<SearchResponse> {
  const params = new URLSearchParams({ q: query, top_k: String(topK) })
  if (project) params.set('project', project)
  return fetchJSON<SearchResponse>(`${API_BASE}/search?${params}`)
}

export async function getMemoryFiles(): Promise<MemoryFile[]> {
  return fetchJSON<MemoryFile[]>(`${API_BASE}/memory/files`)
}

export async function openMemoryFile(path: string): Promise<void> {
  await fetchJSON(`${API_BASE}/memory/open`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ path }),
  })
}

export async function getGraph(project?: string, opts?: { connected_only?: boolean; max_nodes?: number }): Promise<GraphResponse> {
  const params = new URLSearchParams()
  if (project) params.set('project', project)
  if (opts?.connected_only === false) params.set('connected_only', 'false')
  if (opts?.max_nodes) params.set('max_nodes', String(opts.max_nodes))
  return fetchJSON<GraphResponse>(`${API_BASE}/graph?${params}`)
}

export async function getArchitecture(project: string): Promise<Architecture> {
  const params = new URLSearchParams({ project })
  return fetchJSON<Architecture>(`${API_BASE}/architecture?${params}`)
}

export async function getDeadCode(project: string, kind?: string, topK?: number): Promise<DeadCodeResponse> {
  const params = new URLSearchParams({ project })
  if (kind) params.set('kind', kind)
  if (topK) params.set('top_k', String(topK))
  return fetchJSON<DeadCodeResponse>(`${API_BASE}/dead-code?${params}`)
}

export async function getChanges(project: string): Promise<ChangesResponse> {
  const params = new URLSearchParams({ project })
  return fetchJSON<ChangesResponse>(`${API_BASE}/changes?${params}`)
}

export function subscribeToEvents(onMessage: (state: StatusResponse) => void): () => void {
  const es = new EventSource(`${API_BASE}/events`)
  es.onmessage = (e) => {
    try {
      onMessage(JSON.parse(e.data))
    } catch {
      // ignore parse errors
    }
  }
  return () => es.close()
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
