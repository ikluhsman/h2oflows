<template>
  <div class="min-h-screen bg-neutral-50 dark:bg-neutral-950">
    <AppHeader />

    <main class="max-w-5xl mx-auto px-4 py-6 space-y-6">

      <!-- Not authorized -->
      <div v-if="!isDataAdmin && authReady" class="mt-20 flex flex-col items-center gap-3 text-center">
        <svg class="w-10 h-10 text-neutral-300" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
        </svg>
        <h2 class="text-lg font-semibold">Access restricted</h2>
        <p class="text-sm text-neutral-500">You need data admin or site admin permissions to view this page.</p>
      </div>

      <!-- Loading auth -->
      <div v-else-if="!authReady" class="mt-20 flex justify-center">
        <div class="w-6 h-6 rounded-full border-2 border-primary-500 border-t-transparent animate-spin" />
      </div>

      <!-- Admin UI -->
      <template v-else>
        <div class="flex items-center justify-between">
          <h1 class="text-xl font-bold text-neutral-900 dark:text-white">Admin</h1>
        </div>

        <!-- Tabs -->
        <div class="flex gap-1 border-b border-neutral-200 dark:border-neutral-800">
          <button
            v-for="tab in visibleTabs" :key="tab.key"
            class="px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors"
            :class="activeTab === tab.key
              ? 'border-primary-500 text-primary-600 dark:text-primary-400'
              : 'border-transparent text-neutral-500 hover:text-neutral-700 dark:hover:text-neutral-300'"
            @click="activeTab = tab.key"
          >{{ tab.label }}</button>
        </div>

        <!-- Rivers tab -->
        <div v-if="activeTab === 'rivers'">
          <div class="flex items-center justify-between mb-3">
            <p class="text-sm text-neutral-500">
              {{ rivers.length }} rivers
              <template v-if="unassignedReaches.length"> · <span class="text-amber-500">{{ unassignedReaches.length }} unassigned</span></template>
            </p>
            <div class="flex items-center gap-2">
              <UButton size="xs" variant="outline" color="neutral" icon="i-heroicons-plus" @click="authorModalOpen = true">New reach</UButton>
              <UButton size="xs" icon="i-heroicons-plus" @click="createRiverOpen = true">New river</UButton>
            </div>
          </div>

          <UInput v-model="riverSearch" icon="i-heroicons-magnifying-glass" placeholder="Search reaches…" class="mb-3" />

          <div v-if="riversLoading" class="space-y-2">
            <div v-for="i in 5" :key="i" class="h-12 rounded-lg bg-neutral-100 dark:bg-neutral-800 animate-pulse" />
          </div>

          <!-- Needs review -->
          <div v-if="needsReviewRivers.length" class="mb-3 rounded-lg border border-amber-200 dark:border-amber-800 overflow-hidden">
            <div
              class="flex items-center gap-3 px-4 py-2.5 bg-amber-50 dark:bg-amber-950/40 cursor-pointer"
              @click="needsReviewExpanded = !needsReviewExpanded"
            >
              <span class="text-xs font-semibold text-amber-700 dark:text-amber-300 flex-1">Needs review ({{ needsReviewRivers.length }})</span>
              <span class="text-xs text-amber-500">Rivers created without GNIS confirmation</span>
              <svg class="w-4 h-4 text-amber-400 shrink-0 transition-transform" :class="needsReviewExpanded ? 'rotate-90' : ''" viewBox="0 0 20 20" fill="currentColor">
                <path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd"/>
              </svg>
            </div>
            <div v-if="needsReviewExpanded" class="divide-y divide-amber-100 dark:divide-amber-900/40">
              <div v-for="rv in needsReviewRivers" :key="rv.id" class="flex items-center gap-3 px-4 py-2 bg-white dark:bg-neutral-900/60 text-sm">
                <span class="flex-1 truncate font-medium">{{ rv.name }}</span>
                <span v-if="rv.state_abbr" class="text-xs font-mono text-neutral-400">{{ rv.state_abbr }}</span>
                <span v-if="rv.basin" class="text-xs text-neutral-400">{{ rv.basin }}</span>
                <UButton size="xs" variant="ghost" color="neutral" @click="openEditRiverBySlug(rv.slug)">Edit</UButton>
              </div>
            </div>
          </div>

          <div v-if="!riversLoading" class="divide-y divide-neutral-100 dark:divide-neutral-800 rounded-lg border border-neutral-200 dark:border-neutral-700 overflow-hidden">
            <template v-for="stateGroup in groupedByStateBasin" :key="stateGroup.state">
              <!-- State header -->
              <div class="px-4 py-1.5 bg-neutral-100 dark:bg-neutral-800/80 border-t border-neutral-200 dark:border-neutral-700 first:border-t-0">
                <p class="text-xs font-semibold text-neutral-500 dark:text-neutral-400 uppercase tracking-wide">
                  {{ stateGroup.state === '—' ? 'No state' : stateGroup.state }}
                </p>
              </div>

              <template v-for="basinGroup in stateGroup.basins" :key="basinGroup.basin + stateGroup.state">
                <!-- Basin sub-header -->
                <div class="px-6 py-1 bg-neutral-50 dark:bg-neutral-900/60 border-t border-neutral-100 dark:border-neutral-800">
                  <p class="text-xs text-neutral-400 dark:text-neutral-500">{{ basinGroup.basin === '—' ? 'No basin' : basinGroup.basin }}</p>
                </div>

              <template v-for="river in basinGroup.rivers.filter(r => isRiverVisible(r.river_id))" :key="river.river_id + stateGroup.state">
                <!-- River row -->
                <div
                  class="flex items-center gap-3 px-4 py-2.5 bg-white dark:bg-neutral-900 hover:bg-neutral-50 dark:hover:bg-neutral-800/50 cursor-pointer transition-colors"
                  @click="toggleRiverGroup(stateGroup.state, river.river_id)"
                >
                  <div class="flex-1 min-w-0">
                    <p class="text-sm font-medium text-neutral-900 dark:text-white truncate">{{ river.river_name }}</p>
                    <p class="text-xs text-neutral-400 flex items-center gap-1.5 flex-wrap">
                      <span>{{ river.reaches.length }} reach{{ river.reaches.length !== 1 ? 'es' : '' }}<template v-if="river.river_basin"> · {{ river.river_basin }}</template></span>
                      <template v-if="riverHealthMap.get(river.river_slug) as any">
                        <span v-if="(riverHealthMap.get(river.river_slug)?.gauges_unreachable ?? 0) > 0" class="inline-flex items-center gap-0.5 text-red-500 dark:text-red-400 font-medium">
                          <span class="w-1.5 h-1.5 rounded-full bg-red-500 dark:bg-red-400 inline-block" />
                          {{ riverHealthMap.get(river.river_slug)?.gauges_unreachable }} offline
                        </span>
                        <span v-else-if="(riverHealthMap.get(river.river_slug)?.gauges_stale ?? 0) > 0" class="inline-flex items-center gap-0.5 text-amber-500 dark:text-amber-400 font-medium">
                          <span class="w-1.5 h-1.5 rounded-full bg-amber-500 dark:bg-amber-400 inline-block" />
                          {{ riverHealthMap.get(river.river_slug)?.gauges_stale }} stale
                        </span>
                      </template>
                    </p>
                  </div>
                  <svg
                    class="w-4 h-4 text-neutral-400 shrink-0 transition-transform"
                    :class="isRiverExpanded(stateGroup.state, river.river_id) ? 'rotate-90' : ''"
                    viewBox="0 0 20 20" fill="currentColor"
                  >
                    <path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd"/>
                  </svg>
                </div>

                <!-- Expanded reaches -->
                <div v-if="isRiverExpanded(stateGroup.state, river.river_id)" class="bg-neutral-50 dark:bg-neutral-950 border-t border-neutral-100 dark:border-neutral-800">
                  <!-- River actions bar -->
                  <div class="flex items-center justify-between px-6 py-2 border-b border-neutral-100 dark:border-neutral-800">
                    <span class="text-xs text-neutral-400 font-mono">{{ river.river_slug }}</span>
                    <div class="flex items-center gap-2">
                      <UButton size="xs" variant="ghost" color="neutral" @click.stop="openEditRiverBySlug(river.river_slug)">Edit</UButton>
                      <UButton size="xs" variant="ghost" color="neutral" icon="i-heroicons-arrows-up-down" :loading="reorderingRiver === river.river_slug" @click.stop="reorderReaches(river.river_slug)">Re-order</UButton>
                      <UButton size="xs" variant="ghost" color="error" @click.stop="deleteRiver(river.river_slug, river.river_name)">Delete river</UButton>
                    </div>
                  </div>
                  <!-- Reach rows -->
                  <div class="divide-y divide-neutral-100 dark:divide-neutral-800">
                    <div
                      v-for="(reach, reachIdx) in river.reaches" :key="reach.id"
                      class="flex items-center gap-2 px-6 py-2.5 bg-white dark:bg-neutral-900/60"
                    >
                      <!-- Up/down order buttons -->
                      <div class="flex flex-col gap-0.5 shrink-0">
                        <button
                          class="w-4 h-4 flex items-center justify-center text-neutral-300 hover:text-neutral-500 dark:hover:text-neutral-400 disabled:opacity-20 disabled:cursor-default"
                          :disabled="reachIdx === 0"
                          @click.stop="moveReach(stateGroup.state, river.river_id, reachIdx, -1)"
                        >
                          <svg viewBox="0 0 16 16" fill="currentColor" class="w-3 h-3"><path d="M8 4l-5 5h10z"/></svg>
                        </button>
                        <button
                          class="w-4 h-4 flex items-center justify-center text-neutral-300 hover:text-neutral-500 dark:hover:text-neutral-400 disabled:opacity-20 disabled:cursor-default"
                          :disabled="reachIdx === river.reaches.length - 1"
                          @click.stop="moveReach(stateGroup.state, river.river_id, reachIdx, 1)"
                        >
                          <svg viewBox="0 0 16 16" fill="currentColor" class="w-3 h-3"><path d="M8 12l5-5H3z"/></svg>
                        </button>
                      </div>

                      <div class="flex-1 min-w-0">
                        <p class="text-sm font-medium truncate">{{ reach.common_name ?? reach.name }}</p>
                        <p class="text-xs text-neutral-400 truncate">{{ reach.slug }}</p>
                      </div>

                      <div class="flex items-center gap-2 shrink-0 flex-wrap justify-end">
                        <span v-if="reach.state_abbr" class="text-xs font-mono px-1.5 py-0.5 rounded bg-primary-50 dark:bg-primary-950/40 text-primary-600 dark:text-primary-400">{{ reach.state_abbr }}</span>
                        <span
                          class="text-xs px-1.5 py-0.5 rounded"
                          :class="reach.has_centerline
                            ? 'bg-emerald-50 dark:bg-emerald-950/40 text-emerald-600 dark:text-emerald-400'
                            : 'bg-neutral-100 dark:bg-neutral-800 text-neutral-400'"
                        >{{ reach.has_centerline ? 'Line ✓' : 'No line' }}</span>
                        <UButton size="xs" variant="outline" color="error" @click.stop="deleteReachFromGroup(stateGroup.state, river.river_id, reach.slug, reach.common_name ?? reach.name)">Delete</UButton>
                        <NuxtLink :to="`/reaches/${reach.slug}/edit`" class="text-xs text-primary-500 hover:underline">Edit</NuxtLink>
                      </div>
                    </div>
                    <div v-if="river.reaches.length === 0" class="px-6 py-4 text-center text-sm text-neutral-400">No reaches in this state</div>
                  </div>
                </div>
              </template>
              </template>  <!-- end basin loop -->
            </template>

            <div v-if="groupedByStateBasin.length === 0 && !riversLoading" class="px-4 py-8 text-center text-sm text-neutral-400">
              {{ riverSearch ? 'No reaches match your search' : 'No reaches yet' }}
            </div>

            <!-- Pagination -->
            <div v-if="totalPages > 1 || totalRivers > 10" class="flex items-center justify-between px-4 py-2.5 border-t border-neutral-100 dark:border-neutral-800 bg-neutral-50 dark:bg-neutral-900/40 text-xs text-neutral-500">
              <div class="flex items-center gap-2">
                <span>Show</span>
                <select v-model="pageSize" class="rounded border border-neutral-300 dark:border-neutral-600 bg-white dark:bg-neutral-800 px-1.5 py-0.5 text-xs" @change="currentPage = 1">
                  <option :value="10">10</option>
                  <option :value="50">50</option>
                  <option :value="100">100</option>
                </select>
                <span>per page · {{ totalRivers }} rivers total</span>
              </div>
              <div class="flex items-center gap-1">
                <button class="px-2 py-0.5 rounded hover:bg-neutral-200 dark:hover:bg-neutral-700 disabled:opacity-30" :disabled="currentPage <= 1" @click="currentPage--">‹</button>
                <span>{{ currentPage }} / {{ totalPages }}</span>
                <button class="px-2 py-0.5 rounded hover:bg-neutral-200 dark:hover:bg-neutral-700 disabled:opacity-30" :disabled="currentPage >= totalPages" @click="currentPage++">›</button>
              </div>
            </div>

            <!-- Unassigned reaches group -->
            <template v-if="unassignedReaches.length">
              <div
                class="flex items-center gap-3 px-4 py-3 bg-amber-50 dark:bg-amber-950/30 hover:bg-amber-100 dark:hover:bg-amber-900/30 cursor-pointer transition-colors border-t border-neutral-100 dark:border-neutral-800"
                @click="unassignedExpanded = !unassignedExpanded"
              >
                <div class="flex-1 min-w-0">
                  <p class="text-sm font-medium text-amber-800 dark:text-amber-300">Unassigned</p>
                  <p class="text-xs text-amber-600 dark:text-amber-500">No river association — assign via reach edit page</p>
                </div>
                <span class="text-xs text-amber-600 dark:text-amber-400 shrink-0">{{ unassignedReaches.length }} reach{{ unassignedReaches.length !== 1 ? 'es' : '' }}</span>
                <svg class="w-4 h-4 text-amber-400 shrink-0 transition-transform" :class="unassignedExpanded ? 'rotate-90' : ''" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd"/>
                </svg>
              </div>
              <div v-if="unassignedExpanded" class="bg-neutral-50 dark:bg-neutral-950 border-t border-neutral-100 dark:border-neutral-800 divide-y divide-neutral-100 dark:divide-neutral-800">
                <div
                  v-for="reach in unassignedReaches" :key="reach.id"
                  class="flex items-center gap-3 px-6 py-2.5 bg-white dark:bg-neutral-900/60"
                >
                  <div class="flex-1 min-w-0">
                    <p class="text-sm font-medium truncate">{{ reach.common_name ?? reach.name }}</p>
                    <p class="text-xs text-neutral-400 truncate">
                      {{ reach.slug }}
                      <span v-if="reach.river_name" class="text-amber-500"> · {{ reach.river_name }}</span>
                    </p>
                  </div>
                  <div class="flex items-center gap-2 shrink-0 flex-wrap justify-end">
                    <span
                      class="text-xs px-1.5 py-0.5 rounded"
                      :class="reach.has_centerline
                        ? 'bg-emerald-50 dark:bg-emerald-950/40 text-emerald-600 dark:text-emerald-400'
                        : 'bg-neutral-100 dark:bg-neutral-800 text-neutral-400'"
                    >{{ reach.has_centerline ? 'Line ✓' : 'No line' }}</span>
                    <UButton size="xs" variant="outline" color="error" @click="deleteReachFromGroup('', '', reach.slug, reach.common_name ?? reach.name)">Delete</UButton>
                    <NuxtLink :to="`/reaches/${reach.slug}/edit`" class="text-xs text-primary-500 hover:underline">Edit</NuxtLink>
                  </div>
                </div>
              </div>
            </template>
          </div>
        </div>

        <!-- Users tab (site admin only) -->
        <div v-if="activeTab === 'users'">
          <div class="flex items-center justify-between mb-4">
            <p class="text-sm text-neutral-500">Role assignments</p>
            <UButton size="xs" icon="i-heroicons-plus" @click="assignRoleOpen = true">Assign role</UButton>
          </div>

          <div v-if="rolesLoading" class="space-y-2">
            <div v-for="i in 3" :key="i" class="h-12 rounded-lg bg-neutral-100 dark:bg-neutral-800 animate-pulse" />
          </div>

          <div v-else class="divide-y divide-neutral-100 dark:divide-neutral-800 rounded-lg border border-neutral-200 dark:border-neutral-700 overflow-hidden">
            <div
              v-for="ur in userRoles" :key="ur.id"
              class="flex items-center gap-3 px-4 py-3 bg-white dark:bg-neutral-900"
            >
              <div class="flex-1 min-w-0">
                <p class="text-xs font-mono text-neutral-500 truncate">{{ ur.user_id }}</p>
                <p class="text-xs text-neutral-400">
                  <span class="font-medium text-neutral-700 dark:text-neutral-300">{{ ur.role }}</span>
                  <template v-if="ur.river_name"> · {{ ur.river_name }}</template>
                </p>
              </div>
              <button
                class="text-xs text-red-400 hover:text-red-600 transition-colors shrink-0 px-2 py-1 rounded"
                @click="revokeRole(ur.id)"
              >Revoke</button>
            </div>
            <div v-if="userRoles.length === 0" class="px-4 py-8 text-center text-sm text-neutral-400">No role assignments</div>
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
            <div class="flex items-center gap-2">
              <UInput v-model="newRiver.gnis_id" placeholder="00179365" class="max-w-40" @blur="lookupGNIS(newRiver.gnis_id, newRiver)" />
              <span v-if="gnisLookupLoading" class="text-xs text-neutral-400 animate-pulse">Looking up…</span>
              <span v-else-if="gnisLookupMsg" class="text-xs text-neutral-500">{{ gnisLookupMsg }}</span>
            </div>
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
              <span v-if="editingRiver.basin_locked" class="text-xs text-neutral-400 shrink-0">locked</span>
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
            <div class="flex items-center gap-2">
              <UInput v-model="editingRiver.gnis_id" placeholder="00179365" class="max-w-40" @blur="lookupGNIS(editingRiver!.gnis_id, editingRiver!)" />
              <span v-if="gnisLookupLoading" class="text-xs text-neutral-400 animate-pulse">Looking up…</span>
              <span v-else-if="gnisLookupMsg" class="text-xs text-neutral-500">{{ gnisLookupMsg }}</span>
            </div>
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

    <!-- New reach modal -->
    <Teleport to="body">
      <Transition
        enter-active-class="transition-opacity duration-150"
        enter-from-class="opacity-0"
        enter-to-class="opacity-100"
        leave-active-class="transition-opacity duration-100"
        leave-from-class="opacity-100"
        leave-to-class="opacity-0"
      >
        <div
          v-if="authorModalOpen"
          class="fixed inset-0 z-50 flex flex-col bg-neutral-50 dark:bg-neutral-950 overflow-y-auto"
        >
          <div class="sticky top-0 z-10 flex items-center justify-between px-4 py-3 bg-white/90 dark:bg-neutral-950/90 backdrop-blur-sm border-b border-neutral-200 dark:border-neutral-800">
            <h2 class="text-sm font-semibold text-neutral-800 dark:text-neutral-100">New reach</h2>
            <button
              class="p-1.5 rounded-md text-neutral-400 hover:text-neutral-700 dark:hover:text-neutral-200 hover:bg-neutral-100 dark:hover:bg-neutral-800 transition-colors"
              @click="authorModalOpen = false"
            >
              <svg class="w-4 h-4" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
                <path d="M6 6l8 8M14 6l-8 8"/>
              </svg>
            </button>
          </div>
          <div class="flex-1 max-w-5xl w-full mx-auto px-4 py-6">
            <ReachAuthor @created="onAuthorCreated" @cancel="authorModalOpen = false" />
          </div>
        </div>
      </Transition>
    </Teleport>

    <!-- Assign role modal -->
    <UModal v-model:open="assignRoleOpen" title="Assign role">
      <template #body>
        <div class="space-y-3">
          <UFormField label="User ID (Supabase UUID)">
            <UInput v-model="newRole.user_id" placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" />
          </UFormField>
          <UFormField label="Role">
            <select v-model="newRole.role" class="w-full rounded-md border border-neutral-300 dark:border-neutral-700 bg-white dark:bg-neutral-900 px-3 py-2 text-sm">
              <option value="data_admin">data_admin</option>
              <option value="site_admin">site_admin</option>
            </select>
          </UFormField>
          <UFormField label="River (optional — leave blank for global)">
            <select v-model="newRole.river_id" class="w-full rounded-md border border-neutral-300 dark:border-neutral-700 bg-white dark:bg-neutral-900 px-3 py-2 text-sm">
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
const router = useRouter()

