type PermissionTextItem = {
  key: string
  module: string
  description?: string
}

export const usePermissionText = () => {
  const { t, te } = useI18n()

  const permissionCatalogPath = (permissionKey: string, field: 'label' | 'description') => {
    return `admin.permissionCatalog.${permissionKey}.${field}`
  }

  const modulePath = (module: string) => {
    return `admin.permissionModules.${module}`
  }

  const permissionLabel = (permission: PermissionTextItem) => {
    const path = permissionCatalogPath(permission.key, 'label')
    return te(path) ? t(path) : permission.key
  }

  const permissionDescription = (permission: PermissionTextItem) => {
    const path = permissionCatalogPath(permission.key, 'description')
    return te(path) ? t(path) : (permission.description || t('admin.permissions.noDescription'))
  }

  const permissionModuleLabel = (module: string) => {
    const path = modulePath(module)
    return te(path) ? t(path) : module
  }

  return {
    permissionLabel,
    permissionDescription,
    permissionModuleLabel
  }
}
