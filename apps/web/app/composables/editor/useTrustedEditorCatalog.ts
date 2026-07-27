import {
  admitEditorCatalogModules,
  type AdmittedEditorExtensions
} from '~/runtime/editor-extensions/admit'
import {
  parseEditorCatalog,
  type EditorCatalog
} from '~/runtime/editor-extensions/types'

/**
 * 拉取 Host 公开 editor-catalog 并 digest-verify 准入 L2 扩展。
 * 失败 fail-closed：返回空 extensions，核心 SFEditor 仍可用。
 */
export function useTrustedEditorCatalog() {
  const { apiBaseUrl, request } = useApiClient()

  async function loadAdmittedExtensions(): Promise<AdmittedEditorExtensions & { catalog: EditorCatalog | null }> {
    try {
      const catalogRaw = await request<unknown>('/extensions/runtime/editor-catalog')
      const catalog = parseEditorCatalog(catalogRaw)
      const admitted = await admitEditorCatalogModules(catalog, apiBaseUrl)
      return { ...admitted, catalog }
    } catch {
      return {
        extensions: [],
        toolbars: [],
        quarantined: ['editor-catalog:load-failed'],
        catalog: null
      }
    }
  }

  return { loadAdmittedExtensions }
}