// ── New reach modal ───────────────────────────────────────────────────────────
const authorModalOpen = ref(false)

function onAuthorCreated(slug: string) {
  authorModalOpen.value = false
  router.push(`/reaches/${slug}/edit`)
}

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
  const tabs = [{ key: 'rivers', label: 'Rivers' }]
  if (isAdmin.value) tabs.push({ key: 'users', label: 'Users' })
  return tabs
})

// ── Rivers ────────────────────────────────────────────────────────────────────
interface River { id: string; slug: string; name: string; gnis_id: string | null; basin: string | null; basin_locked: boolean; state_abbr: string | null; huc8: string | null; verified: boolean; reach_count: number; gauges_degraded: number; gauges_stale: number; gauges_unreachable: number }
interface RiverDetail extends River {
  reaches: { id: string; slug: string; name: string; common_name: string | null; has_centerline: boolean; state_abbr: string | null; river_order: number | null }[]
}
interface GroupedReach { id: string; slug: string; name: string; common_name: string | null; river_order: number | null; state_abbr: string; has_centerline: boolean }
interface GroupedRiver { river_id: string; river_slug: string; river_name: string; river_basin: string; reaches: GroupedReach[] }
interface GroupedState { state: string; rivers: GroupedRiver[] }
interface GroupedBasin { basin: string; rivers: GroupedRiver[] }
interface GroupedStateBasin { state: string; basins: GroupedBasin[] }

