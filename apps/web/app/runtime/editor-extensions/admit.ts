import type { EditorCatalog, EditorCatalogModule } from './types'
import { loadTrustedEditorL2Module } from './load'
import { EditorL2ContractError } from './types'

export type AdmittedEditorExtensions = {
  extensions: unknown[]
  toolbars: EditorCatalog['toolbars']
  quarantined: string[]
}

/**
 * Admit every catalog module under trusted L2 load. Any single module failure
 * is quarantined; core editor remains usable with partial plugin surfaces.
 */
export async function admitEditorCatalogModules(
  catalog: EditorCatalog,
  apiBaseUrl: string,
  loadModule: typeof loadTrustedEditorL2Module = loadTrustedEditorL2Module
): Promise<AdmittedEditorExtensions> {
  const extensions: unknown[] = []
  const quarantined: string[] = []
  for (const module of catalog.modules) {
    try {
      assertLoadableModule(module)
      const loaded = await loadModule(module, apiBaseUrl)
      extensions.push(...loaded.extensions)
    } catch (error) {
      const reason = error instanceof Error ? error.message : 'unknown editor L2 failure'
      quarantined.push(`${module.extensionId}:${module.l2Module}:${reason}`)
    }
  }
  return {
    extensions,
    toolbars: catalog.toolbars,
    quarantined
  }
}

function assertLoadableModule(module: EditorCatalogModule) {
  if (!module.l2Module || !module.l2Digest || !module.assetPath.includes(module.packageDigest)) {
    throw new EditorL2ContractError('editor catalog module is not loadable')
  }
}
