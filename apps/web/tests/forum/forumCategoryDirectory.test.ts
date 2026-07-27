import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

import {
  activeCategoryDirectoryCategories,
  buildCategoryDirectoryDisplayGroups,
  categoryDirectoryDistribution,
  categoryDirectoryGroupKey,
  findCategoryDirectoryGroup,
  sortCategoryDirectoryCategories,
  summarizeCategoryDirectory,
  summarizeCategoryDirectoryDisplay,
  visibleCategoryDirectoryGroups
} from '../../app/utils/forum/forumCategoryDirectory'
import type { ForumCategory, ForumCategoryGroup } from '../../app/utils/forum/forumTaxonomy'

function category(input: Partial<ForumCategory> & Pick<ForumCategory, 'id' | 'slug' | 'name' | 'position'>): ForumCategory {
  return {
    id: input.id,
    groupId: input.groupId ?? 1,
    groupSlug: input.groupSlug ?? 'community',
    groupName: input.groupName ?? 'Community',
    slug: input.slug,
    name: input.name,
    description: input.description ?? '',
    icon: input.icon ?? '',
    iconColor: input.iconColor ?? '',
    visibility: input.visibility ?? 'public',
    position: input.position,
    defaultSort: input.defaultSort ?? 'latest',
    topicCount: input.topicCount ?? 0,
    commentCount: input.commentCount ?? 0,
    createdAt: input.createdAt ?? '2026-07-23T00:00:00Z',
    updatedAt: input.updatedAt ?? '2026-07-23T00:00:00Z'
  }
}

function group(input: Partial<ForumCategoryGroup> & Pick<ForumCategoryGroup, 'id' | 'slug' | 'name'>): ForumCategoryGroup {
  return {
    id: input.id,
    slug: input.slug,
    name: input.name,
    description: input.description ?? '',
    visibility: input.visibility ?? 'public',
    position: input.position ?? input.id,
    categories: input.categories ?? [],
    createdAt: input.createdAt ?? '2026-07-23T00:00:00Z',
    updatedAt: input.updatedAt ?? '2026-07-23T00:00:00Z'
  }
}

const fixtures = () => [
  group({
    id: 2,
    slug: 'support',
    name: 'Support',
    categories: [
      category({ id: 20, groupId: 2, groupSlug: 'support', groupName: 'Support', slug: 'bugs', name: 'Bug reports', description: 'Report reproducible bugs', position: 2, topicCount: 5, commentCount: 30 }),
      category({ id: 21, groupId: 2, groupSlug: 'support', groupName: 'Support', slug: 'appeals', name: 'Appeals', position: 1, topicCount: 0, commentCount: 0 }),
      category({ id: 22, groupId: 2, groupSlug: 'support', groupName: 'Support', slug: 'hidden', name: 'Hidden', position: 3, visibility: 'hidden', topicCount: 99 })
    ]
  }),
  group({
    id: 3,
    slug: 'empty',
    name: 'Empty public group',
    categories: []
  }),
  group({
    id: 4,
    slug: 'private',
    name: 'Private',
    visibility: 'hidden',
    categories: [
      category({ id: 40, slug: 'secret', name: 'Secret', position: 1, topicCount: 100 })
    ]
  }),
  group({
    id: 1,
    slug: 'community',
    name: 'Community',
    categories: [
      category({ id: 10, slug: 'general', name: 'General', description: 'Open discussion', position: 1, topicCount: 9, commentCount: 12 }),
      category({ id: 11, slug: 'code', name: 'Code', description: 'Implementation notes', position: 2, topicCount: 1, commentCount: 1 })
    ]
  })
]