const rivers = ref<River[]>([])
const riverHealthMap = computed(() => {
  const m = new Map<string, Pick<River, 'gauges_degraded' | 'gauges_stale' | 'gauges_unreachable'>>()
  for (const rv of rivers.value) m.set(rv.slug, rv)
  return m
})
const groupedReaches = ref<GroupedState[]>([])
const riversLoading = ref(false)
const riverSearch = ref('')
const expandedRiverKeys = ref<Set<string>>(new Set())

// Pagination
const pageSize = ref(50)
const currentPage = ref(1)
watch(riverSearch, () => { currentPage.value = 1 })

function riverKey(state: string, riverId: string) { return `${state}::${riverId}` }
function isRiverExpanded(state: string, riverId: string) { return expandedRiverKeys.value.has(riverKey(state, riverId)) }
function toggleRiverGroup(state: string, riverId: string) {
  const key = riverKey(state, riverId)
  const next = new Set(expandedRiverKeys.value)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  expandedRiverKeys.value = next
}

// Rivers needing review (verified = false) — populated once 000069 migration runs
const needsReviewRivers = computed(() => rivers.value.filter(r => !r.verified))
const needsReviewExpanded = ref(false)

// Filter + basin grouping
const filteredGroupedReaches = computed(() => {
  const q = riverSearch.value.toLowerCase()
  const base = q
    ? groupedReaches.value
        .map(sg => ({
          ...sg,
          rivers: sg.rivers
            .map(rv => ({
              ...rv,
              reaches: rv.reaches.filter(re =>
                re.name.toLowerCase().includes(q) ||
                (re.common_name ?? '').toLowerCase().includes(q) ||
                rv.river_name.toLowerCase().includes(q) ||
                re.state_abbr.toLowerCase().includes(q)
              ),
            }))
            .filter(rv => rv.reaches.length > 0),
        }))
        .filter(sg => sg.rivers.length > 0)
    : groupedReaches.value
  return base
})

