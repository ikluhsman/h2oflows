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
            <div
              v-for="river in rivers" :key="river.id"
              class="flex items-center gap-3 px-4 py-3 bg-white dark:bg-gray-900 hover:bg-gray-50 dark:hover:bg-gray-800/50 cursor-pointer transition-colors"
              @click="openRiver(river)"
            >
              <div class="flex-1 min-w-0">
                <p class="text-sm font-medium text-gray-900 dark:text-white truncate">{{ river.name }}</p>
                <p class="text-xs text-gray-400 truncate flex items-center gap-1">
                  <span v-if="river.basin_locked" title="Basin manually locked" class="text-amber-500">&#x1F512;</span>
                  <span>{{ river.basin ?? 'No basin' }}</span>
                  <span class="text-gray-300">·</span>
                  <span>{{ river.slug }}</span>
                </p>
              </div>
              <span class="text-xs text-gray-400 shrink-0">{{ river.reach_count }} reach{{ river.reach_count !== 1 ? 'es' : '' }}</span>
              <svg class="w-4 h-4 text-gray-300 shrink-0" viewBox="0 0 20 20" fill="currentColor">
                <path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd"/>
              </svg>
            </div>
            <div v-if="rivers.length === 0" class="px-4 py-8 text-center text-sm text-gray-400">No rivers yet</div>
          </div>
        </div>

        <!-- Import tab -->
        <div v-if="activeTab === 'import'">
          <div class="space-y-4">
            <div>
              <h2 class="text-sm font-semibold text-gray-700 dark:text-gray-200 mb-1">Import KMZ / KML</h2>
              <p class="text-xs text-gray-400 mb-3">Upload a KMZ or KML file to import or update reaches. Existing reaches are updated; new ones are created.</p>
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
                  <li><strong>Document name</strong> → sets <code>river_name</code> on all reaches in the file</li>
                  <li><strong>Document description</strong> → optional <code>Basin: South Platte</code> line sets the basin for all reaches (overridable per-folder via metadata placemark)</li>
                  <li><strong>One folder per reach</strong> — folder name becomes the reach display name</li>
                  <li><strong>LineString placemark</strong> → reach centerline geometry</li>
                </ul>
              </div>
              <div>
                <p class="font-semibold text-gray-700 dark:text-gray-200 mb-1">Metadata placemarks (coordinate-less)</p>
                <p class="text-gray-400">Keys: <code>common_name</code>, <code>description</code>, <code>min_class</code>, <code>max_class</code>, <code>gauge</code>, <code>basin</code>, <code>permit_required</code>, <code>multi_day</code></p>
                <p class="mt-1 text-gray-400 text-[11px]">When omitted, basin is auto-inferred from the gauge's watershed data (works for USGS gauges) or the river name. Safe to omit for rivers with unique names.</p>
                <p class="mt-0.5 text-amber-500 dark:text-amber-400 text-[11px]">⚠ <strong>basin</strong> is required when a river name exists in multiple drainages (e.g. "Clear Creek" in both South Platte and Arkansas). Omitting it will merge reaches under the wrong basin.</p>
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

        <!-- NHD Explorer tab -->
        <div v-if="activeTab === 'nhd'">
          <div class="space-y-4">

            <!-- Mode switcher -->
            <div class="flex gap-2 border-b border-gray-100 dark:border-gray-800 pb-3">
              <UButton size="xs" :variant="nhdMode === 'explore' ? 'solid' : 'outline'" :color="nhdMode === 'explore' ? 'primary' : 'neutral'" @click="setNHDMode('explore')">Explore</UButton>
              <UButton size="xs" :variant="nhdMode === 'author' ? 'solid' : 'outline'" :color="nhdMode === 'author' ? 'primary' : 'neutral'" @click="setNHDMode('author')">New reach</UButton>
              <UButton size="xs" :variant="nhdMode === 'repin' ? 'solid' : 'outline'" :color="nhdMode === 'repin' ? 'primary' : 'neutral'" @click="setNHDMode('repin')">Re-pin existing</UButton>
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

              <!-- Step indicator -->
              <div class="flex items-center gap-2 mb-3 text-xs">
                <span :class="authorStep === 'put-in' ? 'text-green-700 dark:text-green-400 font-semibold' : authorPutIn ? 'text-gray-400 line-through' : 'text-gray-400'">1. Pick put-in</span>
                <span class="text-gray-300">→</span>
                <span :class="authorStep === 'take-out' ? 'text-red-700 dark:text-red-400 font-semibold' : authorTakeOut ? 'text-gray-400 line-through' : 'text-gray-400'">2. Pick take-out</span>
                <span class="text-gray-300">→</span>
                <span :class="authorStep === 'form' ? 'text-blue-700 dark:text-blue-400 font-semibold' : 'text-gray-400'">3. Fill &amp; save</span>
                <UButton v-if="authorStep !== 'put-in'" size="xs" variant="ghost" color="neutral" class="ml-auto" @click="resetAuthor">Reset</UButton>
              </div>

              <!-- Map with authoring pins -->
              <div v-if="authorSnapping" class="mb-2 text-xs text-blue-600 dark:text-blue-400 text-center animate-pulse">Snapping to NHD…</div>
              <NHDExplorerMap
                :upstream-flowlines="authorUpstream"
                :downstream-flowlines="authorDownstream"
                :upstream-gauges="null"
                :snap-lat="null"
                :snap-lng="null"
                :put-in-pin="authorPutIn ? { lat: authorPutIn.lat, lng: authorPutIn.lng, label: authorPutIn.name || 'Put-in' } : null"
                :take-out-pin="authorTakeOut ? { lat: authorTakeOut.lat, lng: authorTakeOut.lng, label: authorTakeOut.name || 'Take-out' } : null"
                :pick-mode="authorStep === 'put-in' || authorStep === 'take-out'"
                @pick="onAuthorPick"
              />

              <!-- Reach form — shown once both pins placed -->
              <div v-if="authorStep === 'form'" class="mt-4 space-y-3 rounded-xl border border-gray-200 dark:border-gray-700 p-4 bg-white dark:bg-gray-900">
                <h3 class="text-sm font-semibold text-gray-800 dark:text-gray-100">New reach details</h3>

                <!-- Pin summary -->
                <div class="grid grid-cols-2 gap-2 text-xs">
                  <div class="flex items-center gap-2 px-2 py-1.5 rounded-lg bg-green-50 dark:bg-green-950 border border-green-200 dark:border-green-800">
                    <span class="w-2 h-2 rounded-full bg-green-600 shrink-0" />
                    <div>
                      <div class="font-medium text-green-800 dark:text-green-200">Put-in · {{ authorPutIn!.comid }}</div>
                      <div class="text-green-600 dark:text-green-400 font-mono">{{ authorPutIn!.lat.toFixed(5) }}, {{ authorPutIn!.lng.toFixed(5) }}</div>
                    </div>
                  </div>
                  <div class="flex items-center gap-2 px-2 py-1.5 rounded-lg bg-red-50 dark:bg-red-950 border border-red-200 dark:border-red-800">
                    <span class="w-2 h-2 rounded-full bg-red-600 shrink-0" />
                    <div>
                      <div class="font-medium text-red-800 dark:text-red-200">Take-out · {{ authorTakeOut!.comid }}</div>
                      <div class="text-red-600 dark:text-red-400 font-mono">{{ authorTakeOut!.lat.toFixed(5) }}, {{ authorTakeOut!.lng.toFixed(5) }}</div>
                    </div>
                  </div>
                </div>

                <div class="grid grid-cols-2 gap-3">
                  <div>
                    <label class="block text-xs text-gray-500 mb-1">Put-in name</label>
                    <input v-model="authorForm.putInName" class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm" placeholder="e.g. Kremmling" />
                  </div>
                  <div>
                    <label class="block text-xs text-gray-500 mb-1">Take-out name</label>
                    <input v-model="authorForm.takeOutName" class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm" placeholder="e.g. Pumphouse" />
                  </div>
                </div>

                <div>
                  <label class="block text-xs text-gray-500 mb-1">Reach name <span class="text-red-400">*</span></label>
                  <input v-model="authorForm.name" class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm" placeholder="e.g. Kremmling to Pumphouse" />
                </div>

                <div class="grid grid-cols-2 gap-3">
                  <div>
                    <label class="block text-xs text-gray-500 mb-1">Common name</label>
                    <input v-model="authorForm.commonName" class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm" placeholder="e.g. Gore Canyon" />
                  </div>
                  <div>
                    <label class="block text-xs text-gray-500 mb-1">River name</label>
                    <input v-model="authorForm.riverName" class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm" placeholder="e.g. Colorado River" />
                  </div>
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

                <label class="flex items-center gap-2 text-sm cursor-pointer select-none">
                  <input v-model="authorForm.fetchCenterline" type="checkbox" class="rounded" />
                  <span>Fetch NLDI centerline after save</span>
                </label>

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
              <p class="text-xs text-gray-400 mb-3">Enter a reach slug to update its NLDI centerline without modifying its access points.</p>

              <!-- Reach selector -->
              <div class="flex gap-2 mb-3">
                <input
                  v-model="repinSlug"
                  class="flex-1 rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm"
                  placeholder="Reach slug (e.g. colorado-gore-canyon)"
                  @keydown.enter="loadRepinReach"
                />
                <UButton size="sm" :loading="repinLoadingReach" @click="loadRepinReach">Load</UButton>
              </div>
              <div v-if="repinLoadError" class="text-xs text-red-500 mb-2">{{ repinLoadError }}</div>

              <div v-if="repinReach" class="space-y-3">
                <p class="text-xs text-gray-500">Re-pinning: <span class="font-medium text-gray-800 dark:text-gray-100">{{ repinReach.name }}</span></p>

                <!-- Step indicator -->
                <div class="flex items-center gap-2 text-xs">
                  <span :class="repinStep === 'put-in' ? 'text-green-700 dark:text-green-400 font-semibold' : repinPutIn ? 'text-gray-400 line-through' : 'text-gray-400'">1. Pick put-in</span>
                  <span class="text-gray-300">→</span>
                  <span :class="repinStep === 'take-out' ? 'text-red-700 dark:text-red-400 font-semibold' : repinTakeOut ? 'text-gray-400 line-through' : 'text-gray-400'">2. Pick take-out</span>
                  <span class="text-gray-300">→</span>
                  <span :class="repinStep === 'confirm' ? 'text-blue-700 dark:text-blue-400 font-semibold' : 'text-gray-400'">3. Confirm</span>
                  <UButton v-if="repinStep !== 'put-in'" size="xs" variant="ghost" color="neutral" class="ml-auto" @click="resetRepin">Reset</UButton>
                </div>

                <div v-if="repinSnapping" class="text-xs text-blue-600 dark:text-blue-400 text-center animate-pulse">Snapping to NHD…</div>

                <NHDExplorerMap
                  :upstream-flowlines="repinUpstream"
                  :downstream-flowlines="repinDownstream"
                  :upstream-gauges="null"
                  :snap-lat="null"
                  :snap-lng="null"
                  :put-in-pin="repinPutIn ? { lat: repinPutIn.lat, lng: repinPutIn.lng, label: repinPutIn.name || 'Put-in' } : null"
                  :take-out-pin="repinTakeOut ? { lat: repinTakeOut.lat, lng: repinTakeOut.lng, label: repinTakeOut.name || 'Take-out' } : null"
                  :pick-mode="repinStep === 'put-in' || repinStep === 'take-out'"
                  @pick="onRepinPick"
                />

                <!-- Confirm panel -->
                <div v-if="repinStep === 'confirm'" class="rounded-xl border border-gray-200 dark:border-gray-700 p-4 bg-white dark:bg-gray-900 space-y-3">
                  <div class="grid grid-cols-2 gap-3 text-xs">
                    <div class="rounded-lg bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-700 p-2">
                      <div class="font-medium text-green-800 dark:text-green-200">Put-in · {{ repinPutIn!.comid }}</div>
                      <div class="text-green-600 dark:text-green-400 font-mono">{{ repinPutIn!.lat.toFixed(5) }}, {{ repinPutIn!.lng.toFixed(5) }}</div>
                    </div>
                    <div class="rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-700 p-2">
                      <div class="font-medium text-red-800 dark:text-red-200">Take-out · {{ repinTakeOut!.comid }}</div>
                      <div class="text-red-600 dark:text-red-400 font-mono">{{ repinTakeOut!.lat.toFixed(5) }}, {{ repinTakeOut!.lng.toFixed(5) }}</div>
                    </div>
                  </div>
                  <div v-if="repinError" class="text-xs text-red-500">{{ repinError }}</div>
                  <div v-if="repinSuccess" class="text-xs text-green-600 dark:text-green-400">{{ repinSuccess }}</div>
                  <div class="flex gap-2">
                    <UButton size="sm" color="neutral" variant="outline" @click="resetRepin">Start over</UButton>
                    <UButton size="sm" :loading="repinSaving" @click="submitRepin">Replace centerline</UButton>
                  </div>
                </div>

                <div v-if="repinError && repinStep !== 'confirm'" class="text-xs text-red-500">{{ repinError }}</div>
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

    <!-- River detail slide-over -->
    <UModal v-if="selectedRiver" v-model:open="riverDetailOpen" :ui="{ width: 'max-w-2xl' }">
      <template #header>
        <div class="flex items-center justify-between w-full">
          <div>
            <h2 class="text-lg font-bold">{{ selectedRiver.name }}</h2>
            <p class="text-xs text-gray-400 mt-0.5">{{ selectedRiver.basin }} · {{ selectedRiver.slug }}</p>
          </div>
          <div class="flex items-center gap-2">
            <UButton size="xs" variant="outline" color="error" @click="deleteRiver(selectedRiver.slug, selectedRiver.name)">
              Delete river
            </UButton>
            <button class="p-1 rounded text-gray-400 hover:text-gray-600" @click="riverDetailOpen = false">
              <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6 6 18M6 6l12 12"/></svg>
            </button>
          </div>
        </div>
      </template>
      <template #body>
        <div class="space-y-4">

          <!-- Basin override editor -->
          <div class="rounded-lg border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/50 px-4 py-3 space-y-2">
            <p class="text-xs font-semibold text-gray-500 uppercase tracking-wide">Basin assignment</p>

            <!-- HUC source row -->
            <div v-if="selectedRiver.gauge_basin" class="flex items-center gap-2 text-xs text-gray-400">
              <span class="text-gray-300">HUC-derived:</span>
              <span class="font-medium text-gray-600 dark:text-gray-300">{{ selectedRiver.gauge_basin }}</span>
              <template v-if="selectedRiver.gauge_watershed">
                <span class="text-gray-300">via</span>
                <span>{{ selectedRiver.gauge_watershed }}</span>
              </template>
              <template v-if="selectedRiver.gauge_huc8">
                <span class="text-gray-300">·</span>
                <span class="font-mono">HUC{{ selectedRiver.gauge_huc8.slice(0,4) }}</span>
              </template>
            </div>
            <p v-else class="text-xs text-gray-400 italic">No gauge with HUC data linked yet</p>

            <!-- Edit row -->
            <div class="flex items-center gap-2">
              <input
                v-model="basinEdit"
                class="flex-1 rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-1.5 text-sm"
                placeholder="e.g. South Platte"
              />
              <label class="flex items-center gap-1.5 text-xs text-gray-600 dark:text-gray-300 cursor-pointer select-none shrink-0">
                <input type="checkbox" v-model="basinLockEdit" class="rounded" />
                Lock
              </label>
              <UButton size="xs" :loading="basinSaving" @click="saveBasin">Save</UButton>
            </div>
            <p v-if="selectedRiver.basin_locked && !basinLockEdit" class="text-xs text-amber-500">Removing the lock will allow the metadata sync to overwrite this basin.</p>
            <p v-if="!selectedRiver.basin_locked && basinLockEdit" class="text-xs text-blue-500">Locking prevents the sync from overwriting this basin in the future.</p>
          </div>

          <p class="text-sm text-gray-500">{{ selectedRiver.reaches?.length ?? 0 }} reaches</p>
          <div class="divide-y divide-gray-100 dark:divide-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
            <div
              v-for="reach in selectedRiver.reaches" :key="reach.id"
              class="flex items-center gap-3 px-3 py-2.5 bg-white dark:bg-gray-900"
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
                <UButton
                  size="xs" variant="outline" color="neutral"
                  :loading="fetchingCenterlines.has(reach.slug)"
                  @click="fetchCenterline(reach.slug)"
                >Fetch line</UButton>
                <UButton
                  size="xs" variant="outline" color="error"
                  @click="deleteReach(reach.slug, reach.common_name ?? reach.name)"
                >Delete</UButton>
                <NuxtLink :to="`/reaches/${reach.slug}`" class="text-xs text-blue-500 hover:underline">View</NuxtLink>
              </div>
            </div>
            <div v-if="!selectedRiver.reaches?.length" class="px-3 py-6 text-center text-sm text-gray-400">No reaches linked to this river</div>
          </div>

        </div>
      </template>
    </UModal>

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
    { key: 'import', label: 'Import' },
    { key: 'nhd',    label: 'NHD Explorer' },
  ]
  if (isAdmin.value) tabs.push({ key: 'users', label: 'Users' })
  return tabs
})

