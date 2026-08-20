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

export function subscribeToEvents(onMessage: (state: ProgressState) => void): () => void {
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
