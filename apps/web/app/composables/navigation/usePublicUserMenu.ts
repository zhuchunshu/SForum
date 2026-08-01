import { FORUM_PERMISSIONS, usePermissions } from '~/composables/identity/usePermissions'
import { useAuthSession } from '~/composables/identity/useAuthSession'
import { apiErrorMessage, useApiClient } from '~/composables/useApiClient'
import { normalizeEnabledOption, useWebOptions } from '~/composables/useWebOptions'

export type PublicUserMenuEntry = {
  key: string
  label: string
  description?: string
  icon: string
  to?: string
  tone?: 'danger'
  disabled?: boolean
  keepOpen?: boolean
  onSelect?: () => void
}

type PublicUserDropdownItem = {
  label: string
  description?: string
  icon?: string
  to?: string
  type?: 'label'
  color?: 'error'
  disabled?: boolean
  onSelect?: (event: Event) => void
}

export function usePublicUserMenu() {
  const { t } = useI18n()
  const localePath = useLocalePath()
  const router = useRouter()
  const { user, status, refresh } = useAuthSession()
  const { can } = usePermissions()
  const { request } = useApiClient()
  const { webOption } = useWebOptions()
  const toast = useToast()
  const resendingVerification = useState<boolean>('public-user-menu:resending-verification', () => false)

  const displayName = computed(() => user.value?.displayName || user.value?.username || '')
  const emailVerificationRequired = computed(() => normalizeEnabledOption(
    webOption('identity.registration.require_email_verification', 'off'),
    false
  ))
  const needsEmailVerification = computed(() => Boolean(
    user.value && emailVerificationRequired.value && !user.value.emailVerified
  ))
  const canReviewContent = computed(() => can(FORUM_PERMISSIONS.moderationReview))

  const menuGroups = computed<PublicUserMenuEntry[][]>(() => {
    if (!user.value) return []

    return [
      ...(needsEmailVerification.value
        ? [[{
            key: 'resend-email-verification',
            label: resendingVerification.value
              ? t('auth.emailVerificationSending')
              : t('auth.emailVerificationResend'),
            description: t('auth.emailVerificationRequiredHint'),
            icon: 'i-lucide-mail-warning',
            disabled: resendingVerification.value,
            keepOpen: true,
            onSelect: () => {
              void resendEmailVerification()
            }
          }]]
        : []),
      [
        {
          key: 'profile',
          label: t('nav.myProfile'),
          icon: 'i-lucide-user',
          to: localePath(`/u/${user.value.username}`)
        },
        {
          key: 'profile-settings',
          label: t('nav.profileSettings'),
          icon: 'i-lucide-settings',
          to: localePath('/settings/profile')
        },
        ...(canReviewContent.value
          ? [{
              key: 'moderation',
              label: t('nav.moderationWorkbench'),
              icon: 'i-lucide-shield-check',
              to: localePath('/moderation')
            }]
          : [])
      ],
      [{
        key: 'logout',
        label: t('nav.logout'),
        icon: 'i-lucide-log-out',
        tone: 'danger',
        onSelect: () => {
          void logout()
        }
      }]
    ]
  })

  const userMenuItems = computed<PublicUserDropdownItem[][]>(() => {
    if (!user.value) return []

    return [
      [{
        label: displayName.value,
        description: `@${user.value.username}`,
        type: 'label'
      }],
      ...menuGroups.value.map(group => group.map(entry => ({
        label: entry.label,
        ...(entry.description ? { description: entry.description } : {}),
        icon: entry.icon,
        ...(entry.to ? { to: entry.to } : {}),
        ...(entry.tone === 'danger' ? { color: 'error' as const } : {}),
        ...(entry.disabled ? { disabled: true } : {}),
        ...(entry.onSelect
          ? {
              onSelect: (event: Event) => {
                if (entry.keepOpen) event.preventDefault()
                entry.onSelect?.()
              }
            }
          : {})
      })))
    ]
  })

  async function resendEmailVerification() {
    if (resendingVerification.value) return
    resendingVerification.value = true
    try {
      await request<{ sent: boolean }>('/auth/email-verification/request', { method: 'POST' })
      toast.add({
        color: 'success',
        icon: 'i-lucide-mail-check',
        title: t('auth.emailVerificationSent'),
        duration: 10000
      })
    } catch (error) {
      toast.add({
        color: 'error',
        icon: 'i-lucide-circle-alert',
        title: apiErrorMessage(error) || t('auth.emailVerificationSendFailed'),
        duration: 0
      })
    } finally {
      resendingVerification.value = false
    }
  }

  async function logout() {
    try {
      await request('/auth/logout', { method: 'POST' })
    } catch {
      // 服务端退出失败时仍刷新会话，以服务端实际状态为准。
    }

    await refresh()
    await router.push(localePath('/login'))
  }

  return {
    user,
    status,
    displayName,
    needsEmailVerification,
    menuGroups,
    userMenuItems
  }
}
