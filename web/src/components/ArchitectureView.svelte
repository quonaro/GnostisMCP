<script lang="ts">
  import { onMount } from 'svelte'
  import { architectureData, architectureLoading, selectedProject, status, pushToast } from '../lib/stores'
  import { getArchitecture } from '../lib/api'
  import Card from './ui/Card.svelte'
  import { Building2, Loader2, FileCode, Box, ArrowLeftRight, Zap, DoorOpen } from '@lucide/svelte'

  let s = $derived($status)
  let project = $derived($selectedProject || s?.projects?.[0] || '')
  let isLoading = $derived($architectureLoading)
  let arch = $derived($architectureData)

  async function load() {
    if (!project) return
    architectureLoading.set(true)
    try {
      const data = await getArchitecture(project)
      architectureData.set(data)
    } catch (e) {
      pushToast('error', `Architecture load failed: ${String(e)}`)
    } finally {
      architectureLoading.set(false)
    }
  }

  onMount(load)
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <Building2 class="w-5 h-5 text-primary" />
      <h2 class="text-lg font-semibold">Architecture</h2>
      {#if arch}
        <span class="text-sm text-muted-foreground">{arch.project}</span>
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
  {:else if arch}
    <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
      <Card class="p-4">
        <div class="flex items-center gap-2 mb-2">
          <FileCode class="w-4 h-4 text-muted-foreground" />
          <span class="text-xs text-muted-foreground uppercase">Files</span>
        </div>
        <div class="text-2xl font-bold text-primary">{arch.total_files}</div>
      </Card>
      <Card class="p-4">
        <div class="flex items-center gap-2 mb-2">
          <Box class="w-4 h-4 text-muted-foreground" />
          <span class="text-xs text-muted-foreground uppercase">Symbols</span>
        </div>
        <div class="text-2xl font-bold text-success">{arch.total_symbols}</div>
      </Card>
      <Card class="p-4">
        <div class="flex items-center gap-2 mb-2">
          <ArrowLeftRight class="w-4 h-4 text-muted-foreground" />
          <span class="text-xs text-muted-foreground uppercase">Edges</span>
        </div>
        <div class="text-2xl font-bold text-foreground">{arch.total_edges}</div>
      </Card>
      <Card class="p-4">
        <div class="flex items-center gap-2 mb-2">
          <DoorOpen class="w-4 h-4 text-muted-foreground" />
          <span class="text-xs text-muted-foreground uppercase">Entry Points</span>
        </div>
        <div class="text-2xl font-bold text-warning">{arch.entry_points.length}</div>
      </Card>
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
      <Card class="p-4">
        <h3 class="text-sm font-semibold mb-3">Languages</h3>
        <div class="space-y-2">
          {#each Object.entries(arch.languages).sort((a, b) => b[1] - a[1]) as [lang, count]}
            <div class="flex items-center justify-between text-sm">
              <span class="text-foreground">{lang}</span>
              <span class="text-muted-foreground">{count} files</span>
            </div>
          {/each}
        </div>
      </Card>

      <Card class="p-4">
        <h3 class="text-sm font-semibold mb-3">Symbols by Kind</h3>
        <div class="space-y-2">
          {#each Object.entries(arch.symbols_by_kind).sort((a, b) => b[1] - a[1]) as [kind, count]}
            <div class="flex items-center justify-between text-sm">
              <span class="text-foreground">{kind}</span>
              <span class="text-muted-foreground">{count}</span>
            </div>
          {/each}
        </div>
      </Card>
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
      <Card class="p-4">
        <div class="flex items-center gap-2 mb-3">
          <Zap class="w-4 h-4 text-warning" />
          <h3 class="text-sm font-semibold">Hotspots</h3>
        </div>
        <div class="space-y-2">
          {#each arch.hotspots.slice(0, 10) as h}
            <div class="flex items-center justify-between text-sm">
              <div class="min-w-0">
                <div class="font-medium text-foreground truncate">{h.symbol}</div>
                <div class="text-xs text-muted-foreground truncate">{h.path}</div>
              </div>
              <div class="flex gap-2 text-xs shrink-0">
                <span class="text-primary">↓{h.incoming}</span>
                <span class="text-muted-foreground">↑{h.outgoing}</span>
              </div>
            </div>
          {/each}
        </div>
      </Card>

      <Card class="p-4">
        <h3 class="text-sm font-semibold mb-3">Packages</h3>
        <div class="space-y-2">
          {#each arch.packages.slice(0, 15) as pkg}
            <div class="flex items-center justify-between text-sm">
              <span class="text-foreground">{pkg.name}</span>
              <span class="text-muted-foreground">{pkg.files} files</span>
            </div>
          {/each}
        </div>
      </Card>
    </div>

    {#if arch.recently_changed && arch.recently_changed.length > 0}
      <Card class="p-4">
        <h3 class="text-sm font-semibold mb-3">Recently Changed</h3>
        <div class="space-y-1">
          {#each arch.recently_changed.slice(0, 20) as c}
            <div class="flex items-center gap-2 text-sm">
              <span class="inline-block w-2 h-2 rounded-full {c.status === 'new' ? 'bg-success' : c.status === 'modified' ? 'bg-warning' : 'bg-destructive'}"></span>
              <span class="text-foreground truncate">{c.path}</span>
              <span class="text-xs text-muted-foreground shrink-0">{c.status}</span>
            </div>
          {/each}
        </div>
      </Card>
    {/if}
  {:else}
    <div class="text-center py-12 text-muted-foreground">
      {#if project}No architecture data available{:else}Select a project first{/if}
    </div>
  {/if}
</div>
