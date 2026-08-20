<script lang="ts">
  import { status } from '../lib/stores'
  import { Cpu, Box, Hash, Sigma } from '@lucide/svelte'

  let s = $derived($status)
</script>

{#if s}
  <div class="grid grid-cols-2 gap-3">
    <div class="rounded-lg border border-border bg-card p-3 shadow-sm">
      <div class="flex items-center gap-2 mb-1.5">
        <Cpu class="w-3.5 h-3.5 text-muted-foreground" />
        <span class="text-xs text-muted-foreground uppercase">Provider</span>
      </div>
      <div class="text-sm font-semibold text-foreground">{s.provider}</div>
    </div>
    <div class="rounded-lg border border-border bg-card p-3 shadow-sm">
      <div class="flex items-center gap-2 mb-1.5">
        <Box class="w-3.5 h-3.5 text-muted-foreground" />
        <span class="text-xs text-muted-foreground uppercase">Model</span>
      </div>
      <div class="text-sm font-semibold text-foreground truncate">{s.model}</div>
    </div>
    <div class="rounded-lg border border-border bg-card p-3 shadow-sm">
      <div class="flex items-center gap-2 mb-1.5">
        <Hash class="w-3.5 h-3.5 text-muted-foreground" />
        <span class="text-xs text-muted-foreground uppercase">Chunks</span>
      </div>
      <div class="text-sm font-semibold text-primary">{s.total_chunks.toLocaleString()}</div>
    </div>
    <div class="rounded-lg border border-border bg-card p-3 shadow-sm">
      <div class="flex items-center gap-2 mb-1.5">
        <Sigma class="w-3.5 h-3.5 text-muted-foreground" />
        <span class="text-xs text-muted-foreground uppercase">Symbols</span>
      </div>
      <div class="text-sm font-semibold text-success">{s.symbols.toLocaleString()}</div>
    </div>
  </div>
{/if}
