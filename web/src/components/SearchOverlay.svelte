<script lang="ts">
  import { search, type SearchResult } from '../lib/api'
  import { pushToast } from '../lib/stores'
  import Badge from './ui/Badge.svelte'
  import { Search as SearchIcon, FileCode, Hash, X, Loader2 } from '@lucide/svelte'

  let query = $state('')
  let topK = $state(20)
  let busy = $state(false)
  let err = $state<string | null>(null)
  let results = $state<SearchResult[]>([])
  let active = $state(false)
  let inputEl: HTMLInputElement | null = null

  let hasQuery = $derived(query.trim().length > 0)

  async function handleSearch() {
    if (!hasQuery) return
    busy = true
    err = null
    try {
      const resp = await search(query.trim(), topK)
      results = resp.results
    } catch (e) {
      err = String(e)
      results = []
      pushToast('error', String(e))
    } finally {
      busy = false
    }
  }

  function onInput() {
    active = true
    if (!hasQuery) {
      results = []
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      handleSearch()
    } else if (e.key === 'Escape') {
      close()
    }
  }

  function close() {
    active = false
    inputEl?.blur()
  }

  function open() {
    active = true
  }

  function scoreVariant(score: number): 'success' | 'warning' | 'outline' {
    if (score > 0.8) return 'success'
    if (score > 0.6) return 'warning'
    return 'outline'
  }

  function onOverlayClick(e: MouseEvent) {
    if (e.target === e.currentTarget) close()
  }

  function portal(node: HTMLElement) {
    document.body.appendChild(node)
    return { destroy: () => node.remove() }
  }
</script>

<!-- Search input in header -->
<div class="relative flex-1 max-w-md mx-4">
  <SearchIcon class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
  <input
    bind:this={inputEl}
    bind:value={query}
    oninput={onInput}
    onkeydown={handleKeydown}
    onfocus={open}
    type="text"
    placeholder="Search indexed code..."
    class="w-full h-9 pl-9 pr-9 rounded-md border border-input bg-background text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
  />
  {#if active && results.length > 0}
    <button
      onclick={close}
      class="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
      aria-label="Close search"
    >
      <X class="w-4 h-4" />
    </button>
  {/if}
</div>

<!-- Backdrop -->
{#if active}
  <div
    use:portal
    class="fixed inset-0 z-30 bg-black/50 backdrop-blur-sm"
    onclick={onOverlayClick}
    onkeydown={(e) => e.key === 'Escape' && close()}
    role="button"
    tabindex={-1}
  ></div>
{/if}

<!-- Overlay with results -->
{#if active && (results.length > 0 || err || busy)}
  <!-- Results panel -->
  <div use:portal class="fixed top-14 left-0 right-0 z-40 mx-auto max-w-3xl px-4 pt-4">
    <div class="rounded-lg border border-border bg-card shadow-lg overflow-hidden">
      <!-- Header bar -->
      <div class="flex items-center justify-between px-4 py-2.5 border-b border-border bg-secondary/50">
        <div class="flex items-center gap-2 text-sm text-muted-foreground">
          {#if busy}
            <Loader2 class="w-4 h-4 animate-spin" />
            <span>Searching...</span>
          {:else}
            <span>{results.length} result{results.length === 1 ? '' : 's'} for "{query.trim()}"</span>
          {/if}
        </div>
        <div class="flex items-center gap-2">
          <select
            bind:value={topK}
            onchange={handleSearch}
            class="h-7 text-xs px-2 rounded-md border border-input bg-background text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            <option value={5}>5</option>
            <option value={10}>10</option>
            <option value={20}>20</option>
            <option value={50}>50</option>
          </select>
          <button
            onclick={close}
            class="text-muted-foreground hover:text-foreground transition-colors p-1"
            aria-label="Close"
          >
            <X class="w-4 h-4" />
          </button>
        </div>
      </div>

      {#if err}
        <div class="px-4 py-3 text-sm text-destructive">{err}</div>
      {:else if results.length > 0}
        <div class="max-h-[70vh] overflow-y-auto divide-y divide-border">
          {#each results as r (r.id)}
            <div class="p-3.5 hover:bg-secondary/40 transition-colors cursor-pointer">
              <div class="flex items-center justify-between mb-1.5">
                <span class="text-xs font-mono text-muted-foreground truncate flex items-center gap-1.5">
                  <FileCode class="w-3.5 h-3.5 shrink-0" />
                  {r.path}:{r.start_line}-{r.end_line}
                </span>
                <Badge variant={scoreVariant(r.score)} class="ml-2 shrink-0 font-mono">
                  {r.score.toFixed(3)}
                </Badge>
              </div>
              {#if r.symbol}
                <div class="flex items-center gap-1 text-xs text-primary mb-1.5">
                  <Hash class="w-3 h-3" />
                  {r.symbol}
                </div>
              {/if}
              <pre class="text-xs text-foreground/80 overflow-x-auto max-h-32 font-mono leading-relaxed">{r.content}</pre>
            </div>
          {/each}
        </div>
      {:else if !busy}
        <div class="px-4 py-8 text-center text-sm text-muted-foreground">No results found.</div>
      {/if}
    </div>
  </div>
{/if}
