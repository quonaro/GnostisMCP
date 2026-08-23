<script lang="ts">
  import { status, progress, pushToast } from '../lib/stores'
  import { reindexProject, reindexAll, removeProject, openProject } from '../lib/api'
  import { refreshStatus } from '../lib/stores'
  import Card from './ui/Card.svelte'
  import Button from './ui/Button.svelte'
  import Badge from './ui/Badge.svelte'
  import { RefreshCw, Trash2, FolderGit2, Settings2, FolderOpen, ArrowUp, ArrowDown, AlertTriangle, X } from '@lucide/svelte'
  import ProjectAdd from './ProjectAdd.svelte'
  import ProjectEditModal from './ProjectEditModal.svelte'

  let s = $derived($status)
  let busyProject = $state<string | null>(null)
  let editingProject = $state<string | null>(null)
  let showReindexAllConfirm = $state(false)
  let busyReindexAll = $state(false)

  let p = $derived($progress)
  let isRunning = $derived(p?.status === 'running')

  type SortKey = 'name' | 'path' | 'chunks' | 'model' | 'last_indexed_at'
  let sortKey = $state<SortKey>('name')
  let sortAsc = $state(true)

  let sortedProjects = $derived.by(() => {
    if (!s || !s.projects.length) return []
    const arr = [...s.projects]
    arr.sort((a, b) => {
      let cmp: number
      switch (sortKey) {
        case 'name':
          cmp = a.localeCompare(b)
          break
        case 'path':
          cmp = (s.project_stats[a]?.path ?? '').localeCompare(s.project_stats[b]?.path ?? '')
          break
        case 'chunks':
          cmp = (s.project_stats[a]?.chunks ?? 0) - (s.project_stats[b]?.chunks ?? 0)
          break
        case 'model':
          cmp = (s.project_stats[a]?.model ?? '').localeCompare(s.project_stats[b]?.model ?? '')
          break
        case 'last_indexed_at':
          cmp = (s.project_stats[a]?.last_indexed_at ?? '').localeCompare(s.project_stats[b]?.last_indexed_at ?? '')
          break
        default:
          cmp = 0
      }
      return sortAsc ? cmp : -cmp
    })
    return arr
  })

  function toggleSort(key: SortKey) {
    if (sortKey === key) {
      sortAsc = !sortAsc
    } else {
      sortKey = key
      sortAsc = true
    }
  }

  async function handleReindex(name: string) {
    busyProject = name
    try {
      await reindexProject(name)
      await refreshStatus()
      pushToast('success', `Reindex started for "${name}"`)
    } catch (e) {
      pushToast('error', String(e))
    } finally {
      busyProject = null
    }
  }

  async function handleReindexAll() {
    busyReindexAll = true
    try {
      await reindexAll()
      await refreshStatus()
      pushToast('success', 'Reindex started')
      showReindexAllConfirm = false
    } catch (e) {
      pushToast('error', String(e))
    } finally {
      busyReindexAll = false
    }
  }

  async function handleRemove(name: string) {
    if (!confirm(`Remove project "${name}"?`)) return
    busyProject = name
    try {
      await removeProject(name)
      await refreshStatus()
      pushToast('success', `Project "${name}" removed`)
    } catch (e) {
      pushToast('error', String(e))
    } finally {
      busyProject = null
    }
  }

  async function handleOpen(name: string) {
    try {
      await openProject(name)
    } catch (e) {
      pushToast('error', String(e))
    }
  }

  function formatDate(iso: string): string {
    if (!iso || iso === '0001-01-01T00:00:00Z') return '—'
    return new Date(iso).toLocaleString()
  }
</script>