// State→Basin→River hierarchy for rendering
const groupedByStateBasin = computed((): GroupedStateBasin[] => {
  return filteredGroupedReaches.value.map(sg => {
    const basinMap = new Map<string, GroupedRiver[]>()
    for (const rv of sg.rivers) {
      const b = rv.river_basin || '—'
      if (!basinMap.has(b)) basinMap.set(b, [])
      basinMap.get(b)!.push(rv)
    }
    const basins: GroupedBasin[] = Array.from(basinMap.entries()).map(([basin, rivers]) => ({ basin, rivers }))
    return { state: sg.state, basins }
  })
})

// Paginate by total river count across all state groups
const allRivers = computed(() => filteredGroupedReaches.value.flatMap(sg => sg.rivers))
const totalRivers = computed(() => allRivers.value.length)
const totalPages = computed(() => Math.max(1, Math.ceil(totalRivers.value / pageSize.value)))
const pagedRiverIds = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  return new Set(allRivers.value.slice(start, start + pageSize.value).map(r => r.river_id))
})

function isRiverVisible(riverId: string) { return pagedRiverIds.value.has(riverId) }

const unassignedReaches = ref<{ id: string; slug: string; name: string; common_name: string | null; river_name: string | null; has_centerline: boolean }[]>([])
const unassignedExpanded = ref(false)

