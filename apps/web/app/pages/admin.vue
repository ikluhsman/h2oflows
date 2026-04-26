<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-950">
    <AppHeader />

    <main class="max-w-5xl mx-auto px-4 py-6 space-y-6">

      <!-- Not authorized -->
      <div v-if="!isDataAdmin && authReady" class="mt-20 flex flex-col items-center gap-3 text-center">
        <svg class="w-10 h-10 text-gray-300" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
        </svg>
        <h2 class="text-lg font-semibold">Access restricted</h2>
        <p class="text-sm text-gray-500">You need data admin or site admin permissions to view this page.</p>
      </div>

      <!-- Loading auth -->
      <div v-else-if="!authReady" class="mt-20 flex justify-center">
        <div class="w-6 h-6 rounded-full border-2 border-blue-500 border-t-transparent animate-spin" />
      </div>

      <!-- Admin UI -->
      <template v-else>
        <div class="flex items-center justify-between">
          <h1 class="text-xl font-bold text-gray-900 dark:text-white">Admin</h1>
        </div>

        <!-- Tabs -->
        <div class="flex gap-1 border-b border-gray-200 dark:border-gray-800">
          <button
            v-for="tab in visibleTabs" :key="tab.key"
            class="px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors"
            :class="activeTab === tab.key
              ? 'border-blue-500 text-blue-600 dark:text-blue-400'
              : 'border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300'"
            @click="activeTab = tab.key"
          >{{ tab.label }}</button>
        </div>

        <!-- Rivers tab -->
        <div v-if="activeTab === 'rivers'">
          <div class="flex items-center justify-between mb-4">
            <p class="text-sm text-gray-500">{{ rivers.length }} rivers</p>
            <UButton size="xs" icon="i-heroicons-plus" @click="createRiverOpen = true">New river</UButton>
          </div>

          <div v-if="riversLoading" class="space-y-2">
            <div v-for="i in 5" :key="i" class="h-12 rounded-lg bg-gray-100 dark:bg-gray-800 animate-pulse" />
          </div>

          <div v-else class="divide-y divide-gray-100 dark:divide-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
            <template v-for="river in rivers" :key="river.id">
              <!-- River row -->
              <div
                class="flex items-center gap-3 px-4 py-3 bg-white dark:bg-gray-900 hover:bg-gray-50 dark:hover:bg-gray-800/50 cursor-pointer transition-colors"
                @click="toggleRiver(river)"
              >
                <div class="flex-1 min-w-0">
                  <p class="text-sm font-medium text-gray-900 dark:text-white truncate">{{ river.name }}</p>
                  <p class="text-xs text-gray-400 truncate flex items-center gap-1">
                    <span>{{ river.slug }}</span>
                    <span v-if="river.gnis_id" class="text-gray-300">· gnis {{ river.gnis_id }}</span>
                  </p>
                </div>
                <span class="text-xs text-gray-400 shrink-0">{{ river.reach_count }} reach{{ river.reach_count !== 1 ? 'es' : '' }}</span>
                <svg
                  class="w-4 h-4 text-gray-400 shrink-0 transition-transform"
                  :class="expandedRiverId === river.id ? 'rotate-90' : ''"
                  viewBox="0 0 20 20" fill="currentColor"
                >
                  <path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd"/>
                </svg>
              </div>

              <!-- Expanded reaches -->
              <div v-if="expandedRiverId === river.id" class="bg-gray-50 dark:bg-gray-950 border-t border-gray-100 dark:border-gray-800">
                <div v-if="riverDetailLoading" class="px-6 py-4 text-xs text-gray-400 animate-pulse">Loading reaches…</div>
                <template v-else-if="selectedRiver">
                  <!-- Delete river -->
                  <div class="flex items-center justify-between px-6 py-2 border-b border-gray-100 dark:border-gray-800">
                    <span class="text-xs text-gray-400">{{ selectedRiver.reaches?.length ?? 0 }} reaches</span>
                    <UButton size="xs" variant="ghost" color="error" @click="deleteRiver(selectedRiver.slug, selectedRiver.name)">Delete river</UButton>
                  </div>
                  <!-- Reach rows -->
                  <div class="divide-y divide-gray-100 dark:divide-gray-800">
                    <div
                      v-for="reach in selectedRiver.reaches" :key="reach.id"
                      class="flex items-center gap-3 px-6 py-2.5 bg-white dark:bg-gray-900/60"
                    >
                      <div class="flex-1 min-w-0">
                        <p class="text-sm font-medium truncate">{{ reach.common_name ?? reach.name }}</p>
                        <p class="text-xs text-gray-400 truncate">{{ reach.slug }}</p>
                      </div>
                      <div class="flex items-center gap-2 shrink-0 flex-wrap justify-end">
                        <span
                          class="text-xs px-1.5 py-0.5 rounded"
                          :class="reach.has_centerline
                            ? 'bg-emerald-50 dark:bg-emerald-950/40 text-emerald-600 dark:text-emerald-400'
                            : 'bg-gray-100 dark:bg-gray-800 text-gray-400'"
                        >{{ reach.has_centerline ? 'Line ✓' : 'No line' }}</span>
                        <span v-if="centerlineErrors.get(reach.slug)" class="text-xs text-red-400">{{ centerlineErrors.get(reach.slug) }}</span>
                        <UButton size="xs" variant="outline" color="neutral" :loading="fetchingCenterlines.has(reach.slug)" @click="fetchCenterline(reach.slug)">Fetch line</UButton>
                        <UButton size="xs" variant="outline" color="error" @click="deleteReach(reach.slug, reach.common_name ?? reach.name)">Delete</UButton>
                        <button class="text-xs text-blue-500 hover:underline" @click="openReachInEditor(reach.slug)">Edit</button>
                      </div>
                    </div>
                    <div v-if="!selectedRiver.reaches?.length" class="px-6 py-4 text-center text-sm text-gray-400">No reaches linked to this river</div>
                  </div>
                </template>
              </div>
            </template>
            <div v-if="rivers.length === 0" class="px-4 py-8 text-center text-sm text-gray-400">No rivers yet</div>
          </div>
        </div>

        <!-- Import tab -->
        <div v-if="activeTab === 'import'">
          <div class="space-y-4">
            <div>
              <h2 class="text-sm font-semibold text-gray-700 dark:text-gray-200 mb-1">Import KMZ / KML</h2>
              <p class="text-xs text-gray-400 mb-3">Upload a KMZ or KML file to enrich existing reaches with access points, rapids, and hazards. Each folder must include a slug placemark matching a reach created in the Reaches tab.</p>
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
                  <li>Folders missing a slug placemark are skipped with a warning — create the reach in the Reaches tab first</li>
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
        </div>

        <!-- Reaches tab -->
        <div v-if="activeTab === 'nhd'">
          <div class="space-y-4">

            <!-- Mode switcher -->
            <div class="flex gap-2 border-b border-gray-100 dark:border-gray-800 pb-3">
              <UButton size="xs" :variant="nhdMode === 'explore' ? 'solid' : 'outline'" :color="nhdMode === 'explore' ? 'primary' : 'neutral'" @click="setNHDMode('explore')">Explore</UButton>
              <UButton size="xs" :variant="nhdMode === 'author' ? 'solid' : 'outline'" :color="nhdMode === 'author' ? 'primary' : 'neutral'" @click="setNHDMode('author')">New reach</UButton>
              <UButton size="xs" :variant="nhdMode === 'repin' ? 'solid' : 'outline'" :color="nhdMode === 'repin' ? 'primary' : 'neutral'" @click="setNHDMode('repin')">Load Reach</UButton>
            </div>

            <!-- ── EXPLORE MODE ─────────────────────────────────────────────── -->
            <div v-if="nhdMode === 'explore'">
              <p class="text-xs text-gray-400 mb-3">Click the map to snap a point to the nearest NHD reach. Upstream flowlines (blue), downstream mainstem (teal), and USGS gauges (amber) are drawn automatically.</p>

              <div class="flex flex-wrap items-end gap-3 mb-3">
                <div>
                  <label class="block text-xs text-gray-500 mb-1">Distance (km)</label>
                  <select v-model="nhdDistance" class="rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm">
                    <option value="50">50 km</option>
                    <option value="100">100 km</option>
                    <option value="150">150 km</option>
                    <option value="300">300 km</option>
                    <option value="500">500 km</option>
                  </select>
                </div>
                <UButton size="xs" :color="nhdPickMode ? 'primary' : 'neutral'" :variant="nhdPickMode ? 'solid' : 'outline'" @click="nhdPickMode = !nhdPickMode">
                  {{ nhdPickMode ? 'Cancel pick' : 'Pick point' }}
                </UButton>
                <UButton v-if="nhdSnap" size="xs" variant="ghost" color="neutral" @click="clearNHD">Clear</UButton>
              </div>

              <div v-if="nhdSnap" class="mb-3 flex items-center gap-3 px-3 py-2 rounded-lg bg-blue-50 dark:bg-blue-950 border border-blue-200 dark:border-blue-800 text-xs">
                <span class="w-2.5 h-2.5 rounded-full bg-blue-600 shrink-0" />
                <span class="font-medium text-blue-800 dark:text-blue-200">ComID {{ nhdSnap.comid }}</span>
                <span v-if="nhdSnap.name" class="text-blue-600 dark:text-blue-300">{{ nhdSnap.name }}</span>
                <span class="text-blue-400 font-mono ml-auto">{{ nhdSnap.lat.toFixed(5) }}, {{ nhdSnap.lng.toFixed(5) }}</span>
              </div>

              <div v-if="nhdLoading" class="h-120 rounded-xl bg-gray-100 dark:bg-gray-800 animate-pulse flex items-center justify-center text-sm text-gray-400">Fetching NHD data…</div>
              <div v-else-if="nhdError" class="h-32 rounded-xl border border-red-200 dark:border-red-800 flex items-center justify-center text-sm text-red-500">{{ nhdError }}</div>
              <NHDExplorerMap
                v-else
                :upstream-flowlines="nhdUpstream"
                :downstream-flowlines="nhdDownstream"
                :upstream-gauges="nhdGauges"
                :snap-lat="nhdSnap?.lat ?? null"
                :snap-lng="nhdSnap?.lng ?? null"
                :pick-mode="nhdPickMode"
                @pick="onNHDPick"
              />

              <div v-if="nhdGaugeList.length > 0" class="mt-3">
                <p class="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-1">Upstream USGS gauges</p>
                <div class="divide-y divide-gray-100 dark:divide-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
                  <div v-for="g in nhdGaugeList" :key="g.id" class="flex items-center gap-3 px-3 py-2 bg-white dark:bg-gray-900 text-xs">
                    <span class="w-2 h-2 rounded-full bg-amber-400 shrink-0" />
                    <span class="font-medium text-gray-800 dark:text-gray-100 flex-1 truncate">{{ g.name || g.id }}</span>
                    <span class="text-gray-400 font-mono shrink-0">{{ g.id }}</span>
                  </div>
                </div>
              </div>
            </div>

            <!-- ── AUTHOR MODE ──────────────────────────────────────────────── -->
            <div v-if="nhdMode === 'author'">
              <p class="text-xs text-gray-400 mb-3">Pick an anchor point near the reach to load NHD tributary flowlines. Then click flowline segments to select the upstream and downstream ComIDs — no access-point coordinates needed yet.</p>

              <!-- Anchor controls -->
              <div class="flex flex-wrap items-center gap-2 mb-3">
                <UButton
                  size="xs"
                  :color="authorPickMode ? 'primary' : 'neutral'"
                  :variant="authorPickMode ? 'solid' : 'outline'"
                  @click="authorPickMode = !authorPickMode"
                >{{ authorPickMode ? 'Cancel' : 'Pick anchor point' }}</UButton>
                <UButton v-if="authorAnchorSnap" size="xs" variant="ghost" color="neutral" @click="resetAuthor">Clear</UButton>
                <span v-if="authorAnchorSnapping" class="text-xs text-blue-600 dark:text-blue-400 animate-pulse">Snapping to NHD…</span>
              </div>

              <!-- Anchor badge -->
              <div v-if="authorAnchorSnap" class="mb-3 flex items-center gap-3 px-3 py-2 rounded-lg bg-blue-50 dark:bg-blue-950 border border-blue-200 dark:border-blue-800 text-xs">
                <span class="w-2.5 h-2.5 rounded-full bg-blue-600 shrink-0" />
                <span class="font-medium text-blue-800 dark:text-blue-200">Anchor ComID {{ authorAnchorSnap.comid }}</span>
                <span v-if="authorAnchorSnap.name" class="text-blue-600 dark:text-blue-300">{{ authorAnchorSnap.name }}</span>
              </div>

              <!-- ComID slot selector -->
              <div v-if="authorTributaries" class="flex items-center gap-3 mb-3 text-xs">
                <span class="text-gray-500 shrink-0">Click flowline for:</span>
                <button
                  class="flex items-center gap-1.5 px-2 py-1 rounded-md border transition-colors"
                  :class="authorComIDSlot === 'up' ? 'border-green-500 bg-green-50 dark:bg-green-950 text-green-700 dark:text-green-300 font-medium' : 'border-gray-200 dark:border-gray-700 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300'"
                  @click="authorComIDSlot = 'up'"
                >
                  <span class="w-2 h-2 rounded-full bg-green-500 shrink-0" />
                  Upstream<template v-if="authorUpComID"> · <span class="font-mono">{{ authorUpComID }}</span></template>
                </button>
                <button
                  class="flex items-center gap-1.5 px-2 py-1 rounded-md border transition-colors"
                  :class="authorComIDSlot === 'down' ? 'border-red-500 bg-red-50 dark:bg-red-950 text-red-700 dark:text-red-300 font-medium' : 'border-gray-200 dark:border-gray-700 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300'"
                  @click="authorComIDSlot = 'down'"
                >
                  <span class="w-2 h-2 rounded-full bg-red-500 shrink-0" />
                  Downstream<template v-if="authorDownComID"> · <span class="font-mono">{{ authorDownComID }}</span></template>
                </button>
              </div>

              <NHDExplorerMap
                :upstream-flowlines="authorTributaries"
                :downstream-flowlines="authorDownstreamFlowlines"
                :upstream-gauges="null"
                :snap-lat="null"
                :snap-lng="null"
                :pick-mode="authorPickMode"
                :comid-select-mode="!!authorAnchorSnap && !authorPickMode"
                :comid-select-slot="authorComIDSlot"
                :selected-up-comid="authorUpComID"
                :selected-down-comid="authorDownComID"
                :put-in-pin="authorPutInPin"
                :take-out-pin="authorTakeOutPin"
                :disable-auto-fit="true"
                @pick="onAuthorAnchorPick"
                @comid-select="onAuthorComIDSelect"
              />
              <p v-if="authorDownstreamLoading" class="text-xs text-blue-500 dark:text-blue-400 mt-1 animate-pulse">Loading downstream mainstem…</p>

              <!-- Reach form — shown once both ComIDs selected -->
              <div v-if="authorUpComID && authorDownComID" class="mt-4 space-y-3 rounded-xl border border-gray-200 dark:border-gray-700 p-4 bg-white dark:bg-gray-900">
                <h3 class="text-sm font-semibold text-gray-800 dark:text-gray-100">New reach details</h3>

                <!-- ComID summary -->
                <div class="grid grid-cols-2 gap-2 text-xs">
                  <div class="flex items-center gap-2 px-2 py-1.5 rounded-lg bg-green-50 dark:bg-green-950 border border-green-200 dark:border-green-800">
                    <span class="w-2 h-2 rounded-full bg-green-600 shrink-0" />
                    <div>
                      <div class="font-medium text-green-800 dark:text-green-200">Upstream (put-in) ComID</div>
                      <div class="text-green-600 dark:text-green-400 font-mono">{{ authorUpComID }}</div>
                    </div>
                  </div>
                  <div class="flex items-center gap-2 px-2 py-1.5 rounded-lg bg-red-50 dark:bg-red-950 border border-red-200 dark:border-red-800">
                    <span class="w-2 h-2 rounded-full bg-red-600 shrink-0" />
                    <div>
                      <div class="font-medium text-red-800 dark:text-red-200">Downstream (take-out) ComID</div>
                      <div class="text-red-600 dark:text-red-400 font-mono">{{ authorDownComID }}</div>
                    </div>
                  </div>
                </div>

                <!-- River name (auto from NHD snap, read-only) -->
                <div>
                  <label class="block text-xs text-gray-500 mb-1">River name <span class="text-gray-300">(from NHD)</span></label>
                  <div class="flex items-center gap-2">
                    <input
                      v-model="authorForm.riverName"
                      :readonly="!authorRiverNameOverride"
                      class="flex-1 rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm"
                      :class="authorRiverNameOverride ? '' : 'text-gray-400 dark:text-gray-500 cursor-default'"
                      placeholder="Auto-filled from anchor snap"
                    />
                    <button class="text-xs text-blue-500 hover:text-blue-400 shrink-0" @click="authorRiverNameOverride = !authorRiverNameOverride">
                      {{ authorRiverNameOverride ? 'Lock' : 'Override' }}
                    </button>
                  </div>
                </div>

                <div>
                  <label class="block text-xs text-gray-500 mb-1">Reach name <span class="text-red-400">*</span></label>
                  <input v-model="authorForm.name" class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm" placeholder="e.g. Lees Ferry to Diamond Creek" />
                  <p v-if="authorComputedSlug" class="mt-1 text-xs text-gray-400 font-mono">slug: {{ authorComputedSlug }}</p>
                </div>

                <div>
                  <label class="block text-xs text-gray-500 mb-1">Common name</label>
                  <input v-model="authorForm.commonName" class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm" placeholder="e.g. Grand Canyon" />
                </div>

                <div class="grid grid-cols-2 gap-3">
                  <div>
                    <label class="block text-xs text-gray-500 mb-1">Class min</label>
                    <input v-model.number="authorForm.classMin" type="number" min="1" max="6" step="0.5" class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm" placeholder="3" />
                  </div>
                  <div>
                    <label class="block text-xs text-gray-500 mb-1">Class max</label>
                    <input v-model.number="authorForm.classMax" type="number" min="1" max="6" step="0.5" class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm" placeholder="5" />
                  </div>
                </div>

                <div>
                  <label class="block text-xs text-gray-500 mb-1">Description</label>
                  <textarea
                    v-model="authorForm.description"
                    rows="4"
                    class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm resize-y"
                    placeholder="Overview of the reach — character, season, access notes…"
                  />
                </div>

                <div class="grid grid-cols-2 gap-3">
                  <div>
                    <label class="block text-xs text-gray-500 mb-1">Multi-day (days)</label>
                    <input
                      v-model.number="authorForm.multiDay"
                      type="number" min="1" max="30"
                      class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm"
                      placeholder="1"
                    />
                    <p class="text-xs text-gray-400 mt-0.5">1 = single day</p>
                  </div>
                  <div class="flex items-start pt-5">
                    <label class="flex items-center gap-2 cursor-pointer select-none text-sm text-gray-700 dark:text-gray-300">
                      <input type="checkbox" v-model="authorForm.permitRequired" class="rounded" />
                      Permit required
                    </label>
                  </div>
                </div>

                <!-- Flow ranges -->
                <div class="rounded-lg border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/50 p-3 space-y-2">
                  <p class="text-xs font-semibold text-gray-500 uppercase tracking-wide">Flow ranges (CFS)</p>
                  <div class="grid grid-cols-2 gap-2 text-xs">
                    <div v-for="band in authorFlowBands" :key="band.key" class="space-y-1">
                      <div class="flex items-center gap-1.5">
                        <span class="w-2 h-2 rounded-full shrink-0" :style="{ background: band.dot }" />
                        <span class="font-medium text-gray-600 dark:text-gray-300">{{ band.label }}</span>
                      </div>
                      <div class="flex items-center gap-1">
                        <input
                          v-model.number="authorForm.flowRanges[band.key].min"
                          type="number" min="0"
                          class="w-full rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-1.5 py-1 text-xs"
                          :placeholder="band.showMin ? 'min' : '—'"
                          :disabled="!band.showMin"
                        />
                        <span class="text-gray-300 shrink-0">–</span>
                        <input
                          v-model.number="authorForm.flowRanges[band.key].max"
                          type="number" min="0"
                          class="w-full rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-1.5 py-1 text-xs"
                          :placeholder="band.showMax ? 'max' : '—'"
                          :disabled="!band.showMax"
                        />
                      </div>
                    </div>
                  </div>
                </div>

                <div v-if="authorError" class="text-xs text-red-500">{{ authorError }}</div>
                <div v-if="authorSuccess" class="text-xs text-green-600 dark:text-green-400">{{ authorSuccess }}</div>

                <div class="flex gap-2 justify-end pt-1">
                  <UButton size="sm" variant="ghost" color="neutral" @click="resetAuthor">Cancel</UButton>
                  <UButton size="sm" :loading="authorSaving" :disabled="!authorForm.name.trim()" @click="submitAuthorReach">Save reach</UButton>
                </div>
              </div>
            </div>

            <!-- ── RE-PIN EXISTING MODE ────────────────────────────────────────── -->
            <div v-if="nhdMode === 'repin'">
              <p class="text-xs text-gray-400 mb-3">Enter a reach slug to load it for editing — flow lines, metadata, and description.</p>

              <!-- Reach selector -->
              <div class="flex gap-2 mb-3">
                <input
                  v-model="repinSlug"
                  class="flex-1 rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm"
                  placeholder="Reach slug (e.g. colorado-gore-canyon)"
                  @keydown.enter="loadRepinReach"
                />
                <UButton size="sm" :loading="repinLoadingReach" @click="loadRepinReach">Load Reach</UButton>
              </div>
              <div v-if="repinLoadError" class="text-xs text-red-500 mb-2">{{ repinLoadError }}</div>

              <div v-if="repinReach" class="space-y-4">
                <h3 class="text-sm font-semibold text-gray-800 dark:text-gray-100">{{ repinReach.name }}<span v-if="repinReach.river_name" class="text-gray-400 font-normal"> · {{ repinReach.river_name }}</span></h3>

                <!-- Metadata form -->
                <div class="rounded-xl border border-gray-200 dark:border-gray-700 p-4 bg-white dark:bg-gray-900 space-y-3">
                  <p class="text-xs font-semibold text-gray-500 uppercase tracking-wide">Metadata</p>
                  <div>
                    <label class="block text-xs text-gray-500 mb-1">Reach name <span class="text-red-400">*</span></label>
                    <input v-model="repinForm.name" class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm" />
                  </div>
                  <div>
                    <label class="block text-xs text-gray-500 mb-1">Common name</label>
                    <input v-model="repinForm.commonName" class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm" />
                  </div>
                  <div>
                    <label class="block text-xs text-gray-500 mb-1">River name</label>
                    <div class="flex gap-2">
                      <input v-model="repinForm.riverName" class="flex-1 rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm" />
                      <UButton size="xs" variant="outline" color="neutral" :loading="repinRiverNameFetching" :disabled="!repinUpComID" @click="fetchRepinRiverName">Fetch from NLDI</UButton>
                    </div>
                  </div>
                  <div>
                    <label class="block text-xs text-gray-500 mb-1">Slug</label>
                    <input v-model="repinForm.slug" class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-xs font-mono" />
                    <p class="text-xs text-gray-400 mt-0.5">Changing the slug will break existing links.</p>
                  </div>
                  <div class="grid grid-cols-2 gap-3">
                    <div>
                      <label class="block text-xs text-gray-500 mb-1">Class min</label>
                      <input v-model.number="repinForm.classMin" type="number" min="1" max="6" step="0.5" class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm" />
                    </div>
                    <div>
                      <label class="block text-xs text-gray-500 mb-1">Class max</label>
                      <input v-model.number="repinForm.classMax" type="number" min="1" max="6" step="0.5" class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm" />
                    </div>
                  </div>
                  <div class="grid grid-cols-2 gap-3">
                    <div>
                      <label class="block text-xs text-gray-500 mb-1">Multi-day (days)</label>
                      <input v-model.number="repinForm.multiDay" type="number" min="1" max="30" class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm" />
                      <p class="text-xs text-gray-400 mt-0.5">1 = single day</p>
                    </div>
                    <div class="flex items-start pt-5">
                      <label class="flex items-center gap-2 cursor-pointer select-none text-sm text-gray-700 dark:text-gray-300">
                        <input type="checkbox" v-model="repinForm.permitRequired" class="rounded" />
                        Permit required
                      </label>
                    </div>
                  </div>
                  <div class="flex items-center gap-3 pt-1">
                    <span v-if="repinMetaMsg" class="text-xs" :class="repinMetaMsg === 'Saved' ? 'text-green-600 dark:text-green-400' : 'text-red-500'">{{ repinMetaMsg }}</span>
                    <div class="flex-1" />
                    <UButton size="sm" :loading="repinMetaSaving" :disabled="!repinForm.name.trim()" @click="saveRepinMeta">Save metadata</UButton>
                  </div>
                </div>

                <!-- Description editor -->
                <div class="rounded-xl border border-gray-200 dark:border-gray-700 p-4 bg-white dark:bg-gray-900 space-y-2">
                  <div class="flex items-center justify-between">
                    <p class="text-xs font-semibold text-gray-500 uppercase tracking-wide">Description</p>
                    <UButton size="xs" variant="outline" color="neutral" :loading="repinDescGenerating" @click="generateRepinDescription">Generate with AI</UButton>
                  </div>
                  <textarea
                    v-model="repinDescEdit"
                    rows="5"
                    class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm resize-y"
                    placeholder="No description yet — click Generate to create one with AI, or type directly."
                  />
                  <div class="flex items-center gap-3">
                    <span v-if="repinDescMsg" class="text-xs" :class="repinDescMsg === 'Description saved' ? 'text-green-600 dark:text-green-400' : 'text-red-500'">{{ repinDescMsg }}</span>
                    <div class="flex-1" />
                    <UButton size="xs" :loading="repinDescSaving" @click="saveRepinDescription">Save description</UButton>
                  </div>
                </div>

                <!-- Flow bands editor -->
                <div class="rounded-xl border border-gray-200 dark:border-gray-700 p-4 bg-white dark:bg-gray-900 space-y-3">
                  <p class="text-xs font-semibold text-gray-500 uppercase tracking-wide">Flow bands</p>
                  <div class="space-y-2">
                    <div v-for="band in repinFlowBandsDef" :key="band.key" class="flex items-center gap-2 text-xs">
                      <span class="w-2.5 h-2.5 rounded-full shrink-0" :style="`background:${band.dot}`" />
                      <span class="w-20 shrink-0 text-gray-600 dark:text-gray-400">{{ band.label }}</span>
                      <template v-if="band.showMin">
                        <input
                          v-model.number="repinFlowBands[band.key].min"
                          type="number" min="0" placeholder="min cfs"
                          class="w-24 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-1.5 py-1 text-xs"
                        />
                      </template>
                      <span v-else class="w-24" />
                      <span class="text-gray-400">–</span>
                      <template v-if="band.showMax">
                        <input
                          v-model.number="repinFlowBands[band.key].max"
                          type="number" min="0" placeholder="max cfs"
                          class="w-24 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-1.5 py-1 text-xs"
                        />
                      </template>
                      <span v-else class="w-24 text-gray-400 italic">no limit</span>
                    </div>
                  </div>
                  <div class="flex items-center gap-3 pt-1">
                    <span v-if="repinFlowBandsMsg" class="text-xs" :class="repinFlowBandsMsg === 'Saved' ? 'text-green-600 dark:text-green-400' : 'text-red-500'">{{ repinFlowBandsMsg }}</span>
                    <div class="flex-1" />
                    <UButton size="xs" :loading="repinFlowBandsSaving" @click="saveRepinFlowBands">Save flow bands</UButton>
                  </div>
                </div>

                <!-- Flow lines / ComIDs -->
                <div class="rounded-xl border border-gray-200 dark:border-gray-700 p-4 bg-white dark:bg-gray-900 space-y-3">
                  <p class="text-xs font-semibold text-gray-500 uppercase tracking-wide">Flow lines</p>

                  <div class="grid grid-cols-2 gap-2 text-xs">
                    <div class="flex items-center gap-2 px-2 py-1.5 rounded-lg bg-green-50 dark:bg-green-950 border border-green-200 dark:border-green-800">
                      <span class="w-2 h-2 rounded-full bg-green-600 shrink-0" />
                      <div class="min-w-0 flex-1">
                        <div class="font-medium text-green-800 dark:text-green-200">Put-In</div>
                        <div class="text-green-600 dark:text-green-400 font-mono truncate">{{ repinUpComID || '—' }}</div>
                      </div>
                      <UButton size="xs"
                        :variant="repinComIDEditMode === 'up' ? 'solid' : 'outline'"
                        :color="repinComIDEditMode === 'up' ? 'primary' : 'neutral'"
                        @click="repinComIDEditMode = repinComIDEditMode === 'up' ? null : 'up'"
                      >{{ repinComIDEditMode === 'up' ? 'Cancel' : 'Set Put-In Flow Line' }}</UButton>
                    </div>
                    <div class="flex items-center gap-2 px-2 py-1.5 rounded-lg bg-red-50 dark:bg-red-950 border border-red-200 dark:border-red-800">
                      <span class="w-2 h-2 rounded-full bg-red-600 shrink-0" />
                      <div class="min-w-0 flex-1">
                        <div class="font-medium text-red-800 dark:text-red-200">Take-Out</div>
                        <div class="text-red-600 dark:text-red-400 font-mono truncate">{{ repinDownComID || '—' }}</div>
                      </div>
                      <UButton size="xs"
                        :variant="repinComIDEditMode === 'down' ? 'solid' : 'outline'"
                        :color="repinComIDEditMode === 'down' ? 'primary' : 'neutral'"
                        @click="repinComIDEditMode = repinComIDEditMode === 'down' ? null : 'down'"
                      >{{ repinComIDEditMode === 'down' ? 'Cancel' : 'Set Take-Out Flow Line' }}</UButton>
                    </div>
                  </div>

                  <p v-if="repinFlowlinesLoading" class="text-xs text-blue-500 animate-pulse">Loading downstream mainstem…</p>
                  <p v-if="repinComIDEditMode" class="text-xs text-blue-600 dark:text-blue-400">Click any flowline segment on the map to set the {{ repinComIDEditMode === 'up' ? 'put-in' : 'take-out' }} flow line.</p>

                  <NHDExplorerMap
                    :upstream-flowlines="null"
                    :downstream-flowlines="repinDownstream"
                    :upstream-gauges="null"
                    :snap-lat="null"
                    :snap-lng="null"
                    :put-in-pin="repinPutInPin"
                    :take-out-pin="repinTakeOutPin"
                    :comid-select-mode="!!repinComIDEditMode"
                    :comid-select-slot="repinComIDEditMode"
                    :selected-up-comid="repinUpComID"
                    :selected-down-comid="repinDownComID"
                    @comid-select="onRepinComIDSelect"
                  />

                  <div class="flex items-center gap-3 pt-1">
                    <span v-if="repinError" class="text-xs text-red-500">{{ repinError }}</span>
                    <span v-if="repinSuccess" class="text-xs text-green-600 dark:text-green-400">{{ repinSuccess }}</span>
                    <div class="flex-1" />
                    <UButton size="sm" variant="outline" color="neutral" @click="resetRepinComIDs" v-if="repinComIDsDirty">Revert</UButton>
                    <UButton size="sm" :loading="repinSaving" :disabled="!repinComIDsDirty || !repinUpComID || !repinDownComID" @click="submitRepinByComID">Save flow lines</UButton>
                  </div>
                </div>
              </div>
            </div>

          </div>
        </div>

        <!-- Users tab (site admin only) -->
        <div v-if="activeTab === 'users'">
          <div class="flex items-center justify-between mb-4">
            <p class="text-sm text-gray-500">Role assignments</p>
            <UButton size="xs" icon="i-heroicons-plus" @click="assignRoleOpen = true">Assign role</UButton>
          </div>

          <div v-if="rolesLoading" class="space-y-2">
            <div v-for="i in 3" :key="i" class="h-12 rounded-lg bg-gray-100 dark:bg-gray-800 animate-pulse" />
          </div>

          <div v-else class="divide-y divide-gray-100 dark:divide-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
            <div
              v-for="ur in userRoles" :key="ur.id"
              class="flex items-center gap-3 px-4 py-3 bg-white dark:bg-gray-900"
            >
              <div class="flex-1 min-w-0">
                <p class="text-xs font-mono text-gray-500 truncate">{{ ur.user_id }}</p>
                <p class="text-xs text-gray-400">
                  <span class="font-medium text-gray-700 dark:text-gray-300">{{ ur.role }}</span>
                  <template v-if="ur.river_name"> · {{ ur.river_name }}</template>
                </p>
              </div>
              <button
                class="text-xs text-red-400 hover:text-red-600 transition-colors shrink-0 px-2 py-1 rounded"
                @click="revokeRole(ur.id)"
              >Revoke</button>
            </div>
            <div v-if="userRoles.length === 0" class="px-4 py-8 text-center text-sm text-gray-400">No role assignments</div>
          </div>
        </div>
      </template>
    </main>

    <!-- Create river modal -->
    <UModal v-model:open="createRiverOpen" title="New river">
      <template #body>
        <div class="space-y-3">
          <UFormField label="Name">
            <UInput v-model="newRiver.name" placeholder="Arkansas River" @input="autoSlug" />
          </UFormField>
          <UFormField label="Slug">
            <UInput v-model="newRiver.slug" placeholder="arkansas-river" />
          </UFormField>
          <UFormField label="Basin (optional)">
            <UInput v-model="newRiver.basin" placeholder="Arkansas River Basin" />
          </UFormField>
          <UFormField label="State (optional)">
            <UInput v-model="newRiver.state_abbr" placeholder="CO" class="max-w-20" />
          </UFormField>
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton variant="ghost" color="neutral" @click="createRiverOpen = false">Cancel</UButton>
          <UButton :loading="createLoading" @click="createRiver">Create</UButton>
        </div>
      </template>
    </UModal>

    <!-- Assign role modal -->
    <UModal v-model:open="assignRoleOpen" title="Assign role">
      <template #body>
        <div class="space-y-3">
          <UFormField label="User ID (Supabase UUID)">
            <UInput v-model="newRole.user_id" placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" />
          </UFormField>
          <UFormField label="Role">
            <select v-model="newRole.role" class="w-full rounded-md border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-900 px-3 py-2 text-sm">
              <option value="data_admin">data_admin</option>
              <option value="site_admin">site_admin</option>
            </select>
          </UFormField>
          <UFormField label="River (optional — leave blank for global)">
            <select v-model="newRole.river_id" class="w-full rounded-md border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-900 px-3 py-2 text-sm">
              <option value="">Global (all rivers)</option>
              <option v-for="r in rivers" :key="r.id" :value="r.id">{{ r.name }}</option>
            </select>
          </UFormField>
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton variant="ghost" color="neutral" @click="assignRoleOpen = false">Cancel</UButton>
          <UButton :loading="assignLoading" @click="assignRole">Assign</UButton>
        </div>
      </template>
    </UModal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'

