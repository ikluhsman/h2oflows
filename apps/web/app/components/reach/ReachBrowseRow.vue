<template>
  <div
    class="flex items-center gap-2 sm:gap-3 px-3 py-2 hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors group"
  >
    <!-- River icon -->
    <svg class="w-3.5 h-3.5 text-blue-500/70 dark:text-blue-400/70 shrink-0" viewBox="0 0 32 32" fill="none" aria-label="Reach">
      <path d="M4 14c3-6 6-9 8-9s5 9 8 9 5-9 8-9" stroke="currentColor" stroke-width="3" stroke-linecap="round"/>
      <path d="M4 22c3-6 6-9 8-9s5 9 8 9 5-9 8-9" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" opacity="0.6"/>
    </svg>

    <!-- Reach name — click navigates to detail page -->
    <NuxtLink
      :to="`/reaches/${reach.slug}`"
      class="flex-1 min-w-0 text-sm font-medium text-gray-800 dark:text-gray-200 truncate hover:text-blue-600 dark:hover:text-blue-400 transition-colors"
    >{{ displayName }}</NuxtLink>

    <!-- Class rating -->
    <span v-if="classDisplay" class="text-xs font-medium text-gray-500 dark:text-gray-400 tabular-nums shrink-0">
      {{ classDisplay }}
    </span>

    <!-- Sparkline -->
    <div v-if="reach.gauge_id" class="w-20 shrink-0 hidden sm:block opacity-60">
      <GaugeSparkline
        :gauge-id="reach.gauge_id"
        :flow-status="reach.flow_status"
        :flow-band-label="reach.flow_label"
        :reach-slug="reach.slug"
        color="#3b82f6"
        compact
        @latest-cfs="liveCfs = $event"
      />
    </div>

    <!-- Flow badge + CFS — click opens gauge modal -->
    <button
      v-if="reach.gauge_id"
      class="flex items-center gap-1.5 shrink-0 rounded-md px-1.5 py-0.5 hover:bg-gray-100 dark:hover:bg-gray-700/50 transition-colors cursor-pointer"
      @click.stop="$emit('openGauge', reach)"
    >
      <span
        v-if="reach.flow_status !== 'unknown' || reach.flow_label"
        :class="['inline-flex items-center rounded-md px-1.5 py-0.5 text-xs font-medium', flowBandBadgeClass(reach.flow_label, reach.flow_status)]"
      >{{ flowBandLabel(reach.flow_label, reach.flow_status) }}</span>
      <span class="text-sm font-bold tabular-nums text-gray-900 dark:text-white">
        {{ displayCfs != null ? displayCfs.toLocaleString() : '—' }}
      </span>
      <span class="text-xs text-gray-400">cfs</span>
    </button>

    <!-- No gauge fallback -->
    <span v-else class="text-xs text-gray-400 shrink-0">No gauge</span>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { classRange } from '~/utils/classRating'
import { flowBandBadgeClass, flowBandLabel } from '~/utils/flowBand'

export interface ReachListItem {
  slug: string
  river_name: string | null
  common_name: string | null
  put_in_name: string | null
  take_out_name: string | null
  basin: string | null
  class_min: number | null
  class_max: number | null
  current_cfs: number | null
  flow_label: string | null
  flow_status: 'runnable' | 'caution' | 'flood' | 'unknown'
  gauge_id: string | null
  gauge_external_id: string | null
  gauge_source: string | null
  gauge_name: string | null
}

const props = defineProps<{ reach: ReachListItem }>()

defineEmits<{ (e: 'openGauge', reach: ReachListItem): void }>()

const liveCfs = ref<number | null>(null)

const displayCfs = computed(() => liveCfs.value ?? props.reach.current_cfs)

const displayName = computed(() =>
  props.reach.common_name
    ?? (props.reach.put_in_name && props.reach.take_out_name
      ? `${props.reach.put_in_name} to ${props.reach.take_out_name}`
      : props.reach.slug)
)

const classDisplay = computed(() => classRange(props.reach.class_min, props.reach.class_max))
</script>
