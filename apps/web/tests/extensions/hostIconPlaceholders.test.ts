import { beforeEach, describe, expect, test } from 'bun:test'

import { installTestDom } from '../helpers/dom'

// 必须在导入被测模块之前安装 DOM：被测函数默认以 document 为根。
installTestDom()

const {
  createHostIconResolver,
  hydrateHostIconPlaceholders,
  resetHostIconCache
} = await import('../../app/utils/icons/hostIcons')

beforeEach(() => {
  resetHostIconCache()
  document.body.replaceChildren()
})

const CHECK_SVG = '<svg viewBox="0 0 24 24"><path d="M0 0h4"/></svg>'

function placeholder(name: string) {
  const span = document.createElement('span')
  span.className = 'sf-host-icon'
  span.setAttribute('data-sf-icon', name)
  span.setAttribute('aria-hidden', 'true')
  return span
}

function stubResolver(icons: Record<string, string>) {
  const calls: string[][] = []
  return {
    calls,
    resolver: {
      resolve: async (names: readonly string[]) => {
        calls.push([...names])
        const result: Record<string, string> = {}
        for (const name of names) {
          if (icons[name]) result[name] = icons[name]
        }
        return result
      }
    }
  }
}

describe('theme template icon placeholders', () => {
  test('fills every placeholder from the host resolver', async () => {
    const first = placeholder('i-lucide-check')
    const second = placeholder('i-tabler-x')
    document.body.append(first, second)
    const { resolver, calls } = stubResolver({ 'i-lucide-check': CHECK_SVG, 'i-tabler-x': CHECK_SVG })

    const filled = await hydrateHostIconPlaceholders(document, resolver)

    expect(filled).toBe(2)
    expect(calls).toEqual([['i-lucide-check', 'i-tabler-x']])
    expect(first.innerHTML).toContain('<path d="M0 0h4"')
    expect(second.innerHTML).toContain('<path d="M0 0h4"')
  })

  test('never processes the same placeholder twice', async () => {
    document.body.append(placeholder('i-lucide-check'))
    const { resolver, calls } = stubResolver({ 'i-lucide-check': CHECK_SVG })

    await hydrateHostIconPlaceholders(document, resolver)
    const second = await hydrateHostIconPlaceholders(document, resolver)

    expect(second).toBe(0)
    expect(calls).toHaveLength(1)
  })

  test('leaves an unresolvable name as an empty placeholder instead of failing', async () => {
    const span = placeholder('i-lucide-missing')
    document.body.append(span)
    const { resolver } = stubResolver({})

    const filled = await hydrateHostIconPlaceholders(document, resolver)

    expect(filled).toBe(0)
    expect(span.innerHTML).toBe('')
    // 已标记：一次失败不会变成每次导航都重试的请求风暴。
    expect(span.hasAttribute('data-sf-icon-ready')).toBe(true)
  })

  test('never throws when the resolver rejects', async () => {
    const span = placeholder('i-lucide-check')
    document.body.append(span)

    await expect(hydrateHostIconPlaceholders(document, {
      resolve: async () => { throw new Error('offline') }
    })).resolves.toBe(0)
    expect(span.innerHTML).toBe('')
  })

  test('skips the resolver entirely when there is nothing to fill', async () => {
    const { resolver, calls } = stubResolver({})

    expect(await hydrateHostIconPlaceholders(document, resolver)).toBe(0)
    expect(await hydrateHostIconPlaceholders(null, resolver)).toBe(0)
    expect(calls).toEqual([])
  })

  test('uses the shared resolver cache across calls', async () => {
    let fetches = 0
    const resolver = createHostIconResolver(async (names) => {
      fetches += 1
      return names.map((name) => ({ name, svg: CHECK_SVG }))
    })

    document.body.append(placeholder('i-lucide-check'))
    await hydrateHostIconPlaceholders(document, resolver)
    document.body.replaceChildren()
    document.body.append(placeholder('i-lucide-check'))
    await hydrateHostIconPlaceholders(document, resolver)

    expect(fetches).toBe(1)
  })
})
