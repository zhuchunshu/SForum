import { Window } from 'happy-dom'
import { compileScript, compileTemplate, parse, rewriteDefault } from '@vue/compiler-sfc'

const testWindow = new Window({ url: 'http://localhost/' })
Object.assign(globalThis, {
  window: testWindow,
  document: testWindow.document,
  navigator: testWindow.navigator,
  Element: testWindow.Element,
  HTMLElement: testWindow.HTMLElement,
  SVGElement: testWindow.SVGElement,
  Node: testWindow.Node,
  Event: testWindow.Event,
  MouseEvent: testWindow.MouseEvent,
  KeyboardEvent: testWindow.KeyboardEvent
})

export const testVue = await import('vue')
export const { mount, flushPromises } = await import('@vue/test-utils')

export async function compileVueSfc(path: string, id = 'test-sfc') {
  const parsed = parse(await Bun.file(path).text(), { filename: path })
  const script = compileScript(parsed.descriptor, { id })
  const template = compileTemplate({
    source: parsed.descriptor.template?.content || '',
    filename: path,
    id,
    compilerOptions: { bindingMetadata: script.bindings }
  })
  if (template.errors.length) throw new Error(String(template.errors[0]))

  const scriptCode = `const { defineComponent: _defineComponent } = Vue\n${script.content
    .replace(/^import(?:[\s\S]*?from\s+)?['"][^'"]+['"];?\s*$/gm, '')
    .replace(/^export type /gm, 'type ')
    .replace(/^export interface /gm, 'interface ')}`
  const templateCode = template.code
    .replace(/import \{([^}]+)\} from "vue"/, (_match, names) => `const {${names.replace(/(\w+) as (\w+)/g, '$1: $2')}} = Vue`)
    .replace('export function render', 'function render')
  const source = `${rewriteDefault(scriptCode, '__sfc__', ['typescript'])}
${templateCode}
__sfc__.render = render
return __sfc__`
  const executable = new Bun.Transpiler({ loader: 'ts', target: 'esnext' }).transformSync(source)
  return new Function('Vue', executable)(testVue)
}
