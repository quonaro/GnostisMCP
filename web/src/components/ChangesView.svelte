<script lang="ts">
  import { onMount } from 'svelte'
  import { changesData, changesLoading, selectedProject, status, pushToast } from '../lib/stores'
  import { getChanges } from '../lib/api'
  import Card from './ui/Card.svelte'
  import { GitBranch, Loader2 } from '@lucide/svelte'

  let s = $derived($status)
  let project = $derived($selectedProject || s?.projects?.[0] || '')
  let isLoading = $derived($changesLoading)
  let data = $derived($changesData)

  const statusColor: Record<string, string> = {
    new: 'bg-success',
    modified: 'bg-warning',
    deleted: 'bg-destructive',
  }

  async function load() {
    if (!project) return
    changesLoading.set(true)
    try {
      const result = await getChanges(project)
      changesData.set(result)
    } catch (e) {
      pushToast('error', `Changes load failed: ${String(e)}`)
    } finally {
      changesLoading.set(false)
    }
  }

  onMount(load)
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <GitBranch class="w-5 h-5 text-primary" />
      <h2 class="text-lg font-semibold">Changes</h2>
      {#if data}
        <span class="text-sm text-muted-foreground">{data.count} files</span>
      {/if}
    </div>
    <button
      onclick={load}
      class="px-3 py-1.5 text-sm rounded-md bg-primary/10 text-primary hover:bg-primary/20 transition-colors"
      disabled={isLoading}
    >
      Refresh
    </button>
  </div>

  {#if isLoading}
    <div class="flex items-center justify-center py-12">
      <Loader2 class="w-6 h-6 animate-spin text-primary" />
    </div>
  {:else if data && data.changes.length > 0}
    <Card class="p-4">
      <div class="space-y-1">
        {#each data.changes as c}
          <div class="flex items-center gap-3 py-1.5 text-sm">
            <span class="inline-block w-2 h-2 rounded-full shrink-0 {statusColor[c.status] || 'bg-muted-foreground'}"></span>
            <span class="text-foreground truncate flex-1">{c.path}</span>
            <span class="text-xs text-muted-foreground shrink-0 uppercase">{c.status}</span>
          </div>
        {/each}
      </div>
    </Card>
  {:else}
    <div class="text-center py-12 text-muted-foreground">
      {#if project}No changes detected{:else}Select a project first{/if}
    </div>
  {/if}
</div>
