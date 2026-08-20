<script lang="ts">
  import { search, type SearchResult } from '../lib/api'
  import { searchResults, pushToast } from '../lib/stores'
  import Card from './ui/Card.svelte'
  import Button from './ui/Button.svelte'
  import Badge from './ui/Badge.svelte'
  import Input from './ui/Input.svelte'
  import { Search as SearchIcon, FileCode, Hash } from '@lucide/svelte'

  let query = $state('')
  let topK = $state(10)
  let busy = $state(false)
  let err = $state<string | null>(null)
  let results = $state<SearchResult[]>([])

  async function handleSearch() {
    if (!query.trim()) return
    busy = true
    err = null
    try {
      const resp = await search(query.trim(), topK)
      results = resp.results
      searchResults.set(resp.results)
    } catch (e) {
      err = String(e)
      results = []
      pushToast('error', String(e))
    } finally {
      busy = false
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') handleSearch()
  }

  function scoreVariant(score: number): 'success' | 'warning' | 'outline' {
    if (score > 0.8) return 'success'
    if (score > 0.6) return 'warning'
    return 'outline'
  }
</script>

<Card class="p-5 h-full flex flex-col">
  <div class="flex gap-2 mb-4">
    <div class="relative flex-1">
      <SearchIcon class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
      <Input
        bind:value={query}
        onkeydown={handleKeydown}
        placeholder="Search indexed code..."
        class="pl-9"
      />
    </div>
    <select
      bind:value={topK}
      class="h-9 text-sm px-3 rounded-md border border-input bg-background text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
    >
      <option value={5}>5</option>
      <option value={10}>10</option>
      <option value={20}>20</option>
      <option value={50}>50</option>
    </select>
    <Button onclick={handleSearch} disabled={busy || !query.trim()}>
      <SearchIcon class="w-4 h-4" />
      {busy ? 'Searching...' : 'Search'}
    </Button>
  </div>

  {#if err}
    <div class="text-sm text-destructive mb-3">{err}</div>
  {/if}

  {#if results.length > 0}
    <div class="space-y-2 max-h-[60vh] overflow-y-auto pr-1">
      {#each results as r (r.id)}
        <div class="rounded-md border border-border bg-secondary/50 p-3 hover:border-primary/30 transition-colors">
          <div class="flex items-center justify-between mb-1.5">
            <span class="text-xs font-mono text-muted-foreground truncate flex items-center gap-1">
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
  {:else if !busy && query}
    <div class="text-sm text-muted-foreground py-8 text-center">No results found.</div>
  {:else if !busy && !query}
    <div class="text-sm text-muted-foreground py-8 text-center">Enter a query to search across all indexed code.</div>
  {/if}
</Card>