definePageMeta({ ssr: false })

const { isAdmin, isDataAdmin, loadAdminRoles, getToken } = useAuth()
const { apiBase } = useRuntimeConfig().public

// Auth readiness — wait for roles to load before showing gated UI
const authReady = ref(false)
onMounted(async () => {
  await loadAdminRoles()
  authReady.value = true
  if (isDataAdmin.value) {
    loadRivers()
    if (isAdmin.value) loadUserRoles()
  }
})

// Hard-refresh race: Supabase restores the session asynchronously, so
// user.value may be null when onMounted runs. Once isDataAdmin flips to
// true we trigger a load if rivers haven't been fetched yet.
watch(isDataAdmin, (val) => {
  if (val && authReady.value && !riversLoading.value && rivers.value.length === 0) {
    loadRivers()
    if (isAdmin.value) loadUserRoles()
  }
})

// ── Tabs ──────────────────────────────────────────────────────────────────────
const activeTab = ref('rivers')
const visibleTabs = computed(() => {
  const tabs = [
    { key: 'rivers', label: 'Rivers' },
    { key: 'nhd',    label: 'Reaches' },
    { key: 'import', label: 'Metadata' },
  ]
  if (isAdmin.value) tabs.push({ key: 'users', label: 'Users' })
  return tabs
})

