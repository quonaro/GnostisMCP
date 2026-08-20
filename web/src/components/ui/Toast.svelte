<script lang="ts">
  import { toasts, dismissToast } from '../../lib/stores'
  import { CheckCircle2, XCircle, Info, X } from '@lucide/svelte'

  let items = $derived($toasts)
</script>

{#if items.length > 0}
  <div class="fixed bottom-4 right-4 z-50 flex flex-col gap-2 w-80">
    {#each items as t (t.id)}
      <div
        class="flex items-start gap-3 rounded-lg border border-border bg-card p-4 shadow-lg animate-in slide-in-from-right"
      >
        {#if t.type === 'success'}
          <CheckCircle2 class="w-5 h-5 text-success shrink-0 mt-0.5" />
        {:else if t.type === 'error'}
          <XCircle class="w-5 h-5 text-destructive shrink-0 mt-0.5" />
        {:else}
          <Info class="w-5 h-5 text-primary shrink-0 mt-0.5" />
        {/if}
        <div class="flex-1 text-sm text-foreground">{t.message}</div>
        <button
          onclick={() => dismissToast(t.id)}
          class="text-muted-foreground hover:text-foreground transition-colors shrink-0"
        >
          <X class="w-4 h-4" />
        </button>
      </div>
    {/each}
  </div>
{/if}
