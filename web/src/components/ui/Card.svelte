<script lang="ts">
  import { cn } from '../../lib/utils'
  import type { Snippet } from 'svelte'

  let {
    class: className = '',
    children,
    onclick,
    onkeydown,
  }: {
    class?: string
    children: Snippet
    onclick?: () => void
    onkeydown?: (e: KeyboardEvent) => void
  } = $props()

  let clickable = $derived(!!onclick)

  function handleKeydown(e: KeyboardEvent) {
    if (clickable && (e.key === 'Enter' || e.key === ' ')) {
      e.preventDefault()
      onclick?.()
    }
    onkeydown?.(e)
  }
</script>

<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
<div
  {onclick}
  onkeydown={clickable ? handleKeydown : onkeydown}
  class={cn(
    'rounded-lg border border-border bg-card text-card-foreground shadow-sm',
    clickable && 'cursor-pointer hover:border-accent/50 transition-colors',
    className,
  )}
  role={clickable ? 'button' : undefined}
  tabindex={clickable ? 0 : undefined}
>
  {@render children()}
</div>
