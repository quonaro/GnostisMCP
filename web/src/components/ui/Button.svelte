<script lang="ts">
  import { cn } from '../../lib/utils'
  import type { Snippet } from 'svelte'

  type Variant = 'default' | 'destructive' | 'outline' | 'secondary' | 'ghost'
  type Size = 'default' | 'sm' | 'lg' | 'icon'

  let {
    variant = 'default',
    size = 'default',
    class: className = '',
    children,
    onclick,
    disabled = false,
    type = 'button',
  }: {
    variant?: Variant
    size?: Size
    class?: string
    children: Snippet
    onclick?: (e: MouseEvent) => void
    disabled?: boolean
    type?: 'button' | 'submit'
  } = $props()

  const variants: Record<Variant, string> = {
    default: 'bg-primary text-primary-foreground hover:bg-primary/90',
    destructive: 'bg-destructive text-destructive-foreground hover:bg-destructive/90',
    outline: 'border border-input bg-transparent hover:bg-secondary text-foreground',
    secondary: 'bg-secondary text-secondary-foreground hover:bg-secondary/80',
    ghost: 'hover:bg-secondary hover:text-foreground text-muted-foreground',
  }

  const sizes: Record<Size, string> = {
    default: 'h-9 px-4 py-2 text-sm',
    sm: 'h-8 rounded-md px-3 text-xs',
    lg: 'h-11 rounded-md px-8 text-sm',
    icon: 'h-9 w-9',
  }
</script>

<button
  {type}
  {onclick}
  {disabled}
  class={cn(
    'inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50 [&_svg]:size-4 [&_svg]:shrink-0',
    variants[variant],
    sizes[size],
    className,
  )}
>
  {@render children()}
</button>
