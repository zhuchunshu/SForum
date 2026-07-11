import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

import {
  adminPageDefinitions,
  adminSidebarNavigation,
  type AdminNavigationFolderEntry
} from '../app/config/adminModules'
import {
  createCategoryPayload,
  createDefaultForumSettings,
  createTagPayload,
  forumSettingsPayload,
  normalizeForumSettings
} from '../app/utils/adminForum'

describe('admin forum helpers', () => {
  test('normalizes forum settings to recommended defaults', () => {
    expect(createDefaultForumSettings()).toEqual({
      defaultCategorySlug: 'general',
      tagCreationMode: 'controlled',
      tagPublicPages: true,
      tagMaxPerTopic: 5,
      topicsPerPage: 20,
      commentsPerPage: 20
    })

    expect(normalizeForumSettings({
      defaultCategorySlug: ' ',
      tagCreationMode: 'chaos',
      tagPublicPages: 'maybe',
      tagMaxPerTopic: '99',
      topicsPerPage: 0,
      commentsPerPage: 101
    })).toEqual(createDefaultForumSettings())

    expect(forumSettingsPayload(normalizeForumSettings({
      topicsPerPage: '30',
      commentsPerPage: '40'
    }))).toMatchObject({ topicsPerPage: 30, commentsPerPage: 40 })
  })

  test('renders permission-aware pagination controls', () => {
    const settingsPage = readFileSync(new URL('../app/pages/admin/forum/settings.vue', import.meta.url), 'utf8')
    expect(settingsPage).toContain("can('settings.manage')")
    expect(settingsPage).toContain('v-model="form.topicsPerPage"')
    expect(settingsPage).toContain('v-model="form.commentsPerPage"')
    expect(settingsPage).toContain(':min="1"')
    expect(settingsPage).toContain(':max="100"')
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
    const categoriesPage = readFileSync(new URL('../app/pages/admin/forum/categories.vue', import.meta.url), 'utf8')
    const tagsPage = readFileSync(new URL('../app/pages/admin/forum/tags.vue', import.meta.url), 'utf8')
    const zhCN = JSON.parse(readFileSync(new URL('../i18n/locales/zh-CN.json', import.meta.url), 'utf8'))
    const enUS = JSON.parse(readFileSync(new URL('../i18n/locales/en-US.json', import.meta.url), 'utf8'))

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
    expect(settings?.requiredPermissions).toEqual(['category.manage', 'tag.manage', 'settings.manage'])
    expect(settings?.permissionMode).toBe('any')
  })

  test('adds forum category, tag, and settings pages to the sidebar folder', () => {
    const forumFolder = adminSidebarNavigation
      .flat()
      .find((entry): entry is AdminNavigationFolderEntry => entry.type === 'folder' && entry.labelKey === 'admin.nav.forum')

    expect(forumFolder?.children.map((entry) => entry.type === 'page' ? entry.pageId : '')).toEqual([
      '/moderation',
      '/forum/categories',
      '/forum/tags',
      '/forum/settings'
    ])
  })
})
