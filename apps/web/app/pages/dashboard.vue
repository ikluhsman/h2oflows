<template>
  <div class="min-h-screen bg-neutral-50 dark:bg-neutral-950">
    <AppHeader />

    <!-- Dashboard tab bar (only when authenticated and dashboards loaded) -->
    <DashboardTabBar
      v-if="isAuthenticated && db.loaded.value && db.dashboards.value.length"
      :dashboards="db.dashboards.value"
      :active-dashboard-id="db.activeDashboardId.value"
      @select="onSelectDashboard"
      @new="newDashboardOpen = true"
      @delete="onDeleteDashboard"
      @rename="(slug, name) => db.rename(slug, name)"
    />

    <!-- Sticky controls bar — always shown when authenticated -->
    <div
      v-if="isAuthenticated && db.loaded.value"
      class="sticky z-10 bg-neutral-50/95 dark:bg-neutral-950/95 backdrop-blur-sm border-b border-neutral-200 dark:border-neutral-800"
      :class="db.dashboards.value.length ? 'top-24' : 'top-12.75'"
    >
      <div class="max-w-5xl mx-auto px-4 py-2 flex items-center justify-between gap-2">
        <div v-if="hasAnyContent" class="flex items-center gap-2">
          <!-- View mode toggle -->
          <div class="flex items-center gap-0.5 bg-neutral-100 dark:bg-neutral-800 rounded-lg p-1">
            <button
              v-for="m in VIEW_MODES" :key="m.key"
              class="p-1.5 rounded-md transition-colors"
              :class="viewMode === m.key
                ? 'bg-white dark:bg-neutral-700 text-neutral-900 dark:text-white shadow-sm'
                : 'text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-300'"
              :title="m.label"
              @click="setViewMode(m.key)"
            >
              <svg v-if="m.key === 'list'" class="w-4 h-4" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round">
                <line x1="2" y1="4" x2="14" y2="4"/><line x1="2" y1="8" x2="14" y2="8"/><line x1="2" y1="12" x2="14" y2="12"/>
              </svg>
              <svg v-else-if="m.key === 'comfortable'" class="w-4 h-4" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
                <rect x="1" y="1" width="6" height="6" rx="1"/><rect x="9" y="1" width="6" height="6" rx="1"/>
                <rect x="1" y="9" width="6" height="6" rx="1"/><rect x="9" y="9" width="6" height="6" rx="1"/>
              </svg>
              <svg v-else class="w-4 h-4" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
                <rect x="2" y="2" width="12" height="4" rx="1"/><rect x="2" y="7" width="12" height="3" rx="1"/><rect x="2" y="11" width="12" height="3" rx="1"/>
              </svg>
            </button>
          </div>
          <!-- Group by gauge toggle -->
          <button
            v-if="hasSharedGauges"
            class="flex items-center gap-1.5 px-2 py-1 rounded-md text-xs font-medium transition-colors"
            :class="groupByGauge
              ? 'bg-primary-100 dark:bg-primary-900/40 text-primary-700 dark:text-primary-300 shadow-sm'
              : 'bg-neutral-100 dark:bg-neutral-800 text-neutral-500 dark:text-neutral-400 hover:text-neutral-700 dark:hover:text-neutral-200'"
            title="Group by gauge"
            @click="groupByGauge = !groupByGauge"
          >
            <svg v-if="groupByGauge" class="w-4 h-4 shrink-0" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
              <rect x="1" y="5" width="5" height="6" rx="1"/>
              <rect x="10" y="5" width="5" height="6" rx="1"/>
              <line x1="6" y1="8" x2="10" y2="8"/>
            </svg>
            <svg v-else class="w-4 h-4 shrink-0" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
              <rect x="1" y="5" width="5" height="6" rx="1"/>
              <rect x="10" y="5" width="5" height="6" rx="1"/>
              <line x1="6.5" y1="6.5" x2="9.5" y2="9.5"/>
            </svg>
            <span class="hidden sm:inline">Group</span>
          </button>
          <!-- Expand / Collapse all -->
          <button
            v-if="byStateTree.length > 0"
            class="text-xs text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-300 font-medium transition-colors whitespace-nowrap"
            @click="toggleAllSections"
          >{{ allExpanded ? 'Collapse all' : 'Expand all' }}</button>
        </div>
        <div v-else class="flex-1" />
        <UButton size="xs" color="neutral" variant="outline" icon="i-heroicons-plus" @click="searchOpen = true">
          Add gauge
        </UButton>
      </div>
    </div>

    <main class="max-w-5xl mx-auto px-2 sm:px-4 py-6 pb-20 sm:pb-6 space-y-8">

      <!-- Empty state -->
      <div v-if="!hasAnyContent" class="mt-20 flex flex-col items-center gap-4 text-center">
        <div class="text-5xl">🌊</div>
        <h2 class="text-xl font-semibold">No reaches yet</h2>
        <p class="text-neutral-500 max-w-sm text-sm">
          Search for a reach or gauge and add it to your dashboard.
        </p>
        <UButton color="primary" @click="searchOpen = true">Find a gauge</UButton>
      </div>

      <!-- Reaches grouped by basin → river -->
      <template v-if="hasAnyContent">

        <section v-for="stateGroup in byStateTree" :key="stateGroup.name" class="mb-4">
          <!-- State header: large, collapsible, h1+hr style -->
          <button class="flex items-center gap-3 w-full text-left mb-3" @click="toggleState(stateGroup.name)">
            <svg
              class="w-3.5 h-3.5 text-neutral-400 dark:text-neutral-500 shrink-0 transition-transform duration-200"
              :class="{ 'rotate-90': !collapsedStates.has(stateGroup.name) }"
              viewBox="0 0 20 20" fill="currentColor"
            >
              <path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd" />
            </svg>
            <h1 class="text-2xl font-bold text-neutral-900 dark:text-white shrink-0">{{ stateGroup.name === '—' ? 'No State' : stateGroup.name }}</h1>
            <div class="flex-1 h-px bg-neutral-300 dark:bg-neutral-700" />
            <span class="text-sm text-neutral-400 shrink-0">({{ stateGroup.reachCount }})</span>
          </button>

          <template v-if="!collapsedStates.has(stateGroup.name)">
            <div v-for="basin in stateGroup.basins" :key="basin.name" class="mb-4">
              <!-- Basin header: collapsible -->
              <div class="flex items-center gap-2 w-full mb-3">
                <button class="flex items-center gap-2 text-left shrink-0" @click="toggleBasin(stateGroup.name, basin.name)">
                  <svg
                    class="w-3 h-3 text-neutral-400 dark:text-neutral-500 shrink-0 transition-transform duration-150"
                    :class="{ 'rotate-90': !collapsedBasins.has(basinKey(stateGroup.name, basin.name)) }"
                    viewBox="0 0 20 20" fill="currentColor"
                  >
                    <path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd" />
                  </svg>
                  <h2 class="text-sm font-bold text-neutral-700 dark:text-neutral-200 uppercase tracking-wide">{{ basin.name }} Basin</h2>
                  <span class="text-xs text-neutral-400">({{ basin.reachCount }})</span>
                </button>
                <NuxtLink
                  :to="`/basin/${slugifyBasin(basin.name)}`"
                  class="p-0.5 rounded text-primary-500 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300 transition-colors shrink-0"
                  title="View basin map"
                >
                  <svg class="w-4 h-4" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
                    <circle cx="8" cy="3" r="1.5"/>
                    <circle cx="4" cy="13" r="1.5"/>
                    <circle cx="12" cy="13" r="1.5"/>
                    <line x1="8" y1="4.5" x2="8" y2="7"/>
                    <path d="M8 7 Q4 9 4 11.5"/>
                    <path d="M8 7 Q12 9 12 11.5"/>
                  </svg>
                </NuxtLink>
                <div class="flex-1 h-px bg-neutral-200 dark:bg-neutral-700/60" />
              </div>

              <template v-if="!collapsedBasins.has(basinKey(stateGroup.name, basin.name))">
              <!-- Reaches grouped by river -->
              <div class="mb-2">
                <template v-for="river in basin.rivers" :key="river.name">
                  <!-- River section divider -->
                  <div class="flex items-center gap-2 mt-4 first:mt-0 mb-2">
                    <svg class="w-4 h-4 text-primary-500/70 dark:text-primary-400/70 shrink-0" viewBox="0 0 32 32" fill="none" aria-hidden="true">
                      <path d="M4 14c3-6 6-9 8-9s5 9 8 9 5-9 8-9" stroke="currentColor" stroke-width="3" stroke-linecap="round"/>
                      <path d="M4 22c3-6 6-9 8-9s5 9 8 9 5-9 8-9" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" opacity="0.6"/>
                    </svg>
                    <span class="text-base font-semibold text-primary-600 dark:text-primary-400 shrink-0">{{ river.name }}</span>
                    <div class="flex-1 h-px bg-neutral-200 dark:bg-neutral-700" />
                  </div>
                  <!-- Cards wrapper -->
                  <template v-if="groupByGauge">
                    <template v-for="split in [splitReachGroups(river.reaches)]" :key="'split'">
                      <div v-if="split.gaugeGroups.length > 0" :class="viewMode === 'list' ? 'space-y-1.5' : cardGridClass">
                        <GaugeReachGroup
                          v-for="group in split.gaugeGroups"
                          :key="group.lead.id"
                          :lead-gauge="group.lead"
                          :reach-items="group.all"
                          :density="viewMode"
                          :hide-river-name="true"
                          @open="(g, mode) => openGauge(g, mode)"
                          @remove-group="group.all.forEach(g => removeAndSync(g.id, g.contextReachSlug))"
                        />
                      </div>
                      <!-- Ungrouped: list = one card; card modes = individual cards in grid -->
                      <template v-if="split.ungrouped.length > 0">
                        <DashboardReachGroup
                          v-if="viewMode === 'list'"
                          :reaches="split.ungrouped"
                          density="list"
                          :hazard-slugs="hazardSlugs"
                          :class="split.gaugeGroups.length > 0 ? 'mt-3' : ''"
                          @open="(g, mode) => openGauge(g, mode)"
                          @remove="(g) => removeAndSync(g.id, g.contextReachSlug)"
                        />
                        <div v-else :class="[cardGridClass, split.gaugeGroups.length > 0 ? 'mt-3' : '']">
                          <DashboardReachGroup
                            v-for="reach in split.ungrouped"
                            :key="`${reach.id}::${reach.contextReachSlug}`"
                            :reaches="[reach]"
                            :density="viewMode"
                            :hazard-slugs="hazardSlugs"
                            @open="(g, mode) => openGauge(g, mode)"
                            @remove="(g) => removeAndSync(g.id, g.contextReachSlug)"
                          />
                        </div>
                      </template>
                    </template>
                  </template>
                  <template v-else>
                    <!-- List: all reaches in one grouped card -->
                    <DashboardReachGroup
                      v-if="viewMode === 'list'"
                      :reaches="river.reaches"
                      density="list"
                      :hazard-slugs="hazardSlugs"
                      @open="(g, mode) => openGauge(g, mode)"
                      @remove="(g) => removeAndSync(g.id, g.contextReachSlug)"
                    />
                    <!-- Card modes: each reach = own card in grid -->
                    <div v-else :class="cardGridClass">
                      <DashboardReachGroup
                        v-for="reach in river.reaches"
                        :key="`${reach.id}::${reach.contextReachSlug}`"
                        :reaches="[reach]"
                        :density="viewMode"
                        :hazard-slugs="hazardSlugs"
                        @open="(g, mode) => openGauge(g, mode)"
                        @remove="(g) => removeAndSync(g.id, g.contextReachSlug)"
                      />
                    </div>
                  </template>
                </template><!-- end v-for river -->
              </div>

              <!-- Standalone gauges (no reach context) -->
              <div v-if="basin.standaloneGauges.length > 0" class="mb-2 mt-1">
                <div class="flex items-center gap-2 py-1">
                  <svg class="w-3.5 h-3.5 text-neutral-400 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M12 14a2 2 0 1 0 0-4 2 2 0 0 0 0 4z"/>
                    <path d="M12 12 16 8"/>
                    <path d="M3 12a9 9 0 0 1 18 0"/>
                  </svg>
                  <span class="text-xs font-semibold text-neutral-500 dark:text-neutral-400">Standalone Gauges</span>
                </div>
                <div :class="reachContainerClass">
                  <div
                    v-for="g in basin.standaloneGauges"
                    :key="g.id"
                    class="rounded-xl border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 cursor-pointer hover:bg-neutral-50 dark:hover:bg-neutral-800/30 transition-colors"
                    :class="viewMode === 'list' ? 'px-3 py-1.5' : viewMode === 'comfortable' ? 'px-3 py-2.5' : 'px-4 py-3'"
                    @click="openGauge(g, 'gauge')"
                  >
                    <div class="flex items-center gap-3">
                      <svg class="w-4 h-4 text-neutral-400 dark:text-neutral-500 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                        <path d="M12 14a2 2 0 1 0 0-4 2 2 0 0 0 0 4z"/>
                        <path d="M12 12 16 8"/>
                        <path d="M3 12a9 9 0 0 1 18 0"/>
                      </svg>
                      <span class="flex-1 min-w-0 text-sm font-semibold text-neutral-800 dark:text-neutral-200 truncate">
                        {{ g.name ?? `${g.source.toUpperCase()} ${g.externalId}` }}
                      </span>
                      <div class="w-20 shrink-0 hidden sm:block h-5 opacity-50 pointer-events-none">
                        <GaugeSparkline :gauge-id="g.id" flow-status="unknown" color="#3b82f6" compact />
                      </div>
                      <span :class="viewMode === 'list' ? 'text-sm font-bold tabular-nums text-neutral-900 dark:text-white' : 'text-2xl font-bold tabular-nums text-neutral-900 dark:text-white leading-none'">
                        {{ g.currentCfs != null ? g.currentCfs.toLocaleString() : '—' }}
                      </span>
                      <span class="text-xs text-neutral-400">cfs</span>
                      <button
                        class="rounded p-1 text-neutral-300 dark:text-neutral-600 hover:text-red-400 dark:hover:text-red-400 transition-colors shrink-0"
                        aria-label="Remove from dashboard"
                        @click.stop="removeAndSync(g.id)"
                      >
                        <svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm4 0a1 1 0 112 0v6a1 1 0 11-2 0V8z" clip-rule="evenodd"/></svg>
                      </button>
                    </div>
                  </div>
                </div>
              </div>
              </template><!-- end basin v-if -->
            </div>
          </template>
        </section>

        <section v-if="aggregateGauges.length >= 2" class="border border-neutral-300 dark:border-neutral-700 rounded-xl p-4">
          <div class="flex items-center justify-between mb-4">
            <h3 class="font-semibold text-sm">{{ aggregateLabel }} · Flow Comparison</h3>
            <UButton size="xs" color="neutral" variant="ghost" icon="i-heroicons-x-mark" @click="closeAggregate" />
          </div>
          <AggregateGraph :gauges="aggregateGauges" />
        </section>

        <!-- My Reaches section — all on primary; watchlist-filtered on secondary dashboards -->
        <section v-if="visibleUserReaches.length > 0">
          <div class="flex items-center gap-2 mb-3">
            <h2 class="text-sm font-semibold text-neutral-500 uppercase tracking-wide">My Reaches</h2>
            <div class="flex-1 h-px bg-neutral-200 dark:bg-neutral-800" />
            <div v-if="activeUserReaches.some(r => hiddenReaches.has(r.id))" class="relative" ref="reachAddWrap">
              <button
                class="text-xs text-primary-500 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300 transition-colors flex items-center gap-1"
                @click="reachAddOpen = !reachAddOpen"
              >
                <svg class="w-3 h-3" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clip-rule="evenodd"/></svg>
                Add
              </button>
              <div v-if="reachAddOpen" class="absolute right-0 top-full mt-1 z-50 w-52 rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 shadow-lg overflow-hidden">
                <button
                  v-for="r in activeUserReaches.filter(r => hiddenReaches.has(r.id))"
                  :key="r.id"
                  class="w-full text-left px-3 py-2 text-sm hover:bg-primary-50 dark:hover:bg-primary-950/30 transition-colors"
                  @click="showReach(r.id); reachAddOpen = false"
                >{{ r.name }}</button>
              </div>
            </div>
            <NuxtLink to="/my/reaches" class="text-xs text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-300 transition-colors">Manage</NuxtLink>
          </div>
          <!-- List view -->
          <div v-if="viewMode === 'list'" class="space-y-1.5">
            <div
              v-for="r in activeUserReaches.filter(r => !hiddenReaches.has(r.id))"
              :key="r.id"
              class="flex items-center gap-2 sm:gap-3 px-3 py-2 rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 transition-all duration-200 cursor-pointer"
              @click="openUserReach(r)"
            >
              <div class="min-w-0 flex-1 flex items-center gap-1.5">
                <!-- My reach origin indicator — list -->
                <svg class="w-3 h-3 shrink-0 hidden sm:block text-neutral-400/40 dark:text-neutral-500/30" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true"><path fill-rule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" clip-rule="evenodd"/></svg>
                <span class="text-sm font-medium truncate">{{ r.name }}</span>
                <NuxtLink
                  :to="`/my/reaches/${r.slug}`"
                  class="shrink-0 text-neutral-300 dark:text-neutral-600 hover:text-primary-500 dark:hover:text-primary-400 transition-colors"
                  title="Edit reach"
                  @click.stop
                >
                  <svg class="w-3 h-3" viewBox="0 0 20 20" fill="currentColor"><path d="M11 3a1 1 0 100 2h2.586l-6.293 6.293a1 1 0 101.414 1.414L15 6.414V9a1 1 0 102 0V4a1 1 0 00-1-1h-5z"/><path d="M5 5a2 2 0 00-2 2v8a2 2 0 002 2h8a2 2 0 002-2v-3a1 1 0 10-2 0v3H5V7h3a1 1 0 000-2H5z"/></svg>
                </NuxtLink>
              </div>
              <div class="flex items-center gap-2 shrink-0">
                <div v-if="r.gauge_id" class="w-32 shrink-0 hidden sm:block">
                  <GaugeSparkline :gauge-id="r.gauge_id" :flow-status="(r.flow_status as any)" :flow-band-label="r.flow_band ?? null" compact class="h-full w-full" />
                </div>
                <div v-else-if="r.custom_gauge_slug" class="w-32 shrink-0 hidden sm:block">
                  <CustomGaugeSparkline :gauge-slug="r.custom_gauge_slug" compact class="h-full w-full" />
                </div>
                <span
                  v-if="r.flow_status !== 'unknown' || r.flow_band"
                  :class="['hidden sm:inline-flex items-center rounded-md px-1.5 py-0.5 text-xs font-medium shrink-0', reachBadgeClass(r)]"
                >{{ reachStatusLabel(r) }}</span>
                <span class="text-base font-bold tabular-nums min-w-14 text-right" :style="{ color: bandSolid(r.flow_band, r.flow_status) }">
                  {{ r.current_cfs != null ? r.current_cfs.toLocaleString() : '—' }}<span class="text-xs font-normal text-neutral-400 dark:text-neutral-500 ml-0.5">cfs</span>
                </span>
              </div>
              <button
                class="rounded p-1 text-neutral-300 dark:text-neutral-600 hover:text-red-400 dark:hover:text-red-400 transition-colors shrink-0"
                aria-label="Remove from dashboard"
                @click.stop="hideReach(r.id)"
              >
                <svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm4 0a1 1 0 112 0v6a1 1 0 11-2 0V8z" clip-rule="evenodd"/></svg>
              </button>
            </div>
          </div>
          <!-- Card views (comfortable / full) -->
          <div v-else :class="cardGridClass">
            <div
              v-for="r in activeUserReaches.filter(r => !hiddenReaches.has(r.id))"
              :key="r.id"
              class="relative rounded-xl border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 p-3 transition-all duration-200 cursor-pointer overflow-hidden"
              @click="openUserReach(r)"
            >
              <!-- Name+badge (left) | CFS + link + remove (right) -->
              <div class="flex items-start gap-3 mb-2">
                <div class="min-w-0 flex-1">
                  <div class="flex items-center gap-1.5 min-w-0">
                    <span class="text-sm font-medium truncate">{{ r.name }}</span>
                    <span
                      v-if="r.flow_status !== 'unknown' || r.flow_band"
                      :class="['shrink-0 inline-flex items-center rounded-md px-1.5 py-0.5 text-xs font-medium', reachBadgeClass(r)]"
                    >{{ reachStatusLabel(r) }}</span>
                  </div>
                  <span v-if="r.river_name" class="text-xs text-primary-400/70 truncate block mt-0.5">{{ r.river_name }}</span>
                </div>
                <div class="shrink-0 flex items-center gap-1">
                  <span class="text-xl font-bold tabular-nums leading-none" :style="{ color: bandSolid(r.flow_band, r.flow_status) }">
                    {{ r.current_cfs != null ? r.current_cfs.toLocaleString() : '—' }}<span class="text-xs font-normal text-neutral-500 dark:text-neutral-400 ml-0.5">cfs</span>
                  </span>
                  <NuxtLink
                    :to="`/my/reaches/${r.slug}`"
                    class="rounded p-1 text-neutral-300 dark:text-neutral-600 hover:text-primary-500 dark:hover:text-primary-400 transition-colors"
                    title="Edit reach"
                    @click.stop
                  >
                    <svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor"><path d="M11 3a1 1 0 100 2h2.586l-6.293 6.293a1 1 0 101.414 1.414L15 6.414V9a1 1 0 102 0V4a1 1 0 00-1-1h-5z"/><path d="M5 5a2 2 0 00-2 2v8a2 2 0 002 2h8a2 2 0 002-2v-3a1 1 0 10-2 0v3H5V7h3a1 1 0 000-2H5z"/></svg>
                  </NuxtLink>
                  <button
                    class="rounded p-1 transition-all duration-150 text-neutral-300 dark:text-neutral-600 hover:text-red-400 dark:hover:text-red-400"
                    aria-label="Remove from dashboard"
                    @click.stop="hideReach(r.id)"
                  >
                    <svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm4 0a1 1 0 112 0v6a1 1 0 11-2 0V8z" clip-rule="evenodd"/></svg>
                  </button>
                </div>
              </div>
              <!-- Sparkline -->
              <GaugeSparkline v-if="r.gauge_id" :gauge-id="r.gauge_id" :flow-status="(r.flow_status as any)" :flow-band-label="r.flow_band ?? null" compact class="mb-1" />
              <CustomGaugeSparkline v-else-if="r.custom_gauge_slug" :gauge-slug="r.custom_gauge_slug" compact class="mb-1" />
              <!-- Last reading — full only -->
              <p v-if="viewMode === 'full' && r.last_reading_at" class="text-xs text-neutral-400 mt-0.5">{{ reachLastUpdated(r) }}</p>
              <!-- My reach origin badge -->
              <div class="flex items-center gap-1 mt-1.5 text-neutral-400/50 dark:text-neutral-500/40">
                <svg class="w-3 h-3 shrink-0" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" clip-rule="evenodd"/></svg>
                <span class="text-[10px] font-medium">My reach</span>
              </div>
            </div>
          </div>
        </section>

        <!-- Custom Gauges section — shows active-dashboard gauges on all tabs -->
        <section v-if="visibleCustomGauges.length > 0">
          <div class="flex items-center gap-2 mb-3">
            <h2 class="text-sm font-semibold text-neutral-500 uppercase tracking-wide">Custom Gauges</h2>
            <div class="flex-1 h-px bg-neutral-200 dark:bg-neutral-800" />
            <div v-if="activeCustomGauges.some(g => hiddenCustomGauges.has(g.id))" class="relative" ref="gaugeAddWrap">
              <button
                class="text-xs text-primary-500 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300 transition-colors flex items-center gap-1"
                @click="gaugeAddOpen = !gaugeAddOpen"
              >
                <svg class="w-3 h-3" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clip-rule="evenodd"/></svg>
                Add
              </button>
              <div v-if="gaugeAddOpen" class="absolute right-0 top-full mt-1 z-50 w-52 rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 shadow-lg overflow-hidden">
                <button
                  v-for="cg in activeCustomGauges.filter(g => hiddenCustomGauges.has(g.id))"
                  :key="cg.id"
                  class="w-full text-left px-3 py-2 text-sm hover:bg-primary-50 dark:hover:bg-primary-950/30 transition-colors"
                  @click="showCustomGauge(cg.id); gaugeAddOpen = false"
                >{{ cg.name }}</button>
              </div>
            </div>
            <NuxtLink to="/my/gauges" class="text-xs text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-300 transition-colors">Manage</NuxtLink>
          </div>
          <!-- List view -->
          <div v-if="viewMode === 'list'" class="space-y-1.5">
            <div
              v-for="cg in activeCustomGauges.filter(g => !hiddenCustomGauges.has(g.id))"
              :key="cg.id"
              class="flex items-center gap-2 sm:gap-3 px-3 py-2 rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 transition-all duration-200 cursor-pointer"
              @click="openStandaloneCustomGauge(cg)"
            >
              <div class="min-w-0 flex-1 flex items-center gap-1.5">
                <!-- Calculated origin indicator — list -->
                <svg class="w-3.5 h-3.5 text-neutral-400/40 dark:text-neutral-500/30 shrink-0 hidden sm:block" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="4" y="2" width="16" height="20" rx="2"/><line x1="8" y1="6" x2="16" y2="6"/><line x1="8" y1="10" x2="10" y2="10"/><line x1="14" y1="10" x2="16" y2="10"/><line x1="8" y1="14" x2="10" y2="14"/><line x1="14" y1="14" x2="16" y2="14"/><line x1="8" y1="18" x2="10" y2="18"/><line x1="14" y1="18" x2="16" y2="18"/></svg>
                <span class="text-sm font-medium truncate">{{ cg.name }}</span>
                <span v-if="cg.any_input_unhealthy" class="hidden sm:inline-flex items-center rounded-md px-1.5 py-0.5 text-xs font-medium bg-amber-100 dark:bg-amber-950/60 text-amber-700 dark:text-amber-300 shrink-0">Stale</span>
              </div>
              <div class="w-32 shrink-0 hidden sm:block">
                <CustomGaugeSparkline :gauge-slug="cg.slug" compact class="h-full w-full" />
              </div>
              <span class="text-base font-bold tabular-nums text-neutral-900 dark:text-white shrink-0">
                {{ cg.last_value_cfs != null ? cg.last_value_cfs.toLocaleString() : '—' }}<span class="text-xs font-normal text-neutral-400 ml-0.5">{{ cg.unit }}</span>
              </span>
              <button
                class="rounded p-1 text-neutral-300 dark:text-neutral-600 hover:text-red-400 dark:hover:text-red-400 transition-colors shrink-0"
                aria-label="Remove from dashboard"
                @click.stop="hideCustomGauge(cg.id)"
              >
                <svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm4 0a1 1 0 112 0v6a1 1 0 11-2 0V8z" clip-rule="evenodd"/></svg>
              </button>
            </div>
          </div>
          <!-- Card views (comfortable / full) -->
          <div v-else :class="cardGridClass">
            <div
              v-for="cg in activeCustomGauges.filter(g => !hiddenCustomGauges.has(g.id))"
              :key="cg.id"
              class="relative rounded-xl border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 p-3 transition-all duration-200 cursor-pointer overflow-hidden"
              @click="openStandaloneCustomGauge(cg)"
            >
              <!-- Name/icon (left) | value + remove (right) -->
              <div class="flex items-start gap-3">
                <div class="min-w-0 flex-1">
                  <div class="flex items-center gap-1.5">
                    <svg class="w-3.5 h-3.5 text-neutral-400 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                      <rect x="4" y="2" width="16" height="20" rx="2"/><line x1="8" y1="6" x2="16" y2="6"/><line x1="8" y1="10" x2="10" y2="10"/><line x1="14" y1="10" x2="16" y2="10"/><line x1="8" y1="14" x2="10" y2="14"/><line x1="14" y1="14" x2="16" y2="14"/><line x1="8" y1="18" x2="10" y2="18"/><line x1="14" y1="18" x2="16" y2="18"/>
                    </svg>
                    <span class="text-sm font-medium truncate block">{{ cg.name }}</span>
                  </div>
                  <p v-if="viewMode === 'full' && cg.description" class="text-xs text-neutral-400 truncate mt-0.5">{{ cg.description }}</p>
                </div>
                <div class="shrink-0 flex items-center gap-1.5">
                  <span class="text-xl font-bold tabular-nums leading-none text-neutral-900 dark:text-white">
                    {{ cg.last_value_cfs != null ? cg.last_value_cfs.toLocaleString() : '—' }}<span class="text-xs font-normal text-neutral-500 dark:text-neutral-400 ml-0.5">{{ cg.unit }}</span>
                  </span>
                  <button
                    class="rounded-lg p-1.5 transition-all duration-150 text-neutral-300 dark:text-neutral-600 hover:text-red-400 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-950/40"
                    aria-label="Remove from dashboard"
                    @click.stop="hideCustomGauge(cg.id)"
                  >
                    <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm4 0a1 1 0 112 0v6a1 1 0 11-2 0V8z" clip-rule="evenodd"/></svg>
                  </button>
                </div>
              </div>
              <!-- Sparkline -->
              <div class="mt-2">
                <CustomGaugeSparkline :gauge-slug="cg.slug" compact />
              </div>
              <!-- Calculated origin badge + stale indicator -->
              <div class="flex items-center gap-2 mt-1.5">
                <div class="flex items-center gap-1 text-neutral-400/50 dark:text-neutral-500/40">
                  <svg class="w-3 h-3 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="4" y="2" width="16" height="20" rx="2"/><line x1="8" y1="6" x2="16" y2="6"/><line x1="8" y1="10" x2="10" y2="10"/><line x1="14" y1="10" x2="16" y2="10"/><line x1="8" y1="14" x2="10" y2="14"/><line x1="14" y1="14" x2="16" y2="14"/><line x1="8" y1="18" x2="10" y2="18"/><line x1="14" y1="18" x2="16" y2="18"/></svg>
                  <span class="text-[10px] font-medium">Calculated</span>
                </div>
                <span v-if="cg.any_input_unhealthy" class="inline-flex items-center rounded-md px-1.5 py-0.5 text-xs font-medium bg-amber-100 dark:bg-amber-950/60 text-amber-700 dark:text-amber-300">Stale</span>
              </div>
            </div>
          </div>
        </section>

        <!-- Dashboard map -->
        <section>
          <div class="flex items-center gap-2 mb-3">
            <h2 class="text-sm font-semibold text-neutral-500 uppercase tracking-wide">Gauge Map</h2>
            <div class="flex-1 h-px bg-neutral-200 dark:bg-neutral-800" />
            <button
              class="text-xs text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-300 transition-colors"
              @click="mapVisible = !mapVisible"
            >{{ mapVisible ? 'Hide' : 'Show' }}</button>
          </div>
          <ClientOnly v-if="mapVisible">
            <DashboardMap
              :gauges="store.gauges"
              @remove-gauge="removeAndSync($event)"
              @open-gauge="(id) => { const g = store.gauges.find(x => x.id === id); if (g) openGauge(g, 'gauge') }"
            />
          </ClientOnly>
        </section>
      </template>
    </main>

    <GaugeSearchModal v-model:open="searchOpen" @add="handleAdd" @added-external="onAddedExternal" />
    <GaugeDetailModal v-if="detailGauge" v-model:open="detailOpen" :gauge="detailGauge" :mode="detailMode" />
    <UserReachCustomGaugeModal
      v-if="customGaugeModalProps"
      v-model:open="customGaugeModalOpen"
      v-bind="customGaugeModalProps"
    />

    <!-- New dashboard modal -->
    <Teleport to="body">
      <div v-if="newDashboardOpen" class="fixed inset-0 z-50 flex items-center justify-center p-4" @click.self="newDashboardOpen = false">
        <div class="absolute inset-0 bg-black/40" @click="newDashboardOpen = false" />
        <div class="relative w-full max-w-xs bg-white dark:bg-neutral-900 rounded-xl border border-neutral-200 dark:border-neutral-700 shadow-xl p-5 space-y-4">
          <h3 class="text-sm font-semibold">New dashboard</h3>
          <input
            v-model="newDashboardName"
            type="text"
            placeholder="Dashboard name"
            class="w-full rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-800 px-3 py-2 text-sm"
            @keydown.enter="createDashboard"
            @keydown.esc="newDashboardOpen = false"
          />
          <div class="flex gap-2 justify-end">
            <button class="px-3 py-1.5 text-sm rounded-lg border border-neutral-200 dark:border-neutral-700" @click="newDashboardOpen = false">Cancel</button>
            <button class="px-3 py-1.5 text-sm rounded-lg bg-primary-600 text-white hover:bg-primary-700" @click="createDashboard">Create</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useWatchlistStore, type WatchedGauge } from '~/stores/watchlist'
