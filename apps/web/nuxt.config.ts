// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2026-01-01',

  modules: [
    '@nuxt/ui',
    '@pinia/nuxt',
    '@pinia-plugin-persistedstate/nuxt',
  ],

  // Register components by filename only, not directory/filename prefix.
  // This keeps templates readable: <GaugeCard> not <GaugeGaugeCard>.
  components: [
    { path: '~/components', pathPrefix: false },
  ],

  // Runtime config — API base URL comes from env in production
  runtimeConfig: {
    public: {
      apiBase: process.env.NUXT_PUBLIC_API_BASE ?? 'http://localhost:8080',
    },
  },

  // SSR on for reach pages (SEO — "arkansas river conditions" should be indexable)
  // but the dashboard itself is client-side only
  ssr: true,

  // Color mode — Nuxt UI handles dark/light switching via the class strategy
  colorMode: {
    classSuffix: '',
  },

  css: ['~/assets/css/main.css'],

  // TypeScript strict mode
  typescript: {
    strict: true,
  },

})
