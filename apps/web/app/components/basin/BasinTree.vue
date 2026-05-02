<template>
  <div ref="container" class="w-full">
    <!-- Orientation toggle -->
    <div class="flex justify-end mb-2">
      <button
        class="flex items-center gap-1 text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors px-2 py-1 rounded"
        :title="orientation === 'horizontal' ? 'Switch to portrait' : 'Switch to landscape'"
        @click="orientation = orientation === 'horizontal' ? 'vertical' : 'horizontal'"
      >
        <!-- landscape icon -->
        <svg v-if="orientation === 'horizontal'" class="w-4 h-4" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <rect x="1" y="3" width="14" height="10" rx="1"/>
          <line x1="8" y1="3" x2="8" y2="13"/>
        </svg>
        <!-- portrait icon -->
        <svg v-else class="w-4 h-4" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <rect x="3" y="1" width="10" height="14" rx="1"/>
          <line x1="3" y1="8" x2="13" y2="8"/>
        </svg>
        {{ orientation === 'horizontal' ? 'Portrait' : 'Landscape' }}
      </button>
    </div>

    <div :class="orientation === 'horizontal' ? 'overflow-x-auto' : 'overflow-x-auto overflow-y-hidden'">
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
          <!-- ── HORIZONTAL layout labels ───────────────────────────── -->
          <template v-if="orientation === 'horizontal'">
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
            <!-- Root -->
            <template v-else>
              <circle :r="3" :fill="isDark ? '#4b5563' : '#d1d5db'" stroke="none" />
            </template>
          </template>

          <!-- ── VERTICAL layout labels ─────────────────────────────── -->
          <template v-else>
            <!-- Leaf: dot + rotated label below -->
            <template v-if="node.isLeaf">
              <circle
                :r="5"
                :fill="node.slug === selectedSlug ? '#3b82f6' : isDark ? '#d1d5db' : '#374151'"
                stroke="none"
              />
              <g transform="rotate(-55) translate(8,0)">
                <text
                  x="0" dy="0.35em"
                  :font-size="10" :font-family="'ui-monospace, monospace'"
                  :font-weight="node.slug === selectedSlug ? 700 : 400"
                  :fill="node.slug === selectedSlug ? '#3b82f6' : isDark ? '#d1d5db' : '#374151'"
                >{{ node.label }}</text>
                <text
                  x="0" dy="1.5em"
                  :font-size="9" font-family="ui-sans-serif, sans-serif"
                  :fill="isDark ? '#9ca3af' : '#6b7280'"
                >{{ node.name }}</text>
              </g>
            </template>
            <!-- Internal river group: label above dot -->
            <template v-else-if="node.depth > 0">
              <circle :r="3" :fill="isDark ? '#4b5563' : '#d1d5db'" stroke="none" />
              <text
                x="0" dy="-0.7em" text-anchor="middle"
                :font-size="10" font-family="ui-sans-serif, sans-serif"
                :fill="isDark ? '#9ca3af' : '#6b7280'"
              >{{ node.name }}</text>
            </template>
            <!-- Root -->
            <template v-else>
              <circle :r="3" :fill="isDark ? '#4b5563' : '#d1d5db'" stroke="none" />
            </template>
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
import { ref, computed } from 'vue'
import { hierarchy, tree } from 'd3-hierarchy'
import { linkHorizontal, linkVertical } from 'd3-shape'
import type { BasinReach } from './BasinMap.vue'

const props = defineProps<{
  reaches:      BasinReach[]
  selectedSlug?: string | null
}>()

const emit = defineEmits<{ (e: 'select', slug: string): void }>()

const orientation = ref<'horizontal' | 'vertical'>('horizontal')

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

  return { name: 'basin', children: rivers }
})

// ── Layout constants ──────────────────────────────────────────────────────────

// Horizontal
const H_LEVEL_W = 180
const H_NODE_H  = 36
const H_LABEL_W = 220
const PAD       = 24

// Vertical
const V_LEAF_W  = 90   // breadth spacing per leaf
const V_LEVEL_H = 56   // depth spacing per level
const V_LABEL_H = 90   // space below leaves for rotated text

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

// ── Computed layout (branches by orientation) ─────────────────────────────────

const layout = computed(() => {
  const root = hierarchy<TreeNode>(treeData.value)
  if (root.leaves().length === 0) return { nodes: [], links: [], width: 0, height: 0 }

  if (orientation.value === 'horizontal') {
    const treeLayout = tree<TreeNode>().nodeSize([H_NODE_H, H_LEVEL_W])
    treeLayout(root)

    let minX = Infinity, maxX = -Infinity
    root.each(n => {
      minX = Math.min(minX, (n as any).x)
      maxX = Math.max(maxX, (n as any).x)
    })

    const svgH   = maxX - minX + H_NODE_H + PAD * 2
    const svgW   = (root.height + 1) * H_LEVEL_W + H_LABEL_W + PAD * 2
    const shiftY = -minX + PAD

    const linkGen = linkHorizontal<any, any>()
      .x((d: any) => d.y + PAD)
      .y((d: any) => d.x + shiftY)

    const links = root.links().map(l => linkGen(l) ?? '')

    const nodes: LayoutNode[] = root.descendants().map(n => {
      const d = n as any
      return {
        id: n.data.slug ?? n.data.name, depth: n.depth,
        isLeaf: !n.children, slug: n.data.slug,
        name: n.data.name, label: n.data.label,
        sx: d.y + PAD, sy: d.x + shiftY,
      }
    })

    return { nodes, links, width: svgW, height: svgH }

  } else {
    // Vertical: root at top, leaves at bottom
    const treeLayout = tree<TreeNode>().nodeSize([V_LEAF_W, V_LEVEL_H])
    treeLayout(root)

    let minX = Infinity, maxX = -Infinity
    root.each(n => {
      minX = Math.min(minX, (n as any).x)
      maxX = Math.max(maxX, (n as any).x)
    })

    const svgW   = maxX - minX + V_LEAF_W + PAD * 2
    const svgH   = (root.height + 1) * V_LEVEL_H + PAD * 2 + V_LABEL_H
    const shiftX = -minX + PAD + V_LEAF_W / 2

    const linkGen = linkVertical<any, any>()
      .x((d: any) => d.x + shiftX)
      .y((d: any) => d.y + PAD)

    const links = root.links().map(l => linkGen(l) ?? '')

    const nodes: LayoutNode[] = root.descendants().map(n => {
      const d = n as any
      return {
        id: n.data.slug ?? n.data.name, depth: n.depth,
        isLeaf: !n.children, slug: n.data.slug,
        name: n.data.name, label: n.data.label,
        sx: d.x + shiftX, sy: d.y + PAD,
      }
    })

    return { nodes, links, width: svgW, height: svgH }
  }
})
</script>
