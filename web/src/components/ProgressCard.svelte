<script lang="ts">
  import { progress, status } from '../lib/stores'
  import Card from './ui/Card.svelte'
  import Badge from './ui/Badge.svelte'
  import { AlertCircle, Clock } from '@lucide/svelte'

  let p = $derived($progress)
  let s = $derived($status)

  let filePct = $derived(
    p && p.total_files > 0
      ? Math.round((p.done_files / p.total_files) * 100)
      : 0,
  )

  let chunkPct = $derived(
    p && p.total_chunks > 0
      ? Math.round((p.done_chunks / p.total_chunks) * 100)
      : 0,
  )

  let isRunning = $derived(p?.status === 'running')
  let isError = $derived(p?.status === 'error')
  let isDone = $derived(p?.status === 'done')

  let isChunking = $derived(p?.phase === 'chunking')
  let isEmbedding = $derived(p?.phase === 'embedding')

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
      <!-- File progress (chunking phase) -->
      {#if p.total_files > 0}
        <div class="mb-2">
          <div class="flex justify-between items-center text-xs text-muted-foreground mb-1">
            <span>Files</span>
            <span>{p.done_files.toLocaleString()} / {p.total_files.toLocaleString()}</span>
          </div>
          <div class="w-full bg-secondary rounded-full h-2 overflow-hidden">
            <div
              class="h-full rounded-full transition-all duration-500 {isDone ? 'bg-success' : 'bg-primary'}"
              style="width: {filePct}%"
            ></div>
          </div>
        </div>
      {/if}

      <!-- Chunk progress (embedding phase) -->
      {#if p.total_chunks > 0}
        <div class="mb-1.5">
          <div class="flex justify-between items-center text-xs text-muted-foreground mb-1">
            <span>Chunks</span>
            <span>{p.done_chunks.toLocaleString()} / {p.total_chunks.toLocaleString()}</span>
          </div>
          <div class="w-full bg-secondary rounded-full h-2 overflow-hidden">
            <div
              class="h-full rounded-full transition-all duration-500 {isDone ? 'bg-success' : 'bg-primary'}"
              style="width: {chunkPct}%"
            ></div>
          </div>
        </div>
      {/if}

      <div class="flex justify-between items-center text-xs text-muted-foreground mt-1">
        <span class="font-medium text-foreground">
          {#if isChunking && p.total_files > 0}
            {filePct}% files
          {:else if isEmbedding && p.total_chunks > 0}
            {chunkPct}% chunks
          {:else if isDone}
            Done
          {/if}
        </span>
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
