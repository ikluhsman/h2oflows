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
          <div class="flex items-center justify-between mb-3">
            <p class="text-sm text-gray-500">{{ rivers.length }} rivers<template v-if="unassignedReaches.length"> · <span class="text-amber-500">{{ unassignedReaches.length }} unassigned</span></template></p>
          </div>

          <UInput v-model="riverSearch" icon="i-heroicons-magnifying-glass" placeholder="Search rivers…" class="mb-3" />

          <div v-if="riversLoading" class="space-y-2">
            <div v-for="i in 5" :key="i" class="h-12 rounded-lg bg-gray-100 dark:bg-gray-800 animate-pulse" />
          </div>

          <div v-else class="divide-y divide-gray-100 dark:divide-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
            <template v-for="stateGroup in groupedRivers" :key="stateGroup.state">
              <!-- State header -->
              <div class="px-4 py-1.5 bg-gray-100 dark:bg-gray-800/80 border-t border-gray-200 dark:border-gray-700 first:border-t-0">
                <p class="text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wide">{{ stateGroup.state === '—' ? 'No state' : stateGroup.state }}</p>
              </div>

              <template v-for="basinGroup in stateGroup.basins" :key="basinGroup.basin">
                <!-- Basin sub-header (only when basin is named) -->
                <div v-if="basinGroup.basin !== '—'" class="px-6 py-1 bg-gray-50 dark:bg-gray-900/60 border-t border-gray-100 dark:border-gray-800">
                  <p class="text-xs text-gray-400 italic">{{ basinGroup.basin }}</p>
                </div>

                <template v-for="river in basinGroup.rivers" :key="river.id">
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
                      <!-- River actions bar -->
                      <div class="flex items-center justify-between px-6 py-2 border-b border-gray-100 dark:border-gray-800">
                        <span class="text-xs text-gray-400">{{ selectedRiver.reaches?.length ?? 0 }} reaches</span>
                        <div class="flex items-center gap-2">
                          <UButton size="xs" variant="ghost" color="neutral" @click="openEditRiver(selectedRiver)">Edit</UButton>
                          <UButton size="xs" variant="ghost" color="error" @click="deleteRiver(selectedRiver.slug, selectedRiver.name)">Delete river</UButton>
                        </div>
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
                            <UButton size="xs" variant="outline" color="error" @click="deleteReach(reach.slug, reach.common_name ?? reach.name)">Delete</UButton>
                            <button class="text-xs text-blue-500 hover:underline" @click="openReachInEditor(reach.slug)">Edit</button>
                          </div>
                        </div>
                        <div v-if="!selectedRiver.reaches?.length" class="px-6 py-4 text-center text-sm text-gray-400">No reaches linked to this river</div>
                      </div>
                    </template>
                  </div>
                </template>
              </template>
            </template>
            <div v-if="groupedRivers.length === 0 && !riversLoading" class="px-4 py-8 text-center text-sm text-gray-400">{{ riverSearch ? 'No rivers match your search' : 'No rivers yet' }}</div>

            <!-- Unassigned reaches group -->
            <template v-if="unassignedReaches.length">
              <div
                class="flex items-center gap-3 px-4 py-3 bg-amber-50 dark:bg-amber-950/30 hover:bg-amber-100 dark:hover:bg-amber-900/30 cursor-pointer transition-colors border-t border-gray-100 dark:border-gray-800"
                @click="unassignedExpanded = !unassignedExpanded"
              >
                <div class="flex-1 min-w-0">
                  <p class="text-sm font-medium text-amber-800 dark:text-amber-300">Unassigned</p>
                  <p class="text-xs text-amber-600 dark:text-amber-500">No river association — assign from Load Reach editor</p>
                </div>
                <span class="text-xs text-amber-600 dark:text-amber-400 shrink-0">{{ unassignedReaches.length }} reach{{ unassignedReaches.length !== 1 ? 'es' : '' }}</span>
                <svg class="w-4 h-4 text-amber-400 shrink-0 transition-transform" :class="unassignedExpanded ? 'rotate-90' : ''" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd"/>
                </svg>
              </div>
              <div v-if="unassignedExpanded" class="bg-gray-50 dark:bg-gray-950 border-t border-gray-100 dark:border-gray-800 divide-y divide-gray-100 dark:divide-gray-800">
                <div
                  v-for="reach in unassignedReaches" :key="reach.id"
                  class="flex items-center gap-3 px-6 py-2.5 bg-white dark:bg-gray-900/60"
                >
                  <div class="flex-1 min-w-0">
                    <p class="text-sm font-medium truncate">{{ reach.common_name ?? reach.name }}</p>
                    <p class="text-xs text-gray-400 truncate">
                      {{ reach.slug }}
                      <span v-if="reach.river_name" class="text-amber-500"> · {{ reach.river_name }}</span>
                    </p>
                  </div>
                  <div class="flex items-center gap-2 shrink-0 flex-wrap justify-end">
                    <span
                      class="text-xs px-1.5 py-0.5 rounded"
                      :class="reach.has_centerline
                        ? 'bg-emerald-50 dark:bg-emerald-950/40 text-emerald-600 dark:text-emerald-400'
                        : 'bg-gray-100 dark:bg-gray-800 text-gray-400'"
                    >{{ reach.has_centerline ? 'Line ✓' : 'No line' }}</span>
                    <UButton size="xs" variant="outline" color="error" @click="deleteReach(reach.slug, reach.common_name ?? reach.name)">Delete</UButton>
                    <button class="text-xs text-blue-500 hover:underline" @click="openReachInEditor(reach.slug)">Edit</button>
                  </div>
                </div>
              </div>
            </template>
          </div>
        </div>

        <!-- Import tab -->
        <div v-if="activeTab === 'import'">
          <KmlImportPanel />
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
              <ReachAuthor @created="onAuthorCreated" @cancel="setNHDMode('explore')" />
            </div>

            <!-- ── RE-PIN EXISTING MODE ────────────────────────────────────────── -->
            <div v-if="nhdMode === 'repin'">
              <p class="text-xs text-gray-400 mb-3">Enter a reach slug to load it for editing — flow lines, metadata, and description.</p>

              <!-- Reach selector -->
              <div class="flex gap-2 mb-3">
                <input
                  v-model="repinSlugInput"
                  class="flex-1 rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm"
                  placeholder="Reach slug (e.g. colorado-gore-canyon)"
                  @keydown.enter="onLoadReach"
                />
                <UButton size="sm" @click="onLoadReach">Load Reach</UButton>
              </div>

              <ReachEditor
                v-if="repinEditorSlug"
                :key="repinEditorKey"
                :slug="repinEditorSlug"
                :rivers="rivers"
                @slug-changed="(s) => { repinEditorSlug = s; repinSlugInput = s }"
                @rivers-updated="loadRivers"
              />
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
          <UFormField label="GNIS ID (optional)">
            <UInput v-model="newRiver.gnis_id" placeholder="00179365" class="max-w-40" />
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

    <!-- Edit river modal -->
    <UModal v-model:open="editRiverOpen" title="Edit river">
      <template #body>
        <div v-if="editingRiver" class="space-y-3">
          <UFormField label="Name">
            <UInput v-model="editingRiver.name" />
          </UFormField>
          <UFormField label="Basin">
            <div class="flex items-center gap-2">
              <UInput v-model="editingRiver.basin" placeholder="South Platte" class="flex-1" :disabled="editingRiver.basin_locked" />
              <span v-if="editingRiver.basin_locked" class="text-xs text-gray-400 shrink-0">locked</span>
            </div>
          </UFormField>
          <UFormField label="State *">
            <UInput v-model="editingRiver.state_abbr" placeholder="CO" class="max-w-20" />
          </UFormField>
          <UButton
            size="xs" variant="outline" color="neutral"
            :loading="autoFillLoading"
            @click="autoFillRiverMeta"
          >Auto-lookup basin &amp; state from NLDI</UButton>
          <p v-if="autoFillError" class="text-xs text-red-500">{{ autoFillError }}</p>
          <UFormField label="GNIS ID (optional)">
            <UInput v-model="editingRiver.gnis_id" placeholder="00179365" class="max-w-40" />
          </UFormField>
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton variant="ghost" color="neutral" @click="editRiverOpen = false">Cancel</UButton>
          <UButton :loading="editRiverLoading" :disabled="!editingRiver?.state_abbr" @click="saveEditRiver">Save</UButton>
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
const riverSearch = ref('')
const expandedRiverId = ref<string | null>(null)