// ── Rivers ────────────────────────────────────────────────────────────────────
interface River { id: string; slug: string; name: string; gnis_id: string | null; basin: string | null; basin_locked: boolean; state_abbr: string | null; reach_count: number }
interface RiverDetail extends River {
  reaches: { id: string; slug: string; name: string; common_name: string | null; has_centerline: boolean }[]
}

const rivers = ref<River[]>([])
const riversLoading = ref(false)
const expandedRiverId = ref<string | null>(null)
const selectedRiver = ref<RiverDetail | null>(null)
const riverDetailLoading = ref(false)

async function loadRivers() {
  riversLoading.value = true
  const token = await getToken()
  if (!token) { riversLoading.value = false; return }
  try {
    const res = await fetch(`${apiBase}/api/v1/admin/rivers`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    if (res.ok) rivers.value = await res.json()
  } finally {
    riversLoading.value = false
  }
}

async function toggleRiver(river: River) {
  if (expandedRiverId.value === river.id) {
    expandedRiverId.value = null
    selectedRiver.value = null
    return
  }
  expandedRiverId.value = river.id
  selectedRiver.value = null
  riverDetailLoading.value = true
  try {
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/admin/rivers/${river.slug}`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    if (res.ok) selectedRiver.value = await res.json()
  } finally {
    riverDetailLoading.value = false
  }
}

const fetchingCenterlines = ref<Set<string>>(new Set())
const centerlineErrors = ref<Map<string, string>>(new Map())

async function fetchCenterline(reachSlug: string) {
  fetchingCenterlines.value = new Set([...fetchingCenterlines.value, reachSlug])
  centerlineErrors.value = new Map([...centerlineErrors.value].filter(([k]) => k !== reachSlug))
  try {
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/reaches/${reachSlug}/fetch-centerline`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
    })
    if (!res.ok) {
      const body = await res.json().catch(() => ({}))
      const msg = body.error ?? `Error ${res.status}`
      centerlineErrors.value = new Map([...centerlineErrors.value, [reachSlug, msg]])
    } else if (selectedRiver.value) {
      // Refresh expanded detail
      const token2 = await getToken()
      const r2 = await fetch(`${apiBase}/api/v1/admin/rivers/${selectedRiver.value.slug}`, {
        headers: { Authorization: `Bearer ${token2}` },
      })
      if (r2.ok) selectedRiver.value = await r2.json()
    }
  } catch (err: any) {
    centerlineErrors.value = new Map([...centerlineErrors.value, [reachSlug, err?.message ?? 'Failed']])
  } finally {
    const s = new Set(fetchingCenterlines.value)
    s.delete(reachSlug)
    fetchingCenterlines.value = s
  }
}

