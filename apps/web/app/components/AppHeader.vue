<template>
  <header class="sticky top-0 z-20 shrink-0 border-b border-neutral-200 dark:border-neutral-800 bg-white/90 dark:bg-neutral-950/90 backdrop-blur-sm">
    <div class="max-w-5xl mx-auto px-4 h-[50px] flex items-center gap-2">

      <!-- Logo -->
      <NuxtLink to="/" class="flex items-center gap-1.5 shrink-0">
        <svg class="w-5 h-5 text-blue-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M2 12c2-4 4-6 6-6s4 6 6 6 4-6 6-6" stroke-linecap="round"/>
        </svg>
        <span class="text-sm font-bold tracking-tight hidden sm:inline">H2OFlows</span>
      </NuxtLink>

      <!-- Dashboard shortcut -->
      <NuxtLink
        to="/dashboard"
        class="shrink-0 hidden sm:flex items-center gap-1 p-1.5 rounded-md transition-colors"
        :class="route.path === '/dashboard'
          ? 'text-primary-600 dark:text-primary-400 bg-primary-50 dark:bg-primary-950/50'
          : 'text-neutral-500 dark:text-neutral-400 hover:text-primary-600 dark:hover:text-primary-400 hover:bg-neutral-50 dark:hover:bg-neutral-900'"
        title="Flow Dashboard"
      >
        <svg class="w-4.5 h-4.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="4" rx="1"/><rect x="14" y="10" width="7" height="11" rx="1"/><rect x="3" y="13" width="7" height="8" rx="1"/>
        </svg>
        <span class="hidden sm:inline text-xs font-medium">Dashboard</span>
      </NuxtLink>

      <!-- Explore shortcut (replaces Map + Rivers) -->
      <NuxtLink
        to="/explore"
        class="shrink-0 hidden sm:flex items-center gap-1 p-1.5 rounded-md transition-colors"
        :class="route.path === '/explore'
          ? 'text-primary-600 dark:text-primary-400 bg-primary-50 dark:bg-primary-950/50'
          : 'text-neutral-500 dark:text-neutral-400 hover:text-primary-600 dark:hover:text-primary-400 hover:bg-neutral-50 dark:hover:bg-neutral-900'"
        title="Explore Rivers"
      >
        <svg class="w-4.5 h-4.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M4 14c3-6 6-9 8-9s5 9 8 9 5-9 8-9"/>
        </svg>
        <span class="hidden sm:inline text-xs font-medium">Explore</span>
      </NuxtLink>

      <!-- Report shortcut -->
      <NuxtLink
        v-if="isAuthenticated"
        to="/reports/new"
        class="shrink-0 hidden sm:flex items-center gap-1 p-1.5 rounded-md transition-colors"
        :class="route.path === '/reports/new'
          ? 'text-primary-600 dark:text-primary-400 bg-primary-50 dark:bg-primary-950/50'
          : 'text-neutral-500 dark:text-neutral-400 hover:text-primary-600 dark:hover:text-primary-400 hover:bg-neutral-50 dark:hover:bg-neutral-900'"
        title="File a Report"
      >
        <svg class="w-4.5 h-4.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="12" y1="18" x2="12" y2="12"/><line x1="9" y1="15" x2="15" y2="15"/>
        </svg>
        <span class="hidden sm:inline text-xs font-medium">Report</span>
      </NuxtLink>

      <!-- AI Ask button — left side (icon always visible, text desktop only) -->
      <button
        class="shrink-0 flex items-center gap-1 p-1.5 rounded-md text-neutral-500 dark:text-neutral-400 hover:text-purple-600 dark:hover:text-purple-400 hover:bg-neutral-50 dark:hover:bg-neutral-900 transition-colors"
        title="Ask AI anything"
        @click="askOpen = true"
      >
        <svg class="w-4.5 h-4.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="m12 3-1.912 5.813a2 2 0 0 1-1.275 1.275L3 12l5.813 1.912a2 2 0 0 1 1.275 1.275L12 21l1.912-5.813a2 2 0 0 1 1.275-1.275L21 12l-5.813-1.912a2 2 0 0 1-1.275-1.275L12 3Z"/>
        </svg>
        <span class="hidden sm:inline text-xs font-medium">Ask</span>
      </button>

      <!-- Breadcrumb / page-level content injected by each page -->
      <div class="flex items-center gap-2 min-w-0 flex-1">
        <slot />
      </div>

      <!-- Page-level action buttons -->
      <slot name="actions" />

      <!-- Find a gauge button — right side -->
      <button
        class="shrink-0 hidden sm:flex items-center gap-1 p-1.5 rounded-md text-neutral-500 dark:text-neutral-400 hover:text-primary-600 dark:hover:text-primary-400 hover:bg-neutral-50 dark:hover:bg-neutral-900 transition-colors"
        title="Find a gauge"
        @click="gaugeSearchOpen = true"
      >
        <svg class="w-4.5 h-4.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/>
          <path d="M11 8v6M8 11h6"/>
        </svg>
        <span class="text-xs font-medium">Gauge</span>
      </button>

      <!-- Hamburger — mobile only -->
      <button
        class="sm:hidden shrink-0 p-1.5 rounded-md text-neutral-500 hover:text-neutral-900 dark:hover:text-white hover:bg-neutral-100 dark:hover:bg-neutral-800 transition-colors"
        aria-label="Open menu"
        @click="menuOpen = !menuOpen"
      >
        <svg v-if="!menuOpen" class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/>
        </svg>
        <svg v-else class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
        </svg>
      </button>

      <!-- User avatar — far right -->
      <!-- No ClientOnly needed: server always renders unauthenticated (gray), client
           reactively updates to blue after Supabase restores session — no mismatch. -->
      <div class="relative shrink-0" data-user-menu>
          <button
            class="flex items-center justify-center w-7 h-7 rounded-full transition-colors"
            :class="isAuthenticated
              ? 'bg-primary-100 dark:bg-primary-900/50 text-primary-600 dark:text-primary-400 hover:bg-primary-200 dark:hover:bg-primary-900'
              : 'bg-neutral-100 dark:bg-neutral-800 text-neutral-400 dark:text-neutral-500 hover:bg-neutral-200 dark:hover:bg-neutral-700'"
            :title="isAuthenticated ? `Signed in as ${user?.email ?? user?.user_metadata?.user_name ?? 'you'}` : 'Sign in'"
            @click="userMenuOpen = !userMenuOpen"
          >
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="12" cy="8" r="4"/><path d="M20 21a8 8 0 0 0-16 0"/>
            </svg>
          </button>
          <div
            v-if="userMenuOpen"
            class="absolute right-0 top-full mt-1 w-52 bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-700 rounded-lg shadow-lg py-1 z-40"
          >
            <template v-if="isAuthenticated">
              <p class="px-3 py-1.5 text-xs text-neutral-400 truncate">{{ user?.email ?? user?.user_metadata?.user_name }}</p>
              <div class="border-t border-neutral-100 dark:border-neutral-800" />
            </template>
            <!-- Appearance: collapsed by default; expand to show palette + mode -->
            <button
              class="w-full flex items-center justify-between px-3 py-1.5 text-sm text-neutral-600 dark:text-neutral-300 hover:bg-neutral-50 dark:hover:bg-neutral-800 transition-colors"
              @click="appearanceOpen = !appearanceOpen"
            >
              <span class="flex items-center gap-2">
                <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <circle cx="13.5" cy="6.5" r=".5"/><circle cx="17.5" cy="10.5" r=".5"/><circle cx="8.5" cy="7.5" r=".5"/><circle cx="6.5" cy="12.5" r=".5"/>
                  <path d="M12 2C6.5 2 2 6.5 2 12s4.5 10 10 10c.926 0 1.648-.746 1.648-1.688 0-.437-.18-.835-.437-1.125-.29-.289-.438-.652-.438-1.125a1.64 1.64 0 0 1 1.668-1.668h1.996c3.051 0 5.555-2.503 5.555-5.554C21.965 6.012 17.461 2 12 2z"/>
                </svg>
                Appearance
              </span>
              <span class="flex items-center gap-1.5">
                <!-- Active palette swatch preview -->
                <span v-if="activePalette" class="w-4 h-4 rounded-full overflow-hidden border border-neutral-200 dark:border-neutral-700 inline-flex">
                  <span class="w-1/2 h-full" :style="{ background: activePalette.neutralSwatch }" />
                  <span class="w-1/2 h-full" :style="{ background: activePalette.primarySwatch }" />
                </span>
                <svg
                  class="w-3 h-3 text-neutral-400 transition-transform"
                  :class="{ 'rotate-180': appearanceOpen }"
                  viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
                ><polyline points="6 9 12 15 18 9"/></svg>
              </span>
            </button>
            <div v-if="appearanceOpen" class="px-3 pb-2 space-y-1.5">
              <!-- 9×2 palette grid: row = neutral (slate/stone), cols wrap -->
              <div class="space-y-1">
                <div v-for="neutralLabel in ['Slate', 'Stone']" :key="neutralLabel" class="flex items-start gap-1">
                  <span class="text-[9px] text-neutral-400 w-8 shrink-0 pt-1">{{ neutralLabel }}</span>
                  <div class="flex flex-wrap items-center gap-1">
                    <button
                      v-for="p in PALETTES.filter(p => p.neutral === neutralLabel.toLowerCase())"
                      :key="p.id"
                      :title="p.label"
                      class="w-5 h-5 rounded-full overflow-hidden border-2 transition-all shrink-0"
                      :class="themeStore.paletteId === p.id
                        ? 'border-neutral-900 dark:border-white scale-110'
                        : 'border-transparent hover:scale-105'"
                      @click="applyPalette(p.id)"
                    >
                      <div class="flex h-full w-full">
                        <div class="w-1/2 h-full" :style="{ background: p.neutralSwatch }" />
                        <div class="w-1/2 h-full" :style="{ background: p.primarySwatch }" />
                      </div>
                    </button>
                  </div>
                </div>
              </div>
              <!-- Light / Dark / System -->
              <div class="flex items-center gap-1 pt-1">
                <span class="text-[9px] text-neutral-400 w-8 shrink-0">Mode</span>
                <div class="flex items-center gap-1">
                  <button
                    v-for="mode in colorModes"
                    :key="mode.value"
                    :title="mode.label"
                    class="flex items-center justify-center w-7 h-7 rounded-md transition-colors"
                    :class="colorMode.preference === mode.value
                      ? 'bg-neutral-200 dark:bg-neutral-700 text-neutral-900 dark:text-white'
                      : 'text-neutral-500 dark:text-neutral-400 hover:bg-neutral-100 dark:hover:bg-neutral-800'"
                    @click="colorMode.preference = mode.value"
                  >
                    <svg v-if="mode.value === 'light'" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="5"/><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/></svg>
                    <svg v-else-if="mode.value === 'dark'" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>
                    <svg v-else class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="3" width="20" height="14" rx="2"/><path d="M8 21h8M12 17v4"/></svg>
                  </button>
                </div>
              </div>
            </div>
            <div class="border-t border-neutral-100 dark:border-neutral-800" />
            <NuxtLink
              v-if="isAuthenticated"
              to="/my/reports"
              class="w-full text-left px-3 py-1.5 text-sm text-neutral-600 dark:text-neutral-300 hover:bg-neutral-50 dark:hover:bg-neutral-800 transition-colors flex items-center gap-2"
              @click="userMenuOpen = false"
            >
              <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/>
              </svg>
              My Reports
            </NuxtLink>
            <NuxtLink
              v-if="isAuthenticated"
              to="/my/reaches"
              class="w-full text-left px-3 py-1.5 text-sm text-neutral-600 dark:text-neutral-300 hover:bg-neutral-50 dark:hover:bg-neutral-800 transition-colors flex items-center gap-2"
              @click="userMenuOpen = false"
            >
              <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M3 6h18M3 12h18M3 18h18" stroke-linecap="round"/>
              </svg>
              My Reaches
            </NuxtLink>
            <NuxtLink
              v-if="isAuthenticated"
              to="/my/gauges"
              class="w-full text-left px-3 py-1.5 text-sm text-neutral-600 dark:text-neutral-300 hover:bg-neutral-50 dark:hover:bg-neutral-800 transition-colors flex items-center gap-2"
              @click="userMenuOpen = false"
            >
              <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M9 7h6m0 10v-3m-3 3v-6m-3 6v-1"/>
              </svg>
              My Gauges
            </NuxtLink>
            <NuxtLink
              v-if="isDataAdmin"
              to="/admin"
              class="w-full text-left px-3 py-1.5 text-sm text-neutral-600 dark:text-neutral-300 hover:bg-neutral-50 dark:hover:bg-neutral-800 transition-colors flex items-center gap-2"
              @click="userMenuOpen = false"
            >
              <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
              </svg>
              Admin
            </NuxtLink>
            <div class="border-t border-neutral-100 dark:border-neutral-800" />
            <template v-if="isAuthenticated">
              <button
                class="w-full text-left px-3 py-1.5 text-sm text-neutral-600 dark:text-neutral-300 hover:bg-neutral-50 dark:hover:bg-neutral-800 transition-colors"
                @click="userMenuOpen = false; handleSignOut()"
              >Sign out</button>
            </template>
            <template v-else>
              <NuxtLink
                to="/login"
                class="block px-3 py-1.5 text-sm font-medium text-primary-600 dark:text-primary-400 hover:bg-neutral-50 dark:hover:bg-neutral-800 transition-colors"
                @click="userMenuOpen = false"
              >Sign in</NuxtLink>
            </template>
          </div>
        </div>
    </div>

    <!-- Mobile menu dropdown -->
    <div v-if="menuOpen" class="sm:hidden border-t border-neutral-100 dark:border-neutral-800 bg-white dark:bg-neutral-950 px-4 py-3 flex flex-col gap-1">
      <!-- Dashboard + Map — mobile -->
      <NuxtLink
        to="/dashboard"
        class="text-left px-3 py-2 rounded-md text-sm flex items-center gap-2 transition-colors"
        :class="route.path === '/dashboard'
          ? 'text-primary-600 dark:text-primary-400 bg-primary-50 dark:bg-primary-950/50'
          : 'text-neutral-600 dark:text-neutral-300 hover:bg-neutral-50 dark:hover:bg-neutral-900'"
        @click="menuOpen = false"
      >
        <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="4" rx="1"/><rect x="14" y="10" width="7" height="11" rx="1"/><rect x="3" y="13" width="7" height="8" rx="1"/>
        </svg>
        Dashboard
      </NuxtLink>
      <NuxtLink
        to="/explore"
        class="text-left px-3 py-2 rounded-md text-sm flex items-center gap-2 transition-colors"
        :class="route.path === '/explore'
          ? 'text-primary-600 dark:text-primary-400 bg-primary-50 dark:bg-primary-950/50'
          : 'text-neutral-600 dark:text-neutral-300 hover:bg-neutral-50 dark:hover:bg-neutral-900'"
        @click="menuOpen = false"
      >
        <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M4 14c3-6 6-9 8-9s5 9 8 9 5-9 8-9"/>
        </svg>
        Explore
      </NuxtLink>
      <!-- Report — mobile (auth only) -->
      <NuxtLink
        v-if="isAuthenticated"
        to="/reports/new"
        class="text-left px-3 py-2 rounded-md text-sm flex items-center gap-2 transition-colors"
        :class="route.path === '/reports/new'
          ? 'text-primary-600 dark:text-primary-400 bg-primary-50 dark:bg-primary-950/50'
          : 'text-neutral-600 dark:text-neutral-300 hover:bg-neutral-50 dark:hover:bg-neutral-900'"
        @click="menuOpen = false"
      >
        <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="12" y1="18" x2="12" y2="12"/><line x1="9" y1="15" x2="15" y2="15"/>
        </svg>
        Report
      </NuxtLink>

      <div class="border-t border-neutral-100 dark:border-neutral-800 mt-1 pt-2 flex flex-col gap-1">
        <!-- Ask — mobile -->
        <button
          class="w-full text-left px-3 py-2 rounded-md text-sm text-neutral-600 dark:text-neutral-300 hover:bg-neutral-50 dark:hover:bg-neutral-900 transition-colors flex items-center gap-2"
          @click="menuOpen = false; askOpen = true"
        >
          <svg class="w-4 h-4 text-purple-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="m12 3-1.912 5.813a2 2 0 0 1-1.275 1.275L3 12l5.813 1.912a2 2 0 0 1 1.275 1.275L12 21l1.912-5.813a2 2 0 0 1 1.275-1.275L21 12l-5.813-1.912a2 2 0 0 1-1.275-1.275L12 3Z"/>
          </svg>
          Ask AI
        </button>
        <!-- Find a gauge — mobile -->
        <button
          class="w-full text-left px-3 py-2 rounded-md text-sm text-neutral-600 dark:text-neutral-300 hover:bg-neutral-50 dark:hover:bg-neutral-900 transition-colors flex items-center gap-2"
          @click="menuOpen = false; gaugeSearchOpen = true"
        >
          <svg class="w-4 h-4 text-neutral-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/><path d="M11 8v6M8 11h6"/>
          </svg>
          Find a gauge
        </button>
      </div>
      <div class="border-t border-neutral-100 dark:border-neutral-800 mt-1 pt-2">
        <button
          v-if="isAuthenticated"
          class="w-full text-left px-3 py-2 rounded-md text-sm text-neutral-500 hover:text-neutral-900 dark:hover:text-white hover:bg-neutral-50 dark:hover:bg-neutral-900 transition-colors"
          @click="handleSignOut"
        >Sign out</button>
        <NuxtLink
          v-else
          to="/login"
          class="block px-3 py-2 rounded-md text-sm font-medium text-primary-600 dark:text-primary-400 hover:bg-primary-50 dark:hover:bg-primary-950 transition-colors"
          @click="menuOpen = false"
        >Sign in</NuxtLink>
      </div>
    </div>
  </header>

  <!-- Global gauge search modal -->
  <GaugeSearchModal v-model:open="gaugeSearchOpen" @add="handleGaugeAdd" @added-external="onAddedExternal" />

  <!-- Global Ask modal (Teleport so it's above everything) -->
  <Teleport to="body">
    <Transition
      enter-active-class="transition duration-150 ease-out"
      enter-from-class="opacity-0 scale-95"
      enter-to-class="opacity-100 scale-100"
      leave-active-class="transition duration-100 ease-in"
      leave-from-class="opacity-100 scale-100"
      leave-to-class="opacity-0 scale-95"
    >
      <div
        v-if="askOpen"
        class="fixed inset-0 z-50 flex items-start justify-center pt-[10vh] px-4 bg-black/20 backdrop-blur-sm"
        @click.self="closeAsk"
      >
        <div class="w-full max-w-xl bg-white dark:bg-neutral-900 rounded-2xl shadow-2xl border border-neutral-200 dark:border-neutral-800 overflow-hidden">
          <form class="flex items-center gap-2 px-4 py-3 border-b border-neutral-100 dark:border-neutral-800" @submit.prevent="askQuestion">
            <svg class="w-4 h-4 text-purple-400 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="m12 3-1.912 5.813a2 2 0 0 1-1.275 1.275L3 12l5.813 1.912a2 2 0 0 1 1.275 1.275L12 21l1.912-5.813a2 2 0 0 1 1.275-1.275L21 12l-5.813-1.912a2 2 0 0 1-1.275-1.275L12 3Z"/>
            </svg>
            <input
              ref="askInputRef"
              v-model="askQuery"
              type="text"
              placeholder='Ask anything — e.g. "Browns Canyon at 800 cfs?"'
              class="flex-1 bg-transparent text-sm text-neutral-900 dark:text-white placeholder-neutral-400 focus:outline-none"
              :disabled="asking"
            />
            <button
              v-if="askQuery"
              type="button"
              class="text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-300"
              @click="askQuery = ''; askResult = null"
            >
              <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6 6 18M6 6l12 12"/></svg>
            </button>
            <button
              type="submit"
              :disabled="asking || !askQuery.trim()"
              class="shrink-0 px-3 py-1.5 rounded-lg bg-primary-600 hover:bg-primary-700 disabled:opacity-40 text-white text-xs font-semibold transition-colors"
            >
              <span v-if="asking" class="flex items-center gap-1">
                <span class="w-3 h-3 border-2 border-white/30 border-t-white rounded-full animate-spin"/>
              </span>
              <span v-else>Ask</span>
            </button>
          </form>

          <div v-if="askResult" class="px-4 py-4 space-y-3 max-h-96 overflow-y-auto">
            <div
              v-for="result in (askResult.results ?? [])"
              :key="result.reach_slug"
              class="rounded-lg border border-neutral-100 dark:border-neutral-800 p-3 space-y-1"
            >
              <div class="flex items-center justify-between gap-2">
                <span class="text-xs font-semibold uppercase tracking-wide text-primary-500">{{ result.reach_name }}</span>
                <button
                  type="button"
                  class="text-xs text-primary-600 dark:text-primary-400 hover:underline font-medium shrink-0"
                  @click="goToReachOnMap(result.reach_slug)"
                >Show on map →</button>
              </div>
              <div class="ask-answer text-sm text-neutral-700 dark:text-neutral-300 leading-relaxed" v-html="md.render(result.answer)" />
            </div>
            <div v-if="!askResult.results?.length && askResult.answer" class="ask-answer text-sm text-neutral-500 leading-relaxed" v-html="md.render(askResult.answer)" />
          </div>
          <p v-else-if="!asking && !askResult" class="px-4 py-3 text-xs text-neutral-400">
            Try: "What's Foxton like at 300 cfs?" or "Best beginner runs near Denver"
          </p>
          <p v-if="askError" class="px-4 py-3 text-sm text-red-400">{{ askError }}</p>

          <div class="px-4 py-2.5 border-t border-neutral-100 dark:border-neutral-800 flex justify-end">
            <button class="text-xs text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-300" @click="closeAsk">Close</button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, nextTick, watch, onMounted, onUnmounted } from 'vue'
