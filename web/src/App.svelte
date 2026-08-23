<script lang="ts">
  import { onMount } from 'svelte'
  import Sidebar from './components/Sidebar.svelte'
  import ProgressCard from './components/ProgressCard.svelte'
  import MemoryProgressCard from './components/MemoryProgressCard.svelte'
  import StatsGrid from './components/StatsGrid.svelte'
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
  import { LayoutGrid } from '@lucide/svelte'
  import { refreshStatus, initEventSource, activeSection } from './lib/stores'

  let closeEs: (() => void) | null = null
  let section = $derived($activeSection)

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
        <div class="grid grid-cols-1 lg:grid-cols-3 gap-4 items-stretch">
          <div class="lg:col-span-2 flex">
            <ProjectList />
          </div>
          <div class="lg:col-span-1 space-y-4">
            <ProgressCard />
            <JobQueue />
            <MemoryProgressCard />
            <SystemMetricsCard />
            <StatsGrid />
            <MemoryCard />
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
