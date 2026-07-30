import { createError, getRequestURL, getRouterParam } from 'h3'

import { proxyRouteRequest } from '../../../utils/pluginRouteProxy'

const PUBLIC_ID_PATTERN = /^[A-Za-z0-9_-]{1,128}$/

export default defineEventHandler((event) => {
  const requestURL = getRequestURL(event)
  const publicId = getRouterParam(event, 'publicId')
  if (
    requestURL.search
    || (event.method !== 'GET' && event.method !== 'HEAD')
    || !publicId
    || publicId !== publicId.trim()
    || !PUBLIC_ID_PATTERN.test(publicId)
  ) {
    throw createError({ statusCode: 404, statusMessage: 'Avatar not found' })
  }

  const apiBaseUrl = (process.env.NUXT_API_INTERNAL_BASE_URL || 'http://api:8080/api/v1').replace(/\/+$/, '')
  const target = new URL(`${apiBaseUrl}/attachments/${encodeURIComponent(publicId)}/content`)
  return proxyRouteRequest(event, target, { omitCredentials: true })
})
