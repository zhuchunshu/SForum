import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

import {
  adminPageDefinitions,
  adminSidebarNavigation,
  type AdminNavigationFolderEntry
} from '../../app/config/adminModules'
import {
  createCategoryPayload,
  createDefaultForumSettings,
  createTagPayload,
  forumSettingsPartialPayload,
  forumSettingsPayload,
  normalizeForumSettings
} from '../../app/utils/admin/adminForum'

describe('admin forum helpers', () => {
  test('normalizes forum settings to recommended defaults', () => {
    expect(createDefaultForumSettings()).toMatchObject({
      defaultCategorySlug: 'general',
      tagCreationMode: 'controlled',
      tagPublicPages: true,
      tagMinPerTopic: 0,
      tagMaxPerTopic: 5,
      topicsPerPage: 20,
      commentsPerPage: 20,
      topicTitleMinRunes: 2,
      topicTitleMaxRunes: 100,
      commentMaxNestingDepth: 5,
      excerptRuneLimit: 180
    })

    expect(normalizeForumSettings({
      defaultCategorySlug: ' ',
      tagCreationMode: 'chaos',
      tagPublicPages: 'maybe',
      tagMaxPerTopic: '99',
      topicsPerPage: 0,
      commentsPerPage: 101,
      topicTitleMinRunes: 999,
      commentMaxNestingDepth: -1
    })).toEqual(createDefaultForumSettings())

    expect(forumSettingsPayload(normalizeForumSettings({
      topicsPerPage: '30',
      commentsPerPage: '40',
      topicTitleMaxRunes: '120',
      commentMaxNestingDepth: '6'
    }))).toMatchObject({
      topicsPerPage: 30,
      commentsPerPage: 40,
      topicTitleMaxRunes: 120,
      commentMaxNestingDepth: 6
    })
  })

  test('builds exact tab-scoped settings payloads', () => {
    const settings = createDefaultForumSettings()
    expect(forumSettingsPartialPayload(settings, ['topicTitleMinRunes', 'topicTitleMaxRunes'])).toEqual({
      topicTitleMinRunes: 2,
      topicTitleMaxRunes: 100
    })
    expect(forumSettingsPartialPayload(settings, ['tagMinPerTopic', 'tagMaxPerTopic'])).toEqual({
      tagMinPerTopic: 0,
      tagMaxPerTopic: 5
    })
    expect(forumSettingsPartialPayload(settings, ['defaultCategorySlug'])).toEqual({
      defaultCategorySlug: 'general'
    })
  })

  test('creates taxonomy payloads with visual field defaults', () => {
    expect(createCategoryPayload(2)).toEqual({
      groupId: 2,
      slug: '',
      name: '',
      description: '',
      icon: '',
      iconColor: '',
      visibility: 'public',
      position: 0,
      defaultSort: 'latest'
    })

    expect(createCategoryPayload(2, {
      slug: 'support',
      name: 'Support',
      icon: 'i-tabler-help',
      iconColor: '#0f766e'
    })).toMatchObject({
      slug: 'support',
      name: 'Support',
      icon: 'i-tabler-help',
      iconColor: '#0f766e'
    })

    expect(createTagPayload()).toEqual({
      slug: '',
      name: '',
      description: '',
      icon: '',
      iconColor: '',
      status: 'active'
    })

    expect(createTagPayload({
      slug: 'go',
      name: 'Go',
      icon: 'i-lucide-tag',
      iconColor: '#2563eb'
    })).toMatchObject({
      slug: 'go',
      name: 'Go',
      icon: 'i-lucide-tag',
      iconColor: '#2563eb'
    })
  })

  test('wires taxonomy visual controls into admin category and tag pages', () => {
    const categoriesPage = readFileSync(new URL('../../app/pages/admin/forum/categories.vue', import.meta.url), 'utf8')
    const tagsPage = readFileSync(new URL('../../app/pages/admin/forum/tags.vue', import.meta.url), 'utf8')
    const zhCN = JSON.parse(readFileSync(new URL('../../i18n/locales/zh-CN.json', import.meta.url), 'utf8'))
    const enUS = JSON.parse(readFileSync(new URL('../../i18n/locales/en-US.json', import.meta.url), 'utf8'))

    expect(categoriesPage).toContain('LazySFIconPicker')
    expect(categoriesPage).toContain('icon: categoryForm.icon.trim()')
    expect(categoriesPage).toContain('iconColor: categoryForm.iconColor.trim()')
    expect(categoriesPage).toContain('categoryPreviewIcon(category)')
    expect(categoriesPage).toContain('taxonomyPreviewColor(category.iconColor)')

    expect(tagsPage).toContain('LazySFIconPicker')
    expect(tagsPage).toContain('icon: form.icon.trim()')
    expect(tagsPage).toContain('iconColor: form.iconColor.trim()')
    expect(tagsPage).toContain('tagPreviewIcon(tag)')
    expect(tagsPage).toContain('taxonomyPreviewColor(tag.iconColor)')

    expect(zhCN.admin.forum.visual).toMatchObject({
      icon: '图标',
      iconColor: '图标颜色'
    })
    expect(enUS.admin.forum.visual).toMatchObject({
      icon: 'Icon',
      iconColor: 'Icon color'
    })
  })

  test('defines permission-aware forum admin pages', () => {
    const categories = adminPageDefinitions.find((page) => page.id === '/forum/categories')
    const tags = adminPageDefinitions.find((page) => page.id === '/forum/tags')
    const settings = adminPageDefinitions.find((page) => page.id === '/forum/settings')

    expect(categories?.requiredPermissions).toEqual(['category.manage'])
    expect(tags?.requiredPermissions).toEqual(['tag.manage'])
    expect(settings?.requiredPermissions).toEqual(['category.manage', 'tag.manage', 'forum.settings.manage', 'search.manage'])
    expect(settings?.permissionMode).toBe('any')
  })

  test('adds forum category, tag, settings, and content pages to the sidebar folder', () => {
    const forumFolder = adminSidebarNavigation
      .flat()
      .find((entry): entry is AdminNavigationFolderEntry => entry.type === 'folder' && entry.labelKey === 'admin.nav.forum')

    expect(forumFolder?.children.map((entry) => entry.type === 'page' ? entry.pageId : '')).toEqual([
      '/moderation',
      '/forum/categories',
      '/forum/tags',
      '/forum/settings',
      '/forum/content'
    ])
  })
})
