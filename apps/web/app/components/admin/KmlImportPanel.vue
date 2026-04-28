<template>
  <div class="space-y-4">
    <div>
      <h2 class="text-sm font-semibold text-gray-700 dark:text-gray-200 mb-1">Import KMZ / KML</h2>
      <p class="text-xs text-gray-400 mb-3">Upload a KMZ or KML file to enrich existing reaches with access points, rapids, and hazards. Each folder must include a slug placemark matching a reach in the database. Note: this is a global import that may update any reach.</p>
      <div class="flex items-center gap-3">
        <UButton
          :loading="importing"
          icon="i-heroicons-arrow-up-tray"
          @click="triggerKmlUpload"
        >{{ importing ? 'Importing…' : 'Choose KMZ / KML' }}</UButton>
        <button
          class="text-sm text-blue-500 hover:text-blue-400 font-medium transition-colors"
          @click="showKmlGuide = !showKmlGuide"
        >{{ showKmlGuide ? 'Hide guide' : 'KML Format Guide' }}</button>
        <span v-if="importMsg" class="text-sm" :class="importError ? 'text-red-500' : 'text-green-600'">{{ importMsg }}</span>
      </div>
      <input ref="kmlInputRef" type="file" accept=".kmz,.kml" class="hidden" @change="onKmlSelected" />
    </div>

    <!-- KML Format Guide -->
    <div v-if="showKmlGuide" class="bg-gray-50 dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-lg px-4 py-4 text-xs text-gray-600 dark:text-gray-400 space-y-4">
      <div>
        <p class="font-semibold text-gray-700 dark:text-gray-200 mb-1">Document / folder structure</p>
        <ul class="list-disc pl-4 space-y-0.5">
          <li><strong>One folder per reach</strong> — folder name is informational; the <strong>slug placemark</strong> is what links it to the DB</li>
          <li><strong>Slug placemark</strong> (required): a coordinate-less <code>&lt;Placemark&gt;&lt;name&gt;slug&lt;/name&gt;&lt;description&gt;reach-slug-here&lt;/description&gt;&lt;/Placemark&gt;</code> inside the folder</li>
          <li>Folders missing a slug placemark are skipped with a warning — create the reach first</li>
          <li>River name and basin are set from NHD data when the reach is created — no document-level overrides needed</li>
        </ul>
      </div>
      <div>
        <p class="font-semibold text-gray-700 dark:text-gray-200 mb-1">Metadata placemarks (coordinate-less)</p>
        <p class="text-gray-400">Keys: <code>common_name</code>, <code>description</code>, <code>min_class</code>, <code>max_class</code>, <code>gauge</code>, <code>permit_required</code>, <code>multi_day</code></p>
        <p class="mt-1 text-gray-400">Flow bands: <code>below</code> (upper Too Low CFS), <code>running</code> (min,max), <code>high</code> (min,max), <code>above</code> (lower Very High CFS)</p>
        <p class="mt-1 text-gray-400">Pin prefixes: <code>Rapid:</code>, <code>Wave:</code>, <code>Put-in:</code>, <code>Take-out:</code>, <code>Parking:</code>, <code>Campsite:</code>, <code>Hazard:</code></p>
      </div>
    </div>

    <!-- Import log -->
    <div v-if="importLog.length > 0">
      <div class="flex items-center justify-between mb-2">
        <p class="text-xs font-semibold text-gray-500 uppercase tracking-wide">Import log</p>
        <button class="text-xs text-gray-400 hover:text-gray-600 transition-colors" @click="importLog = []">Clear</button>
      </div>
      <div class="bg-gray-950 rounded-lg border border-gray-800 px-4 py-3 max-h-72 overflow-y-auto font-mono text-[11px] space-y-0.5">
        <p
          v-for="(line, i) in importLog"
          :key="i"
          :class="line.startsWith('✗') || line.startsWith('⚠') ? 'text-red-400' : line.startsWith('+') ? 'text-emerald-400' : line.startsWith('✓') ? 'text-gray-300' : 'text-gray-500'"
        >{{ line }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const { getToken } = useAuth()
const { apiBase } = useRuntimeConfig().public

const kmlInputRef  = ref<HTMLInputElement | null>(null)
const importing    = ref(false)
const importMsg    = ref('')
const importError  = ref(false)
const showKmlGuide = ref(false)
const importLog    = ref<string[]>([])

function triggerKmlUpload() {
  importMsg.value = ''
  importError.value = false
  importLog.value = []
  kmlInputRef.value?.click()
}

async function onKmlSelected(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0]
  if (!file) return
  ;(event.target as HTMLInputElement).value = ''
  importing.value = true
  importMsg.value = ''
  importError.value = false
  try {
    const token = await getToken()
    const form = new FormData()
    form.append('file', file)
    const res = await fetch(`${apiBase}/api/v1/import/kmz`, {
      method: 'POST',
      headers: token ? { Authorization: `Bearer ${token}` } : {},
      body: form,
    })
    const json = await res.json()
    if (!res.ok) {
      importError.value = true
      importMsg.value = json.error ?? `Error ${res.status}`
    } else {
      const reachCount = Object.keys(json.reaches ?? {}).length
      importMsg.value = `Imported ${reachCount} reach${reachCount !== 1 ? 'es' : ''}`
      importLog.value = json.log ?? []
    }
  } catch (err: any) {
    importError.value = true
    importMsg.value = err?.message ?? 'Upload failed'
  } finally {
    importing.value = false
  }
}
</script>
