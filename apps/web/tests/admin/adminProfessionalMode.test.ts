import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

import {
  adminPageDefinitions,
  adminSidebarNavigation,
  shouldShowAdminPageInNav,
  type AdminNavigationFolderEntry
} from '../../app/config/adminModules'

const alwaysShowExtensionPageIds = ['/extensions', '/extensions/plugins', '/extensions/themes'] as const
const operationsPageIds = ['/database', '/jobs', '/schedules', '/webhooks'] as const
const systemProfessionalPageIds = ['/settings/features', '/entity-meta'] as const

const bothOff = { professionalMode: false, operationsMode: false }
const proOnly = { professionalMode: true, operationsMode: false }
const opsOnly = { professionalMode: false, operationsMode: true }
const bothOn = { professionalMode: true, operationsMode: true }

describe('admin advanced settings navigation', () => {
  test('marks extension advanced pages as professional-mode only', () => {
    const professionalIds = adminPageDefinitions
      .filter(page => page.professionalMode)
      .map(page => page.id)

    expect(professionalIds).toContain('/extensions/pages')
    expect(professionalIds).toContain('/extensions/route-inspector')
    expect(professionalIds).toContain('/extensions/settings')
    expect(professionalIds).toContain('/extensions/events')
    expect(professionalIds).toContain('/extensions/contributions')

    for (const pageId of alwaysShowExtensionPageIds) {
      const page = adminPageDefinitions.find(item => item.id === pageId)
      expect(page?.professionalMode).toBeFalsy()
    }
  })

  test('marks specialized system pages as professional-mode only', () => {
    for (const pageId of systemProfessionalPageIds) {
      const page = adminPageDefinitions.find(item => item.id === pageId)
      expect(page?.professionalMode).toBe(true)
      expect(page?.operationsMode).toBeFalsy()
    }
  })

  test('marks operations pages as operations-mode only', () => {
    for (const pageId of operationsPageIds) {
      const page = adminPageDefinitions.find(item => item.id === pageId)
      expect(page?.operationsMode).toBe(true)
      expect(page?.professionalMode).toBeFalsy()
    }
  })

  test('hides professional pages from nav when mode is off and shows them when on', () => {
    const allowAll = () => true
    const pagesPage = adminPageDefinitions.find(item => item.id === '/extensions/pages')!
    const overviewPage = adminPageDefinitions.find(item => item.id === '/extensions')!

    expect(shouldShowAdminPageInNav(pagesPage, allowAll, bothOff)).toBe(false)
    expect(shouldShowAdminPageInNav(pagesPage, allowAll, proOnly)).toBe(true)
    expect(shouldShowAdminPageInNav(overviewPage, allowAll, bothOff)).toBe(true)
    expect(shouldShowAdminPageInNav(overviewPage, allowAll, bothOn)).toBe(true)

    for (const pageId of systemProfessionalPageIds) {
      const page = adminPageDefinitions.find(item => item.id === pageId)!
      expect(shouldShowAdminPageInNav(page, allowAll, bothOff)).toBe(false)
      expect(shouldShowAdminPageInNav(page, allowAll, proOnly)).toBe(true)
    }
  })

  test('hides operations pages from nav when operations mode is off', () => {
    const allowAll = () => true
    const jobsPage = adminPageDefinitions.find(item => item.id === '/jobs')!

    expect(shouldShowAdminPageInNav(jobsPage, allowAll, bothOff)).toBe(false)
    expect(shouldShowAdminPageInNav(jobsPage, allowAll, proOnly)).toBe(false)
    expect(shouldShowAdminPageInNav(jobsPage, allowAll, opsOnly)).toBe(true)
    expect(shouldShowAdminPageInNav(jobsPage, allowAll, bothOn)).toBe(true)
  })

  test('still respects permissions when advanced modes are enabled', () => {
    const pagesPage = adminPageDefinitions.find(item => item.id === '/extensions/pages')!
    const jobsPage = adminPageDefinitions.find(item => item.id === '/jobs')!
    expect(shouldShowAdminPageInNav(pagesPage, () => false, bothOn)).toBe(false)
    expect(shouldShowAdminPageInNav(jobsPage, () => false, bothOn)).toBe(false)
  })

  test('extension and operations folders still register their children', () => {
    const extensions = adminSidebarNavigation
      .flat()
      .find((entry): entry is AdminNavigationFolderEntry => (
        entry.type === 'folder' && entry.labelKey === 'admin.nav.extensions'
      ))
    const operations = adminSidebarNavigation
      .flat()
      .find((entry): entry is AdminNavigationFolderEntry => (
        entry.type === 'folder' && entry.labelKey === 'admin.nav.operations'
      ))

    expect(extensions).toBeTruthy()
    expect(operations).toBeTruthy()

    const extensionChildIds = (extensions?.children || [])
      .filter(entry => entry.type === 'page')
      .map(entry => entry.pageId)
    const operationsChildIds = (operations?.children || [])
      .filter(entry => entry.type === 'page')
      .map(entry => entry.pageId)

    for (const pageId of alwaysShowExtensionPageIds) {
      expect(extensionChildIds).toContain(pageId)
    }
    expect(extensionChildIds).toContain('/extensions/pages')
    for (const pageId of operationsPageIds) {
      expect(operationsChildIds).toContain(pageId)
    }
  })

  test('admin shell advanced settings modal has professional and operations switches defaulting off', () => {
    const layout = readFileSync(new URL('../../app/layouts/admin.vue', import.meta.url), 'utf8')
    const composable = readFileSync(new URL('../../app/composables/admin/useAdminAdvancedSettings.ts', import.meta.url), 'utf8')

    expect(layout).toContain('advancedSettingsOpen')
    expect(layout).toContain('useAdminAdvancedSettings')
    expect(layout).toContain('admin.shell.advancedSettings.professionalMode')
    expect(layout).toContain('admin.shell.advancedSettings.operationsMode')
    expect(layout).toContain('v-model="professionalMode"')
    expect(layout).toContain('v-model="operationsMode"')
    expect(layout).toContain("operationsMode ? 'ops' : 'noops'")
    expect(composable).toContain("default: () => '0'")
    expect(composable).toContain('sforum-admin-professional-mode')
    expect(composable).toContain('sforum-admin-operations-mode')
  })
})
