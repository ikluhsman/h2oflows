<template>
  <div ref="container" class="w-full overflow-x-auto">
    <svg
      v-if="layout.nodes.length > 0"
      :width="layout.width"
      :height="layout.height"
      class="select-none"
    >
      <!-- Links -->
      <path
        v-for="(link, i) in layout.links"
        :key="`l${i}`"
        :d="link"
        fill="none"
        stroke="currentColor"
        stroke-width="1.5"
        class="text-gray-300 dark:text-gray-600"
      />

      <!-- Nodes -->
      <g
        v-for="node in layout.nodes"
        :key="node.id"
        :transform="`translate(${node.sx},${node.sy})`"
        :style="node.isLeaf ? 'cursor:pointer' : undefined"
        @click="node.isLeaf ? emit('select', node.slug!) : undefined"
      >
        <!-- Leaf: reach node -->
        <template v-if="node.isLeaf">
          <circle
            :r="5"
            :fill="node.slug === selectedSlug ? '#3b82f6' : isDark ? '#d1d5db' : '#374151'"
            stroke="none"
          />
          <!-- comId label -->
          <text
            x="10"
            dy="0.35em"
            :font-size="11"
            :font-family="'ui-monospace, monospace'"
            :font-weight="node.slug === selectedSlug ? 700 : 400"
            :fill="node.slug === selectedSlug
              ? '#3b82f6'
              : isDark ? '#d1d5db' : '#374151'"
          >{{ node.label }}</text>
          <!-- reach name below comId -->
          <text
            x="10"
            dy="1.5em"
            :font-size="10"
            font-family="ui-sans-serif, sans-serif"
            :fill="isDark ? '#9ca3af' : '#6b7280'"
          >{{ node.name }}</text>
        </template>

        <!-- Internal: river group node -->
        <template v-else-if="node.depth > 0">
          <circle :r="3" :fill="isDark ? '#4b5563' : '#d1d5db'" stroke="none" />
          <text
            x="-8"
            dy="0.35em"
            text-anchor="end"
            :font-size="11"
            font-family="ui-sans-serif, sans-serif"
            :fill="isDark ? '#9ca3af' : '#6b7280'"
          >{{ node.name }}</text>
        </template>

        <!-- Root: basin label -->
        <template v-else>
          <circle :r="3" :fill="isDark ? '#4b5563' : '#d1d5db'" stroke="none" />
        </template>
      </g>
    </svg>

    <p v-else-if="reaches.length > 0" class="text-sm text-gray-400 italic py-2">
      Building tree…
    </p>
    <p v-else class="text-sm text-gray-400 italic py-2">
      No reaches loaded yet.
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { hierarchy, tree } from 'd3-hierarchy'
import { linkHorizontal } from 'd3-shape'
import type { BasinReach } from './BasinMap.vue'

const props = defineProps<{
  reaches:      BasinReach[]
  selectedSlug?: string | null
}>()

const emit = defineEmits<{ (e: 'select', slug: string): void }>()

// Detect dark mode via CSS media query — avoids needing a color-mode composable.
const isDark = typeof window !== 'undefined'
  && window.matchMedia('(prefers-color-scheme: dark)').matches

// ── Tree data ─────────────────────────────────────────────────────────────────

interface TreeNode {
  name:     string
  slug?:    string
  label?:   string   // comId or fallback
  children?: TreeNode[]
}

const treeData = computed<TreeNode>(() => {
  const riverMap = new Map<string, BasinReach[]>()
  for (const r of props.reaches) {
    const river = r.river_name ?? 'Unknown River'
    if (!riverMap.has(river)) riverMap.set(river, [])
    riverMap.get(river)!.push(r)
  }

  for (const reaches of riverMap.values()) {
    reaches.sort((a, b) => {
      if (a.river_order != null && b.river_order != null) return a.river_order - b.river_order
      if (a.river_order != null) return -1
      if (b.river_order != null) return 1
      return a.slug.localeCompare(b.slug)
    })
  }

  const rivers = [...riverMap.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([river, reaches]) => ({
      name: river,
      children: reaches.map(r => ({
        name:  r.common_name ?? r.name,
        slug:  r.slug,
        label: r.anchor_comid ?? r.start_comid ?? r.slug,
      })),
    }))

  return { name: 'basin', children: rivers }
})

// ── D3 layout ─────────────────────────────────────────────────────────────────

const LEVEL_W  = 180   // horizontal spacing per depth level
const NODE_H   = 36    // vertical spacing per leaf
const PAD      = 24    // outer padding
const LABEL_W  = 220   // space reserved right of leaves for text

interface LayoutNode {
  id:     string
  depth:  number
  isLeaf: boolean
  slug?:  string
  name:   string
  label?: string
  sx:     number   // svg x
  sy:     number   // svg y
}

const layout = computed(() => {
  const root = hierarchy<TreeNode>(treeData.value)
  if (root.leaves().length === 0) return { nodes: [], links: [], width: 0, height: 0 }

  const treeLayout = tree<TreeNode>().nodeSize([NODE_H, LEVEL_W])
  treeLayout(root)

  // d3 tree: node.x = breadth axis, node.y = depth axis
  let minX = Infinity
  let maxX = -Infinity
  root.each(n => {
    minX = Math.min(minX, (n as any).x)
    maxX = Math.max(maxX, (n as any).x)
  })

  const svgH   = maxX - minX + NODE_H + PAD * 2
  const maxDepth = root.height
  const svgW   = (maxDepth + 1) * LEVEL_W + LABEL_W + PAD * 2

  // Shift so minX lands at PAD
  const shiftY = -minX + PAD

  const linkGen = linkHorizontal<any, any>()
    .x((d: any) => (d as any).y + PAD)
    .y((d: any) => (d as any).x + shiftY)

  const links = root.links().map(l => linkGen(l) ?? '')

  const nodes: LayoutNode[] = root.descendants().map(n => {
    const d = n as any
    return {
      id:     n.data.slug ?? n.data.name,
      depth:  n.depth,
      isLeaf: !n.children,
      slug:   n.data.slug,
      name:   n.data.name,
      label:  n.data.label,
      sx:     d.y + PAD,
      sy:     d.x + shiftY,
    }
  })

  return { nodes, links, width: svgW, height: svgH }
})
</script>
