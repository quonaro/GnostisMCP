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
  try {
    const s = await getStatus()
    status.set(s)
    progress.set(s.progress)
  } catch (e) {
    error.set(String(e))
  }
}

export function initEventSource() {
  return subscribeToEvents((s) => {
    status.set(s)
    progress.set(s.progress)
    loading.set(false)
    error.set(null)
  })
}
