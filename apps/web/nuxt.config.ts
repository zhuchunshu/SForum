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
const defaultThemeLayer = '../../extensions/builtin/themes/sforum-default/layer'
const uploadedThemeLayer = process.env.SFORUM_THEME_LAYER?.trim()
// Nuxt Layers 按数组顺序应用优先级：上传主题先覆盖，默认主题必须始终作为最后的 fallback layer。
const themeLayers = uploadedThemeLayer
  ? [uploadedThemeLayer, defaultThemeLayer]
  : [defaultThemeLayer]
const nitroOutputDir = process.env.SFORUM_NITRO_OUTPUT_DIR?.trim()
const nuxtGeneratedIgnores = [
  '.nuxt/**',
  '.nuxt-build/**',
  '.nuxt-typecheck/**',
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
  '**/.nuxt-build/**',
  '**/.nuxt-typecheck/**',
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
  extends: themeLayers,
  modules: ['@nuxt/ui', '@nuxtjs/i18n', '@nuxtjs/seo'],
  ssr: true,
  buildDir: process.env.NUXT_BUILD_DIR || '.nuxt',
  nitro: nitroOutputDir ? { output: { dir: nitroOutputDir } } : {},
  ignore: nuxtGeneratedIgnores,
  css: ['~/assets/css/main.css', '~/assets/css/sforum-components.css'],
  devtools: { enabled: true },
  ui: {
    fonts: false
  },
  colorMode: {
    classSuffix: ''
  },
  icon: {
    provider: 'server',
    fallbackToApi: false,
    collections: ['lucide', 'tabler'],
    serverBundle: {
      collections: ['lucide', 'tabler']
    }
  },
  vite: {
    // 预声明会被运行时 import 的依赖，让 Vite 冷启动就预打包好。
    // 否则浏览器首次打开页面时才扫描发现这些 CJS 依赖（altcha）或 devtools 子依赖，
    // 触发 full page reload，叠加成肉眼可见的“网页卡住”。
    optimizeDeps: {
      include: [
        'altcha',
        'altcha/i18n/en',
        'altcha/i18n/zh-cn',
        '@vue/devtools-core',
        '@vue/devtools-kit'
      ]
    },
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
  robots: {
    credits: false,
    header: false,
    sitemap: '/sitemap.xml',
    disallow: [
      '/api/',
      '/login',
      '/register',
      '/components',
      adminRoutePrefix,
      `${adminRoutePrefix}/`
    ]
  },
  sitemap: {
    credits: false,
    includeAppSources: false,
    sources: ['/api/_sitemap-urls'],
    exclude: [
      '/api/**',
      '/login',
      '/register',
      '/components',
      `${adminRoutePrefix}/**`
    ]
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
    },
    'prepare:types'(options) {
      options.tsConfig.compilerOptions ||= {}
      options.tsConfig.compilerOptions.paths ||= {}
      // 主题 layer 位于 apps/web 外，TypeScript 不会从 layer 路径向下找到宿主依赖。
      options.tsConfig.compilerOptions.paths.altcha = [
        '../node_modules/altcha/dist/types/generic.d.ts'
      ]
    }
  },
  ogImage: {
    enabled: false
  },
  compatibilityDate: '2026-07-03'
})
