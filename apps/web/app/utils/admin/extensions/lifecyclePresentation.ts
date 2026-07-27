import type { AdminExtension } from '~/utils/admin/adminExtensions'

export type ExtensionLifecycleAction = 'enable' | 'disable' | 'restart' | 'upgrade' | 'rollback'

export function executableTrustPath(
  item: AdminExtension,
  action: 'enable' | 'restart',
  challenge = false
) {
  const target = action === 'restart' && item.stagedVersion ? '?target=staged' : ''
  return `/admin/extensions/${item.id}/trust${challenge ? '/challenge' : ''}${target}`
}

export function lifecycleSuccessIcon(action: ExtensionLifecycleAction) {
  switch (action) {
    case 'enable': return 'i-lucide-play'
    case 'disable': return 'i-lucide-pause'
    case 'restart': return 'i-lucide-refresh-cw'
    case 'upgrade': return 'i-lucide-package-check'
    case 'rollback': return 'i-lucide-history'
  }
}

export function lifecycleSuccessMessage(action: ExtensionLifecycleAction) {
  switch (action) {
    case 'enable': return 'admin.extensions.enabled'
    case 'disable': return 'admin.extensions.disabled'
    case 'restart': return 'admin.extensions.restarted'
    case 'upgrade': return 'admin.extensions.upgraded'
    case 'rollback': return 'admin.extensions.rolledBack'
  }
}
