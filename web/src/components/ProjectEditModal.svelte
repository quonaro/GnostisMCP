<script lang="ts">
  import { editProject } from '../lib/api'
  import { refreshStatus, pushToast } from '../lib/stores'
  import Button from './ui/Button.svelte'
  import Input from './ui/Input.svelte'
  import { X } from '@lucide/svelte'

  let {
    projectName,
    onClose,
  }: { projectName: string; onClose: () => void } = $props()

  let extensions = $state('')
  let include = $state('')
  let exclude = $state('')
  let maxFileSizeMB = $state(0)
  let busy = $state(false)

  function parsePatterns(text: string): string[] {
    return text.split(',').map(s => s.trim()).filter(Boolean)
  }

  async function handleSubmit() {
    busy = true
    try {
      await editProject(projectName, {
        extensions: parsePatterns(extensions),
        include: parsePatterns(include),
        exclude: parsePatterns(exclude),
        max_file_size_mb: maxFileSizeMB || undefined,
      })
      pushToast('success', `Project "${projectName}" updated`)
      onClose()
      await refreshStatus()
    } catch (e) {
      pushToast('error', String(e))
    } finally {
      busy = false
    }
  }

  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && !busy) onClose()
  }
</script>

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
<div
  class="fixed inset-0 z-[60] flex items-center justify-center bg-black/50 backdrop-blur-sm p-4"
  onclick={onClose}
  onkeydown={onKeydown}
  role="presentation"
>
  <!-- svelte-ignore a11y_no_static_element_interactions, a11y_click_events_have_key_events -->
  <div
    class="w-full max-w-md rounded-lg border border-border bg-card shadow-lg p-5"
    onclick={(e) => e.stopPropagation()}
    onkeydown={onKeydown}
    role="presentation"
  >
    <div class="flex items-center justify-between mb-4">
      <h2 class="text-sm font-semibold text-foreground">Edit Project — {projectName}</h2>
      <button
        onclick={onClose}
        disabled={busy}
        class="text-muted-foreground hover:text-foreground transition-colors disabled:opacity-50"
        aria-label="Close"
      >
        <X class="w-4 h-4" />
      </button>
    </div>

    <div class="space-y-3">
      <div>
        <label for="edit-extensions" class="text-xs text-muted-foreground block mb-1.5">Extensions (comma-separated, empty = defaults)</label>
        <Input
          id="edit-extensions"
          bind:value={extensions}
          placeholder=".go, .py, .ts"
        />
      </div>
      <div>
        <label for="edit-include" class="text-xs text-muted-foreground block mb-1.5">Include patterns (comma-separated)</label>
        <Input
          id="edit-include"
          bind:value={include}
          placeholder="src/**, lib/**"
        />
      </div>
      <div>
        <label for="edit-exclude" class="text-xs text-muted-foreground block mb-1.5">Exclude patterns (comma-separated)</label>
        <Input
          id="edit-exclude"
          bind:value={exclude}
          placeholder="vendor/**, **/*_test.go"
        />
      </div>
      <div>
        <label for="edit-maxsize" class="text-xs text-muted-foreground block mb-1.5">Max file size MB (0 = default)</label>
        <Input
          id="edit-maxsize"
          type="number"
          bind:value={maxFileSizeMB}
          placeholder="5"
        />
      </div>
      <div class="flex gap-2 pt-1">
        <Button
          variant="outline"
          onclick={onClose}
          disabled={busy}
          class="flex-1"
        >
          Cancel
        </Button>
        <Button
          variant="default"
          onclick={handleSubmit}
          disabled={busy}
          class="flex-1"
        >
          {busy ? 'Saving...' : 'Save'}
        </Button>
      </div>
    </div>
  </div>
</div>
