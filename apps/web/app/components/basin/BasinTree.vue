<template>
  <div ref="container" class="w-full">
    <div class="overflow-x-auto">
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
          <!-- Leaf -->
          <template v-if="node.isLeaf">
            <circle
              :r="5"
              :fill="node.slug === selectedSlug ? '#3b82f6' : isDark ? '#d1d5db' : '#374151'"
              stroke="none"
            />
            <text
              x="10" dy="0.35em"
              :font-size="11" :font-family="'ui-monospace, monospace'"
              :font-weight="node.slug === selectedSlug ? 700 : 400"
              :fill="node.slug === selectedSlug ? '#3b82f6' : isDark ? '#d1d5db' : '#374151'"
            >{{ node.label }}</text>
            <text
              x="10" dy="1.5em"
              :font-size="10" font-family="ui-sans-serif, sans-serif"
              :fill="isDark ? '#9ca3af' : '#6b7280'"
            >{{ node.name }}</text>
          </template>
          <!-- Internal river group -->
          <template v-else-if="node.depth > 0">
            <circle :r="3" :fill="isDark ? '#4b5563' : '#d1d5db'" stroke="none" />
            <text
              x="-8" dy="0.35em" text-anchor="end"
              :font-size="11" font-family="ui-sans-serif, sans-serif"
              :fill="isDark ? '#9ca3af' : '#6b7280'"
            >{{ node.name }}</text>
          </template>
          <!-- Root (basin) -->
          <template v-else>
            <circle :r="4" :fill="isDark ? '#6b7280' : '#9ca3af'" stroke="none" />
            <text
              x="-10" dy="0.35em" text-anchor="end"
              :font-size="12" font-weight="600" font-family="ui-sans-serif, sans-serif"
              :fill="isDark ? '#e5e7eb' : '#374151'"
            >{{ node.name }}</text>
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
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { hierarchy, tree } from 'd3-hierarchy'
import { linkHorizontal } from 'd3-shape'
import type { BasinReach } from './BasinMap.vue'

const props = defineProps<{
  reaches:      BasinReach[]
  basinName?:   string | null
  selectedSlug?: string | null
}>()

const emit = defineEmits<{ (e: 'select', slug: string): void }>()

const isDark = typeof window !== 'undefined'
  && window.matchMedia('(prefers-color-scheme: dark)').matches

// ── Tree data ─────────────────────────────────────────────────────────────────

interface TreeNode {
  name:     string
  slug?:    string
  label?:   string
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

  const rootName = props.basinName ? `${props.basinName} Basin` : 'Basin'
  return { name: rootName, children: rivers }
})

// ── Layout constants ──────────────────────────────────────────────────────────

const H_LEVEL_W   = 180
const H_NODE_H    = 36
const H_LABEL_W   = 220
const PAD         = 24
const H_ROOT_PAD  = 110  // left margin so root label doesn't clip

interface LayoutNode {
  id:     string
  depth:  number
  isLeaf: boolean
  slug?:  string
  name:   string
  label?: string
  sx:     number
  sy:     number
}

// ── Computed layout ───────────────────────────────────────────────────────────

const layout = computed(() => {
  const root = hierarchy<TreeNode>(treeData.value)
  if (root.leaves().length === 0) return { nodes: [], links: [], width: 0, height: 0 }

  const treeLayout = tree<TreeNode>().nodeSize([H_NODE_H, H_LEVEL_W])
  treeLayout(root)

  let minX = Infinity, maxX = -Infinity
  root.each(n => {
    minX = Math.min(minX, (n as any).x)
    maxX = Math.max(maxX, (n as any).x)
  })

  const svgH   = maxX - minX + H_NODE_H + PAD * 2
  const svgW   = H_ROOT_PAD + (root.height + 1) * H_LEVEL_W + H_LABEL_W + PAD * 2
  const shiftY = -minX + PAD
  const offX   = PAD + H_ROOT_PAD

  const linkGen = linkHorizontal<any, any>()
    .x((d: any) => d.y + offX)
    .y((d: any) => d.x + shiftY)

  const links = root.links().map(l => linkGen(l) ?? '')

  const nodes: LayoutNode[] = root.descendants().map(n => {
    const d = n as any
    return {
      id: n.data.slug ?? n.data.name, depth: n.depth,
      isLeaf: !n.children, slug: n.data.slug,
      name: n.data.name, label: n.data.label,
      sx: d.y + offX, sy: d.x + shiftY,
    }
  })

  return { nodes, links, width: svgW, height: svgH }
})
</script>