// ── Rivers ────────────────────────────────────────────────────────────────────
interface River { id: string; slug: string; name: string; basin: string | null; basin_locked: boolean; state_abbr: string | null; reach_count: number }
interface RiverDetail extends River {
  gauge_basin: string | null      // system-derived canonical basin (from HUC)
  gauge_watershed: string | null  // HUC4 watershed name (e.g. "Cache La Poudre River")
  gauge_huc8: string | null       // raw HUC8 for reference
  reaches: { id: string; slug: string; name: string; common_name: string | null; has_centerline: boolean }[]
}

const rivers = ref<River[]>([])
const riversLoading = ref(false)

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

const selectedRiver = ref<RiverDetail | null>(null)
const riverDetailOpen = ref(false)
const basinEdit = ref('')
const basinLockEdit = ref(false)
const basinSaving = ref(false)

async function openRiver(river: River) {
  const token = await getToken()
  const res = await fetch(`${apiBase}/api/v1/admin/rivers/${river.slug}`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (res.ok) {
    selectedRiver.value = await res.json()
    basinEdit.value = selectedRiver.value?.basin ?? ''
    basinLockEdit.value = selectedRiver.value?.basin_locked ?? false
    riverDetailOpen.value = true
  }
}

async function saveBasin() {
  if (!selectedRiver.value) return
  basinSaving.value = true
  const token = await getToken()
  try {
    await fetch(`${apiBase}/api/v1/admin/rivers/${selectedRiver.value.slug}`, {
      method: 'PUT',
      headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ basin: basinEdit.value || null, basin_locked: basinLockEdit.value }),
    })
    // Refresh detail + list so the lock badge updates.
    await openRiver(selectedRiver.value)
    loadRivers()
  } finally {
    basinSaving.value = false
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
      openRiver(selectedRiver.value)
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
  if (!res.ok) {
    alert(`Delete failed: ${res.status}`)
    return
  }
  if (selectedRiver.value) openRiver(selectedRiver.value)
  loadRivers()
}