import { cleanBasinName, slugifyBasin } from '~/utils/basin'
import { flowBandLabel } from '~/utils/flowBand'

definePageMeta({ ssr: false })

const { bandBadgeClass, bandSolid } = useFlowBandPalette()

const router = useRouter()
const store = useWatchlistStore()
store.deduplicate()
const { refresh } = useWatchlistRefresh()
const { isAuthenticated, getToken } = useAuth()
const { apiBase } = useRuntimeConfig().public
const { addAndSync, removeAndSync, loadFromServer, loadForDashboard, pushLocalToServer } = useWatchlistSync()
const db = useDashboards()

// ── Dashboard tab management ──────────────────────────────────────────────────
const newDashboardOpen = ref(false)
const newDashboardName = ref('')

async function onSelectDashboard(id: string) {
  if (id === db.activeDashboardId.value) return
  db.setActive(id)
  await activateDashboard(id)
  await refresh()
}

async function onDeleteDashboard(slug: string) {
  const target = db.dashboards.value.find(d => d.slug === slug)
  if (!target) return
  const wasActive = target.id === db.activeDashboardId.value
  await db.remove(slug)
  if (wasActive && db.activeDashboard.value) {
    await activateDashboard(db.activeDashboard.value.id)
    await refresh()
  }
}