import MarkdownIt from 'markdown-it'
import type { WatchedGauge } from '~/stores/watchlist'
import { PALETTES } from '../../app.config'
import type { PaletteId } from '../../app.config'
import { useThemeStore } from '~/stores/theme'

const { user, isAuthenticated, isDataAdmin, signOut } = useAuth()
const router = useRouter()
const route = useRoute()
const colorMode = useColorMode()
const appConfig = useAppConfig()
const themeStore = useThemeStore()
const menuOpen = ref(false)
const userMenuOpen = ref(false)
const appearanceOpen = ref(false)

const colorModes = [
  { value: 'light', label: 'Light' },
  { value: 'dark', label: 'Dark' },
  { value: 'system', label: 'System' },
]

const activePalette = computed(() => PALETTES.find(p => p.id === themeStore.paletteId) ?? null)

function applyPalette(id: PaletteId) {
  const palette = PALETTES.find(p => p.id === id)
  if (!palette) return
  themeStore.paletteId = id
  appConfig.ui.colors.primary = palette.primary
  appConfig.ui.colors.neutral = palette.neutral
}

const { apiBase } = useRuntimeConfig().public

// Close menus on route change
watch(() => route.path, () => { menuOpen.value = false; userMenuOpen.value = false })

function onDocClick(e: MouseEvent) {
  if (userMenuOpen.value && !(e.target as HTMLElement).closest('[data-user-menu]')) {
    userMenuOpen.value = false
  }
}
onMounted(() => document.addEventListener('click', onDocClick))
onUnmounted(() => document.removeEventListener('click', onDocClick))

