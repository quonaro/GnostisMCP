<script lang="ts">
  import { status, loading, error } from '../lib/stores'
  import Card from './ui/Card.svelte'
  import { Cpu, Box, Hash, Sigma, AlertCircle } from '@lucide/svelte'

  let s = $derived($status)
</script>

{#if $error}
  <div class="flex items-center gap-2 rounded-lg border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive">
    <AlertCircle class="w-4 h-4 shrink-0" />
    {$error}
  </div>
{/if}

{#if s}
  <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
    <Card class="p-4">
      <div class="flex items-center gap-2 mb-2">
        <Cpu class="w-4 h-4 text-muted-foreground" />
        <span class="text-xs text-muted-foreground uppercase">Provider</span>
      </div>
      <div class="text-base font-semibold text-foreground">{s.provider}</div>
    </Card>
    <Card class="p-4">
      <div class="flex items-center gap-2 mb-2">
        <Box class="w-4 h-4 text-muted-foreground" />
        <span class="text-xs text-muted-foreground uppercase">Model</span>
      </div>
      <div class="text-base font-semibold text-foreground truncate">{s.model}</div>
    </Card>
    <Card class="p-4">
      <div class="flex items-center gap-2 mb-2">
        <Hash class="w-4 h-4 text-muted-foreground" />
        <span class="text-xs text-muted-foreground uppercase">Chunks</span>
      </div>
      <div class="text-base font-semibold text-primary">{s.total_chunks.toLocaleString()}</div>
    </Card>
    <Card class="p-4">
      <div class="flex items-center gap-2 mb-2">
        <Sigma class="w-4 h-4 text-muted-foreground" />
        <span class="text-xs text-muted-foreground uppercase">Symbols</span>
      </div>
      <div class="text-base font-semibold text-success">{s.symbols.toLocaleString()}</div>
    </Card>
  </div>
{:else if $loading}
  <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
    {#each Array(4) as _}
      <Card class="p-4 animate-pulse">
        <div class="h-3 w-16 bg-secondary rounded mb-2"></div>
        <div class="h-5 w-24 bg-secondary rounded"></div>
      </Card>
    {/each}
  </div>
{/if}
