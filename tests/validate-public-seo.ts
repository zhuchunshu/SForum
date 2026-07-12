import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const root = resolve(import.meta.dir, '..')
const pages = {
  home: 'apps/web/app/pages/index.vue',
  category: 'apps/web/app/pages/c/[categorySlug].vue',
  tag: 'apps/web/app/pages/tags/[tagSlug].vue',
  topic: 'apps/web/app/pages/t/[...path].vue',
  profile: 'apps/web/app/pages/u/[username].vue'
} as const

for (const [type, path] of Object.entries(pages)) {
  const source = readFileSync(resolve(root, path), 'utf8')
  if (!source.includes('useSForumSeo(computed(() => ({')) throw new Error(`${type} page must use reactive typed SEO context`)
  if (!source.includes(`type: '${type}'`)) throw new Error(`${type} page is missing its SEO page type`)
  if (!source.includes('path:')) throw new Error(`${type} page is missing canonical path context`)
}

const topic = readFileSync(resolve(root, pages.topic), 'utf8')
for (const marker of ['public:', 'published:', 'excerpt:', 'datePublished:', 'dateModified:', 'authorName:']) {
  if (!topic.includes(marker)) throw new Error(`topic SEO context is missing ${marker}`)
}

const category = readFileSync(resolve(root, pages.category), 'utf8')
if (!category.includes('categoryName:')) throw new Error('category SEO variables are missing categoryName')

const tag = readFileSync(resolve(root, pages.tag), 'utf8')
if (!tag.includes('tagName:')) throw new Error('tag SEO variables are missing tagName')

const profile = readFileSync(resolve(root, pages.profile), 'utf8')
if (!profile.includes('authorName:')) throw new Error('profile SEO variables are missing authorName')

console.log('Public SEO context validation passed')
