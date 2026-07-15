import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const read = relativePath => fs.readFileSync(path.join(root, relativePath), 'utf8')
const assert = (condition, message) => {
  if (!condition) throw new Error(message)
}
const occurrences = (source, value) => source.split(value).length - 1

const config = read('apps/api/config/config.go')
assert(
  config.includes('envBool("SFORUM_V3_TRUST_CHALLENGES", isProd)'),
  'bare production API/worker processes must default exact-artifact trust on'
)

const compose = read('compose.yaml')
assert(
  occurrences(compose, 'SFORUM_V3_TRUST_CHALLENGES: ${SFORUM_V3_TRUST_CHALLENGES:-false}') === 2,
  'base Compose must pass the compatibility-default trust gate to API and worker'
)
assert(
  occurrences(compose, 'SFORUM_V3_TRUST_CHALLENGE_TTL: ${SFORUM_V3_TRUST_CHALLENGE_TTL:-5m}') === 2,
  'base Compose must pass the challenge TTL to API and worker'
)

const productionCompose = read('compose.prod.yaml')
assert(
  occurrences(productionCompose, 'SFORUM_V3_TRUST_CHALLENGES: ${SFORUM_V3_TRUST_CHALLENGES:-true}') === 2,
  'production Compose must enable exact-artifact trust for API and worker'
)

const developmentEnv = read('.env.example')
assert(
  developmentEnv.includes('SFORUM_V3_TRUST_CHALLENGES=false'),
  'development env must document the explicit migration default'
)
const productionEnv = read('.env.production.example')
assert(
  productionEnv.includes('SFORUM_V3_TRUST_CHALLENGES=true'),
  'production env must enable exact-artifact trust'
)
assert(
  productionEnv.includes('SFORUM_V3_TRUST_CHALLENGE_TTL=5m'),
  'production env must document the bounded challenge TTL'
)

console.log('V3 production trust deployment validation passed.')
