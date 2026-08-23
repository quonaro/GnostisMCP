<script lang="ts">
  import { status } from '../lib/stores'
  import Card from './ui/Card.svelte'
  import Badge from './ui/Badge.svelte'
  import { ListChecks, Loader2, CheckCircle2, AlertCircle, Clock, Eye, RefreshCw, FileCode, ChevronDown, ChevronRight } from '@lucide/svelte'
  import type { Job } from '../lib/api'

  let s = $derived($status)
  let jobs = $derived(s?.jobs ?? [])

  let pending = $derived(jobs.filter((j) => j.status === 'pending'))
  let running = $derived(jobs.filter((j) => j.status === 'running'))
  let completed = $derived(jobs.filter((j) => j.status === 'done' || j.status === 'failed'))
  let failedCount = $derived(completed.filter((j) => j.status === 'failed').length)

  let showCompleted = $state(true)

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

  function statusBadge(j: Job) {
    switch (j.status) {
      case 'pending':
        return { variant: 'secondary' as const, icon: Clock, label: 'Pending' }
      case 'running':
        return { variant: 'warning' as const, icon: Loader2, label: 'Running' }
      case 'done':
        return { variant: 'success' as const, icon: CheckCircle2, label: 'Done' }
      case 'failed':
        return { variant: 'destructive' as const, icon: AlertCircle, label: 'Failed' }
    }
  }

  function typeIcon(j: Job) {
    switch (j.type) {
      case 'watcher':
        return Eye
      case 'index':
        return RefreshCw
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
    {#if jobs.length > 0}
      <Badge variant="outline">{jobs.length}</Badge>
    {/if}
    {#if running.length > 0}
      <span class="ml-auto text-xs text-warning flex items-center gap-1">
        <Loader2 class="w-3 h-3 animate-spin" />
        {running.length} running
      </span>
    {:else if pending.length > 0}
      <span class="ml-auto text-xs text-muted-foreground flex items-center gap-1">
        <Clock class="w-3 h-3" />
        {pending.length} queued
      </span>
    {/if}
  </div>

  {#if jobs.length === 0}
    <div class="text-sm text-muted-foreground py-4 text-center">
      No jobs in queue
    </div>
  {:else}
    <div class="space-y-1.5 max-h-96 overflow-y-auto">
      {#each running as j (j.id)}
        {@const b = statusBadge(j)}
        {@const Ti = typeIcon(j)}
        <div class="flex items-center gap-2 p-2 rounded-md border border-warning/30 bg-warning/5">
          <Ti class="w-3.5 h-3.5 text-warning shrink-0" />
          <b.icon class="w-3.5 h-3.5 text-warning shrink-0 animate-spin" />
          <span class="text-sm text-foreground truncate">{j.description}</span>
          {#if j.started_at}
            <span class="text-xs text-muted-foreground shrink-0">{formatDuration(j.started_at)}</span>
          {/if}
          <Badge variant={b.variant} class="ml-auto shrink-0">{b.label}</Badge>
        </div>
      {/each}

      {#each pending as j, i (j.id)}
        {@const b = statusBadge(j)}
        {@const Ti = typeIcon(j)}
        <div class="flex items-center gap-2 p-2 rounded-md border border-border bg-card/50">
          <span class="text-xs text-muted-foreground shrink-0 w-4 text-right tabular-nums">{i + 1}</span>
          <Ti class="w-3.5 h-3.5 text-muted-foreground shrink-0" />
          <span class="text-sm text-muted-foreground truncate">{j.description}</span>
          <Badge variant={b.variant} class="ml-auto shrink-0">{b.label}</Badge>
        </div>
      {/each}

      {#if completed.length > 0}
        <button
          class="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors pt-1 pb-0.5 w-full"
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
          {#each completed as j (j.id)}
            {@const b = statusBadge(j)}
            {@const Ti = typeIcon(j)}
            <div class="flex items-center gap-2 p-2 rounded-md border border-border bg-card/50">
              <Ti class="w-3.5 h-3.5 shrink-0 text-muted-foreground" />
              <b.icon class="w-3.5 h-3.5 shrink-0 {j.status === 'failed' ? 'text-destructive' : 'text-success'}" />
              <span class="text-sm text-muted-foreground truncate">{j.description}</span>
              {#if j.started_at}
                <span class="text-xs text-muted-foreground shrink-0">{formatDuration(j.started_at, j.finished_at)}</span>
              {/if}
              <span class="text-xs text-muted-foreground shrink-0 ml-auto">{formatTime(j.finished_at ?? j.started_at ?? j.created_at)}</span>
              <Badge variant={b.variant} class="shrink-0">{b.label}</Badge>
            </div>
            {#if j.status === 'failed' && j.error}
              <div class="text-xs text-destructive pl-8 pb-1 truncate">{j.error}</div>
            {/if}
          {/each}
        {/if}
      {/if}
    </div>
  {/if}
</Card>
