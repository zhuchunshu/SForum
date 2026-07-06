import { describe, expect, test } from 'bun:test'

import {
  adminPageDefinitions,
  adminSidebarNavigation,
  type AdminNavigationFolderEntry
} from '../app/config/adminModules'
import {
  createDefaultForumSettings,
  normalizeForumSettings
} from '../app/utils/adminForum'

describe('admin forum helpers', () => {
  test('normalizes forum settings to recommended defaults', () => {
    expect(createDefaultForumSettings()).toEqual({
      defaultCategorySlug: 'general',
      tagCreationMode: 'controlled',
      tagPublicPages: true,
      tagMaxPerTopic: 5
    })

    expect(normalizeForumSettings({
      defaultCategorySlug: ' ',
      tagCreationMode: 'chaos',
      tagPublicPages: 'maybe',
      tagMaxPerTopic: '99'
    })).toEqual(createDefaultForumSettings())
  })

  test('defines permission-aware forum admin pages', () => {
    const categories = adminPageDefinitions.find((page) => page.id === '/forum/categories')
    const tags = adminPageDefinitions.find((page) => page.id === '/forum/tags')
    const settings = adminPageDefinitions.find((page) => page.id === '/forum/settings')

    expect(categories?.requiredPermissions).toEqual(['category.manage'])
    expect(tags?.requiredPermissions).toEqual(['tag.manage'])
    expect(settings?.requiredPermissions).toEqual(['category.manage', 'tag.manage'])
    expect(settings?.permissionMode).toBe('any')
  })

  test('adds forum category, tag, and settings pages to the sidebar folder', () => {
    const forumFolder = adminSidebarNavigation
      .flat()
      .find((entry): entry is AdminNavigationFolderEntry => entry.type === 'folder' && entry.labelKey === 'admin.nav.forum')

    expect(forumFolder?.children.map((entry) => entry.type === 'page' ? entry.pageId : '')).toEqual([
      '/forum/categories',
      '/forum/tags',
      '/forum/settings'
    ])
  })
})
