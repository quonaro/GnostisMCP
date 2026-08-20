<script lang="ts">
  import { status, pushToast } from '../lib/stores'
  import { rebuildProject, removeProject } from '../lib/api'
  import { refreshStatus } from '../lib/stores'
  import Card from './ui/Card.svelte'
  import Button from './ui/Button.svelte'
  import Badge from './ui/Badge.svelte'
  import { RefreshCw, Trash2, FolderGit2, Settings2 } from '@lucide/svelte'
  import ProjectAdd from './ProjectAdd.svelte'
  import ProjectEditModal from './ProjectEditModal.svelte'

  let s = $derived($status)
  let busyProject = $state<string | null>(null)
  let editingProject = $state<string | null>(null)

  async function handleRebuild(name: string) {
    busyProject = name
    try {
      await rebuildProject(name)
      await refreshStatus()
      pushToast('success', `Rebuild started for "${name}"`)
    } catch (e) {
      pushToast('error', String(e))
    } finally {
      busyProject = null
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

  function formatDate(iso: string): string {
    if (!iso || iso === '0001-01-01T00:00:00Z') return '—'
    return new Date(iso).toLocaleString()
  }
</script>

<Card class="p-5 h-full flex flex-col">
  <div class="flex items-center justify-between mb-4 shrink-0">
    <h2 class="text-sm font-semibold text-muted-foreground uppercase">Projects</h2>
    <ProjectAdd />
  </div>

  {#if s && s.projects.length > 0}
    <div class="overflow-auto flex-1 max-h-[50vh]">
      <table class="w-full text-sm">
        <thead>
          <tr class="text-left text-xs text-muted-foreground border-b border-border">
            <th class="pb-2 pr-4 font-medium">Name</th>
            <th class="pb-2 pr-4 font-medium">Path</th>
            <th class="pb-2 pr-4 font-medium">Chunks</th>
            <th class="pb-2 pr-4 font-medium">Last Indexed</th>
            <th class="pb-2 font-medium">Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each s.projects as name (name)}
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
              <td class="py-2.5 pr-4 text-muted-foreground text-xs">{formatDate(s.project_stats[name]?.last_indexed_at ?? '')}</td>
              <td class="py-2.5">
                <div class="flex gap-1.5">
                  <Button
                    variant="outline"
                    size="sm"
                    onclick={() => handleRebuild(name)}
                    disabled={busyProject === name}
                  >
                    <RefreshCw class="w-3 h-3" />
                    Rebuild
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
