<script lang="ts">
  import { showAddModal } from '../lib/stores'
  import Button from './ui/Button.svelte'
  import { Plus } from '@lucide/svelte'

  function showModal() {
    showAddModal.set(true)
  }
</script>

<Button variant="default" size="sm" onclick={showModal}>
  <Plus class="w-4 h-4" />
  Add
</Button>