async function deleteReach(reachSlug: string, displayName: string) {
  if (!confirm(`Permanently delete "${displayName}"?\n\nThis removes all rapids, access points, and features. Gauges are unlinked but kept.`)) return
  const token = await getToken()
  const res = await fetch(`${apiBase}/api/v1/reaches/${reachSlug}`, {
    method: 'DELETE',
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) { alert(`Delete failed: ${res.status}`); return }
  // Refresh expanded river detail
  if (selectedRiver.value) {
    const token2 = await getToken()
    const r2 = await fetch(`${apiBase}/api/v1/admin/rivers/${selectedRiver.value.slug}`, {
      headers: { Authorization: `Bearer ${token2}` },
    })
    if (r2.ok) selectedRiver.value = await r2.json()
  }
  loadRivers()
}

async function deleteRiver(riverSlug: string, riverName: string) {
  if (!confirm(`Permanently delete "${riverName}"?\n\nAll reaches will be unlinked but not deleted.`)) return
  const token = await getToken()
  const res = await fetch(`${apiBase}/api/v1/admin/rivers/${riverSlug}`, {
    method: 'DELETE',
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) { alert(`Delete failed: ${res.status}`); return }
  expandedRiverId.value = null
  selectedRiver.value = null
  loadRivers()
}

// Create river
const createRiverOpen = ref(false)
const createLoading = ref(false)
const newRiver = ref({ name: '', slug: '', basin: '', state_abbr: '' })

function autoSlug() {
  newRiver.value.slug = newRiver.value.name
    .toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')
}

async function createRiver() {
  createLoading.value = true
  const token = await getToken()
  try {
    const res = await fetch(`${apiBase}/api/v1/admin/rivers`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({
        slug: newRiver.value.slug,
        name: newRiver.value.name,
        basin: newRiver.value.basin || null,
        state_abbr: newRiver.value.state_abbr || null,
      }),
    })
    if (res.ok) {
      createRiverOpen.value = false
      newRiver.value = { name: '', slug: '', basin: '', state_abbr: '' }
      loadRivers()
    }
  } finally {
    createLoading.value = false
  }
}

// ── User Roles ────────────────────────────────────────────────────────────────
interface UserRole { id: string; user_id: string; role: string; river_id: string | null; river_name: string | null }

const userRoles = ref<UserRole[]>([])
const rolesLoading = ref(false)

async function loadUserRoles() {
  rolesLoading.value = true
  const token = await getToken()
  if (!token) { rolesLoading.value = false; return }
  try {
    const res = await fetch(`${apiBase}/api/v1/admin/users/roles`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    if (res.ok) userRoles.value = await res.json()
  } finally {
    rolesLoading.value = false
  }
}

async function revokeRole(id: string) {
  const token = await getToken()
  await fetch(`${apiBase}/api/v1/admin/users/roles/${id}`, {
    method: 'DELETE',
    headers: { Authorization: `Bearer ${token}` },
  })
  loadUserRoles()
}

// ── KML Import ────────────────────────────────────────────────────────────────
const kmlInputRef   = ref<HTMLInputElement | null>(null)
const importing     = ref(false)
const importMsg     = ref('')
const importError   = ref(false)
const showKmlGuide  = ref(false)
const importLog     = ref<string[]>([])

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

const assignRoleOpen = ref(false)
const assignLoading = ref(false)
const newRole = ref({ user_id: '', role: 'data_admin', river_id: '' })

async function assignRole() {
  assignLoading.value = true
  const token = await getToken()
  try {
    const res = await fetch(`${apiBase}/api/v1/admin/users/roles`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({
        user_id: newRole.value.user_id,
        role: newRole.value.role,
        river_id: newRole.value.river_id || null,
      }),
    })
    if (res.ok) {
      assignRoleOpen.value = false
      newRole.value = { user_id: '', role: 'data_admin', river_id: '' }
      loadUserRoles()
    }
  } finally {
    assignLoading.value = false
  }
}

// ── NHD Explorer + Reach Authoring ───────────────────────────────────────────
interface NHDSnap { comid: string; name: string; lat: number; lng: number }
interface NHDGaugeItem { id: string; name: string }
interface NHDFC { type: string; features: any[] }
interface AuthorPin { lat: number; lng: number; name: string; comid: string }
interface RepinReach {
  slug: string; name: string; river_name: string | null; common_name: string | null
  description: string | null
  class_min: number | null; class_max: number | null
  permit_required: boolean; multi_day_days: number
  put_in: { lat: number; lng: number } | null
  take_out: { lat: number; lng: number } | null
  start_comid: string | null; end_comid: string | null
}

// ---- Shared ----
const nhdMode = ref<'explore' | 'author' | 'repin'>('explore')
function setNHDMode(mode: 'explore' | 'author' | 'repin') {
  nhdMode.value = mode
  if (mode === 'explore') { resetAuthor(); resetRepin() }
  else if (mode === 'author') { clearNHD(); nhdPickMode.value = false; resetRepin() }
  else { clearNHD(); nhdPickMode.value = false; resetAuthor() }
}

// ---- Explore mode ----
const nhdDistance   = ref('150')
const nhdPickMode   = ref(false)
const nhdLoading    = ref(false)
const nhdError      = ref('')
const nhdSnap       = ref<NHDSnap | null>(null)
const nhdUpstream   = ref<NHDFC | null>(null)
const nhdDownstream = ref<NHDFC | null>(null)
const nhdGauges     = ref<NHDFC | null>(null)
const nhdGaugeList  = ref<NHDGaugeItem[]>([])

function clearNHD() {
  nhdSnap.value = null; nhdUpstream.value = null; nhdDownstream.value = null
  nhdGauges.value = null; nhdGaugeList.value = []; nhdError.value = ''
}

async function onNHDPick(lat: number, lng: number) {
  nhdPickMode.value = false
  nhdLoading.value = true
  nhdError.value = ''
  const token = await getToken()
  if (!token) { nhdLoading.value = false; return }
  try {
    const url = `${apiBase}/api/v1/admin/nldi/watershed?lat=${lat}&lng=${lng}&distance=${nhdDistance.value}`
    const res = await fetch(url, { headers: { Authorization: `Bearer ${token}` } })
    if (!res.ok) { const b = await res.json().catch(() => ({})); nhdError.value = b.error ?? `HTTP ${res.status}`; return }
    const data = await res.json()
    nhdSnap.value = data.snap; nhdUpstream.value = data.upstream_flowlines
    nhdDownstream.value = data.downstream_flowlines; nhdGauges.value = data.upstream_gauges
    nhdGaugeList.value = (data.upstream_gauges?.features ?? []).map((f: any) => ({
      id: f.properties?.identifier ?? '', name: f.properties?.name ?? '',
    }))
  } catch (e: any) { nhdError.value = e.message ?? 'Unknown error' }
  finally { nhdLoading.value = false }
}

// ---- Author mode (ComID-first) ----
const authorPickMode            = ref(false)
const authorAnchorSnapping      = ref(false)
const authorAnchorSnap          = ref<{ comid: string; name: string } | null>(null)
const authorTributaries         = ref<NHDFC | null>(null)
const authorDownstreamFlowlines = ref<NHDFC | null>(null)
const authorDownstreamLoading   = ref(false)
const authorComIDSlot           = ref<'up' | 'down'>('up')
const authorUpComID             = ref<string | null>(null)
const authorDownComID           = ref<string | null>(null)
const authorStartLat            = ref<number | null>(null)
const authorStartLng            = ref<number | null>(null)
const authorEndLat              = ref<number | null>(null)
const authorEndLng              = ref<number | null>(null)
const authorRiverNameOverride   = ref(false)
const authorError               = ref('')
const authorSuccess             = ref('')
const authorSaving              = ref(false)
const authorForm = ref({
  name: '', commonName: '', riverName: '',
  classMin: null as number | null, classMax: null as number | null,
  description: '',
  permitRequired: false,
  multiDay: 1,
  flowRanges: {
    too_low:  { min: null as number | null, max: null as number | null },
    running:  { min: null as number | null, max: null as number | null },
    high:     { min: null as number | null, max: null as number | null },
    very_high: { min: null as number | null, max: null as number | null },
  },
})

const authorFlowBands = [
  { key: 'too_low',  label: 'Too Low',   dot: '#64748b', showMin: false, showMax: true  },
  { key: 'running',  label: 'Runnable',  dot: '#22c55e', showMin: true,  showMax: true  },
  { key: 'high',     label: 'High',      dot: '#f97316', showMin: true,  showMax: true  },
  { key: 'very_high', label: 'Very High', dot: '#ef4444', showMin: true,  showMax: false },
] as const

const authorPutInPin = computed(() =>
  authorStartLat.value != null && authorStartLng.value != null
    ? { lat: authorStartLat.value, lng: authorStartLng.value, label: 'Start' }
    : null
)
const authorTakeOutPin = computed(() =>
  authorEndLat.value != null && authorEndLng.value != null
    ? { lat: authorEndLat.value, lng: authorEndLng.value, label: 'End' }
    : null
)

const authorComputedSlug = computed(() => {
  const river = authorForm.value.riverName.trim()
  const name  = authorForm.value.name.trim()
  if (!river || !name) return ''
  const slugify = (s: string) => s.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')
  return `${slugify(river)}-${slugify(name)}`
})

function resetAuthor() {
  authorPickMode.value = false
  authorAnchorSnapping.value = false
  authorAnchorSnap.value = null
  authorTributaries.value = null
  authorDownstreamFlowlines.value = null
  authorDownstreamLoading.value = false
  authorComIDSlot.value = 'up'
  authorUpComID.value = null
  authorDownComID.value = null
  authorStartLat.value = null; authorStartLng.value = null
  authorEndLat.value = null;   authorEndLng.value = null
  authorRiverNameOverride.value = false
  authorError.value = ''; authorSuccess.value = ''
  authorSaving.value = false
  authorForm.value = {
    name: '', commonName: '', riverName: '',
    classMin: null, classMax: null,
    description: '',
    permitRequired: false,
    multiDay: 1,
    flowRanges: {
      too_low:  { min: null, max: null },
      running:  { min: null, max: null },
      high:     { min: null, max: null },
      very_high: { min: null, max: null },
    },
  }
}

async function onAuthorAnchorPick(lat: number, lng: number) {
  authorPickMode.value = false
  authorAnchorSnapping.value = true
  authorAnchorSnap.value = null
  authorTributaries.value = null
  // Don't reset ComID picks or downstream — anchor is just a viewport hint.
  authorError.value = ''
  try {
    const token = await getToken()
    if (!token) return
    const url = `${apiBase}/api/v1/admin/nldi/upstream-tributaries?lat=${lat}&lng=${lng}&distance=50`
    const res = await fetch(url, { headers: { Authorization: `Bearer ${token}` } })
    if (!res.ok) { authorError.value = `Snap failed: HTTP ${res.status}`; return }
    const data = await res.json()
    authorAnchorSnap.value = { comid: data.snap.comid, name: data.snap.name ?? '' }
    authorTributaries.value = data.tributaries
    // Pre-fill river name from NHD snap (unless user has overridden it)
    if (!authorRiverNameOverride.value && data.snap.name) {
      authorForm.value.riverName = data.snap.name
    }
  } catch (e: any) {
    authorError.value = e.message ?? 'Snap failed'
  } finally {
    authorAnchorSnapping.value = false
  }
}

function onAuthorComIDSelect(comid: string, lat: number, lng: number) {
  if (authorComIDSlot.value === 'up') {
    authorUpComID.value = comid
    authorStartLat.value = lat; authorStartLng.value = lng
    if (!authorDownComID.value) authorComIDSlot.value = 'down'
  } else {
    authorDownComID.value = comid
    authorEndLat.value = lat; authorEndLng.value = lng
  }
}

// When upstream ComID is picked, fetch the full downstream mainstem so the user
// can click anywhere along it to set the take-out — even 300mi downstream.
watch(authorUpComID, async (comid) => {
  authorDownstreamFlowlines.value = null
  if (!comid) return
  authorDownstreamLoading.value = true
  try {
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/admin/nldi/downstream?comid=${comid}&distance=800`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    if (res.ok) {
      const data = await res.json()
      authorDownstreamFlowlines.value = data.downstream_flowlines
    }
  } catch { /* non-fatal — user can still pick downstream from tributaries */ }
  finally { authorDownstreamLoading.value = false }
})

// ---- Load Reach mode ----
const repinSlug             = ref('')
const repinLoadingReach     = ref(false)
const repinLoadError        = ref('')
const repinReach            = ref<RepinReach | null>(null)
const repinDownstream       = ref<NHDFC | null>(null)
const repinFlowlinesLoading = ref(false)
const repinError            = ref('')
const repinSuccess          = ref('')
const repinSaving           = ref(false)
const repinDescEdit         = ref('')
const repinDescGenerating   = ref(false)
const repinDescSaving       = ref(false)
const repinDescMsg          = ref('')

const repinForm = ref({
  name: '', commonName: '', riverName: '', slug: '',
  classMin: null as number | null, classMax: null as number | null,
  permitRequired: false, multiDay: 1,
})
const repinMetaSaving        = ref(false)
const repinMetaMsg           = ref('')
const repinRiverNameFetching = ref(false)
const repinFlowBands = ref({
  too_low:   { min: null as number | null, max: null as number | null },
  running:   { min: null as number | null, max: null as number | null },
  high:      { min: null as number | null, max: null as number | null },
  very_high: { min: null as number | null, max: null as number | null },
})
const repinFlowBandsDef = [
  { key: 'too_low',   label: 'Too Low',   dot: '#64748b', showMin: false, showMax: true  },
  { key: 'running',   label: 'Runnable',  dot: '#22c55e', showMin: true,  showMax: true  },
  { key: 'high',      label: 'High',      dot: '#f97316', showMin: true,  showMax: true  },
  { key: 'very_high', label: 'Very High', dot: '#ef4444', showMin: true,  showMax: false },
] as const
const repinFlowBandsSaving = ref(false)
const repinFlowBandsMsg    = ref('')

const repinUpComID        = ref<string | null>(null)
const repinDownComID      = ref<string | null>(null)
const repinOrigUpComID    = ref<string | null>(null)
const repinOrigDownComID  = ref<string | null>(null)
const repinComIDEditMode  = ref<'up' | 'down' | null>(null)
const repinStartLat       = ref<number | null>(null)
const repinStartLng       = ref<number | null>(null)
const repinEndLat         = ref<number | null>(null)
const repinEndLng         = ref<number | null>(null)

const repinComIDsDirty = computed(() =>
  repinUpComID.value !== repinOrigUpComID.value ||
  repinDownComID.value !== repinOrigDownComID.value
)

const repinPutInPin = computed(() =>
  repinReach.value?.put_in
    ? { lat: repinReach.value.put_in.lat, lng: repinReach.value.put_in.lng, label: 'Put-in' }
    : null
)
const repinTakeOutPin = computed(() =>
  repinReach.value?.take_out
    ? { lat: repinReach.value.take_out.lat, lng: repinReach.value.take_out.lng, label: 'Take-out' }
    : null
)

function resetRepin() {
  repinReach.value = null
  repinDownstream.value = null
  repinError.value = ''; repinSuccess.value = ''
  repinSaving.value = false
  repinDescMsg.value = ''
  repinMetaMsg.value = ''
  repinUpComID.value = null; repinDownComID.value = null
  repinOrigUpComID.value = null; repinOrigDownComID.value = null
  repinComIDEditMode.value = null
  repinStartLat.value = null; repinStartLng.value = null
  repinEndLat.value = null;   repinEndLng.value = null
  repinForm.value = {
    name: '', commonName: '', riverName: '', slug: '',
    classMin: null, classMax: null,
    permitRequired: false, multiDay: 1,
  }
  repinFlowBands.value = {
    too_low:   { min: null, max: null },
    running:   { min: null, max: null },
    high:      { min: null, max: null },
    very_high: { min: null, max: null },
  }
  repinFlowBandsSaving.value = false
  repinFlowBandsMsg.value = ''
}

function resetRepinComIDs() {
  repinUpComID.value = repinOrigUpComID.value
  repinDownComID.value = repinOrigDownComID.value
  repinComIDEditMode.value = null
  repinError.value = ''; repinSuccess.value = ''
}

async function loadRepinReach() {
  const slug = repinSlug.value.trim()
  if (!slug) return
  repinLoadingReach.value = true
  repinLoadError.value = ''
  resetRepin()
  try {
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/admin/reaches/${slug}`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
    if (!res.ok) { repinLoadError.value = res.status === 404 ? `Reach "${slug}" not found` : `HTTP ${res.status}`; return }
    const data = await res.json()
    repinReach.value = {
      slug,
      name: data.name ?? slug,
      common_name: data.common_name ?? null,
      river_name: data.river_name ?? null,
      description: data.description ?? null,
      class_min: data.class_min ?? null,
      class_max: data.class_max ?? null,
      permit_required: !!data.permit_required,
      multi_day_days: data.multi_day_days ?? 1,
      put_in: data.put_in ?? null,
      take_out: data.take_out ?? null,
      start_comid: data.start_comid ?? null,
      end_comid:   data.end_comid   ?? null,
    }
    repinForm.value = {
      name:           data.name ?? '',
      commonName:     data.common_name ?? '',
      riverName:      data.river_name ?? '',
      slug,
      classMin:       data.class_min ?? null,
      classMax:       data.class_max ?? null,
      permitRequired: !!data.permit_required,
      multiDay:       data.multi_day_days ?? 1,
    }
    repinDescEdit.value = data.description ?? ''

    repinUpComID.value       = data.start_comid ?? null
    repinDownComID.value     = data.end_comid   ?? null
    repinOrigUpComID.value   = data.start_comid ?? null
    repinOrigDownComID.value = data.end_comid   ?? null

    // Load existing flow bands
    try {
      const token2 = await getToken()
      const frRes = await fetch(`${apiBase}/api/v1/reaches/${slug}/flow-ranges`, {
        headers: token2 ? { Authorization: `Bearer ${token2}` } : {},
      })
      if (frRes.ok) {
        const bands: Array<{ label: string; min_cfs: number | null; max_cfs: number | null }> = await frRes.json()
        repinFlowBands.value = {
          too_low:   { min: null, max: null },
          running:   { min: null, max: null },
          high:      { min: null, max: null },
          very_high: { min: null, max: null },
        }
        for (const b of bands) {
          const k = b.label as keyof typeof repinFlowBands.value
          if (k in repinFlowBands.value) {
            repinFlowBands.value[k] = { min: b.min_cfs ?? null, max: b.max_cfs ?? null }
          }
        }
      }
    } catch { /* non-fatal */ }

    if (data.start_comid) {
      await fetchRepinFlowlines(data.start_comid)
    }
  } catch (e: any) {
    repinLoadError.value = e.message ?? 'Unknown error'
  } finally {
    repinLoadingReach.value = false
  }
}

async function openReachInEditor(slug: string) {
  activeTab.value = 'nhd'
  setNHDMode('repin')
  repinSlug.value = slug
  await loadRepinReach()
}

async function fetchRepinFlowlines(comid: string) {
  repinFlowlinesLoading.value = true
  try {
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/admin/nldi/downstream?comid=${comid}&distance=800`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
    if (res.ok) {
      const data = await res.json()
      repinDownstream.value = data.downstream_flowlines
    }
  } catch { /* non-fatal */ }
  finally { repinFlowlinesLoading.value = false }
}

function onRepinComIDSelect(comid: string, lat: number, lng: number) {
  if (!repinComIDEditMode.value) return
  if (repinComIDEditMode.value === 'up') {
    repinUpComID.value = comid
    repinStartLat.value = lat; repinStartLng.value = lng
  } else {
    repinDownComID.value = comid
    repinEndLat.value = lat; repinEndLng.value = lng
  }
  repinComIDEditMode.value = null
}

async function saveRepinMeta() {
  if (!repinReach.value) return
  if (!repinForm.value.name.trim()) return
  repinMetaSaving.value = true
  repinMetaMsg.value = ''
  try {
    const f = repinForm.value
    const days = (f.multiDay ?? 1) < 1 ? 1 : (f.multiDay ?? 1)
    const newSlug = f.slug.trim() || repinReach.value.slug
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/admin/reaches/${repinReach.value.slug}/meta`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) },
      body: JSON.stringify({
        name:            f.name.trim(),
        new_slug:        newSlug !== repinReach.value.slug ? newSlug : undefined,
        common_name:     f.commonName.trim(),
        river_name:      f.riverName.trim(),
        class_min:       f.classMin,
        class_max:       f.classMax,
        permit_required: f.permitRequired,
        multi_day_days:  days,
      }),
    })
    if (!res.ok) {
      const d = await res.json().catch(() => ({}))
      repinMetaMsg.value = d.error ?? `HTTP ${res.status}`
      return
    }
    const d = await res.json().catch(() => ({}))
    const savedSlug = d.slug ?? newSlug
    repinMetaMsg.value = 'Saved'
    if (repinReach.value) {
      repinReach.value.name = f.name.trim()
      repinReach.value.river_name = f.riverName.trim() || null
      if (savedSlug !== repinReach.value.slug) {
        repinReach.value.slug = savedSlug
        repinSlug.value = savedSlug
        repinForm.value.slug = savedSlug
      }
    }
  } catch (e: any) {
    repinMetaMsg.value = e.message ?? 'Save failed'
  } finally {
    repinMetaSaving.value = false
  }
}

