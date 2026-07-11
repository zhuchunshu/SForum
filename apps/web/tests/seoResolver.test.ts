import { describe, expect, test } from 'bun:test'
import { resolveSEO, type SEOResolverSettings } from '../app/utils/seoResolver'

const baseSettings = (overrides: Partial<SEOResolverSettings> = {}): SEOResolverSettings => ({
  siteName: 'SForum',
  siteUrl: 'https://example.com',
  seoSiteName: 'SForum',
  homeTitle: 'SForum',
  homeDescription: 'Community discussions.',
  homeKeywords: '',
  homeOGTitle: '',
  homeOGDescription: '',
  homeOGImageUrl: '',
  pageTitleTemplate: '{pageTitle} | {seoSiteName}',
  pageDefaultDescription: 'Community discussions.',
  allowIndexing: true,
  policies: {
    home: { titleTemplate: '{seoSiteName}', descriptionSources: ['content', 'site_default'], defaultImageUrl: '', indexMode: 'index', includeInSitemap: true, schemaType: 'WebSite' },
    category: { titleTemplate: '{categoryName} | {seoSiteName}', descriptionSources: ['content', 'site_default'], defaultImageUrl: '', indexMode: 'index', includeInSitemap: true, schemaType: 'CollectionPage' },
    tag: { titleTemplate: '{tagName} | {seoSiteName}', descriptionSources: ['content', 'site_default'], defaultImageUrl: '', indexMode: 'index', includeInSitemap: true, schemaType: 'CollectionPage' },
    topic: { titleTemplate: '{topicTitle} | {seoSiteName}', descriptionSources: ['summary', 'excerpt', 'site_default'], defaultImageUrl: '', indexMode: 'index', includeInSitemap: true, schemaType: 'DiscussionForumPosting' },
    profile: { titleTemplate: '{authorName} | {seoSiteName}', descriptionSources: ['content', 'site_default'], defaultImageUrl: '', indexMode: 'noindex', includeInSitemap: false, schemaType: 'ProfilePage' },
    static: { titleTemplate: '{pageTitle} | {seoSiteName}', descriptionSources: ['content', 'site_default'], defaultImageUrl: '', indexMode: 'index', includeInSitemap: true, schemaType: 'WebPage' }
  },
  ...overrides
})

describe('resolveSEO', () => {
  test('homepage SEO title is independent from product site name', () => {
    const result = resolveSEO(baseSettings({
      siteName: 'SForum',
      seoSiteName: 'SForum Developers',
      homeTitle: 'Developer Q&A and Open Source Forum'
    }), { type: 'home', path: '/' })

    expect(result.title).toBe('Developer Q&A and Open Source Forum')
    expect(result.siteName).toBe('SForum Developers')
  })

  test('configured homepage description wins over page fallback copy', () => {
    const result = resolveSEO(baseSettings({ homeDescription: 'Configured homepage description.' }), {
      type: 'home', path: '/', description: 'Theme fallback description.'
    })
    expect(result.description).toBe('Configured homepage description.')
  })

  test('private topic cannot be made indexable', () => {
    const result = resolveSEO(baseSettings(), {
      type: 'topic', path: '/t/1/private', title: 'Private', public: false
    })

    expect(result.robots).toBe('noindex,nofollow')
    expect(result.includeInSitemap).toBe(false)
  })

  test('pagination keeps a self canonical URL', () => {
    const result = resolveSEO(baseSettings(), {
      type: 'category', path: '/c/general?page=3', canonicalPath: '/c/general?page=3', title: 'General', variables: { categoryName: 'General' }
    })

    expect(result.canonicalUrl).toBe('https://example.com/c/general?page=3')
    expect(result.robots).toContain('index,follow')
  })

  test('profile is noindex by recommended policy', () => {
    const result = resolveSEO(baseSettings(), {
      type: 'profile', path: '/u/alice', title: 'Alice', variables: { authorName: 'Alice' }
    })

    expect(result.robots).toBe('noindex,follow')
    expect(result.includeInSitemap).toBe(false)
  })

  test('topic description prefers summary then plain text excerpt', () => {
    const withSummary = resolveSEO(baseSettings(), {
      type: 'topic', path: '/t/1', title: 'Deploy', summary: 'Short summary', excerpt: 'Long excerpt', variables: { topicTitle: 'Deploy' }
    })
    const withExcerpt = resolveSEO(baseSettings(), {
      type: 'topic', path: '/t/1', title: 'Deploy', excerpt: 'Long excerpt', variables: { topicTitle: 'Deploy' }
    })

    expect(withSummary.description).toBe('Short summary')
    expect(withExcerpt.description).toBe('Long excerpt')
  })
})
