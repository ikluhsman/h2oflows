// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2026-01-01',

  modules: [
    '@nuxt/ui',
    '@pinia/nuxt',
    '@pinia-plugin-persistedstate/nuxt',
    '@nuxtjs/supabase',
  ],

  supabase: {
    redirect: false,
    redirectOptions: {
      login:    '/login',
      callback: '/confirm',
      exclude:  ['/', '/map', '/reaches/*', '/dashboard'],
    },
    // Use implicit flow — tokens come back in the URL hash, no PKCE code
    // verifier needed. Required for static hosting (Netlify) where sessionStorage
    // doesn't survive the OAuth redirect.
    clientOptions: {
      auth: {
        flowType: 'implicit',
      },
    },
  },

  // Exclude client-only / dynamic routes from static prerender
  routeRules: {
    '/confirm':   { prerender: false },
    '/dashboard': { prerender: false },
    '/trips':     { prerender: false },
  },

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
