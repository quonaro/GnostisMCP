<script lang="ts">
  import { status } from '../lib/stores'
  import Card from './ui/Card.svelte'
  import { Cpu, MemoryStick, Gpu, Thermometer } from '@lucide/svelte'

  let s = $derived($status)
  let sm = $derived(s?.sys_metrics)

  function formatBytes(bytes: number): string {
    if (bytes >= 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`
    if (bytes >= 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(0)} MB`
    return `${(bytes / 1024).toFixed(0)} KB`
  }

  function usageColor(pct: number): string {
    if (pct >= 90) return 'bg-destructive'
    if (pct >= 70) return 'bg-warning'
    return 'bg-primary'
  }
</script>

{#if sm}
  <Card class="p-4">
    <h2 class="text-sm font-semibold text-muted-foreground uppercase mb-3">System Resources</h2>

    <!-- CPU -->
    <div class="mb-3">
      <div class="flex items-center gap-2 mb-1">
        <Cpu class="w-3.5 h-3.5 text-muted-foreground" />
        <span class="text-xs text-muted-foreground truncate">{sm.cpu.name || 'CPU'}</span>
        <span class="text-xs text-muted-foreground ml-auto shrink-0">{sm.cpu.usage_percent.toFixed(1)}% · {sm.cpu.cores} cores</span>
      </div>
      <div class="w-full bg-secondary rounded-full h-2 overflow-hidden">
        <div
          class="h-full rounded-full transition-all duration-500 {usageColor(sm.cpu.usage_percent)}"
          style="width: {sm.cpu.usage_percent}%"
        ></div>
      </div>
    </div>

    <!-- RAM -->
    <div class="mb-3">
      <div class="flex items-center gap-2 mb-1">
        <MemoryStick class="w-3.5 h-3.5 text-muted-foreground" />
        <span class="text-xs text-muted-foreground truncate">{sm.memory.type || 'RAM'}</span>
        <span class="text-xs text-muted-foreground ml-auto">
          {formatBytes(sm.memory.used_bytes)} / {formatBytes(sm.memory.total_bytes)}
        </span>
      </div>
      <div class="w-full bg-secondary rounded-full h-2 overflow-hidden">
        <div
          class="h-full rounded-full transition-all duration-500 {usageColor(sm.memory.usage_percent)}"
          style="width: {sm.memory.usage_percent}%"
        ></div>
      </div>
    </div>

    <!-- GPUs -->
    {#if sm.gpus && sm.gpus.length > 0}
      {#each sm.gpus as gpu}
        <div class="mb-3 last:mb-0">
          <div class="flex items-center gap-2 mb-1">
            <Gpu class="w-3.5 h-3.5 text-muted-foreground" />
            <span class="text-xs text-muted-foreground truncate">{gpu.name}</span>
            <span class="text-xs text-muted-foreground ml-auto shrink-0">
              {gpu.utilization_percent.toFixed(0)}%
              · {formatBytes(gpu.memory_used_bytes)} / {formatBytes(gpu.memory_total_bytes)}
            </span>
          </div>
          <div class="w-full bg-secondary rounded-full h-2 overflow-hidden">
            <div
              class="h-full rounded-full transition-all duration-500 {usageColor(gpu.utilization_percent)}"
              style="width: {gpu.utilization_percent}%"
            ></div>
          </div>
          <div class="flex items-center gap-1 mt-1">
            <Thermometer class="w-3 h-3 text-muted-foreground" />
            <span class="text-xs text-muted-foreground">{gpu.temperature_c.toFixed(0)}°C</span>
          </div>
        </div>
      {/each}
    {/if}
  </Card>
{/if}
