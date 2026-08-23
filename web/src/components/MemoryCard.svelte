<script lang="ts">
  import { status, memoryProgress, pushToast } from '../lib/stores'
  import { getMemoryFiles, openMemoryFile, type MemoryFile } from '../lib/api'
  import Card from './ui/Card.svelte'
  import Badge from './ui/Badge.svelte'
  import { MessageSquare, FileText, Hash, ExternalLink, ChevronDown, ChevronRight, AlertCircle, Loader2 } from '@lucide/svelte'

  let s = $derived($status)
  let memStats = $derived(s?.memory_stats ?? [])
  let hasMemory = $derived(memStats.length > 0)

  let memoryFiles = $state<MemoryFile[]>([])
  let loadingFiles = $state(false)
  let expanded = $state(false)

  let totalFiles = $derived(memStats.reduce((sum, st) => sum + st.files, 0))
  let totalChunks = $derived(memStats.reduce((sum, st) => sum + st.chunks, 0))

  let mp = $derived($memoryProgress)
  let memSyncRunning = $derived(mp?.status === 'running')
  let memSyncError = $derived(mp?.status === 'error')
  let memFilePct = $derived(
    mp && mp.total_files > 0
      ? Math.round((mp.done_files / mp.total_files) * 100)
      : 0,
  )

  async function loadFiles() {
    if (loadingFiles) return
    loadingFiles = true
    try {
      memoryFiles = await getMemoryFiles()
    } catch (e) {
      pushToast('error', `Failed to load memory files: ${String(e)}`)
    } finally {
      loadingFiles = false
    }
  }

  async function toggleExpand() {
    expanded = !expanded
    if (expanded && memoryFiles.length === 0) {
      await loadFiles()
    }
  }

  async function handleOpen(path: string) {
    try {
      await openMemoryFile(path)
    } catch (e) {
      pushToast('error', `Failed to open file: ${String(e)}`)
    }
  }

  function formatDate(dateStr?: string): string {
    if (!dateStr) return ''
    try {
      const d = new Date(dateStr)
      return d.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
    } catch {
      return dateStr
    }
  }

  function formatSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  }
</script>

{#if hasMemory}
  <Card class="p-4">
    <div class="flex items-center gap-2 mb-3">
      <MessageSquare class="w-4 h-4 text-muted-foreground" />
      <h2 class="text-sm font-semibold text-muted-foreground uppercase">Memory</h2>
    </div>

    {#if memSyncError && mp?.error}
      <div class="flex items-start gap-2 rounded-md border border-destructive/50 bg-destructive/10 p-2 text-sm text-destructive mb-3">
        <AlertCircle class="w-4 h-4 shrink-0 mt-0.5" />
        {mp.error}
      </div>
    {/if}

    {#if memSyncRunning && mp}
      <div class="mb-3">
        <div class="flex items-center gap-2 mb-1">
          <Loader2 class="w-3 h-3 animate-spin text-warning" />
          <span class="text-xs text-muted-foreground">Syncing dialogues</span>
          {#if mp.total_files > 0}
            <span class="text-xs text-muted-foreground ml-auto">{mp.done_files.toLocaleString()} / {mp.total_files.toLocaleString()}</span>
          {/if}
        </div>
        {#if mp.total_files > 0}
          <div class="w-full bg-secondary rounded-full h-1.5 overflow-hidden">
            <div
              class="h-full rounded-full bg-warning transition-all duration-500"
              style="width: {memFilePct}%"
            ></div>
          </div>
        {/if}
      </div>
    {/if}

    <div class="space-y-2">
      {#each memStats as stat}
        <div class="flex items-center justify-between rounded-md border border-border bg-card/50 p-2.5">
          <div class="flex items-center gap-2">
            <span
              class="inline-block w-2 h-2 rounded-full {stat.enabled ? 'bg-success' : 'bg-muted-foreground/30'}"
            ></span>
            <span class="text-sm font-medium text-foreground capitalize">{stat.name}</span>
            {#if !stat.enabled}
              <Badge variant="outline">disabled</Badge>
            {/if}
          </div>

          {#if stat.enabled}
            <div class="flex items-center gap-3 text-xs text-muted-foreground">
              <span class="flex items-center gap-1">
                <FileText class="w-3 h-3" />
                {stat.files.toLocaleString()} files
              </span>
              <span class="flex items-center gap-1">
                <Hash class="w-3 h-3" />
                {stat.chunks.toLocaleString()} chunks
              </span>
            </div>
          {/if}
        </div>
      {/each}
    </div>

    {#if totalFiles > 0}
      <button
        onclick={toggleExpand}
        class="flex items-center gap-1.5 mt-3 text-xs text-muted-foreground hover:text-foreground transition-colors"
      >
        {#if expanded}
          <ChevronDown class="w-3.5 h-3.5" />
        {:else}
          <ChevronRight class="w-3.5 h-3.5" />
        {/if}
        {expanded ? 'Hide files' : `Show ${totalFiles} files`}
      </button>

      {#if expanded}
        {#if loadingFiles}
          <div class="mt-2 text-xs text-muted-foreground animate-pulse">Loading files...</div>
        {:else if memoryFiles.length === 0}
          <div class="mt-2 text-xs text-muted-foreground">No files found.</div>
        {:else}
          <div class="mt-2 space-y-1.5 max-h-96 overflow-y-auto">
            {#each memoryFiles as file}
              <div
                class="flex items-center justify-between rounded-md border border-border bg-card/30 px-2.5 py-2 hover:border-accent/50 transition-colors"
              >
                <div class="flex items-center gap-2 min-w-0 flex-1">
                  <FileText class="w-3.5 h-3.5 text-muted-foreground shrink-0" />
                  <div class="min-w-0 flex-1">
                    <div class="flex items-center gap-1.5">
                      <span class="text-sm font-medium text-foreground truncate">{file.title}</span>
                      <Badge variant={file.type === 'chat' ? 'default' : 'secondary'}>
                        {file.type}
                      </Badge>
                    </div>
                    <div class="flex items-center gap-2 text-xs text-muted-foreground mt-0.5">
                      <span class="capitalize">{file.provider}</span>
                      {#if file.date}
                        <span>·</span>
                        <span>{formatDate(file.date)}</span>
                      {/if}
                      <span>·</span>
                      <span>{formatSize(file.size)}</span>
                    </div>
                  </div>
                </div>
                <button
                  onclick={() => handleOpen(file.path)}
                  class="flex items-center gap-1 text-xs text-muted-foreground hover:text-primary transition-colors shrink-0 ml-2"
                  title="Open file"
                >
                  <ExternalLink class="w-3.5 h-3.5" />
                </button>
              </div>
            {/each}
          </div>
        {/if}
      {/if}
    {/if}
  </Card>
{/if}
