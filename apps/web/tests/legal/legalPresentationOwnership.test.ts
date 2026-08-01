import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const root = join(import.meta.dir, '../..')
const read = (rel: string) => readFileSync(join(root, rel), 'utf8')

describe('legal page presentation ownership', () => {
  const pages = [
    ['terms', 'site.terms', 'SFTermsPage', 'site.component.terms'],
    ['privacy', 'site.privacy', 'SFPrivacyPage', 'site.component.privacy'],
    ['guidelines', 'site.guidelines', 'SFGuidelinesPage', 'site.component.guidelines']
  ] as const

  for (const [slug, pageId, island, componentId] of pages) {
    test(`${slug} route is thin SEO + outlet with island fallback`, () => {
      const route = read(`app/pages/${slug}.vue`)
      expect(route).toContain('SFPageOutlet')
      expect(route).toContain(`page="${pageId}"`)
      expect(route).toContain(`<${island}`)
      expect(route).toContain('useSForumSeo')
      expect(route).not.toContain('useSeoMeta')
      expect(route).not.toContain('class="legal-page"')
      expect(route).not.toContain('legalBody(')
    })

    test(`${slug} theme shells mark presentation ownership`, () => {
      for (const theme of ['sforum-default']) {
        const tpl = read(`../../extensions/builtin/themes/${theme}/templates/${slug}.html`)
        expect(tpl).toContain('data-theme-owned="presentation"')
        expect(tpl).toContain(`data-page="${pageId}"`)
        expect(tpl).toContain('<sf-navbar')
        expect(tpl).toContain('<sf-footer')
      }
    })

    test(`${slug} ThemeTemplate maps ${componentId} to island component`, () => {
      const template = read('app/components/SFThemeTemplate.vue')
      expect(template).toContain(`'${componentId}': defineAsyncComponent(() => import('./legal/${island}.vue'))`)
    })
  }

  test('shared legal document island owns body markup', () => {
    const body = read('app/components/legal/SFLegalDocumentPage.vue')
    expect(body).toContain('class="sf-public-page legal-page"')
    expect(body).toContain('sf-prose')
    expect(body).toContain('renderLegalMarkdown')
    expect(body).toContain("kind: 'terms' | 'privacy' | 'guidelines'")
    expect(body).not.toContain('white-space: pre-wrap')
    expect(body).not.toContain('border-radius: 10px')
  })

  test('default theme legal shell fills the public canvas', () => {
    const theme = read('app/assets/css/sforum-theme.css')
    expect(theme).toContain('.sf-theme-template[data-extension-id="sforum.default-theme"] .sf-page.sf-theme--default')
    expect(theme).toContain('min-height: 100vh')
    expect(theme).toContain('flex-direction: column')
    expect(theme).toContain('> .sf-theme-chrome-body')
    expect(theme).toContain('background: var(--sf-public-bg)')
  })
})
