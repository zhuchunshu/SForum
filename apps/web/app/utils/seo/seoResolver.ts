export type SEOPageType = 'home' | 'category' | 'tag' | 'topic' | 'profile' | 'static'
export type SEOIndexMode = 'index' | 'noindex'

export type SEOContentPolicy = {
  titleTemplate: string
  descriptionSources: string[]
  defaultImageUrl: string
  indexMode: SEOIndexMode
  includeInSitemap: boolean
  schemaType: 'WebSite' | 'CollectionPage' | 'DiscussionForumPosting' | 'ProfilePage' | 'WebPage'
}

export type SEOResolverSettings = {
  siteName: string
  siteUrl: string
  seoSiteName: string
  homeTitle: string
  homeDescription: string
  homeKeywords: string
  homeOGTitle: string
  homeOGDescription: string
  homeOGImageUrl: string
  pageTitleTemplate: string
  pageDefaultDescription: string
  allowIndexing: boolean
  policies: Record<SEOPageType, SEOContentPolicy>
}

export type SEOPageContext = {
  type: SEOPageType
  path: string
  canonicalPath?: string
  title?: string
  description?: string
  summary?: string
  excerpt?: string
  image?: string
  public?: boolean
  published?: boolean
  deleted?: boolean
  hidden?: boolean
  noindex?: boolean
  nofollow?: boolean
  variables?: Record<string, string | undefined>
  datePublished?: string
  dateModified?: string
  authorName?: string
  breadcrumbs?: Array<{ name: string, path: string }>
}

export type ResolvedSEO = {
  title: string
  description: string
  keywords: string
  siteName: string
  canonicalUrl: string
  image: string
  ogTitle: string
  ogDescription: string
  robots: string
  includeInSitemap: boolean
  schemaType: SEOContentPolicy['schemaType']
  sources: {
    title: string
    description: string
    image: string
  }
  context: SEOPageContext
}

export function resolveSEO(settings: SEOResolverSettings, context: SEOPageContext): ResolvedSEO {
  const policy = settings.policies[context.type]
  const variables = {
    pageTitle: context.title,
    categoryName: context.variables?.categoryName,
    tagName: context.variables?.tagName,
    topicTitle: context.variables?.topicTitle || context.title,
    authorName: context.variables?.authorName || context.authorName || context.title,
    seoSiteName: settings.seoSiteName,
    ...context.variables
  }
  const isHome = context.type === 'home'
  const title = isHome
    ? firstText(settings.homeTitle, settings.seoSiteName, settings.siteName)
    : firstText(renderTemplate(policy.titleTemplate || settings.pageTitleTemplate, variables), context.title, settings.seoSiteName)
  const description = isHome
    ? firstText(settings.homeDescription, context.description, settings.pageDefaultDescription, settings.seoSiteName)
    : resolveDescription(policy.descriptionSources, context, settings.pageDefaultDescription, settings.seoSiteName)
  const image = firstText(context.image, isHome ? settings.homeOGImageUrl : '', policy.defaultImageUrl)
  const hardNoindex = context.public === false || context.published === false || context.deleted === true || context.hidden === true
  const noindex = hardNoindex || context.noindex === true || !settings.allowIndexing || policy.indexMode === 'noindex'
  const nofollow = hardNoindex || context.nofollow === true

  return {
    title,
    description,
    keywords: isHome ? settings.homeKeywords : '',
    siteName: settings.seoSiteName,
    canonicalUrl: absoluteUrl(settings.siteUrl, context.canonicalPath || context.path),
    image,
    ogTitle: firstText(isHome ? settings.homeOGTitle : '', title),
    ogDescription: firstText(isHome ? settings.homeOGDescription : '', description),
    robots: `${noindex ? 'noindex' : 'index'},${nofollow ? 'nofollow' : 'follow'}${noindex ? '' : ',max-image-preview:large,max-snippet:-1,max-video-preview:-1'}`,
    includeInSitemap: !noindex && policy.includeInSitemap,
    schemaType: policy.schemaType,
    sources: {
      title: isHome && settings.homeTitle ? 'seo.home.title' : `seo.content_type.${context.type}.title_template`,
      description: descriptionSource(policy.descriptionSources, context, isHome),
      image: context.image ? 'page' : isHome && settings.homeOGImageUrl ? 'seo.home.og_image_url' : `seo.content_type.${context.type}.default_image_url`
    },
    context
  }
}

function resolveDescription(sources: string[], context: SEOPageContext, siteDefault: string, siteName: string) {
  for (const source of sources) {
    const value = sourceValue(source, context, siteDefault)
    if (value) {
      return value
    }
  }
  return firstText(context.description, context.summary, context.excerpt, siteDefault, siteName)
}

function descriptionSource(sources: string[], context: SEOPageContext, isHome: boolean) {
  if (isHome) {
    return context.description ? 'page' : 'seo.home.description'
  }
  return sources.find(source => sourceValue(source, context, '')) || 'seo.page.default_description'
}

function sourceValue(source: string, context: SEOPageContext, siteDefault: string) {
  switch (source) {
    case 'content':
    case 'category_description':
    case 'tag_description':
    case 'profile_bio':
    case 'page_description':
      return cleanText(context.description)
    case 'summary':
    case 'topic_summary':
      return cleanText(context.summary)
    case 'excerpt':
    case 'topic_excerpt':
      return cleanText(context.excerpt)
    case 'site_default':
      return cleanText(siteDefault)
    default:
      return ''
  }
}

function renderTemplate(template: string, variables: Record<string, string | undefined>) {
  return cleanText(template.replace(/\{([a-zA-Z][a-zA-Z0-9]*)\}/g, (_, name: string) => variables[name]?.trim() || ''))
}

function cleanText(value?: string) {
  return value?.trim().replace(/\s+/g, ' ') || ''
}

function firstText(...values: Array<string | undefined>) {
  for (const value of values) {
    const text = cleanText(value)
    if (text) {
      return text
    }
  }
  return ''
}

function absoluteUrl(siteUrl: string, path: string) {
  const base = siteUrl.replace(/\/+$/, '')
  const normalizedPath = path.startsWith('/') ? path : `/${path}`
  return `${base}${normalizedPath}`
}
