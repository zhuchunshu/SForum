import { ref } from 'vue'
import { apiErrorMessage } from '~/composables/useApiClient'
import type { WebOption } from '~/composables/useWebOptions'
import {
  BUILTIN_AUTO_UPDATE_OPTION,
  isBuiltinAutoUpdateEnabledValue
} from '~/utils/admin/extensions/builtinAutoUpdate'

/**
 * 内置插件自动更新开关的管理端状态。
 *
 * 只读写既有的 admin web-options：`extension.builtin_auto_update` 是 Host 选项，
 * 不需要（也不应该有）新的 API 端点。加载失败时 `supported` 保持 false，
 * 页面据此隐藏开关，而不是猜一个默认值。
 */
export function useAdminBuiltinAutoUpdate() {
  const { fetchAdminEnvelope, saveMany } = useWebOptions()

  const supported = ref(false)
  const enabled = ref(false)
  const saving = ref(false)
  const loadError = ref('')
  const saveError = ref('')

  function applyOptions(items: WebOption[]) {
    const option = items.find(item => item.name === BUILTIN_AUTO_UPDATE_OPTION)
    supported.value = Boolean(option)
    enabled.value = option ? isBuiltinAutoUpdateEnabledValue(option.value) : false
  }

  async function load() {
    loadError.value = ''
    try {
      const envelope = await fetchAdminEnvelope()
      applyOptions(envelope.data || [])
      return true
    } catch (error) {
      supported.value = false
      enabled.value = false
      loadError.value = apiErrorMessage(error)
      return false
    }
  }

  async function save(next: boolean) {
    if (saving.value) {
      return false
    }

    saving.value = true
    saveError.value = ''
    try {
      const updated = await saveMany([{
        name: BUILTIN_AUTO_UPDATE_OPTION,
        value: next ? 'enabled' : 'disabled'
      }])
      const option = updated.find(item => item.name === BUILTIN_AUTO_UPDATE_OPTION)
      if (option) {
        supported.value = true
        enabled.value = isBuiltinAutoUpdateEnabledValue(option.value)
      } else {
        // 后端可能只回传被接受的选项；此时以本次提交的意图为准。
        enabled.value = next
      }
      return true
    } catch (error) {
      saveError.value = apiErrorMessage(error)
      return false
    } finally {
      saving.value = false
    }
  }

  async function restoreRecommended() {
    const previous = enabled.value
    enabled.value = true
    const ok = await save(true)
    if (!ok) {
      enabled.value = previous
    }
    return ok
  }

  return {
    supported,
    enabled,
    saving,
    loadError,
    saveError,
    load,
    save,
    restoreRecommended
  }
}
