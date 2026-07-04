import type { NuxtPage } from 'nuxt/schema'
import {
  LEGACY_ADMIN_ROUTE_PREFIX,
  normalizeAdminRoutePrefix
} from './app/utils/adminRoutePrefix'

const appName = process.env.APP_NAME || 'SForum'
const appUrl =
  process.env.NUXT_PUBLIC_I18N_BASE_URL ||
  process.env.APP_URL ||
  'http://127.0.0.1:3000'
const adminRoutePrefix = normalizeAdminRoutePrefix(
  process.env.NUXT_PUBLIC_ADMIN_ROUTE_PREFIX ||
  process.env.ADMIN_ROUTE_PREFIX
)
const nuxtGeneratedIgnores = [
  '.nuxt/**',
  '.output/**',
  '.nitro/**',
  '.vite/**',
  '.cache/**',
  'coverage/**',
  'playwright-report/**',
  'test-results/**'
]

const generatedOutputWatchIgnores = [
  '**/.nuxt/**',
  '**/.output/**',
  '**/.nitro/**',
  '**/.vite/**',
  '**/.cache/**',
  '**/dist/**',
  '**/coverage/**',
  '**/playwright-report/**',
  '**/test-results/**'
]

function rewriteAdminPageRoutes(pages: NuxtPage[]) {
  for (const page of pages) {
    if (
      page.path === LEGACY_ADMIN_ROUTE_PREFIX ||
      page.path.startsWith(`${LEGACY_ADMIN_ROUTE_PREFIX}/`)
    ) {
      page.path = page.path.replace(
        LEGACY_ADMIN_ROUTE_PREFIX,
        adminRoutePrefix
      )
    }

    if (page.children?.length) {
      rewriteAdminPageRoutes(page.children)
    }
  }
}

export default defineNuxtConfig({
  modules: ['@nuxt/ui', '@nuxtjs/i18n', '@nuxtjs/seo'],
  ssr: true,
  buildDir: process.env.NUXT_BUILD_DIR || '.nuxt',
  ignore: nuxtGeneratedIgnores,
  css: ['~/assets/css/main.css', '~/assets/css/sforum-components.css'],
  devtools: { enabled: true },
  ui: {
    fonts: false
  },
  vite: {
    server: {
      watch: {
        ignored: generatedOutputWatchIgnores
      }
    }
  },
  vue: {
    compilerOptions: {
      isCustomElement: (tag) => tag === 'altcha-widget'
    }
  },
  schemaOrg: {
    enabled: false
  },
  runtimeConfig: {
    public: {
      apiBaseUrl: process.env.NUXT_PUBLIC_API_BASE_URL || '/api/v1',
      adminRoutePrefix,
      appLocale: process.env.APP_LOCALE || 'zh-CN',
      supportedLocales: process.env.SUPPORTED_LOCALES || 'zh-CN,en-US',
      i18n: {
        baseUrl: appUrl
      }
    }
  },
  site: {
    name: appName,
    url: appUrl
  },
  app: {
    head: {
      titleTemplate: `%s - ${appName}`,
      meta: [
        { name: 'viewport', content: 'width=device-width, initial-scale=1' }
      ]
    }
  },
  i18n: {
    baseUrl: appUrl,
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
  hooks: {
    'pages:extend'(pages) {
      rewriteAdminPageRoutes(pages)
    }
  },
  ogImage: {
    enabled: false
  },
  compatibilityDate: '2026-07-03'
})
