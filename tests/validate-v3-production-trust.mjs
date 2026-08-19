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
  'bare production API processes must default exact-artifact trust on'
)
assert(
  config.includes('envBool("SFORUM_V3_PUBLIC_L2", false)'),
  'bare processes must keep public L2 opt-in until the P9 production gate closes'
)

const compose = read('compose.yaml')
assert(
  occurrences(compose, 'SFORUM_V3_TRUST_CHALLENGES: ${SFORUM_V3_TRUST_CHALLENGES:-false}') === 1,
  'base Compose must pass the compatibility-default trust gate to the API'
)
assert(
  occurrences(compose, 'SFORUM_V3_TRUST_CHALLENGE_TTL: ${SFORUM_V3_TRUST_CHALLENGE_TTL:-5m}') === 1,
  'base Compose must pass the challenge TTL to the API'
)
assert(
  occurrences(compose, 'SFORUM_V3_PUBLIC_L2: ${SFORUM_V3_PUBLIC_L2:-false}') === 1,
  'base Compose must pass the opt-in public L2 gate only to the API'
)

const productionCompose = read('compose.prod.yaml')
assert(
  occurrences(productionCompose, 'SFORUM_V3_TRUST_CHALLENGES: ${SFORUM_V3_TRUST_CHALLENGES:-true}') === 1,
  'production Compose must enable exact-artifact trust for the API'
)
assert(
  occurrences(productionCompose, 'SFORUM_V3_PUBLIC_L2: ${SFORUM_V3_PUBLIC_L2:-false}') === 1,
  'production Compose must keep public L2 opt-in until its production gate closes'
)

const developmentEnv = read('.env.example')
assert(
  developmentEnv.includes('SFORUM_V3_TRUST_CHALLENGES=false'),
  'development env must document the explicit migration default'
)
assert(
  developmentEnv.includes('SFORUM_V3_PUBLIC_L2=false'),
  'development env must document that public L2 is opt-in'
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
assert(
  productionEnv.includes('SFORUM_V3_PUBLIC_L2=false'),
  'production env must not enable public L2 before the P9 acceptance gate'
)

console.log('V3 production trust deployment validation passed.')
