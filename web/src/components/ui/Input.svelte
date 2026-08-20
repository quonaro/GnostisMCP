<script lang="ts">
  import { cn } from '../../lib/utils'

  let {
    value = $bindable(''),
    type = 'text',
    placeholder = '',
    class: className = '',
    onkeydown,
    oninput,
    id,
    ...rest
  }: {
    value?: string
    type?: string
    placeholder?: string
    class?: string
    onkeydown?: (e: KeyboardEvent) => void
    oninput?: (e: Event) => void
    id?: string
    [key: string]: unknown
  } = $props()
</script>

<input
  {type}
  bind:value
  {placeholder}
  {onkeydown}
  {oninput}
  {id}
  {...rest}
  class={cn(
    'flex h-9 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50',
    className,
  )}
/>