async function createDashboard() {
  const name = newDashboardName.value.trim()
  if (!name) return
  const created = await db.create(name)
  if (created) {
    newDashboardName.value = ''
    newDashboardOpen.value = false
    await onSelectDashboard(created.id)
  }
}

// ── Server sync ───────────────────────────────────────────────────────────────
let serverSynced = false
async function syncWithServer() {
  if (serverSynced) return
  serverSynced = true
  await db.load()
  const activeId = db.activeDashboard.value?.id
  if (activeId) {
    await activateDashboard(activeId)
  } else {
    await loadFromServer()
    await pushLocalToServer()
  }
}

watch(isAuthenticated, (val) => { if (val) { syncWithServer(); loadUserReaches(); loadCustomGauges() } })

let refreshTimer: ReturnType<typeof setInterval> | null = null
// ── Active hazards ────────────────────────────────────────────────────────────
const hazardSlugs = ref(new Set<string>())

async function loadActiveHazards() {
  const data = await $fetch<{ slug: string }[]>(`${apiBase}/api/v1/reaches/active-hazards`).catch(() => [])
  hazardSlugs.value = new Set((data ?? []).map(h => h.slug))
}

onMounted(() => {
  if (isAuthenticated.value) { syncWithServer(); loadUserReaches(); loadCustomGauges() }
  refresh()
  loadActiveHazards()
  refreshTimer = setInterval(refresh, 60_000)
})

