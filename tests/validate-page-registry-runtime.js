/**
 * Runtime Page Registry 浏览器/HTTP 验收（隔离端口，不杀 3000）。
 *
 * 用法：
 *   PAGE_REGISTRY_API=http://127.0.0.1:18080 PAGE_REGISTRY_WEB=http://127.0.0.1:13000 \
 *     node tests/validate-page-registry-runtime.js
 *
 * 若环境未启动隔离服务，脚本以 skip 退出 0（门禁不失败），并打印如何手动跑。
 * 完整浏览器断言需用户/CI 起 API+Web 后设置上述环境变量。
 */
import assert from 'node:assert/strict'

const API = process.env.PAGE_REGISTRY_API || process.env.API_BASE || ''
const WEB = process.env.PAGE_REGISTRY_WEB || process.env.WEB_BASE || ''

async function getJSON(url, opts = {}) {
  const res = await fetch(url, opts)
  const text = await res.text()
  let body
  try {
    body = JSON.parse(text)
  } catch {
    body = text
  }
  return { status: res.status, body, headers: res.headers }
}

async function main() {
  if (!API) {
    console.log('[page-registry] SKIP: set PAGE_REGISTRY_API to run live checks (isolated port; do not kill :3000)')
    console.log('  Example: PAGE_REGISTRY_API=http://127.0.0.1:18080 node tests/validate-page-registry-runtime.js')
    return
  }
  console.log('[page-registry] API=', API, 'WEB=', WEB || '(none)')

  // 1) resolve core home
  {
    const r = await getJSON(`${API}/api/v1/pages/resolve?id=forum.home`)
    assert.equal(r.status, 200, 'resolve core status')
    assert.equal(r.body?.data?.provider, 'core', 'default provider is core')
    console.log('✓ resolve core home')
  }

  // 2) resolve-path 404 for unknown
  {
    const r = await getJSON(`${API}/api/v1/pages/resolve-path?path=/no-such-page-xyz`)
    assert.equal(r.status, 404)
    console.log('✓ resolve-path 404')
  }

  // 3) catalog
  {
    const r = await getJSON(`${API}/api/v1/pages/catalog`)
    assert.equal(r.status, 200)
    assert.ok(Array.isArray(r.body?.data) || Array.isArray(r.body))
    console.log('✓ public catalog')
  }

  // 4) active skin (may be empty)
  {
    const r = await getJSON(`${API}/api/v1/site/active-theme/skin`)
    assert.equal(r.status, 200)
    console.log('✓ active theme skin')
  }

  // 5) optional: demo-docs if fixture plugin enabled
  {
    const r = await getJSON(`${API}/api/v1/pages/resolve-path?path=/demo-docs/hello`)
    if (r.status === 200) {
      assert.equal(r.body?.data?.action, 'add')
      console.log('✓ demo-docs/hello add page live')
    } else {
      console.log('· demo-docs not registered (plugin not enabled in this env)')
    }
  }

  // 6) optional web homepage HTML (no L2 script execution — check source only as smoke)
  if (WEB) {
    const res = await fetch(WEB + '/')
    const html = await res.text()
    assert.ok(res.status === 200 || res.status === 302, 'web home status')
    // L2 已关闭：不应出现动态 import 扩展 widget 的脚本入口
    assert.ok(!html.includes('SFExtensionWidget') || !html.includes('import('), 'no dynamic extension widget import in HTML smoke')
    console.log('✓ web home smoke (status + no L2 dynamic import)')
  }

  console.log('[page-registry] all live checks passed')
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})
