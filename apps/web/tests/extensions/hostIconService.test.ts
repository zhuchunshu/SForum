import { describe, expect, test } from 'bun:test'

import {
  getHostIconAsset,
  getHostIconAssets,
  renderHostIconDocument,
  renderHostIconInline,
  sanitizeIconBody
} from '../../server/utils/iconSvg'
import { HOST_ICON_BATCH_LIMIT } from '../../app/utils/iconPicker'

describe('host icon service', () => {
  test('resolves a real icon from both approved collections', () => {
    const lucide = getHostIconAsset('i-lucide-arrow-up-right')
    expect(lucide?.set).toBe('lucide')
    expect(lucide?.body).toContain('<path')
    // 图标体必须使用 currentColor，否则宿主主题色无法生效。
    expect(lucide?.body).toContain('currentColor')

    const tabler = getHostIconAsset('i-tabler-brand-github')
    expect(tabler?.set).toBe('tabler')
    expect(tabler?.body.length).toBeGreaterThan(0)
  })

  test('resolves aliases through the iconify parent chain', () => {
    // tabler 的 brand-livewire 走 alias + defs/use 片段引用。
    const asset = getHostIconAsset('i-tabler-brand-livewire')
    expect(asset?.body).toContain('<use href="#tabler-brand-livewire-')
  })

  test('rejects unknown names and malformed input', () => {
    expect(getHostIconAsset('i-lucide-definitely-not-a-real-icon')).toBeUndefined()
    expect(getHostIconAsset('i-mdi-home')).toBeUndefined()
    expect(getHostIconAsset('i-lucide-../../etc/passwd')).toBeUndefined()
    expect(getHostIconAsset('')).toBeUndefined()
    expect(getHostIconAsset(null)).toBeUndefined()
    expect(getHostIconAsset(42)).toBeUndefined()
  })

  test('batches deduplicate, skip unknown names, and stay bounded', () => {
    const assets = getHostIconAssets(['i-lucide-check', 'i-lucide-check', 'bogus', '', null])
    expect(assets.map((item) => item.name)).toEqual(['i-lucide-check'])

    // 去重后最多处理 HOST_ICON_BATCH_LIMIT 个名称：再多一个唯一名称就超出上限被丢弃。
    const unknown = Array.from({ length: HOST_ICON_BATCH_LIMIT }, (_, index) => `i-lucide-none-${index}`)
    expect(getHostIconAssets([...unknown, 'i-lucide-check'])).toEqual([])
    expect(getHostIconAssets([...unknown.slice(0, HOST_ICON_BATCH_LIMIT - 1), 'i-lucide-check']))
      .toHaveLength(1)
  })

  test('renders both inlinable and standalone documents', () => {
    const asset = getHostIconAsset('i-lucide-check')!
    expect(renderHostIconInline(asset)).toContain('width="1em"')
    expect(renderHostIconDocument(asset)).toContain('width="24"')
    expect(renderHostIconDocument(asset)).toContain('viewBox="0 0 24 24"')
  })
})

describe('host icon sanitizer', () => {
  test('drops script and style blocks with their content', () => {
    const body = sanitizeIconBody('<script>alert(1)</script><style>*{x:1}</style><path d="M0 0h4"/>', 'i-lucide-x')
    expect(body).toBe('<path d="M0 0h4"/>')
  })

  test('drops every event handler attribute', () => {
    const body = sanitizeIconBody('<path d="M0 0h4" onload="alert(1)" onclick="x()" onERROR="y()"/>', 'i-lucide-x')
    expect(body).toBe('<path d="M0 0h4"/>')
  })

  test('drops elements outside the allowlist', () => {
    const body = sanitizeIconBody('<foreignObject><p>x</p></foreignObject><a href="https://evil.example/"><path d="M0 0h4"/></a><image src="x.png"/>', 'i-lucide-x')
    expect(body).toBe('<path d="M0 0h4"/>')
  })

  test('drops non-allowlisted attributes but keeps geometry and presentation', () => {
    const body = sanitizeIconBody('<path d="M0 0h4" fill="none" stroke="currentColor" data-x="1" style="background:url(x)"/>', 'i-lucide-x')
    expect(body).toBe('<path d="M0 0h4" fill="none" stroke="currentColor"/>')
  })

  test('rejects javascript and data-text URLs outright', () => {
    expect(sanitizeIconBody('<path d="M0 0h4" fill="javascript:alert(1)"/>', 'i-lucide-x')).toBe('')
    expect(sanitizeIconBody('<path d="M0 0h4"/><iframe src="data:text/html,<script>1</script>"/>', 'i-lucide-x')).toBe('')
  })

  test('keeps only same-document fragment references and namespaces their ids', () => {
    const body = sanitizeIconBody('<defs><path id="shape" d="M0 0h4"/></defs><use href="#shape"/><use href="#missing"/><use href="https://evil.example/x"/>', 'i-lucide-x')
    expect(body).toBe('<defs><path id="i-lucide-x-shape" d="M0 0h4"/></defs><use href="#i-lucide-x-shape"/>')
  })

  test('keeps fragment references inside presentation attributes', () => {
    const body = sanitizeIconBody('<defs><path id="c" d="M0 0h4"/></defs><rect fill="url(#c)" clip-path="url(#c)" stroke="#fff" opacity="0.5"/>', 'i-lucide-x')
    expect(body).toContain('fill="url(#i-lucide-x-c)"')
    expect(body).toContain('clip-path="url(#i-lucide-x-c)"')
    expect(body).toContain('stroke="#fff"')
    expect(body).toContain('opacity="0.5"')
    expect(body).not.toContain('id="c"')
  })

  test('re-strips nested script blocks until the body stops changing', () => {
    // 单次替换会把 `<script>a</script>` 挖掉，剩下的片段重新拼出 `<script>`；
    // 剥离到不动点后必须由整块检测判死，而不是把拼出来的标签带进输出。
    expect(sanitizeIconBody('<scr<script>a</script>ipt><path d="M0 0h4"/>', 'i-lucide-x')).toBe('')
    expect(sanitizeIconBody('<scr<script>a</script>ipt>b</script>', 'i-lucide-x')).toBe('')
  })

  test('returns an empty body when nothing survives', () => {
    expect(sanitizeIconBody('', 'i-lucide-x')).toBe('')
    expect(sanitizeIconBody('plain text', 'i-lucide-x')).toBe('')
    expect(sanitizeIconBody('<script>alert(1)</script>', 'i-lucide-x')).toBe('')
  })
})
