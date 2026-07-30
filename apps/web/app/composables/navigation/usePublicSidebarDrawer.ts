type PublicSidebarOwner = {
  id: string
  token: string
}

export function usePublicSidebarDrawer() {
  const open = useState<boolean>('public-sidebar-drawer-open', () => false)
  const owner = useState<PublicSidebarOwner | null>('public-sidebar-drawer-owner', () => null)

  function openDrawer() {
    open.value = true
  }

  function closeDrawer() {
    open.value = false
  }

  function toggleDrawer() {
    open.value = !open.value
  }

  function claimOwner(id: string, token: string) {
    owner.value = { id, token }
  }

  function releaseOwner(token: string) {
    if (owner.value?.token !== token) return
    owner.value = null
    open.value = false
  }

  return {
    open,
    owner: readonly(owner),
    hasPageOwner: computed(() => Boolean(owner.value)),
    openDrawer,
    closeDrawer,
    toggleDrawer,
    claimOwner,
    releaseOwner
  }
}