async function saveRepinFlowBands() {
  if (!repinReach.value) return
  repinFlowBandsSaving.value = true
  repinFlowBandsMsg.value = ''
  try {
    const b = repinFlowBands.value
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/reaches/${repinReach.value.slug}/flow-ranges`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) },
      body: JSON.stringify({
        too_low:   { min_cfs: null,          max_cfs: b.too_low.max  },
        running:   { min_cfs: b.running.min,  max_cfs: b.running.max  },
        high:      { min_cfs: b.high.min,     max_cfs: b.high.max     },
        very_high: { min_cfs: b.very_high.min, max_cfs: null          },
      }),
    })
    repinFlowBandsMsg.value = res.ok ? 'Saved' : `HTTP ${res.status}`
  } catch (e: any) {
    repinFlowBandsMsg.value = e.message ?? 'Save failed'
  } finally {
    repinFlowBandsSaving.value = false
  }
}

async function fetchRepinRiverName() {
  if (!repinUpComID.value) return
  repinRiverNameFetching.value = true
  try {
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/admin/nldi/river-name?comid=${repinUpComID.value}`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
    if (res.ok) {
      const d = await res.json()
      if (d.river_name) repinForm.value.riverName = d.river_name
    }
  } catch { /* non-fatal */ }
  finally { repinRiverNameFetching.value = false }
}

