<script lang="ts">
  import { activeSection, searchOpen, type Section } from '../lib/stores'
  import { LayoutGrid, Network, Building2, Trash2, GitBranch, Search as SearchIcon } from '@lucide/svelte'
  import type { Snippet } from 'svelte'

  let section = $derived($activeSection)

  const items: { id: Section; label: string; icon: typeof LayoutGrid }[] = [
    { id: 'overview', label: 'Overview', icon: LayoutGrid },
    { id: 'graph', label: 'Graph', icon: Network },
    { id: 'architecture', label: 'Architecture', icon: Building2 },
    { id: 'dead-code', label: 'Dead Code', icon: Trash2 },
    { id: 'changes', label: 'Changes', icon: GitBranch },
  ]

  function select(id: Section) {
    activeSection.set(id)
  }
</script>

<aside class="hidden md:flex w-64 h-screen sticky top-0 bg-card border-r border-border flex-col shrink-0">
  <div class="p-4 border-b border-border shrink-0">
    <div class="flex items-center gap-2">
      <img src="/gnostis-logo.svg" alt="Gnostis" class="w-8 h-8 rounded-lg" />
      <div class="font-bold text-2xl text-foreground tracking-wide">GNOSTIS</div>
    </div>
  </div>

  <div class="px-3 pt-2">
    <button
      onclick={() => searchOpen.set(true)}
      class="w-full flex items-center gap-3 px-3 py-2 rounded-md text-sm font-medium text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
    >
      <SearchIcon class="w-4 h-4" />
      <span>Search</span>
      <kbd class="ml-auto text-[10px] rounded border border-border px-1.5 py-0.5 text-muted-foreground/70">⌘K</kbd>
    </button>
  </div>

  <nav class="flex-1 p-3 space-y-1">
    {#each items as item (item.id)}
      <button
        onclick={() => select(item.id)}
        class="w-full flex items-center gap-3 px-3 py-2 rounded-md text-sm font-medium transition-colors {section === item.id ? 'bg-primary/10 text-primary' : 'text-muted-foreground hover:bg-secondary hover:text-foreground'}"
      >
        <item.icon class="w-4 h-4" />
        <span>{item.label}</span>
      </button>
    {/each}
  </nav>

</aside>
