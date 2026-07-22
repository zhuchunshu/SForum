/**
 * 内置角色模板权限包，与 apps/api/app/Models/Identity/seeds.go 的 SeedRoleTemplates 对齐。
 * 后台「应用模板」在创建自定义组时预填权限；系统模板角色本身由 migration seed。
 */
export type RoleTemplateDefinition = {
  key: string
  /** i18n：admin.roleCatalog.<key>.alias */
  aliasKey: string
  /** i18n：admin.roleCatalog.<key>.description */
  descriptionKey: string
  permissionKeys: readonly string[]
}

export const ROLE_TEMPLATE_DEFINITIONS = [
  {
    key: 'moderator',
    aliasKey: 'admin.roleCatalog.moderator.alias',
    descriptionKey: 'admin.roleCatalog.moderator.description',
    permissionKeys: [
      'admin.access',
      'moderation.review',
      'moderation.view_ip',
      'topic.lock',
      'topic.pin',
      'topic.edit_any',
      'topic.revision.view_any',
      'topic.delete_any',
      'post.edit_any',
      'post.revision.view_any',
      'post.delete_any',
      'user.ban'
    ]
  },
  {
    key: 'operator',
    aliasKey: 'admin.roleCatalog.operator.alias',
    descriptionKey: 'admin.roleCatalog.operator.description',
    permissionKeys: [
      'admin.access',
      'user.view',
      'user.manage',
      'settings.site.manage',
      'settings.mail.manage',
      'settings.avatar.manage',
      'settings.appearance.manage',
      'forum.settings.manage',
      'seo.manage',
      'category.manage',
      'tag.manage',
      'attachment.manage',
      'attachment.settings.manage'
    ]
  },
  {
    key: 'tech_admin',
    aliasKey: 'admin.roleCatalog.tech_admin.alias',
    descriptionKey: 'admin.roleCatalog.tech_admin.description',
    permissionKeys: [
      'admin.access',
      'extension.view',
      'extension.plugin.manage',
      'extension.theme.manage',
      'jobs.view',
      'jobs.manage',
      'search.manage',
      'database.manage',
      'attachment.settings.manage'
    ]
  }
] as const satisfies readonly RoleTemplateDefinition[]

export function findRoleTemplate(key: string): RoleTemplateDefinition | undefined {
  return ROLE_TEMPLATE_DEFINITIONS.find(template => template.key === key)
}
