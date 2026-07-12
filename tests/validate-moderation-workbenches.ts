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
for (const locale of ['zh-CN', 'en-US']) {
  const messages = JSON.parse(read(`apps/web/i18n/locales/${locale}.json`))
  assert(typeof messages.admin?.nav?.moderation === 'string', `${locale} must define admin.nav.moderation`)
}

const navbar = read('apps/web/app/components/SFNavbar.vue')
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

const workbenchPage = read('apps/web/app/pages/moderation/index.vue')
assert(workbenchPage.includes("'pending'"), 'Workbench must include the pending tab')
assert(workbenchPage.includes("'reports'"), 'Workbench must include the reports tab')
assert(workbenchPage.includes("'history'"), 'Workbench must include the history tab')
assert(workbenchPage.includes('ModerationQueueItem'), 'Workbench must render scannable queue items')
const reviewMiddleware = read('apps/web/app/middleware/moderation-review.ts')
assert(reviewMiddleware.includes("can('moderation.review')"), 'Workbench middleware must require moderation.review')
const contextPanel = read('apps/web/app/components/moderation/ModerationContextPanel.vue')
for (const action of ['approve', 'reject', 'keep_and_close', 'hide_and_close', 'delete_and_close']) {
  assert(contextPanel.includes(action), `Context panel must expose ${action}`)
}
const authorPage = read('apps/web/app/pages/my/content-review.vue')
assert(authorPage.includes('listAuthorReviewItems'), 'Author page must use the authenticated review-status endpoint')
const composer = read('apps/web/app/pages/topics/new.vue')
assert(composer.includes("created.status === 'pending'"), 'Topic composer must handle pending publication')
const topicPage = read('apps/web/app/pages/t/[...path].vue')
assert(topicPage.includes('replySubmittedForReview'), 'Comment composer must handle pending publication')

console.log('Moderation workbench validation passed')
