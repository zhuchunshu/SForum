import type { ComputedRef, Ref } from 'vue'
import type { SEOSettings } from './useWebOptions'
import { resolveSEO, type SEOPageContext } from '~/utils/seoResolver'

type MaybeReactive<T> = T | Ref<T> | ComputedRef<T> | (() => T)

export type SForumSchemaInput = {
  type?: 'WebPage' | 'DiscussionForumPosting'
  datePublished?: string
  dateModified?: string
  authorName?: string
}

export type SForumSEOInput = {
  title: MaybeReactive<string>
  description?: MaybeReactive<string | undefined>
  path?: MaybeReactive<string | undefined>
  image?: MaybeReactive<string | undefined>
  type?: MaybeReactive<'website' | 'article'>
  noindex?: MaybeReactive<boolean | undefined>
  schema?: MaybeReactive<SForumSchemaInput | undefined>
}

export const useSForumSeo = (input: SForumSEOInput | MaybeReactive<SEOPageContext>) => {
  const route = useRoute()
  const { siteUrl, seoSettings, seoIndexable } = useWebOptions()

  const pageContext = computed(() => normalizePageContext(input, route.path || '/'))
  const resolved = computed(() => resolveSEO({
    ...seoSettings.value,
    siteUrl: siteUrl.value,
    allowIndexing: seoIndexable.value
  }, pageContext.value))

  useSeoMeta({
    title: () => resolved.value.title,
    description: () => resolved.value.description,
    keywords: () => resolved.value.keywords || undefined,
    ogTitle: () => resolved.value.ogTitle,
    ogDescription: () => resolved.value.ogDescription,
    ogType: () => pageContext.value.type === 'topic' ? 'article' : 'website',
    ogUrl: () => resolved.value.canonicalUrl,
    ogImage: () => resolved.value.image || undefined,
    twitterCard: () => seoSettings.value.twitterCard,
    twitterSite: () => seoSettings.value.twitterSite || undefined,
    twitterTitle: () => resolved.value.ogTitle,
    twitterDescription: () => resolved.value.ogDescription,
    twitterImage: () => resolved.value.image || undefined
  })

  useHead(() => ({
    link: [
      { key: 'canonical', rel: 'canonical', href: resolved.value.canonicalUrl }
    ],
    meta: [
      { key: 'robots', name: 'robots', content: resolved.value.robots },
      ...verificationMeta(seoSettings.value)
    ],
    script: structuredDataScript({
      settings: seoSettings.value,
      siteName: resolved.value.siteName,
      siteUrl: siteUrl.value,
      canonicalUrl: resolved.value.canonicalUrl,
      title: resolved.value.title,
      description: resolved.value.description,
      image: resolved.value.image,
      schema: legacySchema(input, pageContext.value)
    })
  }))

  return {
    title: computed(() => resolved.value.title),
    description: computed(() => resolved.value.description),
    canonicalUrl: computed(() => resolved.value.canonicalUrl),
    robotsRule: computed(() => resolved.value.robots),
    resolved
  }
}

function normalizePageContext(input: SForumSEOInput | MaybeReactive<SEOPageContext>, routePath: string): SEOPageContext {
  const value = resolveReactive(input as MaybeReactive<SForumSEOInput | SEOPageContext>)
  if (value && isPageContext(value)) {
    return value
  }
  const legacy = value as SForumSEOInput | undefined
  const title = resolveReactive(legacy?.title)?.trim() || ''
  const path = resolveReactive(legacy?.path) || routePath
  const schema = resolveReactive(legacy?.schema)
  return {
    type: schema?.type === 'DiscussionForumPosting' ? 'topic' : path === '/' ? 'home' : 'static',
    path,
    title,
    description: resolveReactive(legacy?.description),
    image: resolveReactive(legacy?.image),
    noindex: resolveReactive(legacy?.noindex),
    variables: {
      pageTitle: title,
      topicTitle: title,
      authorName: schema?.authorName
    },
    datePublished: schema?.datePublished,
    dateModified: schema?.dateModified,
    authorName: schema?.authorName
  }
}

