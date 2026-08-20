<script lang="ts">
  import { onMount } from 'svelte'
  import ProgressCard from './components/ProgressCard.svelte'
  import StatsGrid from './components/StatsGrid.svelte'
  import ProjectList from './components/ProjectList.svelte'
  import SearchOverlay from './components/SearchOverlay.svelte'
  import ProjectAddModal from './components/ProjectAddModal.svelte'
  import Toast from './components/ui/Toast.svelte'
  import { refreshStatus, initEventSource } from './lib/stores'
  import { Database } from '@lucide/svelte'

  let closeEs: (() => void) | null = null

  onMount(() => {
    refreshStatus()
    closeEs = initEventSource()
    const interval = setInterval(refreshStatus, 5000)

    return () => {
      clearInterval(interval)
      closeEs?.()
    }
  })
</script>

<div class="min-h-screen bg-background text-foreground">
  <header class="sticky top-0 z-40 border-b border-border bg-background/80 backdrop-blur-sm">
    <div class="px-4 md:px-6 h-14 flex items-center gap-2">
      <div class="flex items-center gap-2 shrink-0 w-[200px]">
        <div class="w-8 h-8 rounded-lg bg-primary/15 flex items-center justify-center">
          <Database class="w-5 h-5 text-primary" />
        </div>
        <div class="hidden sm:block">
          <span class="font-semibold text-foreground">Gnostis</span>
          <span class="text-xs text-muted-foreground ml-2">Code Index</span>
        </div>
      </div>

      <div class="flex-1 flex justify-center">
        <SearchOverlay />
      </div>

      <div class="w-[200px] shrink-0"></div>
    </div>
  </header>

  <main class="p-4 md:p-6 space-y-4">
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
      <div class="lg:col-span-2">
        <ProjectList />
      </div>
      <div class="lg:col-span-1 space-y-4">
        <ProgressCard />
        <StatsGrid />
      </div>
    </div>
  </main>

  <ProjectAddModal />
  <Toast />
</div>
