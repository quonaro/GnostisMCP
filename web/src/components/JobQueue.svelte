<script lang="ts">
  import { status, progress } from '../lib/stores'
  import Card from './ui/Card.svelte'
  import {
    ListChecks, Loader2, CheckCircle2, AlertCircle, Clock,
    Eye, RefreshCw, FileCode, ChevronDown, ChevronRight,
    Zap, Bot, Timer,
  } from '@lucide/svelte'
  import type { Job } from '../lib/api'

  let s = $derived($status)
  let p = $derived($progress)
  let jobs = $derived(s?.jobs ?? [])

  let pending = $derived(jobs.filter((j) => j.status === 'pending'))
  let running = $derived(jobs.filter((j) => j.status === 'running'))
  let completed = $derived(jobs.filter((j) => j.status === 'done' || j.status === 'failed'))
  let failedCount = $derived(completed.filter((j) => j.status === 'failed').length)

  let showCompleted = $state(false)

  let runningJob = $derived(running[0] ?? null)

  let isProgressRunning = $derived(!!runningJob && !!p && p.status === 'running')

  let hasProgress = $derived(
    isProgressRunning &&
    (p.total_files > 0 || p.total_chunks > 0),
  )

  let phaseLabel = $derived.by(() => {
    if (!isProgressRunning || !p) return ''
    switch (p.phase) {
      case 'indexing': return 'Scanning files'
      case 'chunking': return 'Chunking'
      case 'embedding': return 'Embedding'
      default: return p.phase || 'Processing'
    }
  })

  let currentProject = $derived(isProgressRunning ? p?.project ?? '' : '')

  let progressPct = $derived.by(() => {
    if (!hasProgress || !p) return 0
    if (p.total_chunks > 0) return Math.round((p.done_chunks / p.total_chunks) * 100)
    if (p.total_files > 0) return Math.round((p.done_files / p.total_files) * 100)
    return 0
  })

  let progressLabel = $derived.by(() => {
    if (!hasProgress || !p) return ''
    if (p.total_chunks > 0) return `${p.done_chunks}/${p.total_chunks} chunks`
    if (p.total_files > 0) return `${p.done_files}/${p.total_files} files`
    return ''
  })

  let isActive = $derived(running.length > 0 || pending.length > 0)

  function isAuto(j: Job): boolean {
    return j.type === 'watcher'
  }

  function formatTime(ts: string): string {
    return new Date(ts).toLocaleTimeString()
  }

  function formatDuration(start?: string, end?: string): string {
    if (!start) return ''
    const startMs = new Date(start).getTime()
    const endMs = end ? new Date(end).getTime() : Date.now()
    const seconds = Math.round((endMs - startMs) / 1000)
    if (seconds < 60) return `${seconds}s`
    const minutes = Math.floor(seconds / 60)
    const rem = seconds % 60
    return `${minutes}m ${rem}s`
  }

  function formatETA(seconds: number): string {
    if (seconds <= 0) return ''
    if (seconds < 60) return `${seconds}s`
    const m = Math.floor(seconds / 60)
    const sec = seconds % 60
    return `${m}m ${sec}s`
  }

  let etaSeconds = $derived(s?.eta_seconds ?? 0)

  function typeIcon(j: Job) {
    switch (j.type) {
      case 'watcher': return Eye
      case 'index': return RefreshCw
      default:
        if (j.type.startsWith('project:')) return FileCode
        return ListChecks
    }
  }
</script>