// My Reaches and Custom Gauges are not dashboard-scoped in the backend, so show
// them only on the primary dashboard to avoid cluttering every custom dashboard.
const isDefaultDashboard = computed(() =>
  db.dashboards.value.length <= 1 || db.activeDashboard.value?.id === db.dashboards.value[0]?.id
)

// ── User reaches ──────────────────────────────────────────────────────────────
interface UserReachSummary {
  id: string; slug: string; name: string; river_name: string | null
  current_cfs: number | null; flow_band: string | null
  flow_status: 'runnable' | 'caution' | 'flood' | 'unknown'
  last_reading_at: string | null; gauge_id: string | null
  custom_gauge_id: string | null; custom_gauge_slug: string | null; custom_gauge_name: string | null
}

function reachBadgeClass(r: UserReachSummary): string {
  return bandBadgeClass(r.flow_band, r.flow_status)
}
function reachStatusLabel(r: UserReachSummary): string {
  return flowBandLabel(r.flow_band, r.flow_status)
}
function reachLastUpdated(r: UserReachSummary): string {
  if (!r.last_reading_at) return ''
  const ms = Date.now() - new Date(r.last_reading_at).getTime()
  const minutes = Math.floor(ms / 60_000)
  if (minutes < 1)  return 'Updated just now'
  if (minutes < 60) return `Updated ${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24)   return `Updated ${hours}h ago`
  return `Updated ${Math.floor(hours / 24)}d ago`
}

// Custom gauge modal state (used for user reaches backed by custom gauges AND standalone cards)
const customGaugeModalOpen  = ref(false)
const customGaugeModalProps = ref<{
  reachName: string; reachSlug: string
  gaugeName: string; gaugeSlug: string
  currentCfs: number | null; flowBand: string | null; flowStatus: string
} | null>(null)

function openStandaloneCustomGauge(cg: CustomGaugeSummary) {
  customGaugeModalProps.value = {
    reachName:  cg.name,
    reachSlug:  '',           // empty = standalone (no reach link in modal)
    gaugeName:  cg.name,
    gaugeSlug:  cg.slug,
    currentCfs: cg.last_value_cfs,
    flowBand:   null,
    flowStatus: 'unknown',
  }
  customGaugeModalOpen.value = true
}

// Click a user reach card → open appropriate modal or navigate.
function openUserReach(r: UserReachSummary) {
  if (r.custom_gauge_id && r.custom_gauge_slug) {
    customGaugeModalProps.value = {
      reachName:  r.name,
      reachSlug:  r.slug,
      gaugeName:  r.custom_gauge_name ?? r.name,
      gaugeSlug:  r.custom_gauge_slug,
      currentCfs: r.current_cfs,
      flowBand:   r.flow_band,
      flowStatus: r.flow_status,
    }
    customGaugeModalOpen.value = true
    return
  }
  if (r.gauge_id) {
    const gauge = store.gauges.find(g => g.id === r.gauge_id)
    if (gauge) { openGauge(gauge, 'gauge'); return }
  }
  router.push(`/my/reaches/${r.slug}`)
}
const userReaches = ref<UserReachSummary[]>([])
async function loadUserReaches() {
  const token = await getToken()
  if (!token) return
  const res = await fetch(`${apiBase}/api/v1/me/reaches`, {
    headers: { Authorization: `Bearer ${token}` },
  }).catch(() => null)
  if (!res?.ok) return
  userReaches.value = await res.json() ?? []
}

function loadSet(key: string): Set<string> {
  try { return new Set(JSON.parse(localStorage.getItem(key) ?? '[]')) } catch { return new Set() }
}
function saveSet(key: string, s: Set<string>) {
  localStorage.setItem(key, JSON.stringify([...s]))
}

// Per-dashboard hidden sets — keyed by dashboardId so trashing on secondary
// doesn't bleed into primary and vice-versa.
const hiddenReachesKey = computed(() => `h2oflow_hidden_reaches_${db.activeDashboard.value?.id ?? 'default'}`)
const hiddenGaugesKey  = computed(() => `h2oflow_hidden_custom_gauges_${db.activeDashboard.value?.id ?? 'default'}`)

const hiddenReaches      = ref<Set<string>>(new Set())
const hiddenCustomGauges = ref<Set<string>>(new Set())
const reachAddOpen       = ref(false)
const gaugeAddOpen       = ref(false)
const reachAddWrap       = ref<HTMLElement | null>(null)
const gaugeAddWrap       = ref<HTMLElement | null>(null)

onMounted(() => {
  hiddenReaches.value      = loadSet(hiddenReachesKey.value)
  hiddenCustomGauges.value = loadSet(hiddenGaugesKey.value)
})

// Reload per-dashboard hidden sets when user switches tabs.
watch(() => db.activeDashboard.value?.id, () => {
  hiddenReaches.value      = loadSet(hiddenReachesKey.value)
  hiddenCustomGauges.value = loadSet(hiddenGaugesKey.value)
})

function hideReach(id: string) {
  hiddenReaches.value = new Set([...hiddenReaches.value, id])
  saveSet(hiddenReachesKey.value, hiddenReaches.value)
}
function showReach(id: string) {
  const s = new Set(hiddenReaches.value); s.delete(id)
  hiddenReaches.value = s; saveSet(hiddenReachesKey.value, s)
}
function hideCustomGauge(id: string) {
  hiddenCustomGauges.value = new Set([...hiddenCustomGauges.value, id])
  saveSet(hiddenGaugesKey.value, hiddenCustomGauges.value)
}
function showCustomGauge(id: string) {
  const s = new Set(hiddenCustomGauges.value); s.delete(id)
  hiddenCustomGauges.value = s; saveSet(hiddenGaugesKey.value, s)
}

// Close add-dropdowns on outside click
if (import.meta.client) {
  document.addEventListener('click', (e) => {
    if (reachAddWrap.value && !reachAddWrap.value.contains(e.target as Node)) reachAddOpen.value = false
    if (gaugeAddWrap.value && !gaugeAddWrap.value.contains(e.target as Node)) gaugeAddOpen.value = false
  })
}

// ── Custom gauges ─────────────────────────────────────────────────────────────
interface CustomGaugeSummary { id: string; slug: string; name: string; description: string | null; unit: string; last_value_cfs: number | null; any_input_unhealthy: boolean }
const customGauges = ref<CustomGaugeSummary[]>([])
// IDs of custom gauges and reach slugs pinned to the currently active (non-primary) dashboard.
const dashboardCustomGaugeIds = ref<string[]>([])
const dashboardReachSlugs     = ref<string[]>([])

// Items visible on the active dashboard — always filtered by watchlist.
// Every tab (including primary) uses its own watchlist entries so that
// adding a reach to secondary doesn't bleed it onto primary.
const activeCustomGauges = computed(() =>
  customGauges.value.filter(cg => dashboardCustomGaugeIds.value.includes(cg.id))
)
const activeUserReaches = computed(() =>
  userReaches.value.filter(r => dashboardReachSlugs.value.includes(r.slug))
)

// After applying the localStorage hidden-set: what actually renders.
const visibleUserReaches  = computed(() => activeUserReaches.value.filter(r => !hiddenReaches.value.has(r.id)))
const visibleCustomGauges = computed(() => activeCustomGauges.value.filter(cg => !hiddenCustomGauges.value.has(cg.id)))

// Whether there's anything to render anywhere on the page.
const hasAnyContent = computed(() =>
  store.gauges.length > 0 || visibleUserReaches.value.length > 0 || visibleCustomGauges.value.length > 0
)

async function loadCustomGauges() {
  const token = await getToken()
  if (!token) return
  const res = await fetch(`${apiBase}/api/v1/me/custom-gauges`, {
    headers: { Authorization: `Bearer ${token}` },
  }).catch(() => null)
  if (!res?.ok) return
  const data = await res.json()
  customGauges.value = data.items ?? []
}

// Wrapper: load dashboard watchlist + update per-dashboard filter lists.
async function activateDashboard(id: string) {
  const { customGaugeIds, reachSlugs } = await loadForDashboard(id)
  dashboardCustomGaugeIds.value = customGaugeIds
  dashboardReachSlugs.value = reachSlugs
}
onUnmounted(() => { if (refreshTimer) clearInterval(refreshTimer) })

// ── Reach-primary grouping: state → basin → river → reaches ─────────────────

interface RiverGroup { name: string; reaches: WatchedGauge[] }
interface BasinGroup { name: string; reachCount: number; rivers: RiverGroup[]; standaloneGauges: WatchedGauge[] }
interface StateGroup { name: string; reachCount: number; basins: BasinGroup[] }

const byStateTree = computed<StateGroup[]>(() => {
  // state → basin → river → reaches
  type BasinEntry = { rivers: Map<string, WatchedGauge[]>; standalone: WatchedGauge[] }
  const stateMap = new Map<string, Map<string, BasinEntry>>()

  // De-duplicate: same gauge+reach should only appear once
  const seen = new Set<string>()
  for (const g of store.gauges) {
    const dedupeKey = `${g.id}::${g.contextReachSlug ?? ''}`
    if (seen.has(dedupeKey)) continue
    seen.add(dedupeKey)

    const state = g.stateAbbr ?? '—'
    const basin = cleanBasinName(g.contextReachBasinGroup)
      ?? cleanBasinName(g.watershedName)
      ?? cleanBasinName(g.basinName)
      ?? cleanBasinName(g.contextReachRiverName)
      ?? cleanBasinName(g.riverName)
      ?? 'Other'

    if (!stateMap.has(state)) stateMap.set(state, new Map())
    const basinMap = stateMap.get(state)!
    if (!basinMap.has(basin)) basinMap.set(basin, { rivers: new Map(), standalone: [] })
    const entry = basinMap.get(basin)!

    if (g.contextReachSlug) {
      const river = g.contextReachRiverName ?? g.riverName ?? 'Unknown River'
      if (!entry.rivers.has(river)) entry.rivers.set(river, [])
      entry.rivers.get(river)!.push(g)
    } else {
      entry.standalone.push(g)
    }
  }

  return [...stateMap.entries()]
    .sort(([a], [b]) => a === '—' ? 1 : b === '—' ? -1 : a.localeCompare(b))
    .map(([state, basinMap]) => {
      const basins = [...basinMap.entries()]
        .sort(([a], [b]) => a === 'Other' ? 1 : b === 'Other' ? -1 : a.localeCompare(b))
        .map(([bName, { rivers, standalone }]) => {
          const riverGroups = [...rivers.entries()]
            .sort(([a], [b]) => a.localeCompare(b))
            .map(([rName, reaches]) => ({
              name: rName,
              reaches: [...reaches].sort((a, b) => {
                // river_order (stored, admin-set) preferred; fall back to center longitude
                const ao = a.contextReachRiverOrder
                const bo = b.contextReachRiverOrder
                if (ao != null && bo != null) return ao - bo
                if (ao != null) return -1
                if (bo != null) return 1
                const al = a.contextReachCenterLng ?? a.lng
                const bl = b.contextReachCenterLng ?? b.lng
                if (al == null && bl == null) return 0
                if (al == null) return 1
                if (bl == null) return -1
                return al - bl
              }),
            }))
          const reachCount = riverGroups.reduce((s, r) => s + r.reaches.length, 0) + standalone.length
          return { name: bName, reachCount, rivers: riverGroups, standaloneGauges: standalone }
        })
      const reachCount = basins.reduce((s, b) => s + b.reachCount, 0)
      return { name: state, reachCount, basins }
    })
})

// ── Collapsible sections ────────────────────────────────────────────────────
const COLLAPSED_STATES_KEY = 'h2oflow_dashboard_collapsed_states'
const COLLAPSED_BASINS_KEY = 'h2oflow_dashboard_collapsed_basins'
const collapsedStates = ref<Set<string>>(new Set())
const collapsedBasins = ref<Set<string>>(new Set())

onMounted(() => {
  try {
    const s = localStorage.getItem(COLLAPSED_STATES_KEY)
    if (s) collapsedStates.value = new Set(JSON.parse(s))
    const b = localStorage.getItem(COLLAPSED_BASINS_KEY)
    if (b) collapsedBasins.value = new Set(JSON.parse(b))
  } catch {}
})

function toggleState(state: string) {
  const s = new Set(collapsedStates.value)
  if (s.has(state)) s.delete(state); else s.add(state)
  collapsedStates.value = s
  localStorage.setItem(COLLAPSED_STATES_KEY, JSON.stringify([...s]))
}

function basinKey(stateName: string, basinName: string) { return `${stateName}::${basinName}` }

function toggleBasin(stateName: string, basinName: string) {
  const key = basinKey(stateName, basinName)
  const s = new Set(collapsedBasins.value)
  if (s.has(key)) s.delete(key); else s.add(key)
  collapsedBasins.value = s
  localStorage.setItem(COLLAPSED_BASINS_KEY, JSON.stringify([...s]))
}

// ── Expand / collapse all ────────────────────────────────────────────────────
const allExpanded = computed(() => collapsedStates.value.size === 0 && collapsedBasins.value.size === 0)

function toggleAllSections() {
  if (allExpanded.value) {
    const states = new Set(byStateTree.value.map(s => s.name))
    collapsedStates.value = states
    localStorage.setItem(COLLAPSED_STATES_KEY, JSON.stringify([...states]))
  } else {
    collapsedStates.value = new Set()
    collapsedBasins.value = new Set()
    localStorage.setItem(COLLAPSED_STATES_KEY, '[]')
    localStorage.setItem(COLLAPSED_BASINS_KEY, '[]')
  }
}


// ── View mode ────────────────────────────────────────────────────────────────
const VIEW_MODE_KEY = 'h2oflow_dashboard_view_mode'
const VIEW_MODES = [
  { key: 'list',        label: 'List'        },
  { key: 'comfortable', label: 'Comfortable' },
  { key: 'full',        label: 'Full'        },
] as const
type ViewMode = 'list' | 'comfortable' | 'full'
const viewMode = ref<ViewMode>('comfortable')
onMounted(() => {
  const saved = localStorage.getItem(VIEW_MODE_KEY)
  if (saved === 'list' || saved === 'comfortable' || saved === 'full') {
    viewMode.value = saved
  } else if (saved === 'compact') {
    viewMode.value = 'comfortable' // migrate old compact saves
    localStorage.setItem(VIEW_MODE_KEY, 'comfortable')
  }
})
function setViewMode(m: ViewMode) {
  viewMode.value = m
  localStorage.setItem(VIEW_MODE_KEY, m)
}


// ── Group by gauge ────────────────────────────────────────────────────────────
const GROUP_KEY = 'h2oflow_dashboard_group_by_gauge'
const groupByGauge = ref(false)
onMounted(() => {
  const saved = localStorage.getItem(GROUP_KEY)
  if (saved !== null) groupByGauge.value = saved === 'true'
})
watch(groupByGauge, val => localStorage.setItem(GROUP_KEY, String(val)))

// True when at least one gauge ID appears on multiple reaches (toggle is meaningful).
const hasSharedGauges = computed(() => {
  const counts = new Map<string, number>()
  for (const g of store.gauges) counts.set(g.id, (counts.get(g.id) ?? 0) + 1)
  return [...counts.values()].some(c => c > 1)
})

interface GaugeGroup { lead: WatchedGauge; all: WatchedGauge[] }
interface SplitGroups { gaugeGroups: GaugeGroup[]; ungrouped: WatchedGauge[] }
function splitReachGroups(reaches: WatchedGauge[]): SplitGroups {
  const map = new Map<string, WatchedGauge[]>()
  for (const r of reaches) {
    if (!map.has(r.id)) map.set(r.id, [])
    map.get(r.id)!.push(r)
  }
  const gaugeGroups: GaugeGroup[] = []
  const ungrouped: WatchedGauge[] = []
  for (const all of map.values()) {
    if (all.length > 1) gaugeGroups.push({ lead: all[0]!, all })
    else ungrouped.push(all[0]!)
  }
  return { gaugeGroups, ungrouped }
}

const cardGridClass = 'grid grid-cols-1 sm:grid-cols-2 gap-2'

// Container class: multi-col grid for comfortable + full
const reachContainerClass = computed(() =>
  viewMode.value === 'comfortable' || viewMode.value === 'full'
    ? `${cardGridClass} mt-1`
    : 'space-y-2 mt-1'
)

// ── UI state ─────────────────────────────────────────────────────────────────
const searchOpen  = ref(false)
const MAP_VIS_KEY = 'h2oflow_dashboard_map_visible'
const mapVisible  = ref(true)
onMounted(() => {
  const saved = localStorage.getItem(MAP_VIS_KEY)
  if (saved !== null) mapVisible.value = saved !== 'false'
})
watch(mapVisible, val => localStorage.setItem(MAP_VIS_KEY, String(val)))

async function handleAdd(gauge: Omit<WatchedGauge, 'watchState' | 'activeSince'>, dashboardId: string | null) {
  const targetId = dashboardId ?? db.activeDashboard.value?.id ?? null
  await addAndSync(gauge, targetId)
  // Re-hydrate from server so we know the gauge persisted under the right dashboard.
  // Without this round-trip, a stale unique constraint or wrong dashboard_id silently
  // drops the gauge on the next refresh, surfacing as "appears, then disappears."
  if (targetId) {
    await activateDashboard(targetId)
    await refresh()
  }
}

interface AddedExternalPayload {
  kind: 'reach' | 'custom_gauge'
  reachId?: string
  reachSlug?: string
  customGaugeId?: string
}

async function onAddedExternal(payload?: AddedExternalPayload) {
  // Re-adding via the modal should clear the local trash-hidden flag — otherwise
  // the section gate (visibleUserReaches/visibleCustomGauges) keeps the row hidden
  // and the click looks like a no-op.
  if (payload?.kind === 'reach' && payload.reachId && hiddenReaches.value.has(payload.reachId)) {
    showReach(payload.reachId)
  }
  if (payload?.kind === 'custom_gauge' && payload.customGaugeId && hiddenCustomGauges.value.has(payload.customGaugeId)) {
    showCustomGauge(payload.customGaugeId)
  }

  await loadUserReaches()
  await loadCustomGauges()
  const id = db.activeDashboard.value?.id
  if (id) {
    await activateDashboard(id)
    await refresh()
  }
}

const detailOpen  = ref(false)
const detailGauge = ref<WatchedGauge | null>(null)
const detailMode  = ref<'gauge' | 'reach'>('gauge')
function openGauge(gauge: WatchedGauge, mode: 'gauge' | 'reach' = 'gauge') {
  detailGauge.value = gauge
  detailMode.value = mode
  detailOpen.value = true
}

const aggregateLabel  = ref('')
const aggregateGauges = ref<WatchedGauge[]>([])
function openAggregate(label: string, gauges: WatchedGauge[]) { aggregateLabel.value = label; aggregateGauges.value = gauges }
function closeAggregate() { aggregateGauges.value = [] }
</script>
