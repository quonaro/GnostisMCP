<script lang="ts">
  import { cn } from '../../lib/utils'
  import type { Snippet } from 'svelte'

  type Variant = 'default' | 'secondary' | 'destructive' | 'success' | 'warning' | 'outline'

  let {
    variant = 'default',
    class: className = '',
    children,
  }: {
    variant?: Variant
    class?: string
    children: Snippet
  } = $props()

  const variants: Record<Variant, string> = {
    default: 'border-transparent bg-primary/15 text-primary',
    secondary: 'border-transparent bg-secondary text-secondary-foreground',
    destructive: 'border-transparent bg-destructive/15 text-destructive',
    success: 'border-transparent bg-success/15 text-success',
    warning: 'border-transparent bg-warning/15 text-warning',
    outline: 'text-muted-foreground border-border',
  }
</script>

<span
  class={cn(
    'inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-medium transition-colors',
    variants[variant],
    className,
  )}
>
  {@render children()}
</span>
