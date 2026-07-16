import fs from 'node:fs'

const caddy = fs.readFileSync('deploy/caddy/Caddyfile', 'utf8')
const compose = fs.readFileSync('compose.prod.yaml', 'utf8')
const productionEnv = fs.readFileSync('.env.production.example', 'utf8')

const websocketProxy = 'reverse_proxy @host_api_websocket 127.0.0.1:{$API_PORT:18080}'
const webProxy = 'reverse_proxy 127.0.0.1:{$WEB_PORT:3000}'
const requiredCaddyContracts = [
  '@host_api_websocket {',
  'header_regexp connection_upgrade Connection (?i)(^|.*,\\s*)upgrade(\\s*,.*|$)',
  'header_regexp websocket_upgrade Upgrade (?i)^websocket$',
  'not header Sec-WebSocket-Protocol *vite-hmr*',
  websocketProxy,
  webProxy
]

for (const contract of requiredCaddyContracts) {
  if (!caddy.includes(contract)) {
    throw new Error(`production Caddyfile is missing: ${contract}`)
  }
}
if (caddy.indexOf(websocketProxy) >= caddy.indexOf(webProxy)) {
  throw new Error('Host API WebSocket proxy must run before the Nuxt fallback')
}
if (/header_up\s+(?:Host|Origin|Cookie|Authorization|Sec-WebSocket-\S+)/i.test(caddy)) {
  throw new Error('production WebSocket proxy must not rewrite Host authority headers')
}
const apiPortMappings = compose.split('\n').filter(line => line.includes('${API_PORT'))
if (apiPortMappings.length !== 1 || apiPortMappings[0].trim() !== '- "127.0.0.1:${API_PORT:-18080}:8080"') {
  throw new Error('production API WebSocket ingress must bind only to loopback')
}
if (!/^API_PORT=18080$/m.test(productionEnv)) {
  throw new Error('production API_PORT default is missing')
}

console.log('Production WebSocket proxy contract is valid.')
