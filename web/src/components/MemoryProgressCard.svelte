<script lang="ts">
  import { memoryProgress, status } from '../lib/stores'
  import Card from './ui/Card.svelte'
  import { MessageSquare, AlertCircle } from '@lucide/svelte'

  let p = $derived($memoryProgress)
  let s = $derived($status)

  let hasMemory = $derived((s?.memory_stats ?? []).length > 0)

  let filePct = $derived(
    p && p.total_files > 0
      ? Math.round((p.done_files / p.total_files) * 100)
      : 0,
  )

  let isRunning = $derived(p?.status === 'running')
  let isError = $derived(p?.status === 'error')
  let isDone = $derived(p?.status === 'done')
</script>

{#if hasMemory}
  <Card class="p-4">
    <div class="flex items-center gap-2 mb-3">
      <MessageSquare class="w-4 h-4 text-muted-foreground" />
      <h2 class="text-sm font-semibold text-muted-foreground uppercase">Memory Sync</h2>
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
      </div>

      {#if isRunning || isDone}
        {#if p.total_files > 0}
          <div class="mb-2">
            <div class="flex justify-between items-center text-xs text-muted-foreground mb-1">
              <span>Dialogues</span>
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

        <div class="flex justify-between items-center text-xs text-muted-foreground mt-1">
          <span class="font-medium text-foreground">
            {#if isRunning && p.total_files > 0}
              {filePct}%
            {:else if isDone}
              Done
            {/if}
          </span>
        </div>
      {:else if !isError}
        <div class="text-sm text-muted-foreground">Idle — no active memory sync.</div>
      {/if}
    {:else}
      <div class="text-sm text-muted-foreground">No progress data.</div>
    {/if}
  </Card>
{/if}
