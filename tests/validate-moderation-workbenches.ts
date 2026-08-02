import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const root = process.cwd()
const read = (path: string) => readFileSync(resolve(root, path), 'utf8')
const assert = (condition: unknown, message: string) => {
  if (!condition) throw new Error(message)
}

const permissions = read('apps/web/app/composables/identity/usePermissions.ts')
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
assert(navbar.includes('usePublicUserMenu'), 'Public navbar must consume the shared user menu')
const publicUserMenu = read('apps/web/app/composables/navigation/usePublicUserMenu.ts')
assert(publicUserMenu.includes('FORUM_PERMISSIONS.moderationReview'), 'Public moderator entry must require moderation.review')
assert(publicUserMenu.includes("localePath('/moderation')"), 'Public moderator entry must link to /moderation')

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

const workbenchPage = read('apps/web/app/components/moderation/SFModerationReviewPage.vue')
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

const composerShell = read('apps/web/app/pages/topics/new.vue')
assert(composerShell.includes('SFPageOutlet'), 'Topic composer route must use SFPageOutlet')
assert(composerShell.includes('SFTopicComposerPage'), 'Topic composer route must fail-closed to SFTopicComposerPage')
const composer = read('apps/web/app/components/forum/SFTopicComposerPage.vue')
assert(composer.includes("created.status === 'pending'"), 'Topic composer island must handle pending publication')
assert(composer.includes("submittedForReview"), 'Topic composer island must toast pending submissions')
assert(!composer.includes('/my/content-review'), 'Pending submissions must not route to removed /my/content-review')

const topicShell = read('apps/web/app/pages/t/[...path].vue')
assert(topicShell.includes('SFPageOutlet'), 'Topic show route must use SFPageOutlet')
assert(topicShell.includes('SFTopicShowPage'), 'Topic show route must fail-closed to SFTopicShowPage')
const topicPage = read('apps/web/app/components/forum/SFTopicShowPage.vue')
assert(topicPage.includes('useTopicCommentComposerDrawer'), 'Comment composer island must delegate drawer state')
const commentComposerDrawer = read('apps/web/app/composables/forum/useTopicCommentComposerDrawer.ts')
assert(commentComposerDrawer.includes('useTopicCommentSubmission'), 'Comment composer drawer must delegate comment submission')
const commentSubmission = read('apps/web/app/composables/forum/useTopicCommentSubmission.ts')
assert(commentSubmission.includes("created.status === 'pending'"), 'Comment submission must handle pending publication')
assert(commentSubmission.includes('replySubmittedForReview'), 'Comment submission must toast pending publication')

console.log('Moderation workbench validation passed')
