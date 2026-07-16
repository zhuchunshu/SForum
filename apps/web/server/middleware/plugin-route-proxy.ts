import { proxyDeclaredPluginRoute } from '../utils/pluginRouteProxy'

export default defineEventHandler((event) => proxyDeclaredPluginRoute(event))
