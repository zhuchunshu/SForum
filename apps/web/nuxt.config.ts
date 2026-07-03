export default defineNuxtConfig({
  modules: ['@nuxt/ui', '@nuxtjs/i18n', '@nuxtjs/seo'],
  ssr: true,
  css: ['~/assets/css/main.css'],
  devtools: { enabled: true },
  runtimeConfig: {
    public: {
      apiBaseUrl: process.env.NUXT_PUBLIC_API_BASE_URL || '/api/v1',
      appLocale: process.env.APP_LOCALE || 'zh-CN',
      supportedLocales: process.env.SUPPORTED_LOCALES || 'zh-CN,en-US'
    }
  },
  app: {
    head: {
      titleTemplate: '%s - SForum',
      meta: [
        { name: 'viewport', content: 'width=device-width, initial-scale=1' }
      ]
    }
  },
  i18n: {
    defaultLocale: 'zh-CN',
    strategy: 'prefix_except_default',
    langDir: 'locales',
    locales: [
      {
        code: 'zh-CN',
        language: 'zh-CN',
        name: '简体中文',
        file: 'zh-CN.json'
      },
      {
        code: 'en',
        language: 'en-US',
        name: 'English',
        file: 'en-US.json'
      }
    ],
    detectBrowserLanguage: {
      useCookie: true,
      cookieKey: 'sforum_locale',
      redirectOn: 'root',
      fallbackLocale: 'zh-CN'
    }
  },
  ogImage: {
    enabled: false
  },
  compatibilityDate: '2026-07-03'
})
