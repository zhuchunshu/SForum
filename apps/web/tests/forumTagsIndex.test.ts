import { describe, expect, test } from 'bun:test'

import {
  activeForumTags,
  filterTagIndexTags,
  recentTagIndexTags,
  tagDisplayDescription,
  tagHeatEntries,
  tagIndexOverview
} from '../app/utils/forumTagsIndex'
import type { ForumTag } from '../app/utils/forumTaxonomy'

const NOW = Date.parse('2026-07-23T12:00:00Z')

function tag(input: Partial<ForumTag> & Pick<ForumTag, 'id' | 'slug' | 'name' | 'topicCount' | 'createdAt'>): ForumTag {
  return {
    description: '',
    icon: '',
    iconColor: '',
    status: 'active',
    updatedAt: input.createdAt,
    ...input
  }
}

const tags = [
  tag({ id: 1, slug: 'sforum', name: 'SForum', description: 'Host framework', topicCount: 100, createdAt: '2026-07-01T00:00:00Z' }),
  tag({ id: 2, slug: 'vue', name: 'Vue', description: '', topicCount: 60, createdAt: '2026-07-21T00:00:00Z' }),
  tag({ id: 3, slug: 'api', name: 'API', description: 'OpenAPI contracts', topicCount: 20, createdAt: '2026-06-01T00:00:00Z' }),
  tag({ id: 4, slug: '中文标签', name: '中文标签', description: '中文搜索', topicCount: 8, createdAt: '2026-07-22T00:00:00Z' }),
  tag({ id: 5, slug: 'disabled', name: 'Disabled', topicCount: 999, createdAt: '2026-07-23T00:00:00Z', status: 'disabled' })
]

describe('forum tag index presentation helpers', () => {
  test('maps only real active tag fields and preserves missing descriptions', () => {
    expect(activeForumTags(tags).map((item) => item.slug)).toEqual(['sforum', 'vue', 'api', '中文标签'])
    expect(tagDisplayDescription(tags[1]!, 'No description yet')).toBe('No description yet')
    expect(tagDisplayDescription(tags[0]!, 'No description yet')).toBe('Host framework')
  })

  test('builds overview from real topic counts and createdAt', () => {
    expect(tagIndexOverview(tags, NOW)).toEqual({
      totalTags: 4,
      totalTopicReferences: 188,
      weekNewTags: 2,
      hotThreshold: 100
    })
  })

  test('filters all, hot, week, search, and A-Z without demo metrics', () => {
    expect(filterTagIndexTags(tags, { filter: 'all', nowMs: NOW }).map((item) => item.slug))
      .toEqual(['sforum', 'vue', 'api', '中文标签'])
    expect(filterTagIndexTags(tags, { filter: 'hot', nowMs: NOW }).map((item) => item.slug))
      .toEqual(['sforum'])
    expect(filterTagIndexTags(tags, { filter: 'week', nowMs: NOW }).map((item) => item.slug))
      .toEqual(['vue', '中文标签'])
    expect(filterTagIndexTags(tags, { filter: 'all', query: 'openapi', nowMs: NOW }).map((item) => item.slug))
      .toEqual(['api'])
    expect(filterTagIndexTags(tags, { filter: 'az', locale: 'en-US', nowMs: NOW }).map((item) => item.slug))
      .toEqual(['api', 'sforum', 'vue', '中文标签'])
  })

  test('returns empty lists for search or filter misses', () => {
    expect(filterTagIndexTags(tags, { filter: 'all', query: 'missing', nowMs: NOW })).toEqual([])
    expect(filterTagIndexTags([tags[2]!], { filter: 'week', nowMs: NOW })).toEqual([])
  })

  test('derives heat distribution widths only from topicCount', () => {
    const heat = tagHeatEntries(tags, 3, 'en-US')
    expect(heat.map((entry) => [entry.tag.slug, entry.widthPercent])).toEqual([
      ['sforum', 100],
      ['vue', 60],
      ['api', 20]
    ])
  })

  test('uses createdAt for recent tags', () => {
    expect(recentTagIndexTags(tags, 4, NOW, 'zh-CN').map((item) => item.slug))
      .toEqual(['中文标签', 'vue'])
  })

  test('does not classify future timestamps as this-week tags', () => {
    const future = tag({
      id: 6,
      slug: 'future',
      name: 'Future',
      topicCount: 2,
      createdAt: '2026-07-24T12:00:00Z'
    })
    expect(filterTagIndexTags([future], { filter: 'week', nowMs: NOW })).toEqual([])
    expect(recentTagIndexTags([future], 4, NOW, 'en-US')).toEqual([])
  })
})
