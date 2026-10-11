import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

import {
  adminPageDefinitions,
  adminSidebarNavigation,
  shouldShowAdminPageInNav,
  type AdminNavigationEntry,
  type AdminNavigationFolderEntry
} from '../../app/config/adminModules'

const gamificationPermission = 'sforum.gamification.manage'
const bothOff = { professionalMode: false, operationsMode: false }
const allowAll = () => true
const allowNone = () => false

function findFolder(labelKey: string) {
  return adminSidebarNavigation
    .flat()
    .find((entry): entry is AdminNavigationFolderEntry => (
      entry.type === 'folder' && entry.labelKey === labelKey
    ))
}

function pageChildren(labelKey: string) {
  return (findFolder(labelKey)?.children || []).filter(entry => entry.type === 'page')
}

function collectPageIds(entries: readonly AdminNavigationEntry[]): string[] {
  return entries.flatMap((entry) => {
    if (entry.type === 'page') {
      return [entry.pageId]
    }
    return entry.type === 'folder' ? collectPageIds(entry.children) : []
  })
}

function navLabels(locale: 'zh-CN' | 'en-US') {
  const body = readFileSync(new URL(`../../i18n/locales/${locale}.json`, import.meta.url), 'utf8')
  return (JSON.parse(body) as { admin: { nav: Record<string, string> } }).admin.nav
}

describe('admin gamification navigation entry', () => {
  test('registers the page definition behind the plugin permission', () => {
    const page = adminPageDefinitions.find(item => item.id === '/gamification')

    expect(page).toBeTruthy()
    expect(page?.componentName).toBe('AdminGamification')
    expect(page?.requiredPermissions).toEqual([gamificationPermission])
    // 成长值运营是插件承载的产品能力，不属于高级/运维工具，默认应在侧栏可见。
    expect(page?.professionalMode).toBeFalsy()
    expect(page?.operationsMode).toBeFalsy()
  })

  test('reaches the page from the sidebar through exactly one entry', () => {
    const entries = adminSidebarNavigation
      .flatMap(collectPageIds)
      .filter(pageId => pageId === '/gamification')

    expect(entries).toHaveLength(1)

    const engagement = findFolder('admin.nav.engagement')
    expect(engagement).toBeTruthy()
    expect(engagement?.icon).toBe('i-lucide-medal')
    expect(pageChildren('admin.nav.engagement').map(entry => entry.pageId)).toEqual(['/gamification'])
  })

  test('shows the entry once the permission is granted and hides it otherwise', () => {
    const page = adminPageDefinitions.find(item => item.id === '/gamification')!

    expect(shouldShowAdminPageInNav(page, allowAll, bothOff)).toBe(true)
    expect(shouldShowAdminPageInNav(page, allowNone, bothOff)).toBe(false)
    expect(shouldShowAdminPageInNav(page, () => false, { professionalMode: true, operationsMode: true })).toBe(false)
  })

  test('leaves the engagement folder empty when no child page is visible', () => {
    // 侧栏按同一谓词过滤子项，空文件夹整体不渲染；权限被拒时不应留下空分组。
    const visible = pageChildren('admin.nav.engagement').filter((entry) => {
      const page = adminPageDefinitions.find(item => item.id === entry.pageId)!
      return shouldShowAdminPageInNav(page, allowNone, bothOff)
    })

    expect(visible).toHaveLength(0)
  })

  test('localizes the folder and page labels in both locales', () => {
    const folderKeys = ['engagement'] as const
    const pageKeys = ['gamification'] as const

    for (const locale of ['zh-CN', 'en-US'] as const) {
      const nav = navLabels(locale)

      for (const key of [...folderKeys, ...pageKeys]) {
        expect(nav[key]?.trim()).toBeTruthy()
      }

      expect(nav.engagement).not.toBe(nav.gamification)
    }
  })
})
