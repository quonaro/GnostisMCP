<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import { graphData, graphLoading, selectedProject, status, pushToast } from '../lib/stores'
  import { getGraph } from '../lib/api'
  import type { GraphNode, GraphEdge } from '../lib/api'
  import Card from './ui/Card.svelte'
  import { Loader2, Network } from '@lucide/svelte'

  const KIND_COLORS: Record<string, string> = {
    function: '#22c55e',
    method: '#3b82f6',
    type: '#f97316',
  }

  let canvasEl: HTMLCanvasElement
  let ctx: CanvasRenderingContext2D | null = null
  let animationId = 0
  let needsRender = false

  let nodes: GraphNode[] = $state([])
  let edges: GraphEdge[] = []
  let nodeById = new Map<string, GraphNode>()
  let resolvedLinks: { s: GraphNode; t: GraphNode }[] = []

  let hoveredNode: GraphNode | null = $state(null)
  let selectedNodeId: string | null = $state(null)
  let totalNodes = $state(0)
  let totalEdges = $state(0)
  let subsampled = $state(false)
  let hideIsolated = $state(true)
  let isolatedCount = $state(0)

  let highlightSet: Set<string> | null = null

  let scale = 1
  let offsetX = 0
  let offsetY = 0
  let isPanning = false
  let panStartX = 0
  let panStartY = 0

  let hoverRafPending = false
  let lastMoveX = 0
  let lastMoveY = 0

  let s = $derived($status)
  let project = $derived($selectedProject || '')

  async function loadGraph() {
    graphLoading.set(true)
    try {
      const data = await getGraph(project, { connected_only: hideIsolated, max_nodes: 800 })
      graphData.set(data)
      buildGraph(data.nodes, data.edges)
    } catch (e) {
      pushToast('error', `Graph load failed: ${String(e)}`)
    } finally {
      graphLoading.set(false)
    }
  }

  function buildGraph(rawNodes: GraphNode[], rawEdges: GraphEdge[]) {
    totalNodes = rawNodes.length
    totalEdges = rawEdges.length
    subsampled = false
    selectedNodeId = null
    hoveredNode = null

    nodes = rawNodes
    nodeById = new Map(nodes.map((n) => [n.id, n]))
    resolvedLinks = rawEdges
      .filter((e) => nodeById.has(e.from) && nodeById.has(e.to))
      .map((e) => ({ s: nodeById.get(e.from)!, t: nodeById.get(e.to)! }))

    highlightSet = null
    scale = 1
    offsetX = 0
    offsetY = 0
    requestRender()
  }

  function requestRender() {
    if (needsRender) return
    needsRender = true
    animationId = requestAnimationFrame(() => {
      needsRender = false
      render()
    })
  }

  function rebuildHighlight() {
    if (!selectedNodeId) {
      highlightSet = null
      return
    }
    highlightSet = new Set<string>([selectedNodeId])
    for (const link of resolvedLinks) {
      if (link.s.id === selectedNodeId) highlightSet.add(link.t.id)
      if (link.t.id === selectedNodeId) highlightSet.add(link.s.id)
    }
  }

  function render() {
    if (!ctx || !canvasEl) return
    const w = canvasEl.clientWidth
    const h = canvasEl.clientHeight
    ctx.clearRect(0, 0, w, h)
    ctx.fillStyle = getComputedStyle(canvasEl).getPropertyValue('--background').trim() || '#f5f8fa'
    ctx.fillRect(0, 0, w, h)

    ctx.save()
    ctx.translate(offsetX, offsetY)
    ctx.scale(scale, scale)

    const margin = 50
    const visMinX = -offsetX / scale - margin
    const visMaxX = (w - offsetX) / scale + margin
    const visMinY = -offsetY / scale - margin
    const visMaxY = (h - offsetY) / scale + margin

    const drawEdges = scale > 0.3

    if (drawEdges) {
      if (highlightSet) {
        ctx.strokeStyle = 'rgba(150, 150, 150, 0.15)'
        ctx.lineWidth = 0.5
        ctx.beginPath()
        for (const { s, t } of resolvedLinks) {
          if (highlightSet.has(s.id) && highlightSet.has(t.id)) continue
          if ((s.x < visMinX && t.x < visMinX) || (s.x > visMaxX && t.x > visMaxX)) continue
          if ((s.y < visMinY && t.y < visMinY) || (s.y > visMaxY && t.y > visMaxY)) continue
          ctx.moveTo(s.x, s.y)
          ctx.lineTo(t.x, t.y)
        }
        ctx.stroke()

        ctx.strokeStyle = 'rgba(100, 200, 255, 0.6)'
        ctx.lineWidth = 1.5
        ctx.beginPath()
        for (const { s, t } of resolvedLinks) {
          if (!highlightSet.has(s.id) || !highlightSet.has(t.id)) continue
          if ((s.x < visMinX && t.x < visMinX) || (s.x > visMaxX && t.x > visMaxX)) continue
          if ((s.y < visMinY && t.y < visMinY) || (s.y > visMaxY && t.y > visMaxY)) continue
          ctx.moveTo(s.x, s.y)
          ctx.lineTo(t.x, t.y)
        }
        ctx.stroke()
      } else {
        ctx.strokeStyle = 'rgba(100, 120, 100, 0.3)'
        ctx.lineWidth = 0.5
        ctx.beginPath()
        for (const { s, t } of resolvedLinks) {
          if ((s.x < visMinX && t.x < visMinX) || (s.x > visMaxX && t.x > visMaxX)) continue
          if ((s.y < visMinY && t.y < visMinY) || (s.y > visMaxY && t.y > visMaxY)) continue
          ctx.moveTo(s.x, s.y)
          ctx.lineTo(t.x, t.y)
        }
        ctx.stroke()
      }
    }

    if (highlightSet) {
      ctx.globalAlpha = 0.3
      ctx.fillStyle = '#ccc'
      ctx.beginPath()
      for (const n of nodes) {
        if (highlightSet.has(n.id)) continue
        if (n.x < visMinX || n.x > visMaxX || n.y < visMinY || n.y > visMaxY) continue
        const radius = Math.max(2, Math.min(6, 2 + Math.sqrt(n.degree) * 0.8))
        ctx.moveTo(n.x + radius, n.y)
        ctx.arc(n.x, n.y, radius, 0, Math.PI * 2)
      }
      ctx.fill()
      ctx.globalAlpha = 1
    }

    ctx.fillStyle = '#888'
    ctx.beginPath()
    for (const n of nodes) {
      if (highlightSet && !highlightSet.has(n.id)) continue
      if (n.x < visMinX || n.x > visMaxX || n.y < visMinY || n.y > visMaxY) continue
      const color = KIND_COLORS[n.kind] ?? '#888'
      if (color !== '#888') continue
      const radius = Math.max(2, Math.min(6, 2 + Math.sqrt(n.degree) * 0.8))
      ctx.moveTo(n.x + radius, n.y)
      ctx.arc(n.x, n.y, radius, 0, Math.PI * 2)
    }
    ctx.fill()

    for (const [kind, color] of Object.entries(KIND_COLORS)) {
      ctx.fillStyle = color
      ctx.beginPath()
      let hasAny = false
      for (const n of nodes) {
        if (highlightSet && !highlightSet.has(n.id)) continue
        if (n.kind !== kind) continue
        if (selectedNodeId === n.id) continue
        if (n.x < visMinX || n.x > visMaxX || n.y < visMinY || n.y > visMaxY) continue
        const radius = Math.max(2, Math.min(6, 2 + Math.sqrt(n.degree) * 0.8))
        ctx.moveTo(n.x + radius, n.y)
        ctx.arc(n.x, n.y, radius, 0, Math.PI * 2)
        hasAny = true
      }
      if (hasAny) ctx.fill()
    }

    if (selectedNodeId) {
      const sn = nodeById.get(selectedNodeId)
      if (sn) {
        const color = KIND_COLORS[sn.kind] ?? '#888'
        const radius = Math.max(2, Math.min(6, 2 + Math.sqrt(sn.degree) * 0.8))
        ctx.strokeStyle = color
        ctx.lineWidth = 2
        ctx.beginPath()
        ctx.arc(sn.x, sn.y, radius + 4, 0, Math.PI * 2)
        ctx.stroke()
        ctx.fillStyle = color
        ctx.beginPath()
        ctx.arc(sn.x, sn.y, radius, 0, Math.PI * 2)
        ctx.fill()
      }
    }

    ctx.restore()
  }

  function screenToWorld(clientX: number, clientY: number): { x: number; y: number } {
    const rect = canvasEl!.getBoundingClientRect()
    const sx = (clientX - rect.left) * (canvasEl!.width / rect.width)
    const sy = (clientY - rect.top) * (canvasEl!.height / rect.height)
    return {
      x: (sx - offsetX) / scale,
      y: (sy - offsetY) / scale,
    }
  }

  function findNodeAt(wx: number, wy: number): GraphNode | null {
    const minDist = 12 / scale
    let closest: GraphNode | null = null
    let closestDist = minDist
    for (const n of nodes) {
      const ddx = n.x - wx
      const ddy = n.y - wy
      const d = Math.sqrt(ddx * ddx + ddy * ddy)
      if (d < closestDist) {
        closestDist = d
        closest = n
      }
    }
    return closest
  }

  function onPointerDown(e: MouseEvent) {
    if (!canvasEl) return
    isPanning = true
    panStartX = e.clientX - offsetX
    panStartY = e.clientY - offsetY
    canvasEl.style.cursor = 'grabbing'
  }

  function onPointerMove(e: MouseEvent) {
    if (!canvasEl) return

    if (isPanning) {
      offsetX = e.clientX - panStartX
      offsetY = e.clientY - panStartY
      requestRender()
      return
    }

    lastMoveX = e.clientX
    lastMoveY = e.clientY
    if (hoverRafPending) return
    hoverRafPending = true
    requestAnimationFrame(() => {
      hoverRafPending = false
      const { x, y } = screenToWorld(lastMoveX, lastMoveY)
      const node = findNodeAt(x, y)
      if (node) {
        hoveredNode = node
        canvasEl.style.cursor = 'pointer'
      } else {
        hoveredNode = null
        canvasEl.style.cursor = 'default'
      }
    })
  }

  function onPointerUp() {
    isPanning = false
    if (canvasEl) canvasEl.style.cursor = 'default'
  }

  function onClick(e: MouseEvent) {
    if (!canvasEl) return
    const { x, y } = screenToWorld(e.clientX, e.clientY)
    const node = findNodeAt(x, y)
    if (node) {
      selectedNodeId = selectedNodeId === node.id ? null : node.id
    } else {
      selectedNodeId = null
    }
    rebuildHighlight()
    render()
  }

  function onWheel(e: WheelEvent) {
    if (!canvasEl) return
    e.preventDefault()
    const rect = canvasEl.getBoundingClientRect()
    const mx = (e.clientX - rect.left) * (canvasEl.width / rect.width)
    const my = (e.clientY - rect.top) * (canvasEl.height / rect.height)

    const delta = e.deltaY > 0 ? 0.9 : 1.1
    const newScale = Math.max(0.1, Math.min(10, scale * delta))

    offsetX = mx - (mx - offsetX) * (newScale / scale)
    offsetY = my - (my - offsetY) * (newScale / scale)
    scale = newScale
    requestRender()
  }

  function onResize() {
    if (!canvasEl || !ctx) return
    const dpr = window.devicePixelRatio || 1
    const w = canvasEl.clientWidth
    const h = canvasEl.clientHeight
    canvasEl.width = w * dpr
    canvasEl.height = h * dpr
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
    render()
  }

  onMount(() => {
    const dpr = window.devicePixelRatio || 1
    const w = canvasEl.clientWidth || 800
    const h = canvasEl.clientHeight || 600
    canvasEl.width = w * dpr
    canvasEl.height = h * dpr
    ctx = canvasEl.getContext('2d')
    if (ctx) ctx.setTransform(dpr, 0, 0, dpr, 0, 0)

    canvasEl.addEventListener('pointerdown', onPointerDown)
    canvasEl.addEventListener('pointermove', onPointerMove)
    canvasEl.addEventListener('pointerup', onPointerUp)
    canvasEl.addEventListener('click', onClick)
    canvasEl.addEventListener('wheel', onWheel, { passive: false })
    window.addEventListener('resize', onResize)

    loadGraph()
  })

  onDestroy(() => {
    cancelAnimationFrame(animationId)
    canvasEl?.removeEventListener('pointerdown', onPointerDown)
    canvasEl?.removeEventListener('pointermove', onPointerMove)
    canvasEl?.removeEventListener('pointerup', onPointerUp)
    canvasEl?.removeEventListener('click', onClick)
    canvasEl?.removeEventListener('wheel', onWheel)
    window.removeEventListener('resize', onResize)
  })

  let isLoading = $derived($graphLoading)
  let data = $derived($graphData)
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <Network class="w-5 h-5 text-primary" />
      <h2 class="text-lg font-semibold">Graph</h2>
      {#if data}
        <span class="text-sm text-muted-foreground">
          {project ? project : 'All projects'}
          · {subsampled ? `showing ${nodes.length} of ${totalNodes} nodes` : `${totalNodes} nodes`}
          · {totalEdges} edges
          {#if hideIsolated && isolatedCount > 0}· {isolatedCount} isolated hidden{/if}
        </span>
      {/if}
    </div>
    <div class="flex items-center gap-2">
      {#if selectedNodeId}
        <button
          onclick={() => { selectedNodeId = null; rebuildHighlight(); render() }}
          class="px-2 py-1 text-xs rounded bg-muted text-muted-foreground hover:bg-muted/80 transition-colors"
        >
          Clear selection
        </button>
      {/if}
      <button
        onclick={() => { hideIsolated = !hideIsolated; loadGraph() }}
        class="px-2 py-1 text-xs rounded transition-colors {hideIsolated ? 'bg-primary/15 text-primary' : 'bg-muted text-muted-foreground hover:bg-muted/80'}"
        title="Toggle isolated nodes (degree 0)"
      >
        {hideIsolated ? 'Connected only' : 'Show all'}
      </button>
      <button
        onclick={loadGraph}
        class="px-3 py-1.5 text-sm rounded-md bg-primary/10 text-primary hover:bg-primary/20 transition-colors"
        disabled={isLoading}
      >
        Refresh
      </button>
    </div>
  </div>

  <Card class="p-0 overflow-hidden relative">
    <div class="relative" style="height: 70vh; min-height: 400px;">
      {#if isLoading}
        <div class="absolute inset-0 flex items-center justify-center bg-background/80 z-10">
          <Loader2 class="w-6 h-6 animate-spin text-primary" />
        </div>
      {/if}
      <canvas bind:this={canvasEl} class="w-full h-full block"></canvas>

      <!-- Legend -->
      <div class="absolute top-3 right-3 bg-card/90 border border-border rounded-lg p-2 text-xs space-y-1">
        {#each Object.entries(KIND_COLORS) as [kind, color]}
          <div class="flex items-center gap-1.5">
            <span class="inline-block w-2.5 h-2.5 rounded-full" style="background:{color}"></span>
            <span class="text-muted-foreground capitalize">{kind}</span>
          </div>
        {/each}
      </div>

      {#if subsampled}
        <div class="absolute top-3 left-3 bg-card/90 border border-border rounded-lg px-3 py-1.5 text-xs text-muted-foreground">
          Showing top {nodes.length.toLocaleString()} nodes by connectivity
        </div>
      {/if}

      {#if hoveredNode}
        <div class="absolute bottom-4 left-4 bg-card/95 border border-border rounded-lg p-3 text-sm shadow-lg max-w-xs">
          <div class="font-semibold text-foreground">{hoveredNode.symbol}</div>
          <div class="text-xs text-muted-foreground mt-1">{hoveredNode.path}</div>
          <div class="text-xs text-muted-foreground mt-1">{hoveredNode.kind} · lines {hoveredNode.start_line}–{hoveredNode.end_line}</div>
        </div>
      {/if}
    </div>
  </Card>
</div>
