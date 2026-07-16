import { getQuery } from 'h3'

import { proxyRouteRequest } from '../../../utils/pluginRouteProxy'

export default defineEventHandler((event) => {
  const pathParam = event.context.params?.path
  const pathSegments = Array.isArray(pathParam) ? pathParam : pathParam ? pathParam.split('/') : []
  const path = pathSegments.map((segment) => encodeURIComponent(segment)).join('/')
  const apiBaseUrl = (process.env.NUXT_API_INTERNAL_BASE_URL || 'http://api:8080/api/v1').replace(/\/+$/, '')
  const target = new URL(`${apiBaseUrl}/${path}`)

  for (const [key, value] of Object.entries(getQuery(event))) {
    if (Array.isArray(value)) {
      for (const item of value) {
        target.searchParams.append(key, String(item))
      }
      continue
    }

    if (value !== undefined && value !== null) {
      target.searchParams.set(key, String(value))
    }
  }

  return proxyRouteRequest(event, target)
})
