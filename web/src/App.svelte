<script lang="ts">
  import { onMount } from 'svelte'
  import Sidebar from './components/Sidebar.svelte'
  import ProjectList from './components/ProjectList.svelte'
  import MemoryCard from './components/MemoryCard.svelte'
  import SearchOverlay from './components/SearchOverlay.svelte'
  import ProjectAddModal from './components/ProjectAddModal.svelte'
  import Toast from './components/ui/Toast.svelte'
  import JobQueue from './components/JobQueue.svelte'
  import SystemMetricsCard from './components/SystemMetricsCard.svelte'
  import GraphView from './components/GraphView.svelte'
  import ArchitectureView from './components/ArchitectureView.svelte'
  import DeadCodeView from './components/DeadCodeView.svelte'
  import ChangesView from './components/ChangesView.svelte'
  import { LayoutGrid, Cpu, Box, Hash, Sigma } from '@lucide/svelte'
  import { refreshStatus, initEventSource, activeSection, status } from './lib/stores'

  let closeEs: (() => void) | null = null
  let section = $derived($activeSection)
  let s = $derived($status)

  onMount(() => {
    refreshStatus()
    closeEs = initEventSource()

    return () => {
      closeEs?.()
    }
  })
</script>

<div class="min-h-screen bg-background text-foreground flex">
  <Sidebar />

  <div class="flex-1 min-w-0 flex flex-col">
    <main class="flex-1 p-4 md:p-6 space-y-4">
      {#if section === 'overview'}
        <div class="flex items-center gap-2">
          <LayoutGrid class="w-5 h-5 text-primary" />
          <h2 class="text-lg font-semibold">Overview</h2>
        </div>
        {#if s}
          <div class="flex items-center gap-4 text-xs text-muted-foreground flex-wrap">
            <span class="flex items-center gap-1.5">
              <Cpu class="w-3.5 h-3.5" />
              <span class="font-medium text-foreground">{s.provider}</span>
            </span>
            <span class="flex items-center gap-1.5">
              <Box class="w-3.5 h-3.5" />
              <span class="font-medium text-foreground truncate max-w-[12rem]">{s.model}</span>
            </span>
            <span class="flex items-center gap-1.5">
              <Hash class="w-3.5 h-3.5" />
              <span class="font-medium text-primary">{s.total_chunks.toLocaleString()}</span> chunks
            </span>
            <span class="flex items-center gap-1.5">
              <Sigma class="w-3.5 h-3.5" />
              <span class="font-medium text-success">{s.symbols.toLocaleString()}</span> symbols
            </span>
          </div>
        {/if}
        <div class="grid grid-cols-1 lg:grid-cols-3 gap-4 items-stretch">
          <div class="lg:col-span-2 flex">
            <ProjectList />
          </div>
          <div class="lg:col-span-1 space-y-4">
            <JobQueue />
            <MemoryCard />
            <SystemMetricsCard />
          </div>
        </div>
      {:else if section === 'graph'}
        <GraphView />
      {:else if section === 'architecture'}
        <ArchitectureView />
      {:else if section === 'dead-code'}
        <DeadCodeView />
      {:else if section === 'changes'}
        <ChangesView />
      {/if}
    </main>
  </div>

  <ProjectAddModal />
  <SearchOverlay />
  <Toast />
</div>