async function deleteRiver(riverSlug: string, riverName: string) {
  if (!confirm(`Permanently delete "${riverName}"?\n\nAll reaches will be unlinked but not deleted.`)) return
  const token = await getToken()
  const res = await fetch(`${apiBase}/api/v1/admin/rivers/${riverSlug}`, {
    method: 'DELETE',
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) {
    alert(`Delete failed: ${res.status}`)
    return
  }
  riverDetailOpen.value = false
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
interface RepinReach { slug: string; name: string; river_name: string | null }

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

// ---- Author mode ----
const authorStep     = ref<'put-in' | 'take-out' | 'form'>('put-in')
const authorSnapping = ref(false)
const authorPutIn    = ref<AuthorPin | null>(null)
const authorTakeOut  = ref<AuthorPin | null>(null)
const authorUpstream   = ref<NHDFC | null>(null)
const authorDownstream = ref<NHDFC | null>(null)
const authorError    = ref('')
const authorSuccess  = ref('')
const authorSaving   = ref(false)
const authorForm     = ref({ name: '', commonName: '', riverName: '', putInName: '', takeOutName: '', classMin: null as number | null, classMax: null as number | null, fetchCenterline: true })

function resetAuthor() {
  authorStep.value = 'put-in'; authorPutIn.value = null; authorTakeOut.value = null
  authorUpstream.value = null; authorDownstream.value = null
  authorError.value = ''; authorSuccess.value = ''
  authorSaving.value = false
  authorForm.value = { name: '', commonName: '', riverName: '', putInName: '', takeOutName: '', classMin: null, classMax: null, fetchCenterline: true }
}

async function snapAuthorPin(lat: number, lng: number): Promise<AuthorPin | null> {
  const token = await getToken()
  if (!token) return null
  const url = `${apiBase}/api/v1/admin/nldi/watershed?lat=${lat}&lng=${lng}&distance=150`
  const res = await fetch(url, { headers: { Authorization: `Bearer ${token}` } })
  if (!res.ok) return null
  const data = await res.json()
  return { lat, lng, comid: data.snap.comid, name: data.snap.name ?? '' }
}

async function onAuthorPick(lat: number, lng: number) {
  authorSnapping.value = true
  authorError.value = ''
  try {
    if (authorStep.value === 'put-in') {
      const pin = await snapAuthorPin(lat, lng)
      if (!pin) { authorError.value = 'Snap failed — try another point'; return }
      authorPutIn.value = pin
      authorForm.value.putInName = pin.name
      authorStep.value = 'take-out'
    } else if (authorStep.value === 'take-out') {
      const pin = await snapAuthorPin(lat, lng)
      if (!pin) { authorError.value = 'Snap failed — try another point'; return }
      authorTakeOut.value = pin
      authorForm.value.takeOutName = pin.name
      if (authorForm.value.name === '' && authorPutIn.value) {
        authorForm.value.name = `${authorPutIn.value.name || 'Put-in'} to ${pin.name || 'Take-out'}`
      }
      // Fetch mainstem between the two pins for the map preview.
      fetchAuthorMainstem()
      authorStep.value = 'form'
    }
  } finally {
    authorSnapping.value = false
  }
}

async function fetchAuthorMainstem() {
  if (!authorPutIn.value || !authorTakeOut.value) return
  const token = await getToken()
  if (!token) return
  try {
    const url = `${apiBase}/api/v1/admin/nldi/watershed?lat=${authorPutIn.value.lat}&lng=${authorPutIn.value.lng}&distance=500`
    const res = await fetch(url, { headers: { Authorization: `Bearer ${token}` } })
    if (!res.ok) return
    const data = await res.json()
    authorUpstream.value  = data.upstream_flowlines
    authorDownstream.value = data.downstream_flowlines
  } catch { /* non-fatal */ }
}

// ---- Re-pin mode ----
const repinSlug         = ref('')
const repinLoadingReach = ref(false)
const repinLoadError    = ref('')
const repinReach        = ref<RepinReach | null>(null)
const repinStep         = ref<'put-in' | 'take-out' | 'confirm'>('put-in')
const repinSnapping     = ref(false)
const repinPutIn        = ref<AuthorPin | null>(null)
const repinTakeOut      = ref<AuthorPin | null>(null)
const repinUpstream     = ref<NHDFC | null>(null)
const repinDownstream   = ref<NHDFC | null>(null)
const repinError        = ref('')
const repinSuccess      = ref('')
const repinSaving       = ref(false)

function resetRepin() {
  repinStep.value = 'put-in'; repinPutIn.value = null; repinTakeOut.value = null
  repinUpstream.value = null; repinDownstream.value = null
  repinError.value = ''; repinSuccess.value = ''
  repinSaving.value = false
}

async function loadRepinReach() {
  const slug = repinSlug.value.trim()
  if (!slug) return
  repinLoadingReach.value = true
  repinLoadError.value = ''
  repinReach.value = null
  resetRepin()
  try {
    const res = await fetch(`${apiBase}/api/v1/reaches/${slug}`)
    if (!res.ok) { repinLoadError.value = res.status === 404 ? `Reach "${slug}" not found` : `HTTP ${res.status}`; return }
    const data = await res.json()
    repinReach.value = { slug, name: data.name ?? slug, river_name: data.river_name ?? null }
  } catch (e: any) {
    repinLoadError.value = e.message ?? 'Unknown error'
  } finally {
    repinLoadingReach.value = false
  }
}

async function onRepinPick(lat: number, lng: number) {
  repinSnapping.value = true
  repinError.value = ''
  try {
    if (repinStep.value === 'put-in') {
      const pin = await snapAuthorPin(lat, lng)
      if (!pin) { repinError.value = 'Snap failed — try another point'; return }
      repinPutIn.value = pin
      repinStep.value = 'take-out'
    } else if (repinStep.value === 'take-out') {
      const pin = await snapAuthorPin(lat, lng)
      if (!pin) { repinError.value = 'Snap failed — try another point'; return }
      repinTakeOut.value = pin
      fetchRepinMainstem()
      repinStep.value = 'confirm'
    }
  } finally {
    repinSnapping.value = false
  }
}

async function fetchRepinMainstem() {
  if (!repinPutIn.value || !repinTakeOut.value) return
  const token = await getToken()
  if (!token) return
  try {
    const url = `${apiBase}/api/v1/admin/nldi/watershed?lat=${repinPutIn.value.lat}&lng=${repinPutIn.value.lng}&distance=500`
    const res = await fetch(url, { headers: { Authorization: `Bearer ${token}` } })
    if (!res.ok) return
    const data = await res.json()
    repinUpstream.value = data.upstream_flowlines
    repinDownstream.value = data.downstream_flowlines
  } catch { /* non-fatal */ }
}

async function submitRepin() {
  if (!repinReach.value || !repinPutIn.value || !repinTakeOut.value) return
  repinSaving.value = true
  repinError.value = ''
  repinSuccess.value = ''
  const token = await getToken()
  if (!token) { repinSaving.value = false; return }
  try {
    const body = {
      put_in:   { lat: repinPutIn.value.lat,  lng: repinPutIn.value.lng  },
      take_out: { lat: repinTakeOut.value.lat, lng: repinTakeOut.value.lng },
    }
    const res = await fetch(`${apiBase}/api/v1/admin/reaches/${repinReach.value.slug}/nldi-centerline`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify(body),
    })
    const data = await res.json()
    if (!res.ok) { repinError.value = data.error ?? `HTTP ${res.status}`; return }
    const lengthStr = data.length_mi != null ? ` · ${data.length_mi} mi` : ''
    repinSuccess.value = `Centerline updated${lengthStr}`
  } catch (e: any) {
    repinError.value = e.message ?? 'Unknown error'
  } finally {
    repinSaving.value = false
  }
}

async function submitAuthorReach() {
  if (!authorForm.value.name.trim() || !authorPutIn.value || !authorTakeOut.value) return
  authorSaving.value = true
  authorError.value = ''
  authorSuccess.value = ''
  const token = await getToken()
  if (!token) { authorSaving.value = false; return }
  try {
    const body = {
      name:             authorForm.value.name.trim(),
      common_name:      authorForm.value.commonName.trim(),
      river_name:       authorForm.value.riverName.trim(),
      put_in:  { lat: authorPutIn.value.lat,  lng: authorPutIn.value.lng,  name: authorForm.value.putInName,  comid: authorPutIn.value.comid },
      take_out: { lat: authorTakeOut.value.lat, lng: authorTakeOut.value.lng, name: authorForm.value.takeOutName, comid: authorTakeOut.value.comid },
      class_min:        authorForm.value.classMin,
      class_max:        authorForm.value.classMax,
      fetch_centerline: authorForm.value.fetchCenterline,
    }
    const res = await fetch(`${apiBase}/api/v1/admin/reaches`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify(body),
    })
    const data = await res.json()
    if (!res.ok) { authorError.value = data.error ?? `HTTP ${res.status}`; return }
    const lengthStr = data.length_mi != null ? ` · ${data.length_mi} mi` : ''
    authorSuccess.value = `Saved! Slug: ${data.slug}${lengthStr}`
  } catch (e: any) {
    authorError.value = e.message ?? 'Unknown error'
  } finally {
    authorSaving.value = false
  }
}
</script>
