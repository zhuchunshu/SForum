import type {
  EditorCatalog,
  EditorCatalogContribution,
  EditorCatalogModule,
  EditorL2CommandHandlerV1
} from './types'
import { loadTrustedEditorL2Module } from './load'
import { EditorL2ContractError, isExactEditorAssetPath } from './types'

export type AdmittedEditorExtensions = {
  extensions: unknown[]
  toolbars: EditorCatalog['toolbars']
  commands: Record<string, AdmittedEditorCommand>
  quarantined: string[]
}

export type AdmittedEditorCommand = {
  declaration: EditorCatalogContribution
  handler: EditorL2CommandHandlerV1
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
  if (catalog.safeMode) {
    return {
      extensions: [],
      toolbars: [],
      commands: {},
      quarantined: ['editor-catalog:safe-mode']
    }
  }
  const extensions: unknown[] = []
  const commands: Record<string, AdmittedEditorCommand> = {}
  const toolbars: EditorCatalog['toolbars'] = []
  const quarantined: string[] = []
  for (const module of catalog.modules) {
    try {
      assertLoadableModule(module)
      const loaded = await loadModule(module, apiBaseUrl)
      const admittedCommands = admitModuleCommands(module, loaded.commands)
      extensions.push(...loaded.extensions)
      Object.assign(commands, admittedCommands)
      toolbars.push(...module.toolbars.filter(toolbar => Boolean(commands[toolbar.commandId || ''])))
    } catch (error) {
      const reason = error instanceof Error ? error.message : 'unknown editor L2 failure'
      quarantined.push(`${module.extensionId}:${module.l2Module}:${reason}`)
    }
  }
  return {
    extensions,
    toolbars,
    commands,
    quarantined
  }
}

function admitModuleCommands(
  module: EditorCatalogModule,
  handlers: Record<string, EditorL2CommandHandlerV1>
) {
  const admitted: Record<string, AdmittedEditorCommand> = {}
  for (const declaration of module.commands) {
    const commandKey = declaration.commandKey || ''
    const handler = handlers[commandKey]
    if (!commandKey || typeof handler !== 'function') {
      throw new EditorL2ContractError(`editor command handler is missing: ${commandKey || declaration.id}`)
    }
    admitted[declaration.id] = { declaration, handler }
  }
  return admitted
}

function assertLoadableModule(module: EditorCatalogModule) {
  if (!module.l2Module || !module.l2Digest || !isExactEditorAssetPath(
    module.assetPath, module.extensionId, module.packageDigest, module.l2Module
  )) {
    throw new EditorL2ContractError('editor catalog module is not loadable')
  }
}
