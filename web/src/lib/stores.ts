import { writable } from 'svelte/store'
import type { StatusResponse, ProgressState, MemoryProgressState, SearchResult, GraphResponse, Architecture, DeadCodeResponse, ChangesResponse } from './api'
import { getStatus, subscribeToEvents } from './api'

export type Section = 'overview' | 'graph' | 'architecture' | 'dead-code' | 'changes'

function getSectionFromHash(): Section {
  const hash = window.location.hash.replace('#', '')
  const valid: Section[] = ['overview', 'graph', 'architecture', 'dead-code', 'changes']
  return (valid as string[]).includes(hash) ? (hash as Section) : 'overview'
}

export const activeSection = writable<Section>(getSectionFromHash())

activeSection.subscribe((section) => {
  if (window.location.hash !== '#' + section) {
    window.location.hash = section
  }
})

if (typeof window !== 'undefined') {
  window.addEventListener('hashchange', () => {
    activeSection.set(getSectionFromHash())
  })
}
export const selectedProject = writable<string>('')

export const status = writable<StatusResponse | null>(null)
export const progress = writable<ProgressState | null>(null)
export const memoryProgress = writable<MemoryProgressState | null>(null)
export const searchResults = writable<SearchResult[]>([])
export const loading = writable(false)
export const error = writable<string | null>(null)
export const showAddModal = writable(false)
export const searchOpen = writable(false)

export const graphData = writable<GraphResponse | null>(null)
export const graphLoading = writable(false)
export const architectureData = writable<Architecture | null>(null)
export const architectureLoading = writable(false)
export const deadCodeData = writable<DeadCodeResponse | null>(null)
export const deadCodeLoading = writable(false)
export const changesData = writable<ChangesResponse | null>(null)
export const changesLoading = writable(false)

export interface Toast {
  id: number
  type: 'success' | 'error' | 'info'
  message: string
}

export const toasts = writable<Toast[]>([])
let toastId = 0

export function pushToast(type: Toast['type'], message: string) {
  const id = ++toastId
  toasts.update((t) => [...t, { id, type, message }])
  setTimeout(() => dismissToast(id), 4000)
}

export function dismissToast(id: number) {
  toasts.update((t) => t.filter((x) => x.id !== id))
}

export async function refreshStatus() {
  try {
    const s = await getStatus()
    status.set(s)
    progress.set(s.progress)
    memoryProgress.set(s.memory_progress ?? null)
  } catch (e) {
    error.set(String(e))
  }
}

export function initEventSource() {
  return subscribeToEvents((s) => {
    status.set(s)
    progress.set(s.progress)
    memoryProgress.set(s.memory_progress ?? null)
    loading.set(false)
    error.set(null)
  })
}
