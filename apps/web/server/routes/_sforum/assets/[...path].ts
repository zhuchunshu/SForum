import { createError, getRequestURL } from 'h3'

import { proxyRouteRequest } from '../../../utils/pluginRouteProxy'
import { buildPublicAssetTarget } from '../../../utils/sforumAssetProxy'

export default defineEventHandler(async (event) => {
  const requestURL = getRequestURL(event)
  if (requestURL.search || (event.method !== 'GET' && event.method !== 'HEAD')) {
    throw createError({ statusCode: 404, statusMessage: 'Asset not found' })
  }
  let target: URL
  try {
    const apiBaseURL = process.env.NUXT_API_INTERNAL_BASE_URL || 'http://api:8080/api/v1'
    target = buildPublicAssetTarget(apiBaseURL, requestURL.pathname)
  } catch {
    throw createError({ statusCode: 404, statusMessage: 'Asset not found' })
  }
  return proxyRouteRequest(event, target, { omitCredentials: true })
})
