import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

/**
 * Public SEO context ownership after V3 presentation migration.
 *
 * - Home SEO stays on the thin route shell (pages/index.vue).
 * - Category / tag / topic / profile SEO live on Host body islands so theme L1
 *   chrome can wrap them without duplicating meta contracts on fat routes.
 */
const root = resolve(import.meta.dir, '..')

const surfaces = {
  home: {
    path: 'apps/web/app/pages/index.vue',
    type: 'home',
    markers: ["type: 'home'", 'path:']
  },
  category: {
    path: 'apps/web/app/components/forum/SFCategoryShowPage.vue',
    type: 'category',
    markers: ["type: 'category'", 'path:', 'categoryName:']
  },
  tag: {
    path: 'apps/web/app/components/forum/SFTagShowPage.vue',
    type: 'tag',
    markers: ["type: 'tag'", 'path:', 'tagName:']
  },
  topic: {
    path: 'apps/web/app/components/forum/SFTopicShowPage.vue',
    type: 'topic',
    markers: [
      "type: 'topic'",
      'path:',
      'public:',
      'published:',
      'excerpt:',
      'datePublished:',
      'dateModified:',
      'authorName:'
    ]
  },
  profile: {
    path: 'apps/web/app/components/profile/SFProfileShowPage.vue',
    type: 'profile',
    markers: ["type: 'profile'", 'path:', 'authorName:']
  }
} as const

for (const [name, surface] of Object.entries(surfaces)) {
  const source = readFileSync(resolve(root, surface.path), 'utf8')
  if (!source.includes('useSForumSeo(computed(() => ({')) {
    throw new Error(`${name} surface must use reactive typed SEO context (${surface.path})`)
  }
  if (!source.includes(`type: '${surface.type}'`)) {
    throw new Error(`${name} surface is missing SEO page type '${surface.type}'`)
  }
  for (const marker of surface.markers) {
    if (!source.includes(marker)) {
      throw new Error(`${name} SEO context is missing ${marker}`)
    }
  }
}

// Thin route shells must still mount Page Registry outlets for the same surfaces.
const routeShells = {
  category: 'apps/web/app/pages/c/[categorySlug].vue',
  tag: 'apps/web/app/pages/tags/[tagSlug].vue',
  topic: 'apps/web/app/pages/t/[...path].vue',
  profile: 'apps/web/app/pages/u/[username].vue'
} as const

for (const [name, rel] of Object.entries(routeShells)) {
  const source = readFileSync(resolve(root, rel), 'utf8')
  if (!source.includes('SFPageOutlet')) {
    throw new Error(`${name} route shell must use SFPageOutlet`)
  }
  // SEO 已上岛；路由壳不得再重复完整 SEO 上下文（允许无 SEO 或仅 outlet）。
  if (source.includes("type: 'topic'") || source.includes("type: 'category'") || source.includes("type: 'tag'") || source.includes("type: 'profile'")) {
    throw new Error(`${name} route shell must not own typed SEO body (moved to Host island)`)
  }
}

console.log('Public SEO context validation passed')