<Card class="p-5 flex flex-col h-full flex-1">
  <div class="flex items-center justify-between mb-4 shrink-0">
    <h2 class="text-sm font-semibold text-muted-foreground uppercase">Projects</h2>
    <div class="flex items-center gap-2">
      <Button
        variant="destructive"
        size="sm"
        onclick={() => (showReindexAllConfirm = true)}
        disabled={isRunning}
      >
        <RefreshCw class="w-3.5 h-3.5" />
        Reindex All
      </Button>
      <ProjectAdd />
    </div>
  </div>

  {#if s && s.projects.length > 0}
    <div class="overflow-auto flex-1">
      <table class="w-full text-sm">
        <thead>
          <tr class="text-left text-xs text-muted-foreground border-b border-border">
            <th class="pb-2 pr-4 font-medium cursor-pointer select-none hover:text-foreground transition-colors" onclick={() => toggleSort('name')}>
              <span class="inline-flex items-center gap-1">
                Name
                {#if sortKey === 'name'}{#if sortAsc}<ArrowUp class="w-3 h-3" />{:else}<ArrowDown class="w-3 h-3" />{/if}{/if}
              </span>
            </th>
            <th class="pb-2 pr-4 font-medium cursor-pointer select-none hover:text-foreground transition-colors" onclick={() => toggleSort('path')}>
              <span class="inline-flex items-center gap-1">
                Path
                {#if sortKey === 'path'}{#if sortAsc}<ArrowUp class="w-3 h-3" />{:else}<ArrowDown class="w-3 h-3" />{/if}{/if}
              </span>
            </th>
            <th class="pb-2 pr-4 font-medium cursor-pointer select-none hover:text-foreground transition-colors" onclick={() => toggleSort('chunks')}>
              <span class="inline-flex items-center gap-1">
                Chunks
                {#if sortKey === 'chunks'}{#if sortAsc}<ArrowUp class="w-3 h-3" />{:else}<ArrowDown class="w-3 h-3" />{/if}{/if}
              </span>
            </th>
            <th class="pb-2 pr-4 font-medium cursor-pointer select-none hover:text-foreground transition-colors" onclick={() => toggleSort('model')}>
              <span class="inline-flex items-center gap-1">
                Model
                {#if sortKey === 'model'}{#if sortAsc}<ArrowUp class="w-3 h-3" />{:else}<ArrowDown class="w-3 h-3" />{/if}{/if}
              </span>
            </th>
            <th class="pb-2 pr-4 font-medium cursor-pointer select-none hover:text-foreground transition-colors" onclick={() => toggleSort('last_indexed_at')}>
              <span class="inline-flex items-center gap-1">
                Last Indexed
                {#if sortKey === 'last_indexed_at'}{#if sortAsc}<ArrowUp class="w-3 h-3" />{:else}<ArrowDown class="w-3 h-3" />{/if}{/if}
              </span>
            </th>
            <th class="pb-2 font-medium">Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each sortedProjects as name (name)}
            <tr class="border-b border-border/50 hover:bg-secondary/30 transition-colors">
              <td class="py-2.5 pr-4">
                <div class="flex items-center gap-2">
                  <FolderGit2 class="w-4 h-4 text-muted-foreground shrink-0" />
                  <span class="font-medium text-foreground">{name}</span>
                </div>
              </td>
              <td class="py-2.5 pr-4 text-muted-foreground text-xs truncate max-w-[200px]" title={s.project_stats[name]?.path ?? ''}>
                {s.project_stats[name]?.path ?? '—'}
              </td>
              <td class="py-2.5 pr-4">
                <Badge variant="default">{s.project_stats[name]?.chunks ?? 0}</Badge>
              </td>
              <td class="py-2.5 pr-4 text-muted-foreground text-xs truncate max-w-[160px]" title={s.project_stats[name]?.model ?? ''}>
                {s.project_stats[name]?.model ?? '—'}
              </td>
              <td class="py-2.5 pr-4 text-muted-foreground text-xs">{formatDate(s.project_stats[name]?.last_indexed_at ?? '')}</td>
              <td class="py-2.5">
                <div class="flex gap-1.5">
                  <Button
                    variant="ghost"
                    size="sm"
                    onclick={() => handleOpen(name)}
                    title="Open in file manager"
                  >
                    <FolderOpen class="w-3 h-3" />
                  </Button>
                  <Button
                    variant="destructive"
                    size="sm"
                    onclick={() => handleReindex(name)}
                    disabled={busyProject === name}
                  >
                    <RefreshCw class="w-3 h-3" />
                    Reindex
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onclick={() => (editingProject = name)}
                    disabled={busyProject === name}
                  >
                    <Settings2 class="w-3 h-3" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onclick={() => handleRemove(name)}
                    disabled={busyProject === name}
                    class="text-destructive hover:bg-destructive/10 hover:text-destructive"
                  >
                    <Trash2 class="w-3 h-3" />
                  </Button>
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {:else}
    <div class="text-sm text-muted-foreground py-8 text-center">No projects configured. Add one to get started.</div>
  {/if}
</Card>

{#if editingProject}
  <ProjectEditModal
    projectName={editingProject}
    onClose={() => (editingProject = null)}
  />
{/if}

{#if showReindexAllConfirm}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div
    class="fixed inset-0 z-[60] flex items-center justify-center bg-black/50 backdrop-blur-sm p-4"
    onclick={() => !busyReindexAll && (showReindexAllConfirm = false)}
    onkeydown={(e) => e.key === 'Escape' && !busyReindexAll && (showReindexAllConfirm = false)}
    role="presentation"
  >
    <!-- svelte-ignore a11y_no_static_element_interactions, a11y_click_events_have_key_events -->
    <div
      class="w-full max-w-md rounded-lg border border-border bg-card shadow-lg p-5"
      onclick={(e) => e.stopPropagation()}
      onkeydown={(e) => e.stopPropagation()}
      role="presentation"
    >
      <div class="flex items-center justify-between mb-4">
        <div class="flex items-center gap-2">
          <AlertTriangle class="w-5 h-5 text-destructive" />
          <h2 class="text-sm font-semibold text-foreground">Reindex All Projects</h2>
        </div>
        <button
          onclick={() => !busyReindexAll && (showReindexAllConfirm = false)}
          disabled={busyReindexAll}
          class="text-muted-foreground hover:text-foreground transition-colors disabled:opacity-50"
          aria-label="Close"
        >
          <X class="w-4 h-4" />
        </button>
      </div>

      <p class="text-sm text-muted-foreground mb-4">
        This will reindex <strong class="text-foreground">all projects</strong>. This action cannot be undone and may take a while depending on the size of your projects.
      </p>

      <div class="flex gap-2">
        <Button
          variant="outline"
          onclick={() => (showReindexAllConfirm = false)}
          disabled={busyReindexAll}
          class="flex-1"
        >
          Cancel
        </Button>
        <Button
          variant="destructive"
          onclick={handleReindexAll}
          disabled={busyReindexAll}
          class="flex-1"
        >
          <RefreshCw class="w-3.5 h-3.5" />
          {busyReindexAll ? 'Starting...' : 'Reindex All'}
        </Button>
      </div>
    </div>
  </div>
{/if}
