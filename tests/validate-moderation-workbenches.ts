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

console.log('Moderation workbench validation passed')
