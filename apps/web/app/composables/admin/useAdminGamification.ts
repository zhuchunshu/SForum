import type {
  AdminGamificationApi,
  GamificationBadge,
  GamificationBadgePayload,
  GamificationGrant,
  GamificationLevel,
  GamificationLevelPayload
} from '~/utils/admin/adminGamification'
import { createAdminGamificationApi } from '~/utils/admin/adminGamification'
import type { AdminExtension } from '~/utils/admin/adminExtensions'

/** 插件 ID：未启用时管理面所有写操作都不可用。 */
export const GAMIFICATION_EXTENSION_ID = 'sforum.gamification'

export const ADMIN_GAMIFICATION_KEY = Symbol('admin-gamification') as InjectionKey<AdminGamificationState>

export type AdminGamificationState = ReturnType<typeof useAdminGamification>

/**
 * 页面壳是唯一的数据所有者：三个标签页通过注入共享同一份上下文，
 * 避免同一个 useAsyncData key 被多个组件持有而在卸载时被清理。
 */
export function useAdminGamification() {
  const { request } = useApiClient()
  const api = createAdminGamificationApi(request)
  const toast = useToast()
  const { t } = useI18n()

  const {
    data: badges,
    pending: badgesPending,
    error: badgesError,
    refresh: refreshBadges
  } = useAsyncData<GamificationBadge[]>('admin-gamification-badges', () => api.listBadges(), {
    default: (): GamificationBadge[] => [],
    server: false
  })

  const {
    data: levels,
    pending: levelsPending,
    error: levelsError,
    refresh: refreshLevels
  } = useAsyncData<GamificationLevel[]>('admin-gamification-levels', () => api.listLevels(), {
    default: (): GamificationLevel[] => [],
    server: false
  })

  // 插件启用状态：未启用时 Host 侧路由不存在，写操作必然失败。
  const { data: cachedExtensions } = useNuxtData<AdminExtension[]>('admin-extensions')
  const {
    data: extensions,
    pending: extensionsPending,
    refresh: refreshExtensions
  } = useAsyncData<AdminExtension[]>('admin-extensions', () => request<AdminExtension[]>('/admin/extensions'), {
    default: (): AdminExtension[] => [],
    server: false
  })

  const extension = computed(() => {
    const list = extensions.value?.length ? extensions.value : (cachedExtensions.value ?? [])
    return list.find(item => item.id === GAMIFICATION_EXTENSION_ID) ?? null
  })
  const pluginEnabled = computed(() => extension.value?.status === 'enabled')
  const pluginMissing = computed(() => !extension.value)
  // 扩展列表未就绪时无法判断启用状态；用于避免「未启用」告警闪烁。
  const pluginStatusResolved = computed(() => !extensionsPending.value || Boolean(cachedExtensions.value?.length))

  const saving = ref(false)
  const error = computed(() => badgesError.value ?? levelsError.value ?? null)
  const pending = computed(() => badgesPending.value || levelsPending.value)

  async function refreshAll() {
    await Promise.all([refreshBadges(), refreshLevels(), refreshExtensions()])
  }

  /** 统一包装：失败时弹 Toast，成功时刷新列表。 */
  async function runAction(options: {
    action: () => Promise<unknown>
    successTitle: string
    failureTitle: string
    refresh?: boolean
  }) {
    saving.value = true
    try {
      await options.action()
      toast.add({ color: 'success', icon: 'i-lucide-check', title: options.successTitle, duration: 10000 })
      if (options.refresh !== false) {
        await refreshAll()
      }
      return true
    } catch (err) {
      toast.add({
        color: 'error',
        icon: 'i-lucide-triangle-alert',
        title: options.failureTitle,
        description: describeApiError(err, t),
        duration: 10000
      })
      return false
    } finally {
      saving.value = false
    }
  }

  async function saveBadge(payload: GamificationBadgePayload, slug?: string) {
    return runAction({
      action: () => (slug ? api.updateBadge(slug, payload) : api.createBadge(payload)),
      successTitle: t('admin.gamification.badges.saved'),
      failureTitle: t('admin.gamification.badges.saveFailed')
    })
  }

  async function disableBadge(slug: string) {
    return runAction({
      action: () => api.disableBadge(slug),
      successTitle: t('admin.gamification.badges.disabled'),
      failureTitle: t('admin.gamification.badges.disableFailed')
    })
  }

  async function saveLevel(payload: GamificationLevelPayload) {
    return runAction({
      action: () => api.upsertLevel(payload),
      successTitle: t('admin.gamification.levels.saved'),
      failureTitle: t('admin.gamification.levels.saveFailed')
    })
  }

  async function seedDefaults() {
    return runAction({
      action: () => api.seedDefaults(),
      successTitle: t('admin.gamification.badges.seeded'),
      failureTitle: t('admin.gamification.badges.seedFailed')
    })
  }

  async function grantBadge(userId: number, badgeSlug: string) {
    return runAction({
      action: () => api.grantBadge(userId, badgeSlug),
      successTitle: t('admin.gamification.grants.granted'),
      failureTitle: t('admin.gamification.grants.failed')
    })
  }

  async function revokeBadge(userId: number, badgeSlug: string) {
    return runAction({
      action: () => api.revokeBadge(userId, badgeSlug),
      successTitle: t('admin.gamification.grants.revoked'),
      failureTitle: t('admin.gamification.grants.failed')
    })
  }

  async function backfill(users: Array<{ userId: number, topicCount?: number, commentCount?: number }>) {
    return runAction({
      action: () => api.backfill({ users }),
      successTitle: t('admin.gamification.backfill.done'),
      failureTitle: t('admin.gamification.backfill.failed')
    })
  }

  /** 授予日志：按用户 id 查询，不做全量拉取。 */
  async function loadUserBadges(userId: number): Promise<GamificationGrant[]> {
    try {
      return await api.listUserBadges(userId)
    } catch {
      return []
    }
  }

  return {
    api: api as AdminGamificationApi,
    badges,
    levels,
    pending,
    error,
    saving,
    extension,
    pluginEnabled,
    pluginMissing,
    pluginStatusResolved,
    refreshAll,
    refreshBadges,
    refreshLevels,
    saveBadge,
    disableBadge,
    saveLevel,
    seedDefaults,
    grantBadge,
    revokeBadge,
    backfill,
    loadUserBadges
  }
}

function describeApiError(error: unknown, t: (key: string) => string): string {
  const data = (error as { data?: { reason?: string, message?: string } } | null)?.data
  if (data?.reason === 'gamification.badge_not_manual') return t('admin.gamification.errors.notManual')
  if (data?.reason === 'gamification.grant_payload_invalid') return t('admin.gamification.errors.payloadInvalid')
  if (data?.reason === 'gamification.badge_invalid') return t('admin.gamification.errors.badgeInvalid')
  if (data?.reason === 'gamification.grant_failed') return t('admin.gamification.errors.grantFailed')
  return data?.message || (error as Error | undefined)?.message || t('admin.gamification.errors.unknown')
}
