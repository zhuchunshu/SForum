import type { InjectionKey } from 'vue'

import type { SForumAdminHost } from './host'

// 宿主专用入口，插件公共 API 不导出注入键，避免跨扩展伪造宿主能力。
export const ADMIN_HOST_INJECTION_KEY: InjectionKey<SForumAdminHost> = Symbol('sforum.admin.host')
