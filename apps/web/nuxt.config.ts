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
// 公开主题经 Page Registry 运行时注入；管理端预构建组件按 digest 动态加载。
const nitroOutputDir = process.env.SFORUM_NITRO_OUTPUT_DIR?.trim()
const devtoolsEnabled = process.env.NUXT_DEVTOOLS === 'true'
// Nuxt 4.4 的提取式 payload 依赖路由缓存保持开启；会话页在请求内切到
// no-store 后，`_payload.json` 会退化成 HTML，进而中断客户端导航。
// SSR 数据内联可继续保留 SPA 导航，同时让开发与生产行为一致。
const payloadExtractionEnabled = false
const publicHomepageRouteRule = {
  cache: false,
  headers: {
    'cache-control': 's-maxage=600, stale-while-revalidate'
  }
} as const
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
  modules: ['@nuxt/ui', '@nuxtjs/i18n', '@nuxtjs/seo', '@nuxt/image'],
  ssr: true,
  buildDir: process.env.NUXT_BUILD_DIR || '.nuxt',
  experimental: {
    // 开发态 HMR 后的 SWR payload 可能短暂指向旧路由并返回 HTML/404；内联数据可保持导航与 SSR 一致。
    payloadExtraction: payloadExtractionEnabled
  },
  nitro: {
    // 静态资源（带 hash 的 _nuxt 文件）压缩为 brotli + gzip。
    compressPublicAssets: { brotli: true, gzip: true },
    // 路由级渲染模式与缓存：公开内容页走 stale-while-revalidate，全部页面保持 SSR 彻底避免空壳白屏。
    // i18n strategy=no_prefix：URL 不含语言前缀，无需 /en/** 镜像规则。
    // 非默认语 cookie 由 server/middleware/locale-cache.ts 绕过共享 SWR，避免串语言。
    routeRules: {
      // 公开内容页：短到中等 swr，命中缓存的同时保持最终一致。
      // 根路由的 query 变体由 middleware 设为 no-store；基础页仍交给 CDN 做 SWR。
      '/': publicHomepageRouteRule,
      // 分类/标签详情包含分页 query；Nuxt payload 路径不携带该 query，不能共享 SWR 缓存。
      '/c/**': { cache: false },
      '/categories': { swr: 600 },
      '/tags': { swr: 600 },
      '/tags/**': { cache: false },
      '/u/**': { swr: 3600 },
      // 主题详情：匿名短 SWR（SEO/首屏）；登录或 ?edit= 由 server/middleware/topic-page-cache 禁缓存。
      // varies：扩展公开贡献设置变更会 bump site.public_surface_revision，中间件写入同名请求头。
      '/t/**': {
        swr: 60,
        cache: {
          varies: ['x-sforum-public-surface-revision']
        }
      },
      // 登录/注册/密码找回与受保护用户页保持 SSR，受保护页显式禁缓存，避免继承公开内容页 SWR。
      '/settings/**': { cache: false, robots: { index: false } },
      '/topics/new': { cache: false, robots: { index: false } },
      // 管理后台：SSR + 禁缓存 + 禁止索引。未登录由 admin 中间件服务端重定向到 /login，不再返回空壳。
      [`${adminRoutePrefix}/**`]: { cache: false, robots: { index: false } },
      // 组件预览页：SSR（生产环境直接渲染 404 错误页，不再先返回空壳）。
      '/components': { robots: { index: false } },
      // 静态图标目录：数据完全静态，长缓存。
      '/api/icon-collections/**': { cache: { maxAge: 86400 } },
      '/api/_sitemap-urls': { cache: { maxAge: 600 } }
    },
    ...(nitroOutputDir ? { output: { dir: nitroOutputDir } } : {})
  },
  ignore: nuxtGeneratedIgnores,
  css: [
    '~/assets/css/main.css',
    '~/assets/css/sforum-components.css',
    '~/assets/css/highlight-theme.css',
    '~/assets/css/sforum-theme.css',
    '~/assets/css/sforum-home.css',
    '~/assets/css/sforum-topic.css',
    '~/assets/css/sforum-taxonomy.css',
    '~/assets/css/sforum-profile.css'
  ],
  // DevTools 会扩大开发期依赖扫描和常驻内存；需要调试时用 NUXT_DEVTOOLS=true 显式开启。
  devtools: { enabled: devtoolsEnabled },
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
  image: {
    // 头像与用户上传图片优化：默认 webp 格式，惰性加载。
    format: ['webp'],
    quality: 80,
    // 头像尺寸预设，SFAvatar 按 size prop 选用对应宽度。
    screens: {
      xs: 48,
      sm: 96,
      md: 128,
      lg: 256
    }
  },
  vite: {
    resolve: {
      dedupe: ['vue', 'vue-router', 'nuxt', '@nuxt/ui']
    },
    // 预声明会被运行时 import 的依赖，让 Vite 冷启动就预打包好。
    // 否则浏览器首次打开页面时才扫描发现这些 CJS 依赖（altcha、highlight、tiptap）
    // 或 devtools 子依赖，触发 full page reload，叠加成肉眼可见的“网页卡住”。
    optimizeDeps: {
      include: [
        ...(devtoolsEnabled ? ['@vue/devtools-core', '@vue/devtools-kit'] : []),
        'altcha',
        'altcha/i18n/en',
        'altcha/i18n/zh-cn',
        'dompurify',
        'parse5',
        'highlight.js/lib/core',
        'highlight.js/lib/languages/bash',
        'highlight.js/lib/languages/css',
        'highlight.js/lib/languages/go',
        'highlight.js/lib/languages/ini',
        'highlight.js/lib/languages/javascript',
        'highlight.js/lib/languages/json',
        'highlight.js/lib/languages/markdown',
        'highlight.js/lib/languages/python',
        'highlight.js/lib/languages/ruby',
        'highlight.js/lib/languages/rust',
        'highlight.js/lib/languages/shell',
        'highlight.js/lib/languages/sql',
        'highlight.js/lib/languages/typescript',
        'highlight.js/lib/languages/xml',
        'highlight.js/lib/languages/yaml',
        '@tiptap/vue-3',
        '@tiptap/extension-character-count',
        '@tiptap/extension-image',
        '@tiptap/extension-link',
        '@tiptap/extension-placeholder',
        '@tiptap/extension-underline',
        '@tiptap/markdown',
        '@tiptap/starter-kit'
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
  // multi-sitemaps 下顶层 includeAppSources 会被忽略；子 sitemap 默认不启用
  // app sources（opt-in），我们只用自定义 API sources，故不在此设置。
  sitemap: {
    credits: false,
    sitemaps: {
      static: { sources: ['/api/_sitemap-urls'] },
      categories: { sources: ['/api/_sitemap-categories'] },
      tags: { sources: ['/api/_sitemap-tags'] },
      topics: { sources: ['/api/_sitemap-topics'], chunks: true },
      profiles: { sources: ['/api/_sitemap-profiles'] }
    },
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
    // 无感切换：URL 不带语言前缀；locale 由 cookie / setLocale 决定。
    strategy: 'no_prefix',
    langDir: 'locales',
    // i18n 10.4 的 Nitro context 实验路径在 dev 冷启动/HMR 窗口可能先进入
    // render:before、后执行 request 初始化，导致首页偶发 500。常规路由检测保留。
    experimental: {
      nitroContextDetection: false
    },
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
      // no_prefix 下没有可跳转的 locale 路径；只做 cookie/浏览器协商，不 302。
      alwaysRedirect: false,
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