async function loadRivers() {
  riversLoading.value = true
  const token = await getToken()
  if (!token) { riversLoading.value = false; return }
  try {
    const [rRes, uRes, gRes] = await Promise.all([
      fetch(`${apiBase}/api/v1/admin/rivers`, { headers: { Authorization: `Bearer ${token}` } }),
      fetch(`${apiBase}/api/v1/admin/reaches/unassigned`, { headers: { Authorization: `Bearer ${token}` } }),
      fetch(`${apiBase}/api/v1/admin/reaches/grouped`, { headers: { Authorization: `Bearer ${token}` } }),
    ])
    if (rRes.ok) rivers.value = await rRes.json()
    if (uRes.ok) unassignedReaches.value = await uRes.json()
    if (gRes.ok) groupedReaches.value = await gRes.json()
  } finally {
    riversLoading.value = false
  }
}

async function openEditRiverBySlug(slug: string) {
  const token = await getToken()
  const res = await fetch(`${apiBase}/api/v1/admin/rivers/${slug}`, { headers: { Authorization: `Bearer ${token}` } })
  if (res.ok) {
    const rv: RiverDetail = await res.json()
    openEditRiver(rv)
  }
}

async function moveReach(state: string, riverId: string, reachIdx: number, direction: 1 | -1) {
  const sg = groupedReaches.value.find(s => s.state === state)
  if (!sg) return
  const rv = sg.rivers.find(r => r.river_id === riverId)
  if (!rv) return
  const swapIdx = reachIdx + direction
  if (swapIdx < 0 || swapIdx >= rv.reaches.length) return

  const a = rv.reaches[reachIdx]
  const b = rv.reaches[swapIdx]
  const orderA = a.river_order ?? (reachIdx + 1)
  const orderB = b.river_order ?? (swapIdx + 1)

  // Optimistic update
  rv.reaches[reachIdx] = { ...b, river_order: orderA }
  rv.reaches[swapIdx] = { ...a, river_order: orderB }
  groupedReaches.value = [...groupedReaches.value]

  const token = await getToken()
  await Promise.all([
    fetch(`${apiBase}/api/v1/admin/reaches/${a.slug}`, {
      method: 'PATCH',
      headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ river_order: orderB }),
    }),
    fetch(`${apiBase}/api/v1/admin/reaches/${b.slug}`, {
      method: 'PATCH',
      headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ river_order: orderA }),
    }),
  ])
}

