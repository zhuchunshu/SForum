import type { SEOContentPolicy, SEOPageContext } from './seoResolver'

export type SEOStructuredDataInput = {
  siteUrl: string
  siteName: string
  canonicalUrl: string
  title: string
  description: string
  image: string
  schemaType: SEOContentPolicy['schemaType']
  indexable: boolean
  searchActionEnabled: boolean
  discussionEnabled: boolean
  organizationLogoUrl: string
  breadcrumbs?: SEOPageContext['breadcrumbs']
  authorName?: string
  datePublished?: string
  dateModified?: string
}

export function buildSEOStructuredData(input: SEOStructuredDataInput) {
  const siteUrl = input.siteUrl.replace(/\/+$/, '')
  const website: Record<string, unknown> = { '@type': 'WebSite', '@id': `${siteUrl}/#website`, url: siteUrl, name: input.siteName }
  if (input.searchActionEnabled) {
    website.potentialAction = { '@type': 'SearchAction', target: `${siteUrl}/search?q={search_term_string}`, 'query-input': 'required name=search_term_string' }
  }
  const organization: Record<string, unknown> = { '@type': 'Organization', '@id': `${siteUrl}/#organization`, name: input.siteName, url: siteUrl }
  if (input.organizationLogoUrl) organization.logo = input.organizationLogoUrl
  const graph: Array<Record<string, unknown>> = [website, organization]

  const pageType = input.schemaType === 'DiscussionForumPosting' ? 'WebPage' : input.schemaType
  if (input.schemaType !== 'DiscussionForumPosting') {
    graph.push(clean({ '@type': pageType, '@id': `${input.canonicalUrl}#webpage`, url: input.canonicalUrl, name: input.title, description: input.description, image: input.image || undefined, isPartOf: { '@id': `${siteUrl}/#website` } }))
  }
  if (input.schemaType === 'DiscussionForumPosting' && input.indexable && input.discussionEnabled) {
    graph.push(clean({ '@type': 'DiscussionForumPosting', '@id': `${input.canonicalUrl}#discussion`, headline: input.title, text: input.description, url: input.canonicalUrl, image: input.image || undefined, datePublished: input.datePublished, dateModified: input.dateModified, author: input.authorName ? { '@type': 'Person', name: input.authorName } : undefined, isPartOf: { '@id': `${siteUrl}/#website` } }))
  }
  if (input.breadcrumbs?.length) {
    graph.push({ '@type': 'BreadcrumbList', '@id': `${input.canonicalUrl}#breadcrumb`, itemListElement: input.breadcrumbs.map((item, index) => ({ '@type': 'ListItem', position: index + 1, name: item.name, item: absoluteUrl(siteUrl, item.path) })) })
  }
  return { '@context': 'https://schema.org', '@graph': graph }
}

/**
 * 把结构化数据序列化成可内联进 `<script>` 的 JSON-LD 文本。
 *
 * `JSON.stringify` 不转义 `<`，而标题、描述等字段可能来自主题/插件产出的 HTML
 * 文本；其中只要出现 `</script>` 就会提前闭合脚本块。把 `<` 写成 `\u003c`
 * 仍是合法 JSON，消费端解析时解码回 `<`，页面文本不变。
 */
export function serializeStructuredData(graph: object): string {
  return JSON.stringify(graph).replace(/</g, '\\u003c')
}

function absoluteUrl(siteUrl: string, path: string) {
  return `${siteUrl}${path.startsWith('/') ? path : `/${path}`}`
}

function clean(value: Record<string, unknown>) {
  return Object.fromEntries(Object.entries(value).filter(([, item]) => item !== undefined && item !== ''))
}