async function submitRepinByComID() {
  if (!repinReach.value || !repinUpComID.value || !repinDownComID.value) return
  repinSaving.value = true
  repinError.value = ''
  repinSuccess.value = ''
  try {
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/admin/reaches/${repinReach.value.slug}/nldi-centerline-by-comid`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) },
      body: JSON.stringify({
        up_comid:  repinUpComID.value,
        down_comid: repinDownComID.value,
        start_lat: repinStartLat.value,
        start_lng: repinStartLng.value,
        end_lat:   repinEndLat.value,
        end_lng:   repinEndLng.value,
      }),
    })
    const data = await res.json()
    if (!res.ok) { repinError.value = data.error ?? `HTTP ${res.status}`; return }
    const lengthStr = data.length_mi != null ? ` · ${data.length_mi} mi` : ''
    repinSuccess.value = `Flow lines saved${lengthStr}`
    repinOrigUpComID.value = repinUpComID.value
    repinOrigDownComID.value = repinDownComID.value
    if (repinReach.value) {
      repinReach.value.start_comid = repinUpComID.value
      repinReach.value.end_comid   = repinDownComID.value
    }
  } catch (e: any) {
    repinError.value = e.message ?? 'Unknown error'
  } finally {
    repinSaving.value = false
  }
}

async function generateRepinDescription() {
  if (!repinReach.value) return
  repinDescGenerating.value = true
  repinDescMsg.value = ''
  try {
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/admin/reaches/${repinReach.value.slug}/generate-description`, {
      method: 'POST',
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
    const data = await res.json()
    if (!res.ok) { repinDescMsg.value = data.error ?? `HTTP ${res.status}`; return }
    repinDescEdit.value = data.description ?? ''
  } catch (e: any) {
    repinDescMsg.value = e.message ?? 'Generate failed'
  } finally {
    repinDescGenerating.value = false
  }
}

async function saveRepinDescription() {
  if (!repinReach.value) return
  repinDescSaving.value = true
  repinDescMsg.value = ''
  try {
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/admin/reaches/${repinReach.value.slug}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) },
      body: JSON.stringify({ description: repinDescEdit.value || null }),
    })
    if (!res.ok) {
      const d = await res.json().catch(() => ({}))
      repinDescMsg.value = d.error ?? `HTTP ${res.status}`
      return
    }
    repinDescMsg.value = 'Description saved'
  } catch (e: any) {
    repinDescMsg.value = e.message ?? 'Save failed'
  } finally {
    repinDescSaving.value = false
  }
}