describe('category directory helpers', () => {
  test('maps real ForumCategoryGroup and ForumCategory DTO fields without fabricating hidden data', () => {
    const visible = visibleCategoryDirectoryGroups(fixtures())
    expect(visible.map((item) => item.slug)).toEqual(['support', 'empty', 'community'])
    expect(visible[0]?.categories.map((item) => item.slug)).toEqual(['bugs', 'appeals'])
    expect(visible.flatMap((item) => item.categories).some((item) => item.slug === 'hidden')).toBe(false)
    expect(visible.flatMap((item) => item.categories).some((item) => item.slug === 'secret')).toBe(false)
    expect(visible[0]?.categories[0]).toMatchObject({
      id: 20,
      groupId: 2,
      groupSlug: 'support',
      groupName: 'Support',
      slug: 'bugs',
      name: 'Bug reports',
      description: 'Report reproducible bugs',
      icon: '',
      iconColor: '',
      visibility: 'public',
      position: 2,
      defaultSort: 'latest',
      topicCount: 5,
      commentCount: 30
    })
  })

  test('preserves empty public groups and records current backend ungrouped category semantics', () => {
    const visible = visibleCategoryDirectoryGroups(fixtures())
    expect(visible.find((item) => item.slug === 'empty')?.categories).toEqual([])

    const migration = readFileSync(new URL('../../../api/database/migrations/202607070003_forum_taxonomy.sql', import.meta.url), 'utf8')
    expect(migration).toContain('ALTER COLUMN group_id SET NOT NULL')
    expect(migration).toContain("WHERE group_id IS NULL")
  })

  test('sorts by default position, existing active topic count, and locale name with stable fallback', () => {
    const categories = visibleCategoryDirectoryGroups(fixtures())[0]!.categories
    expect(sortCategoryDirectoryCategories(categories, 'default').map((item) => item.slug)).toEqual(['appeals', 'bugs'])
    expect(sortCategoryDirectoryCategories(categories, 'active').map((item) => item.slug)).toEqual(['bugs', 'appeals'])
    expect(sortCategoryDirectoryCategories(categories, 'name', 'en-US').map((item) => item.slug)).toEqual(['appeals', 'bugs'])
  })

  test('focuses a group by stable id, returns all groups, and combines focus with sorting', () => {
    const visible = visibleCategoryDirectoryGroups(fixtures())
    const focusedKey = categoryDirectoryGroupKey(visible[0]!)
    expect(focusedKey).toBe('2')
    expect(findCategoryDirectoryGroup(visible, focusedKey)?.slug).toBe('support')

    const focused = buildCategoryDirectoryDisplayGroups(visible, { sort: 'active', focusedGroupKey: focusedKey })
    expect(focused.map((item) => item.slug)).toEqual(['support'])
    expect(focused[0]?.categories.map((item) => item.slug)).toEqual(['bugs', 'appeals'])

    const all = buildCategoryDirectoryDisplayGroups(visible, { sort: 'default' })
    expect(all.map((item) => item.slug)).toEqual(['support', 'empty', 'community'])
  })

  test('filters only the returned directory by category name, description, or slug', () => {
    const visible = visibleCategoryDirectoryGroups(fixtures())
    const byDescription = buildCategoryDirectoryDisplayGroups(visible, { sort: 'default', query: 'implementation' })
    expect(byDescription.map((item) => [item.slug, item.categories.map((category) => category.slug)])).toEqual([
      ['community', ['code']]
    ])

    const bySlug = buildCategoryDirectoryDisplayGroups(visible, { sort: 'default', focusedGroupKey: '2', query: 'appeals' })
    expect(bySlug.map((item) => [item.slug, item.categories.map((category) => category.slug)])).toEqual([
      ['support', ['appeals']]
    ])

    const empty = buildCategoryDirectoryDisplayGroups(visible, { sort: 'default', query: 'does-not-exist' })
    expect(empty).toEqual([])
  })

  test('derives right-rail statistics from visible DTO counts only', () => {
    const visible = visibleCategoryDirectoryGroups(fixtures())
    expect(summarizeCategoryDirectory(visible)).toEqual({
      groupCount: 3,
      categoryCount: 4,
      topicCount: 15,
      commentCount: 43
    })

    const display = buildCategoryDirectoryDisplayGroups(visible, { sort: 'default', focusedGroupKey: '2' })
    expect(summarizeCategoryDirectoryDisplay(display)).toEqual({
      groupCount: 1,
      categoryCount: 2,
      topicCount: 5,
      commentCount: 30
    })
    expect(categoryDirectoryDistribution(visible).map((item) => [item.group.slug, item.count, item.percent])).toEqual([
      ['support', 2, 100],
      ['empty', 0, 0],
      ['community', 2, 100]
    ])
    expect(activeCategoryDirectoryCategories(visible, 2).map((item) => item.slug)).toEqual(['general', 'bugs'])
  })
})
