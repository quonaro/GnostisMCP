import { writable } from 'svelte/store'
import type { StatusResponse, ProgressState, SearchResult } from './api'
import { getStatus, subscribeToEvents } from './api'

export const status = writable<StatusResponse | null>(null)
export const progress = writable<ProgressState | null>(null)
export const searchResults = writable<SearchResult[]>([])
export const loading = writable(false)
export const error = writable<string | null>(null)
export const showAddModal = writable(false)

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
  loading.set(true)
  error.set(null)
  try {
    const s = await getStatus()
    status.set(s)
    progress.set(s.progress)
  } catch (e) {
    error.set(String(e))
  } finally {
    loading.set(false)
  }
}

export function initEventSource() {
  return subscribeToEvents((state) => {
    progress.set(state)
  })
}
