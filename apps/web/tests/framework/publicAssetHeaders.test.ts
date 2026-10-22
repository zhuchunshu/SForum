import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

const source = (path: string) => readFileSync(fileURLToPath(new URL(path, import.meta.url)), 'utf8')
const nuxtConfig = source('../../nuxt.config.ts')

/**
 * 公开只读资源必须对"换源"链路可加载：QQ 内置浏览器（X5 云加速）、CDN、企业
 * 代理会把文档或子资源换源，浏览器随后按跨源校验。这里钉住三件事：
 * 1. 应用自己发 ACAO + CORP，不依赖某个 ingress；
 * 2. 私有资产不被这条策略误伤；
 * 3. 两个 ingress 生成路径保持同样的冗余头。
 */
describe('public asset cross-origin policy', () => {
  test('marks build assets and public extension assets as cross-origin readable', () => {
    expect(nuxtConfig).toContain("'access-control-allow-origin': '*'")
    expect(nuxtConfig).toContain("'cross-origin-resource-policy': 'cross-origin'")
    expect(nuxtConfig).toContain("'timing-allow-origin': '*'")

    for (const route of ["'/_nuxt/**'", "'/_sforum/assets/**'", "'/_sforum/icons/**'"]) {
      expect(nuxtConfig).toContain(`${route}: { headers: publicAssetCorsHeaders }`)
    }
  })

  test('keeps private extension assets on the same-origin policy', () => {
    expect(nuxtConfig).not.toContain("'/_sforum/private-assets/**'")
    expect(nuxtConfig).not.toContain("'/_sforum/**': { headers: publicAssetCorsHeaders }")
  })

  test('keeps both ingress definitions redundant with the same asset headers', () => {
    const legacyCaddy = source('../../../../deploy/caddy/Caddyfile')
    const routerExample = source('../../../../deploy/router/Caddyfile.example')
    const upgradeScript = source('../../../../upgrade.sh')

    for (const config of [legacyCaddy, routerExample, upgradeScript]) {
      expect(config).toContain('Access-Control-Allow-Origin "*"')
      expect(config).toContain('Cross-Origin-Resource-Policy "cross-origin"')
      expect(config).toContain('/_nuxt/* /_sforum/*')
      // Caddy 默认在反代之前写头，上游的 same-origin 会再追加一份；defer 才会覆盖。
      expect(config).toContain('defer')
    }
  })
})