async function submitAuthorReach() {
  if (!authorForm.value.name.trim() || !authorUpComID.value || !authorDownComID.value) return
  authorSaving.value = true
  authorError.value = ''
  authorSuccess.value = ''
  const token = await getToken()
  if (!token) { authorSaving.value = false; return }
  try {
    const f = authorForm.value
    const days = (f.multiDay ?? 1) < 1 ? 1 : (f.multiDay ?? 1)
    const body = {
      name:            f.name.trim(),
      common_name:     f.commonName.trim(),
      river_name:      f.riverName.trim(),
      up_comid:        authorUpComID.value,
      down_comid:      authorDownComID.value,
      start_lat:       authorStartLat.value,
      start_lng:       authorStartLng.value,
      end_lat:         authorEndLat.value,
      end_lng:         authorEndLng.value,
      class_min:       f.classMin,
      class_max:       f.classMax,
      description:     f.description.trim() || undefined,
      permit_required: f.permitRequired,
      multi_day_days:  days,
    }
    const res = await fetch(`${apiBase}/api/v1/admin/reaches`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify(body),
    })
    const data = await res.json()
    if (!res.ok) { authorError.value = data.error ?? `HTTP ${res.status}`; return }

    const slug: string = data.slug

    // Submit flow ranges if any band has at least one value set
    const ranges = f.flowRanges
    const hasRanges = Object.values(ranges).some(b => b.min != null || b.max != null)
    if (hasRanges) {
      await fetch(`${apiBase}/api/v1/reaches/${slug}/flow-ranges`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({
          too_low:   { min_cfs: null,                  max_cfs: ranges.too_low.max   },
          running:   { min_cfs: ranges.running.min,    max_cfs: ranges.running.max   },
          high:      { min_cfs: ranges.high.min,       max_cfs: ranges.high.max      },
          very_high: { min_cfs: ranges.very_high.min,  max_cfs: null                 },
        }),
      })
    }

    // Redirect to Load Reach with the newly created reach already loaded.
    setNHDMode('repin')
    repinSlug.value = slug
    await loadRepinReach()
  } catch (e: any) {
    authorError.value = e.message ?? 'Unknown error'
  } finally {
    authorSaving.value = false
  }
}
</script>
