import { describe, expect, test } from 'bun:test'
import { buildSEOStructuredData } from '../app/utils/seoStructuredData'

const base = {
  siteUrl: 'https://example.com', siteName: 'SForum Developers', canonicalUrl: 'https://example.com/c/general',
  title: 'General | SForum Developers', description: 'General discussions', image: '',
  schemaType: 'CollectionPage' as const, indexable: true, searchActionEnabled: true, discussionEnabled: true,
  organizationLogoUrl: '', breadcrumbs: [{ name: 'Home', path: '/' }, { name: 'General', path: '/c/general' }]
}

describe('buildSEOStructuredData', () => {
  test('builds site, organization, collection, and breadcrumb nodes', () => {
    const graph = buildSEOStructuredData(base)['@graph']
    expect(graph.map(node => node['@type'])).toEqual(['WebSite', 'Organization', 'CollectionPage', 'BreadcrumbList'])
    expect(graph[0].name).toBe('SForum Developers')
  })

  test('builds a discussion node only for indexable public topics', () => {
    const topic = buildSEOStructuredData({ ...base, canonicalUrl: 'https://example.com/t/1/hello', schemaType: 'DiscussionForumPosting', authorName: 'Alice', datePublished: '2026-07-10T00:00:00Z' })['@graph']
    const hidden = buildSEOStructuredData({ ...base, schemaType: 'DiscussionForumPosting', indexable: false })['@graph']
    expect(topic.some(node => node['@type'] === 'DiscussionForumPosting')).toBe(true)
    expect(hidden.some(node => node['@type'] === 'DiscussionForumPosting')).toBe(false)
  })
})
