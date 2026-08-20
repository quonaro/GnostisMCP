<script lang="ts">
  import { progress, status, pushToast } from '../lib/stores'
  import { rebuildIndex } from '../lib/api'
  import { refreshStatus } from '../lib/stores'
  import Card from './ui/Card.svelte'
  import Button from './ui/Button.svelte'
  import Badge from './ui/Badge.svelte'
  import { RefreshCw, AlertCircle, Clock } from '@lucide/svelte'

  let p = $derived($progress)
  let s = $derived($status)

  let pct = $derived(
    p && p.total_chunks > 0
      ? Math.round((p.done_chunks / p.total_chunks) * 100)
      : 0,
  )

  let isRunning = $derived(p?.status === 'running')
  let isError = $derived(p?.status === 'error')
  let isDone = $derived(p?.status === 'done')

  let busy = $state(false)

  async function handleRebuildAll() {
    busy = true
    try {
      await rebuildIndex()
      await refreshStatus()
      pushToast('success', 'Index rebuild started')
    } catch (e) {
      pushToast('error', String(e))
    } finally {
      busy = false
    }
  }

  function formatETA(seconds: number): string {
    if (seconds <= 0) return '—'
    if (seconds < 60) return `${seconds}s`
    const m = Math.floor(seconds / 60)
    const sec = seconds % 60
    return `${m}m ${sec}s`
  }
</script>

<Card class="p-4">
  <div class="flex items-center justify-between mb-3">
    <h2 class="text-sm font-semibold text-muted-foreground uppercase">Indexing Progress</h2>
    <Button variant="default" size="sm" onclick={handleRebuildAll} disabled={busy || isRunning}>
      <RefreshCw class="w-3.5 h-3.5" />
      {busy ? 'Starting...' : 'Rebuild All'}
    </Button>
  </div>

  {#if isError && p?.error}
    <div class="flex items-start gap-2 rounded-md border border-destructive/50 bg-destructive/10 p-2 text-sm text-destructive mb-2">
      <AlertCircle class="w-4 h-4 shrink-0 mt-0.5" />
      {p.error}
    </div>
  {/if}

  {#if p}
    <div class="flex items-center gap-2 mb-2">
      <span
        class="inline-block w-2 h-2 rounded-full {isRunning ? 'bg-warning animate-pulse' : isDone ? 'bg-success' : isError ? 'bg-destructive' : 'bg-muted-foreground/40'}"
      ></span>
      <span class="text-sm text-foreground capitalize">{p.status}</span>
      {#if p.phase}
        <Badge variant="outline">{p.phase}</Badge>
      {/if}
      {#if p.project}
        <Badge variant="secondary">{p.project}</Badge>
      {/if}
    </div>

    {#if isRunning || isDone}
      <div class="w-full bg-secondary rounded-full h-2 mb-1.5 overflow-hidden">
        <div
          class="h-full rounded-full transition-all duration-500 {isDone ? 'bg-success' : 'bg-primary'}"
          style="width: {pct}%"
        ></div>
      </div>
      <div class="flex justify-between items-center text-xs text-muted-foreground">
        <span>{p.done_chunks.toLocaleString()} / {p.total_chunks.toLocaleString()} chunks</span>
        <span class="font-medium text-foreground">{pct}%</span>
        {#if s?.eta_seconds && s.eta_seconds > 0}
          <span class="flex items-center gap-1">
            <Clock class="w-3 h-3" />
            ETA: {formatETA(s.eta_seconds)}
          </span>
        {/if}
      </div>
    {:else}
      <div class="text-sm text-muted-foreground">Idle — no active indexing job.</div>
    {/if}
  {:else}
    <div class="text-sm text-muted-foreground">No progress data.</div>
  {/if}
</Card>
