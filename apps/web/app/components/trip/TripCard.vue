<template>
  <div
    class="rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 p-4 cursor-pointer hover:border-blue-200 dark:hover:border-blue-800 hover:shadow-sm transition-all"
    @click="emit('open')"
  >
    <!-- Header: reach name + date -->
    <div class="flex items-start justify-between gap-2 mb-3">
      <div class="min-w-0">
        <p class="font-semibold text-sm truncate">{{ title }}</p>
        <p class="text-xs text-gray-400 mt-0.5">{{ dateLabel }}</p>
      </div>
      <span
        v-if="trip.share_consent === true"
        class="shrink-0 text-xs font-medium text-blue-500 bg-blue-50 dark:bg-blue-950 rounded-full px-2 py-0.5"
      >Shared</span>
      <span
        v-else-if="trip.share_consent === false"
        class="shrink-0 text-xs font-medium text-gray-400 bg-gray-100 dark:bg-gray-800 rounded-full px-2 py-0.5"
      >Private</span>
    </div>

    <!-- Stats row -->
    <div class="flex items-center gap-4 text-xs text-gray-500">
      <span v-if="trip.duration_min != null" class="flex items-center gap-1">
        <svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="10" cy="10" r="8"/><path d="M10 6v4l2.5 2.5" stroke-linecap="round"/></svg>
        {{ durationLabel }}
      </span>
      <span v-if="trip.distance_mi != null" class="flex items-center gap-1">
        <svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M3 10 C5 5 8 3 10 3 S15 5 17 10 S15 17 10 17 S3 15 3 10Z" stroke-linecap="round"/></svg>
        {{ trip.distance_mi.toFixed(1) }} mi
      </span>
      <span v-if="trip.start_cfs != null" class="flex items-center gap-1">
        <svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M2 10 C5 6 8 6 10 10 S15 14 18 10" stroke-linecap="round"/></svg>
        {{ trip.start_cfs.toLocaleString() }} cfs
      </span>
    </div>

    <!-- Notes preview -->
    <p v-if="trip.notes" class="mt-2 text-xs text-gray-500 dark:text-gray-400 line-clamp-2">{{ trip.notes }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { TripSummary } from '~/composables/useTrips'

const props = defineProps<{ trip: TripSummary }>()
const emit  = defineEmits<{ (e: 'open'): void }>()

const title = computed(() =>
  props.trip.reach_name || props.trip.gauge_name || 'Unnamed trip'
)

const dateLabel = computed(() => {
  const d = new Date(props.trip.started_at)
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })
})

const durationLabel = computed(() => {
  const m = props.trip.duration_min!
  if (m < 60) return `${m}m`
  return `${Math.floor(m / 60)}h ${m % 60}m`
})
</script>