function isPageContext(value: SForumSEOInput | SEOPageContext): value is SEOPageContext {
  return ['home', 'category', 'tag', 'topic', 'profile', 'static'].includes(String(value.type))
}

function legacySchema(input: SForumSEOInput | MaybeReactive<SEOPageContext>, context: SEOPageContext): SForumSchemaInput | undefined {
  const value = resolveReactive(input as MaybeReactive<SForumSEOInput | SEOPageContext>)
  if (value && !isPageContext(value)) {
    return resolveReactive(value.schema)
  }
  return context.type === 'topic'
    ? { type: 'DiscussionForumPosting', datePublished: context.datePublished, dateModified: context.dateModified, authorName: context.authorName }
    : { type: 'WebPage' }
}

function resolveReactive<T>(value: MaybeReactive<T> | undefined): T | undefined {
  if (typeof value === 'function') {
    return (value as () => T)()
  }
  return unref(value)
}

function absoluteSiteUrl(siteUrl: string, path: string) {
  const base = siteUrl.replace(/\/+$/, '') || 'http://127.0.0.1:3000'
  const normalizedPath = path.startsWith('/') ? path : `/${path}`
  return `${base}${normalizedPath}`
}

function verificationMeta(settings: SEOSettings) {
  return [
    settings.googleVerification ? { key: 'google-site-verification', name: 'google-site-verification', content: settings.googleVerification } : null,
    settings.bingVerification ? { key: 'msvalidate.01', name: 'msvalidate.01', content: settings.bingVerification } : null,
    settings.baiduVerification ? { key: 'baidu-site-verification', name: 'baidu-site-verification', content: settings.baiduVerification } : null,
    settings.yandexVerification ? { key: 'yandex-verification', name: 'yandex-verification', content: settings.yandexVerification } : null
  ].filter((item): item is { key: string, name: string, content: string } => item !== null)
}

function structuredDataScript(input: {
  settings: SEOSettings
  siteName: string
  siteUrl: string
  canonicalUrl: string
  title: string
  description: string
  image: string
  schema?: SForumSchemaInput
}) {
  if (!input.settings.schemaOrgEnabled) {
    return []
  }

  const settings = input.settings
  const websiteNode: Record<string, unknown> = {
    '@type': 'WebSite',
    '@id': `${input.siteUrl.replace(/\/+$/, '')}/#website`,
    url: input.siteUrl,
    name: input.siteName
  }
  const organizationNode: Record<string, unknown> = {
    '@type': 'Organization',
    '@id': `${input.siteUrl.replace(/\/+$/, '')}/#organization`,
    name: input.siteName,
    url: input.siteUrl
  }
  const graph: Array<Record<string, unknown>> = [
    websiteNode,
    organizationNode,
    {
      '@type': 'WebPage',
      '@id': `${input.canonicalUrl}#webpage`,
      url: input.canonicalUrl,
      name: input.title,
      description: input.description,
      isPartOf: { '@id': `${input.siteUrl.replace(/\/+$/, '')}/#website` }
    }
  ]

  const logoUrl = settings.schemaOrgOrganizationLogoUrl
  if (logoUrl) {
    organizationNode.logo = logoUrl
  }

  if (settings.schemaOrgSearchActionEnabled) {
    websiteNode.potentialAction = {
      '@type': 'SearchAction',
      target: `${input.siteUrl.replace(/\/+$/, '')}/?q={search_term_string}`,
      'query-input': 'required name=search_term_string'
    }
  }

  if (
    input.schema?.type === 'DiscussionForumPosting' &&
    settings.schemaOrgDiscussionEnabled
  ) {
    graph.push({
      '@type': 'DiscussionForumPosting',
      '@id': `${input.canonicalUrl}#discussion`,
      headline: input.title,
      text: input.description,
      url: input.canonicalUrl,
      image: input.image || undefined,
      datePublished: input.schema.datePublished,
      dateModified: input.schema.dateModified,
      author: input.schema.authorName ? { '@type': 'Person', name: input.schema.authorName } : undefined
    })
  }

  return [
    {
      key: 'sforum-schema-org',
      type: 'application/ld+json',
      innerHTML: JSON.stringify({
        '@context': 'https://schema.org',
        '@graph': graph
      })
    }
  ]
}
