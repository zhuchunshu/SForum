export { default as SPluginAlert } from './components/feedback/SPluginAlert.vue'
export { default as SPluginEmptyState } from './components/feedback/SPluginEmptyState.vue'
export { default as SPluginButton } from './components/forms/SPluginButton.vue'
export { default as SPluginField } from './components/forms/SPluginField.vue'
export { default as SPluginInput } from './components/forms/SPluginInput.vue'
export { default as SPluginSelect } from './components/forms/SPluginSelect.vue'
export { default as SPluginIcon } from './components/icons/SPluginIcon.vue'
export { default as SPluginPage } from './components/layout/SPluginPage.vue'
export { default as SPluginSection } from './components/layout/SPluginSection.vue'
export { default as SPluginTable } from './components/data/SPluginTable.vue'

export {
  PLUGIN_ICON_NAME_PATTERN,
  PLUGIN_ICON_RESOLVER,
  providePluginIcons
} from './icons/resolver'
export { definePluginPage, mountPluginPage } from './mount'

export type { PluginIconResolver } from './icons/resolver'
export type { PluginPageModule, PluginPageMountBridge, PluginPageMountCleanup } from './mount'
export type {
  SPluginSelectOption,
  SPluginTableColumn,
  SPluginTableRow
} from './types'