const groupedRivers = computed(() => {
  const q = riverSearch.value.toLowerCase()
  const filtered = q
    ? rivers.value.filter(rv =>
        rv.name.toLowerCase().includes(q) ||
        (rv.state_abbr ?? '').toLowerCase().includes(q) ||
        (rv.basin ?? '').toLowerCase().includes(q)
      )
    : rivers.value

  const stateMap = new Map<string, Map<string, River[]>>()
  for (const rv of filtered) {
    const state = rv.state_abbr ?? '—'
    const basin = rv.basin ?? '—'
    if (!stateMap.has(state)) stateMap.set(state, new Map())
    const basinMap = stateMap.get(state)!
    if (!basinMap.has(basin)) basinMap.set(basin, [])
    basinMap.get(basin)!.push(rv)
  }

  return [...stateMap.entries()]
    .sort(([a], [b]) => a === '—' ? 1 : b === '—' ? -1 : a.localeCompare(b))
    .map(([state, basinMap]) => ({
      state,
      basins: [...basinMap.entries()]
        .sort(([a], [b]) => a === '—' ? 1 : b === '—' ? -1 : a.localeCompare(b))
        .map(([basin, rvs]) => ({
          basin,
          rivers: [...rvs].sort((a, b) => a.name.localeCompare(b.name)),
        })),
    }))
})

