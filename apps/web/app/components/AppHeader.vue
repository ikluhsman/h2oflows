<template>
  <header class="sticky top-0 z-20 border-b border-gray-200 dark:border-gray-800 bg-white/90 dark:bg-gray-950/90 backdrop-blur-sm">
    <div class="max-w-full px-4 py-3 flex items-center gap-3">
      <NuxtLink to="/" class="flex items-center gap-2 shrink-0">
        <span class="text-xl font-bold tracking-tight">H2OFlows</span>
      </NuxtLink>

      <!-- Breadcrumb / page-level nav injected by each page -->
      <slot />

      <div class="flex-1" />

      <!-- Page-level action buttons (trip banner, Add gauge, etc.) -->
      <slot name="actions" />

      <UColorModeButton size="sm" color="neutral" variant="ghost" />
    </div>
  </header>
</template>