async function handleSignOut() {
  menuOpen.value = false
  await signOut()
  router.push('/')
}

// ── Find a gauge ─────────────────────────────────────────────────────────────
const { addAndSync, loadForDashboard } = useWatchlistSync()
const dashboardsForAdd = useDashboards()
const gaugeSearchOpen = ref(false)
function handleGaugeAdd(gauge: Omit<WatchedGauge, 'watchState' | 'activeSince'>, dashboardId: string | null) {
  addAndSync(gauge, dashboardId)
}
async function onAddedExternal() {
  // User reach or custom gauge added directly to watchlist via API — reload
  // the active dashboard so the new item shows up immediately.
  const id = dashboardsForAdd.activeDashboard.value?.id
  if (id) await loadForDashboard(id)
}

// ── Global Ask ────────────────────────────────────────────────────────────────
const md          = new MarkdownIt({ html: false, linkify: false, breaks: true })
const askOpen     = ref(false)
const askInputRef = ref<HTMLInputElement>()
const askQuery    = ref('')
const asking      = ref(false)
const askError    = ref('')
const askResult   = ref<{ results?: { answer: string; reach_slug: string; reach_name: string }[]; answer?: string } | null>(null)

watch(askOpen, async (open) => {
  if (open) {
    askQuery.value  = ''
    askResult.value = null
    askError.value  = ''
    await nextTick()
    askInputRef.value?.focus()
  }
})

function closeAsk() { askOpen.value = false }

function goToReachOnMap(slug: string) {
  closeAsk()
  router.push({ path: '/explore', query: { focus: slug } })
}

async function askQuestion() {
  const q = askQuery.value.trim()
  if (!q) return
  asking.value    = true
  askError.value  = ''
  askResult.value = null
  try {
    const res = await fetch(`${apiBase}/api/v1/ask`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ question: q }),
    })
    if (!res.ok) throw new Error(`${res.status}`)
    askResult.value = await res.json()
  } catch {
    askError.value = 'Something went wrong. Try again.'
  } finally {
    asking.value = false
  }
}
</script>

