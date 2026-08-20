<script lang="ts">
  import { addProject, pickDirectory } from '../lib/api'
  import { refreshStatus, pushToast, showAddModal } from '../lib/stores'
  import Button from './ui/Button.svelte'
  import Input from './ui/Input.svelte'
  import { Plus, FolderOpen, X } from '@lucide/svelte'

  let path = $state('')
  let name = $state('')
  let exclude = $state('')
  let busy = $state(false)
  let picking = $state(false)

  let open = $derived($showAddModal)

  function closeModal() {
    if (busy) return
    showAddModal.set(false)
  }

  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') closeModal()
  }

  async function handlePick() {
    picking = true
    try {
      const result = await pickDirectory()
      if (result.path) {
        path = result.path
        if (!name.trim()) {
          const parts = result.path.split('/').filter(Boolean)
          name = parts[parts.length - 1] ?? ''
        }
      }
    } catch (e) {
      pushToast('error', String(e))
    } finally {
      picking = false
    }
  }

  function parsePatterns(text: string): string[] {
    return text.split(',').map(s => s.trim()).filter(Boolean)
  }

  async function handleSubmit() {
    if (!path.trim()) return
    busy = true
    try {
      const result = await addProject(path.trim(), name.trim(), {
        exclude: parsePatterns(exclude),
      })
      pushToast('success', `Project "${result.name}" added`)
      path = ''
      name = ''
      exclude = ''
      showAddModal.set(false)
      await refreshStatus()
    } catch (e) {
      pushToast('error', String(e))
    } finally {
      busy = false
    }
  }
</script>

{#if open}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div
    class="fixed inset-0 z-[60] flex items-center justify-center bg-black/50 backdrop-blur-sm p-4"
    onclick={closeModal}
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
        <h2 class="text-sm font-semibold text-foreground">Add Project</h2>
        <button
          onclick={closeModal}
          disabled={busy}
          class="text-muted-foreground hover:text-foreground transition-colors disabled:opacity-50"
          aria-label="Close"
        >
          <X class="w-4 h-4" />
        </button>
      </div>

      <div class="space-y-3">
        <div>
          <label for="add-path" class="text-xs text-muted-foreground block mb-1.5">Directory</label>
          <div class="flex gap-2">
            <Input
              id="add-path"
              bind:value={path}
              placeholder="Click Browse to select a folder"
              readonly
              class="flex-1"
            />
            <Button
              variant="outline"
              onclick={handlePick}
              disabled={picking || busy}
              class="shrink-0"
            >
              <FolderOpen class="w-4 h-4" />
              {picking ? '...' : 'Browse'}
            </Button>
          </div>
        </div>
        <div>
          <label for="add-name" class="text-xs text-muted-foreground block mb-1.5">Name (optional)</label>
          <Input
            id="add-name"
            bind:value={name}
            placeholder="my-project"
          />
        </div>
        <div>
          <label for="add-exclude" class="text-xs text-muted-foreground block mb-1.5">Exclude patterns (optional, comma-separated)</label>
          <Input
            id="add-exclude"
            bind:value={exclude}
            placeholder="vendor/**, **/*_test.go"
          />
        </div>
        <div class="flex gap-2 pt-1">
          <Button
            variant="outline"
            onclick={closeModal}
            disabled={busy}
            class="flex-1"
          >
            Cancel
          </Button>
          <Button
            variant="default"
            onclick={handleSubmit}
            disabled={busy || !path.trim()}
            class="flex-1"
          >
            <Plus class="w-4 h-4" />
            {busy ? 'Adding...' : 'Add'}
          </Button>
        </div>
      </div>
    </div>
  </div>
{/if}
