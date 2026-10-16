import { beforeEach, describe, expect, test } from 'bun:test'

import {
  createHostIconResolver,
  resetHostIconCache,
  type HostIconFetcher,
  type HostIconResponseItem
} from '../../app/utils/icons/hostIcons'
import {
  HOST_ICON_BATCH_LIMIT,
  HOST_ICON_NAME_PATTERN,
  isHostIconName,
  parseHostIconName
} from '../../app/utils/iconPicker'

beforeEach(() => {
  resetHostIconCache()
})

function recordingFetcher(items?: (names: readonly string[]) => HostIconResponseItem[]) {
  const calls: string[][] = []
  const fetcher: HostIconFetcher = async (names) => {
    calls.push([...names])
    return items ? items(names) : names.map((name) => ({ name, svg: `<svg data-icon="${name}"></svg>` }))
  }
  return { calls, fetcher }
}

describe('host icon name grammar', () => {
  test('accepts only the two host collections', () => {
    expect(isHostIconName('i-lucide-arrow-up-right')).toBe(true)
    expect(isHostIconName('i-tabler-brand-github')).toBe(true)
    expect(isHostIconName('i-mdi-home')).toBe(false)
    expect(isHostIconName('i-lucide-')).toBe(false)
    expect(isHostIconName('i-lucide-../etc/passwd')).toBe(false)
    expect(isHostIconName('i-lucide-Alarm')).toBe(false)
    expect(isHostIconName(42)).toBe(false)
    expect(isHostIconName(undefined)).toBe(false)
  })

  test('splits the collection from the icon name', () => {
    expect(parseHostIconName('i-lucide-check')).toEqual({ set: 'lucide', icon: 'check' })
    expect(parseHostIconName('i-tabler-chevron-down')).toEqual({ set: 'tabler', icon: 'chevron-down' })
    expect(parseHostIconName('nope')).toBeUndefined()
  })

  test('pins one shared pattern for Host and SDK', () => {
    expect(HOST_ICON_NAME_PATTERN.test('i-lucide-check')).toBe(true)
    expect(HOST_ICON_NAME_PATTERN.test('lucide-check')).toBe(false)
  })
})

describe('host icon resolver', () => {
  test('resolves a batch and reuses the cache on later calls', async () => {
    const { calls, fetcher } = recordingFetcher()
    const resolver = createHostIconResolver(fetcher)

    const first = await resolver.resolve(['i-lucide-check', 'i-tabler-x'])
    expect(first).toEqual({
      'i-lucide-check': '<svg data-icon="i-lucide-check"></svg>',
      'i-tabler-x': '<svg data-icon="i-tabler-x"></svg>'
    })

    const second = await resolver.resolve(['i-lucide-check'])
    expect(second).toEqual({ 'i-lucide-check': '<svg data-icon="i-lucide-check"></svg>' })
    expect(calls).toEqual([['i-lucide-check', 'i-tabler-x']])
  })

  test('deduplicates names and drops anything outside the grammar', async () => {
    const { calls, fetcher } = recordingFetcher()
    const resolver = createHostIconResolver(fetcher)

    await resolver.resolve(['i-lucide-check', 'i-lucide-check', 'i-mdi-home', '', '  '])

    expect(calls).toEqual([['i-lucide-check']])
  })

  test(`splits requests at the ${HOST_ICON_BATCH_LIMIT} name bound`, async () => {
    const { calls, fetcher } = recordingFetcher()
    const resolver = createHostIconResolver(fetcher)
    const names = Array.from({ length: HOST_ICON_BATCH_LIMIT + 5 }, (_, index) => `i-lucide-icon-${index}`)

    const result = await resolver.resolve(names)

    expect(calls.map((batch) => batch.length)).toEqual([HOST_ICON_BATCH_LIMIT, 5])
    expect(Object.keys(result)).toHaveLength(names.length)
  })

  test('shares one request between concurrent callers', async () => {
    const { calls, fetcher } = recordingFetcher()
    const resolver = createHostIconResolver(fetcher)

    await Promise.all([
      resolver.resolve(['i-lucide-check']),
      resolver.resolve(['i-lucide-check'])
    ])

    expect(calls).toEqual([['i-lucide-check']])
  })

  test('omits names the Host could not resolve instead of failing the batch', async () => {
    const { fetcher } = recordingFetcher((names) =>
      names.filter((name) => name !== 'i-lucide-missing').map((name) => ({ name, svg: `<svg data-icon="${name}"></svg>` }))
    )
    const resolver = createHostIconResolver(fetcher)

    const result = await resolver.resolve(['i-lucide-check', 'i-lucide-missing'])

    expect(result).toEqual({ 'i-lucide-check': '<svg data-icon="i-lucide-check"></svg>' })
    expect('i-lucide-missing' in result).toBe(false)
  })

  test('never throws when the fetch fails, and remembers the failure', async () => {
    let calls = 0
    const resolver = createHostIconResolver(async () => {
      calls += 1
      throw new Error('offline')
    })

    await expect(resolver.resolve(['i-lucide-check'])).resolves.toEqual({})
    await expect(resolver.resolve(['i-lucide-check'])).resolves.toEqual({})
    expect(calls).toBe(1)
  })

  test('skips the network entirely for an empty or invalid-only request', async () => {
    const { calls, fetcher } = recordingFetcher()
    const resolver = createHostIconResolver(fetcher)

    expect(await resolver.resolve([])).toEqual({})
    expect(await resolver.resolve(['not-an-icon'])).toEqual({})
    expect(calls).toEqual([])
  })
})
