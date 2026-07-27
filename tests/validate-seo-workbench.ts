import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const root = resolve(import.meta.dir, '..')
const read = (path: string) => readFileSync(resolve(root, path), 'utf8')
const tabRoot = 'apps/web/app/components/admin/settings/seo/tabs'
const tabFiles = [
  'SFAdminSeoOverviewTab.vue',
  'SFAdminSeoSearchTab.vue',
  'SFAdminSeoContentTab.vue',
  'SFAdminSeoMetaTab.vue',
  'SFAdminSeoRobotsTab.vue',
  'SFAdminSeoSitemapTab.vue',
  'SFAdminSeoSchemaTab.vue',
  'SFAdminSeoVerificationTab.vue',
  'SFAdminSeoPermalinksTab.vue'
]
const sources = Object.fromEntries(tabFiles.map(file => [file, read(`${tabRoot}/${file}`)]))
const imagePicker = read(`${tabRoot}/SFSEOImagePicker.vue`)
const appearance = read(`${tabRoot}/SFSEOSearchAppearance.vue`)
const contentTypes = read(`${tabRoot}/SFSEOContentTypes.vue`)

for (const [file, source] of Object.entries(sources)) {
  if (/<svg\b/i.test(source)) throw new Error(`${file} must use the approved icon library`)
}
for (const marker of ['seo.site.inherit_site_name', 'seo.home.title', 'seo.home.description', 'seo.home.keywords']) {
  if (!sources['SFAdminSeoSearchTab.vue'].includes(marker)) throw new Error(`Search tab is missing owned payload field ${marker}`)
}
for (const marker of ['seo.content_type.', 'include_in_sitemap', 'schema_type']) {
  if (!sources['SFAdminSeoContentTab.vue'].includes(marker)) throw new Error(`Content tab is missing owned payload behavior ${marker}`)
}
for (const source of [appearance, contentTypes]) {
  if (!source.includes("import SFSEOImagePicker from './SFSEOImagePicker.vue'")) throw new Error('SEO editors must explicitly import the colocated image picker')
}
for (const marker of ['dragover', '@drop', 'type="file"', 'type="url"', 'uploading', '<img', 'removeImage', '/admin/seo/assets']) {
  if (!imagePicker.includes(marker)) throw new Error(`SEO image picker is missing ${marker}`)
}
for (const type of ['category', 'tag', 'topic', 'profile', 'static']) {
  if (!contentTypes.includes(`'${type}'`)) throw new Error(`Content type editor is missing ${type}`)
}

console.log('SEO workbench validation passed')
