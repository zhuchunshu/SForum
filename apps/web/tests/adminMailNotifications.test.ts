import { describe, expect, test } from 'bun:test'

const page = await Bun.file(new URL('../app/pages/admin/settings/mail.vue', import.meta.url)).text()
const modules = await Bun.file(new URL('../app/config/adminModules.ts', import.meta.url)).text()

describe('mail and notification admin center', () => {
  test('is visible in the System navigation with settings permission', () => {
    expect(modules).toContain("pageId: '/settings/mail'")
    expect(modules).toContain("requiredPermissions: ['settings.manage']")
  })

  test('exposes four views and provider-neutral test actions', () => {
    expect(page).toContain("value: 'overview'")
    expect(page).toContain("value: 'mail'")
    expect(page).toContain("value: 'notifications'")
    expect(page).toContain("value: 'deliveries'")
    expect(page).toContain("'/admin/mail/policy/restore'")
    expect(page).toContain("'/admin/notifications/test'")
    expect(page).toContain("recipient: testRecipient.value")
  })
})
