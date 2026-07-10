import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const root = process.cwd()
const read = (path: string) => readFileSync(resolve(root, path), 'utf8')
const assert = (condition: unknown, message: string) => {
  if (!condition) throw new Error(message)
}

const permissions = read('apps/web/app/composables/usePermissions.ts')
assert(permissions.includes("moderationManage: 'moderation.manage'"), 'Missing moderation.manage frontend permission')
assert(permissions.includes("moderationReview: 'moderation.review'"), 'Missing moderation.review frontend permission')

const modules = read('apps/web/app/config/adminModules.ts')
assert(modules.includes("requiredPermissions: ['moderation.manage']"), 'Admin moderation must require moderation.manage')
assert(modules.includes("{ type: 'page', pageId: '/moderation' }"), 'Admin moderation must have a sidebar entry')

const navbar = read('extensions/builtin/themes/sforum-default/layer/app/components/SFNavbar.vue')
assert(navbar.includes('FORUM_PERMISSIONS.moderationReview'), 'Public moderator entry must require moderation.review')
assert(navbar.includes("localePath('/moderation')"), 'Public moderator entry must link to /moderation')

const adminPage = read('apps/web/app/pages/admin/moderation.vue')
assert(adminPage.includes('ModerationSettingsForm'), 'Admin moderation page must render the settings form')
assert(adminPage.includes('ModerationDecisionTable'), 'Admin moderation page must render audit history')
const settingsForm = read('apps/web/app/components/moderation/ModerationSettingsForm.vue')
assert(settingsForm.includes("value: 'rules'"), 'Settings form must expose rules mode')
assert(settingsForm.includes('reviewNewUsers'), 'Settings form must expose the new-user rule')
assert(settingsForm.includes('reviewExternalLinks'), 'Settings form must expose the external-link rule')
assert(settingsForm.includes('resetSettings'), 'Settings form must support restoring defaults')

console.log('Moderation workbench validation passed')
