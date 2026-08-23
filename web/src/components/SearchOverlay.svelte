<script lang="ts">
  import { search, type SearchResult } from '../lib/api'
  import { searchOpen, pushToast } from '../lib/stores'
  import Badge from './ui/Badge.svelte'
  import { Search as SearchIcon, FileCode, Hash, X, Loader2 } from '@lucide/svelte'

  let query = $state('')
  let topK = $state(20)
  let busy = $state(false)
  let err = $state<string | null>(null)
  let results = $state<SearchResult[]>([])
  let inputEl = $state<HTMLInputElement | null>(null)
  let debounceTimer: ReturnType<typeof setTimeout> | null = null

  let open = $derived($searchOpen)
  let hasQuery = $derived(query.trim().length > 0)

  async function handleSearch() {
    if (!hasQuery) {
      results = []
      return
    }
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
    if (debounceTimer) clearTimeout(debounceTimer)
    debounceTimer = setTimeout(handleSearch, 300)
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      close()
    }
  }

  function close() {
    searchOpen.set(false)
  }

  function scoreVariant(score: number): 'success' | 'warning' | 'outline' {
    if (score > 0.8) return 'success'
    if (score > 0.6) return 'warning'
    return 'outline'
  }

  function onBackdropClick(e: MouseEvent) {
    if (e.target === e.currentTarget) close()
  }

  function portal(node: HTMLElement) {
    document.body.appendChild(node)
    return { destroy: () => node.remove() }
  }

  $effect(() => {
    if (open) {
      query = ''
      results = []
      err = null
      queueMicrotask(() => inputEl?.focus())
    }
  })

  $effect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        searchOpen.set(true)
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  })
</script>

{#if open}
  <!-- Backdrop -->
  <div
    use:portal
    class="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm flex items-start justify-center pt-[10vh] p-4"
    onclick={onBackdropClick}
    onkeydown={handleKeydown}
    role="button"
    tabindex={-1}
  >
    <!-- Dialog -->
    <div class="flex max-h-[80vh] w-full max-w-2xl flex-col overflow-hidden rounded-2xl border border-border bg-card shadow-2xl">
      <!-- Search input -->
      <div class="flex items-center gap-3 border-b border-border p-4">
        <SearchIcon class="w-5 h-5 shrink-0 text-muted-foreground" />
        <input
          bind:this={inputEl}
          bind:value={query}
          oninput={onInput}
          onkeydown={handleKeydown}
          type="text"
          placeholder="Search indexed code..."
          class="flex-1 bg-transparent text-base text-foreground outline-none placeholder:text-muted-foreground"
        />
        {#if busy}
          <Loader2 class="w-5 h-5 shrink-0 animate-spin text-muted-foreground" />
        {/if}
        <button
          onclick={close}
          class="rounded-lg p-1.5 text-muted-foreground transition hover:bg-secondary hover:text-foreground"
          aria-label="Close"
        >
          <X class="w-4 h-4" />
        </button>
      </div>

      <!-- Results -->
      <div class="flex-1 overflow-y-auto p-2">
        {#if err}
          <div class="px-4 py-3 text-sm text-destructive">{err}</div>
        {:else if results.length > 0}
          <div class="px-3 pb-1 pt-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            {results.length} result{results.length === 1 ? '' : 's'}
          </div>
          {#each results as r (r.id)}
            <div class="rounded-lg p-3 hover:bg-secondary/40 transition-colors cursor-pointer">
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
        {:else if busy}
          <div class="flex items-center justify-center py-12">
            <Loader2 class="w-6 h-6 animate-spin text-muted-foreground" />
          </div>
        {:else if hasQuery}
          <div class="flex items-center justify-center py-12 text-sm text-muted-foreground">No results found.</div>
        {:else}
          <div class="flex items-center justify-center py-12 text-sm text-muted-foreground">Enter a query to search across all indexed code.</div>
        {/if}
      </div>

      <!-- Footer with top-k selector -->
      <div class="flex items-center justify-between border-t border-border px-4 py-2.5">
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
          <span class="text-xs text-muted-foreground">results</span>
        </div>
        <div class="flex items-center gap-2 text-xs text-muted-foreground">
          <kbd class="rounded border border-border px-1.5 py-0.5">Esc</kbd>
          <span>to close</span>
        </div>
      </div>
    </div>
  </div>
{/if}
