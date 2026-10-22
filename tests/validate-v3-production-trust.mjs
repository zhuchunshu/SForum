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

// 公开 L2 不再是迁移门：它恒开，只有 Safe Mode 与 exact-artifact trust 能关闭它。
// 下面这组断言防止该开关被重新引入（配置、装配、Compose、env 模板、生产初始化脚本）。
const FORBIDDEN_GATE_KEY = 'SFORUM_V3_PUBLIC_L2'
assert(
  !config.includes(FORBIDDEN_GATE_KEY),
  'public L2 must not reintroduce a configuration gate in the API config'
)
assert(
  !read('apps/api/bootstrap/api_assembly_extensions.go').includes('WithPublicL2'),
  'frontend assembly must not accept a public L2 switch'
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
  !compose.includes(FORBIDDEN_GATE_KEY),
  'base Compose must not pass a public L2 switch to the API'
)

const productionCompose = read('compose.prod.yaml')
assert(
  occurrences(productionCompose, 'SFORUM_V3_TRUST_CHALLENGES: ${SFORUM_V3_TRUST_CHALLENGES:-true}') === 1,
  'production Compose must enable exact-artifact trust for the API'
)
assert(
  !productionCompose.includes(FORBIDDEN_GATE_KEY),
  'production Compose must not pass a public L2 switch to the API'
)

const developmentEnv = read('.env.example')
assert(
  developmentEnv.includes('SFORUM_V3_TRUST_CHALLENGES=false'),
  'development env must document the explicit migration default'
)
assert(
  !developmentEnv.includes(FORBIDDEN_GATE_KEY),
  'development env must not document a public L2 switch'
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
  !productionEnv.includes(FORBIDDEN_GATE_KEY),
  'production env must not offer a public L2 switch'
)
assert(
  !read('deploy/scripts/configure-production.sh').includes(FORBIDDEN_GATE_KEY),
  'production env bootstrap must not emit a public L2 switch'
)

console.log('V3 production trust deployment validation passed.')
