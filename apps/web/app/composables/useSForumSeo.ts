import type { ComputedRef, Ref } from 'vue'
import type { SEOSettings } from './useWebOptions'

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

export const useSForumSeo = (input: SForumSEOInput) => {
  const route = useRoute()
  const { siteName, siteUrl, seoSettings, seoIndexable } = useWebOptions()

  const title = computed(() => (resolveReactive(input.title) || '').trim())
  const description = computed(() => resolveReactive(input.description)?.trim() || seoSettings.value.metaDescription)
  const canonicalPath = computed(() => resolveReactive(input.path) || route.path || '/')
  const canonicalUrl = computed(() => absoluteSiteUrl(siteUrl.value, canonicalPath.value))
  const image = computed(() => resolveReactive(input.image)?.trim() || seoSettings.value.ogImageUrl)
  const robotsRule = computed(() => {
    const pageNoindex = resolveReactive(input.noindex) === true
    return pageNoindex || !seoIndexable.value
      ? 'noindex, nofollow'
      : 'index, follow, max-image-preview:large, max-snippet:-1, max-video-preview:-1'
  })

  useSeoMeta({
    title,
    description,
    ogTitle: title,
    ogDescription: description,
    ogType: () => resolveReactive(input.type) || 'website',
    ogUrl: canonicalUrl,
    ogImage: image,
    twitterCard: () => seoSettings.value.twitterCard,
    twitterSite: () => seoSettings.value.twitterSite || undefined,
    twitterTitle: title,
    twitterDescription: description,
    twitterImage: image
  })

  useHead(() => ({
    link: [
      { key: 'canonical', rel: 'canonical', href: canonicalUrl.value }
    ],
    meta: [
      { key: 'robots', name: 'robots', content: robotsRule.value },
      ...verificationMeta(seoSettings.value)
    ],
    script: structuredDataScript({
      settings: seoSettings.value,
      siteName: siteName.value,
      siteUrl: siteUrl.value,
      canonicalUrl: canonicalUrl.value,
      title: title.value,
      description: description.value,
      image: image.value,
      schema: resolveReactive(input.schema)
    })
  }))

  return {
    title,
    description,
    canonicalUrl,
    robotsRule
  }
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
      target: `${input.siteUrl.replace(/\/+$/, '')}/search?q={search_term_string}`,
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
