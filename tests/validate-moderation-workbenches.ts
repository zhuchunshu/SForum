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

// 呈现所有权迁移后：路由壳只挂 outlet + 岛；业务逻辑在 Host body islands。
const workbenchShell = read('apps/web/app/pages/moderation/index.vue')
assert(workbenchShell.includes('SFPageOutlet'), 'Workbench route must use SFPageOutlet')
assert(workbenchShell.includes('page="moderation.review"'), 'Workbench route must declare moderation.review')
assert(workbenchShell.includes('SFModerationReviewPage'), 'Workbench route must fail-closed to SFModerationReviewPage')

const workbenchPage = read('apps/web/app/components/SFModerationReviewPage.vue')
assert(workbenchPage.includes("'pending'"), 'Workbench island must include the pending tab')
assert(workbenchPage.includes("'reports'"), 'Workbench island must include the reports tab')
assert(workbenchPage.includes("'history'"), 'Workbench island must include the history tab')
assert(workbenchPage.includes('ModerationQueueItem'), 'Workbench island must render scannable queue items')

const reviewMiddleware = read('apps/web/app/middleware/moderation-review.ts')
assert(reviewMiddleware.includes("can('moderation.review')"), 'Workbench middleware must require moderation.review')
const contextPanel = read('apps/web/app/components/moderation/ModerationContextPanel.vue')
for (const action of ['approve', 'reject', 'keep_and_close', 'hide_and_close', 'delete_and_close']) {
  assert(contextPanel.includes(action), `Context panel must expose ${action}`)
}

const authorShell = read('apps/web/app/pages/my/content-review.vue')
assert(authorShell.includes('SFPageOutlet'), 'Author review route must use SFPageOutlet')
assert(authorShell.includes('SFMyContentReviewPage'), 'Author review route must fail-closed to SFMyContentReviewPage')
const authorPage = read('apps/web/app/components/SFMyContentReviewPage.vue')
assert(authorPage.includes('listAuthorReviewItems'), 'Author island must use the authenticated review-status endpoint')

const composerShell = read('apps/web/app/pages/topics/new.vue')
assert(composerShell.includes('SFPageOutlet'), 'Topic composer route must use SFPageOutlet')
assert(composerShell.includes('SFTopicComposerPage'), 'Topic composer route must fail-closed to SFTopicComposerPage')
const composer = read('apps/web/app/components/SFTopicComposerPage.vue')
assert(composer.includes("created.status === 'pending'"), 'Topic composer island must handle pending publication')

const topicShell = read('apps/web/app/pages/t/[...path].vue')
assert(topicShell.includes('SFPageOutlet'), 'Topic show route must use SFPageOutlet')
assert(topicShell.includes('SFTopicShowPage'), 'Topic show route must fail-closed to SFTopicShowPage')
const topicPage = read('apps/web/app/components/SFTopicShowPage.vue')
assert(topicPage.includes('replySubmittedForReview'), 'Comment composer island must handle pending publication')

console.log('Moderation workbench validation passed')