const unassignedReaches = ref<{ id: string; slug: string; name: string; common_name: string | null; river_name: string | null; has_centerline: boolean }[]>([])
const unassignedExpanded = ref(false)
const selectedRiver = ref<RiverDetail | null>(null)
const riverDetailLoading = ref(false)

async function loadRivers() {
  riversLoading.value = true
  const token = await getToken()
  if (!token) { riversLoading.value = false; return }
  try {
    const [rRes, uRes] = await Promise.all([
      fetch(`${apiBase}/api/v1/admin/rivers`, { headers: { Authorization: `Bearer ${token}` } }),
      fetch(`${apiBase}/api/v1/admin/reaches/unassigned`, { headers: { Authorization: `Bearer ${token}` } }),
    ])
    if (rRes.ok) rivers.value = await rRes.json()
    if (uRes.ok) unassignedReaches.value = await uRes.json()
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
const newRiver = ref({ name: '', slug: '', basin: '', state_abbr: '', gnis_id: '' })

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
        gnis_id: newRiver.value.gnis_id || null,
      }),
    })
    if (res.ok) {
      createRiverOpen.value = false
      newRiver.value = { name: '', slug: '', basin: '', state_abbr: '', gnis_id: '' }
      loadRivers()
    }
  } finally {
    createLoading.value = false
  }
}

// Edit river
const editRiverOpen = ref(false)
const editRiverLoading = ref(false)
const autoFillLoading = ref(false)
const autoFillError = ref('')
const editingRiver = ref<{ slug: string; name: string; basin: string; basin_locked: boolean; state_abbr: string; gnis_id: string } | null>(null)

function openEditRiver(river: RiverDetail) {
  editingRiver.value = {
    slug: river.slug,
    name: river.name,
    basin: river.basin ?? '',
    basin_locked: river.basin_locked,
    state_abbr: river.state_abbr ?? '',
    gnis_id: river.gnis_id ?? '',
  }
  autoFillError.value = ''
  editRiverOpen.value = true
}

async function autoFillRiverMeta() {
  if (!editingRiver.value) return
  autoFillLoading.value = true
  autoFillError.value = ''
  try {
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/admin/rivers/${editingRiver.value.slug}/auto-fill`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    if (!res.ok) {
      const body = await res.json().catch(() => ({}))
      autoFillError.value = body.error ?? `Error ${res.status}`
      return
    }
    const data = await res.json()
    if (data.state_abbr) editingRiver.value.state_abbr = data.state_abbr
    if (data.basin && !editingRiver.value.basin_locked) editingRiver.value.basin = data.basin
  } catch (err: any) {
    autoFillError.value = err?.message ?? 'Lookup failed'
  } finally {
    autoFillLoading.value = false
  }
}

async function saveEditRiver() {
  if (!editingRiver.value) return
  editRiverLoading.value = true
  const token = await getToken()
  try {
    const res = await fetch(`${apiBase}/api/v1/admin/rivers/${editingRiver.value.slug}`, {
      method: 'PUT',
      headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: editingRiver.value.name || null,
        basin: editingRiver.value.basin || null,
        state_abbr: editingRiver.value.state_abbr || null,
        gnis_id: editingRiver.value.gnis_id || null,
      }),
    })
    if (res.ok) {
      editRiverOpen.value = false
      loadRivers()
    }
  } finally {
    editRiverLoading.value = false
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
  if (mode === 'explore') clearNHD()
  else { clearNHD(); nhdPickMode.value = false }
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

// ---- Load Reach (re-pin) mode ----
const repinSlugInput  = ref('')
const repinEditorSlug = ref('')
const repinEditorKey  = ref(0)

function onLoadReach() {
  const slug = repinSlugInput.value.trim()
  if (!slug) return
  repinEditorSlug.value = slug
  repinEditorKey.value++
}

async function openReachInEditor(slug: string) {
  activeTab.value = 'nhd'
  nhdMode.value = 'repin'
  repinSlugInput.value = slug
  repinEditorSlug.value = slug
  repinEditorKey.value++
}

// ---- Author mode ----
async function onAuthorCreated(slug: string) {
  nhdMode.value = 'repin'
  repinSlugInput.value = slug
  repinEditorSlug.value = slug
  repinEditorKey.value++
}

</script>