<Card class="p-4">
  <div class="flex items-center gap-2 mb-3">
    <ListChecks class="w-4 h-4 text-muted-foreground" />
    <h2 class="text-sm font-semibold text-muted-foreground uppercase">Job Queue</h2>
    {#if isActive}
      <div class="ml-auto flex items-center gap-1.5">
        {#if running.length > 0}
          <span class="flex items-center gap-1 text-xs text-warning">
            <span class="inline-block w-1.5 h-1.5 rounded-full bg-warning animate-pulse"></span>
            {running.length} active
          </span>
        {/if}
        {#if pending.length > 0}
          <span class="flex items-center gap-1 text-xs text-muted-foreground">
            <Clock class="w-3 h-3" />
            {pending.length} queued
          </span>
        {/if}
      </div>
    {/if}
  </div>

  {#if jobs.length === 0}
    <div class="flex flex-col items-center gap-2 py-6 text-center">
      <CheckCircle2 class="w-6 h-6 text-success/50" />
      <span class="text-sm text-muted-foreground">Queue is empty — nothing pending</span>
    </div>
  {:else}
    <div class="space-y-2 max-h-[28rem] overflow-y-auto">

      {#if runningJob}
        {@const Ti = typeIcon(runningJob)}
        {@const auto = isAuto(runningJob)}
        <div class="rounded-lg border border-warning/40 bg-warning/5 p-3 transition-all">
          <div class="flex items-center gap-2 mb-1">
            <Ti class="w-4 h-4 text-warning shrink-0" />
            <span class="text-sm font-medium text-foreground truncate flex-1">{runningJob.description}</span>
            {#if auto}
              <span class="flex items-center gap-0.5 text-[10px] text-muted-foreground bg-secondary/80 rounded-full px-1.5 py-0.5 shrink-0">
                <Bot class="w-2.5 h-2.5" />
                Auto
              </span>
            {:else}
              <span class="flex items-center gap-0.5 text-[10px] text-primary bg-primary/10 rounded-full px-1.5 py-0.5 shrink-0">
                <Zap class="w-2.5 h-2.5" />
                Manual
              </span>
            {/if}
          </div>

          {#if currentProject}
            <div class="flex items-center gap-2 mb-2 text-xs">
              <span class="flex items-center gap-1 text-muted-foreground">
                <FileCode class="w-3 h-3" />
                <span class="font-medium text-foreground">{currentProject}</span>
              </span>
              {#if phaseLabel}
                <span class="flex items-center gap-1 text-warning">
                  <Loader2 class="w-3 h-3 animate-spin" />
                  {phaseLabel}
                </span>
              {/if}
            </div>
          {/if}

          {#if hasProgress}
            <div class="mb-1.5">
              <div class="flex justify-between items-center text-[11px] text-muted-foreground mb-1">
                <span>{progressLabel}</span>
                <span>{progressPct}%</span>
              </div>
              <div class="w-full bg-secondary rounded-full h-1.5 overflow-hidden">
                <div
                  class="h-full rounded-full bg-warning transition-all duration-500"
                  style="width: {progressPct}%"
                ></div>
              </div>
              <div class="flex justify-end items-center text-[10px] text-muted-foreground mt-1">
                <span class="flex items-center gap-1">
                  <Timer class="w-2.5 h-2.5" />
                  {formatDuration(runningJob.started_at)}
                  {#if etaSeconds > 0}
                    <span class="text-muted-foreground/60">·</span>
                    <Clock class="w-2.5 h-2.5" />
                    ETA {formatETA(etaSeconds)}
                  {/if}
                </span>
              </div>
            </div>
          {:else if isProgressRunning}
            <div class="flex items-center gap-1.5 text-[11px] text-muted-foreground">
              <Loader2 class="w-3 h-3 animate-spin text-warning" />
              <span>{formatDuration(runningJob.started_at)}</span>
            </div>
          {:else}
            <div class="flex items-center gap-1.5 text-[11px] text-muted-foreground">
              <Loader2 class="w-3 h-3 animate-spin text-warning" />
              <span>{formatDuration(runningJob.started_at)}</span>
            </div>
          {/if}
        </div>
      {/if}

      {#if pending.length > 0}
        <div class="space-y-1">
          <div class="flex items-center gap-1.5 text-[11px] font-medium text-muted-foreground/70 uppercase tracking-wide px-1 pb-0.5">
            <Clock class="w-3 h-3" />
            Queued ({pending.length})
          </div>
          {#each pending as j, i (j.id)}
            {@const Ti = typeIcon(j)}
            {@const auto = isAuto(j)}
            <div class="flex items-center gap-2 px-2 py-1.5 rounded-md hover:bg-secondary/40 transition-colors group">
              <span class="text-[10px] text-muted-foreground/60 shrink-0 w-4 text-right tabular-nums font-mono">{i + 1}</span>
              <Ti class="w-3.5 h-3.5 text-muted-foreground shrink-0" />
              <span class="text-xs text-muted-foreground truncate flex-1">{j.description}</span>
              {#if auto}
                <span class="flex items-center gap-0.5 text-[10px] text-muted-foreground/60 shrink-0">
                  <Bot class="w-2.5 h-2.5" />
                </span>
              {/if}
            </div>
          {/each}
        </div>
      {/if}

      {#if completed.length > 0}
        <div class="pt-1">
          <button
            class="flex items-center gap-1 text-[11px] font-medium text-muted-foreground/70 hover:text-foreground transition-colors px-1 pb-0.5 w-full uppercase tracking-wide"
            onclick={() => (showCompleted = !showCompleted)}
          >
            {#if showCompleted}
              <ChevronDown class="w-3 h-3" />
            {:else}
              <ChevronRight class="w-3 h-3" />
            {/if}
            <span>Completed ({completed.length}{#if failedCount > 0}, {failedCount} failed{/if})</span>
          </button>

          {#if showCompleted}
            <div class="space-y-0.5 mt-1">
              {#each completed as j (j.id)}
                {@const Ti = typeIcon(j)}
                {@const auto = isAuto(j)}
                <div class="flex items-center gap-2 px-2 py-1.5 rounded-md hover:bg-secondary/40 transition-colors">
                  <Ti class="w-3.5 h-3.5 shrink-0 text-muted-foreground/60" />
                  {#if j.status === 'failed'}
                    <AlertCircle class="w-3.5 h-3.5 shrink-0 text-destructive" />
                  {:else}
                    <CheckCircle2 class="w-3.5 h-3.5 shrink-0 text-success/70" />
                  {/if}
                  <span class="text-xs text-muted-foreground truncate flex-1">{j.description}</span>
                  {#if auto}
                    <span class="shrink-0">
                      <Bot class="w-2.5 h-2.5 text-muted-foreground/40" />
                    </span>
                  {/if}
                  {#if j.started_at}
                    <span class="text-[10px] text-muted-foreground/60 shrink-0 tabular-nums">{formatDuration(j.started_at, j.finished_at)}</span>
                  {/if}
                  <span class="text-[10px] text-muted-foreground/60 shrink-0 tabular-nums">{formatTime(j.finished_at ?? j.started_at ?? j.created_at)}</span>
                </div>
                {#if j.status === 'failed' && j.error}
                  <div class="text-[10px] text-destructive/80 pl-8 pb-1 truncate">{j.error}</div>
                {/if}
              {/each}
            </div>
          {/if}
        </div>
      {/if}
    </div>
  {/if}
</Card>
