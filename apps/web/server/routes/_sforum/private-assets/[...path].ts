import { createError, getRequestURL } from 'h3'

import { proxyRouteRequest } from '../../../utils/pluginRouteProxy'
import { buildPrivateAssetTarget } from '../../../utils/sforumAssetProxy'

export default defineEventHandler((event) => {
  const requestURL = getRequestURL(event)
  if (requestURL.search || (event.method !== 'GET' && event.method !== 'HEAD')) {
    throw createError({ statusCode: 404, statusMessage: 'Asset not found' })
  }
  try {
    const apiBaseURL = process.env.NUXT_API_INTERNAL_BASE_URL || 'http://api:8080/api/v1'
    return proxyRouteRequest(event, buildPrivateAssetTarget(apiBaseURL, requestURL.pathname))
  } catch {
    throw createError({ statusCode: 404, statusMessage: 'Asset not found' })
  }
})