async function deleteReachFromGroup(state: string, riverId: string, reachSlug: string, displayName: string) {
  if (!confirm(`Permanently delete "${displayName}"?\n\nThis removes all rapids, access points, and features. Gauges are unlinked but kept.`)) return
  const token = await getToken()
  const res = await fetch(`${apiBase}/api/v1/reaches/${reachSlug}`, {
    method: 'DELETE',
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) { alert(`Delete failed: ${res.status}`); return }
  // Remove optimistically from grouped state, then full refresh
  const sg = groupedReaches.value.find(s => s.state === state)
  if (sg) {
    const rv = sg.rivers.find(r => r.river_id === riverId)
    if (rv) rv.reaches = rv.reaches.filter(r => r.slug !== reachSlug)
    groupedReaches.value = [...groupedReaches.value]
  }
  loadRivers()
}

const reorderingRiver = ref<string | null>(null)
async function reorderReaches(riverSlug: string) {
  reorderingRiver.value = riverSlug
  try {
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/admin/rivers/${riverSlug}/reorder-reaches`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
    })
    if (!res.ok) { alert(`Reorder failed: ${res.status}`); return }
    loadRivers()
  } finally {
    reorderingRiver.value = null
  }
}

async function deleteRiver(riverSlug: string, riverName: string) {
  if (!confirm(`Permanently delete "${riverName}"?\n\nAll reaches will be unlinked but not deleted.`)) return
  const token = await getToken()
  const res = await fetch(`${apiBase}/api/v1/admin/rivers/${riverSlug}`, {
    method: 'DELETE',
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) { alert(`Delete failed: ${res.status}`); return }
  loadRivers()
}

// Create river
const createRiverOpen = ref(false)
const createLoading = ref(false)
const newRiver = ref({ name: '', slug: '', basin: '', state_abbr: '', gnis_id: '', huc8: '' })
const gnisLookupLoading = ref(false)
const gnisLookupMsg = ref('')

function autoSlug() {
  newRiver.value.slug = newRiver.value.name
    .toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-/, '')
}

async function lookupGNIS(gnisId: string, target: { state_abbr: string; basin: string; huc8: string }) {
  if (!gnisId.trim()) return
  gnisLookupLoading.value = true
  gnisLookupMsg.value = ''
  try {
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/admin/rivers/gnis-lookup?gnis_id=${encodeURIComponent(gnisId.trim())}`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    const data = await res.json()
    if (!res.ok) { gnisLookupMsg.value = data.error ?? `Error ${res.status}`; return }
    if (data.state_abbr && !target.state_abbr) target.state_abbr = data.state_abbr
    if (data.basin && !target.basin) target.basin = data.basin
    if (data.huc8) target.huc8 = data.huc8
    gnisLookupMsg.value = `Found: ${data.states ?? data.state_abbr}${data.huc8 ? ' · HUC8 ' + data.huc8 : ''}`
  } catch (err: any) {
    gnisLookupMsg.value = err?.message ?? 'Lookup failed'
  } finally {
    gnisLookupLoading.value = false
  }
}

async function createRiver() {
  createLoading.value = true
  const token = await getToken()
  try {
    const slug = newRiver.value.slug.replace(/-+$/, '')
    const res = await fetch(`${apiBase}/api/v1/admin/rivers`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({
        slug,
        name: newRiver.value.name,
        basin: newRiver.value.basin || null,
        state_abbr: newRiver.value.state_abbr || null,
        gnis_id: newRiver.value.gnis_id || null,
        huc8: newRiver.value.huc8 || null,
      }),
    })
    if (res.ok) {
      createRiverOpen.value = false
      newRiver.value = { name: '', slug: '', basin: '', state_abbr: '', gnis_id: '', huc8: '' }
      loadRivers()
      // Background auto-fill: resolve state/basin from NHD or GNIS
      fetch(`${apiBase}/api/v1/admin/rivers/${slug}/auto-fill`, {
        headers: { Authorization: `Bearer ${token}` },
      }).then(r => r.ok ? r.json() : null).then(data => {
        if (!data) return
        const patch: Record<string, string> = {}
        if (data.state_abbr) patch.state_abbr = data.state_abbr
        if (data.basin) patch.basin = data.basin
        if (data.huc8) patch.huc8 = data.huc8
        if (Object.keys(patch).length === 0) return
        fetch(`${apiBase}/api/v1/admin/rivers/${slug}`, {
          method: 'PUT',
          headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
          body: JSON.stringify(patch),
        }).then(() => loadRivers())
      }).catch(() => {})
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
const editingRiver = ref<{ slug: string; name: string; basin: string; basin_locked: boolean; state_abbr: string; gnis_id: string; huc8: string } | null>(null)

function openEditRiver(river: RiverDetail) {
  editingRiver.value = {
    slug: river.slug,
    name: river.name,
    basin: river.basin ?? '',
    basin_locked: river.basin_locked,
    state_abbr: river.state_abbr ?? '',
    gnis_id: river.gnis_id ?? '',
    huc8: (river as any).huc8 ?? '',
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
        huc8: editingRiver.value.huc8 || null,
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


</script>
