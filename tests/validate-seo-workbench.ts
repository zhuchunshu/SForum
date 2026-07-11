import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const root = resolve(import.meta.dir, '..')
const read = (path: string) => readFileSync(resolve(root, path), 'utf8')

const page = read('apps/web/app/pages/admin/seo.vue')
const imagePicker = read('apps/web/app/components/admin/seo/SFSEOImagePicker.vue')
const appearance = read('apps/web/app/components/admin/seo/SFSEOSearchAppearance.vue')
const contentTypes = read('apps/web/app/components/admin/seo/SFSEOContentTypes.vue')

for (const marker of ['SFSEOSearchAppearance', 'SFSEOContentTypes', 'seo.site.inherit_site_name', 'seo.home.title', 'seo.home.description', 'seo.home.keywords']) {
  if (!page.includes(marker)) throw new Error(`SEO workbench is missing ${marker}`)
}

for (const marker of ['dragover', '@drop', 'type="file"', 'type="url"', 'uploading', '<img', 'removeImage', '/admin/seo/assets']) {
  if (!imagePicker.includes(marker)) throw new Error(`SEO image picker is missing ${marker}`)
}

for (const marker of ['SFSEOImagePicker', 'homeTitle', 'homeDescription', 'homeKeywords', 'inheritSiteName']) {
  if (!appearance.includes(marker)) throw new Error(`Search appearance is missing ${marker}`)
}

for (const type of ['category', 'tag', 'topic', 'profile', 'static']) {
  if (!contentTypes.includes(`'${type}'`)) throw new Error(`Content type editor is missing ${type}`)
}

for (const source of [page, imagePicker, appearance, contentTypes]) {
  if (/<svg\b/i.test(source)) throw new Error('SEO workbench must use the approved icon library')
}

console.log('SEO workbench validation passed')
