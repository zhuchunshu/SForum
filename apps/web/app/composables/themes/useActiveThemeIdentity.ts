import {
  mergeActiveThemeIdentity,
  type ActiveThemeIdentity
} from '~/utils/themes/activeThemeClientCache'

export function useActiveThemeIdentity() {
  const identity = useState<ActiveThemeIdentity | null>('sforum-active-theme-identity', () => null)

  function update(next: ActiveThemeIdentity | null) {
    identity.value = mergeActiveThemeIdentity(identity.value, next)
  }

  return { identity, update }
}
