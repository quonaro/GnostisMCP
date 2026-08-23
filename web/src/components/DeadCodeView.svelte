<script lang="ts">
  import { onMount } from 'svelte'
  import { deadCodeData, deadCodeLoading, selectedProject, status, pushToast } from '../lib/stores'
  import { getDeadCode } from '../lib/api'
  import Card from './ui/Card.svelte'
  import { Trash2, Loader2 } from '@lucide/svelte'

  let s = $derived($status)
  let project = $derived($selectedProject || s?.projects?.[0] || '')
  let isLoading = $derived($deadCodeLoading)
  let data = $derived($deadCodeData)
  let kind = $state('')

  async function load() {
    if (!project) return
    deadCodeLoading.set(true)
    try {
      const result = await getDeadCode(project, kind || undefined, 100)
      deadCodeData.set(result)
    } catch (e) {
      pushToast('error', `Dead code load failed: ${String(e)}`)
    } finally {
      deadCodeLoading.set(false)
    }
  }

  onMount(load)
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <Trash2 class="w-5 h-5 text-primary" />
      <h2 class="text-lg font-semibold">Dead Code</h2>
      {#if data}
        <span class="text-sm text-muted-foreground">{data.count} candidates</span>
      {/if}
    </div>
    <div class="flex items-center gap-2">
      <select
        bind:value={kind}
        onchange={load}
        class="px-3 py-1.5 text-sm rounded-md border border-input bg-card text-foreground"
      >
        <option value="">All kinds</option>
        <option value="function">Functions</option>
        <option value="method">Methods</option>
      </select>
      <button
        onclick={load}
        class="px-3 py-1.5 text-sm rounded-md bg-primary/10 text-primary hover:bg-primary/20 transition-colors"
        disabled={isLoading}
      >
        Refresh
      </button>
    </div>
  </div>

  {#if isLoading}
    <div class="flex items-center justify-center py-12">
      <Loader2 class="w-6 h-6 animate-spin text-primary" />
    </div>
  {:else if data && data.candidates.length > 0}
    <Card class="p-0 overflow-hidden">
      <table class="w-full text-sm">
        <thead class="border-b border-border bg-secondary/50">
          <tr>
            <th class="text-left px-4 py-2 font-medium text-muted-foreground">Symbol</th>
            <th class="text-left px-4 py-2 font-medium text-muted-foreground">Kind</th>
            <th class="text-left px-4 py-2 font-medium text-muted-foreground">Path</th>
            <th class="text-right px-4 py-2 font-medium text-muted-foreground">Lines</th>
          </tr>
        </thead>
        <tbody>
          {#each data.candidates as c}
            <tr class="border-b border-border/50 hover:bg-secondary/30">
              <td class="px-4 py-2 font-medium text-foreground">{c.symbol}</td>
              <td class="px-4 py-2 text-muted-foreground">{c.kind}</td>
              <td class="px-4 py-2 text-muted-foreground truncate max-w-xs">{c.path}</td>
              <td class="px-4 py-2 text-right text-muted-foreground">{c.start_line}–{c.end_line}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </Card>
  {:else}
    <div class="text-center py-12 text-muted-foreground">
      {#if project}No dead code candidates found{:else}Select a project first{/if}
    </div>
  {/if}
</div>
