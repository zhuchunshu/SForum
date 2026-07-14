import { parseFragment, type DefaultTreeAdapterTypes } from 'parse5'
import { createCommentVNode, h, type Component, type VNodeChild } from 'vue'

export type ThemeIslandPropType = 'string' | 'boolean' | 'integer' | 'url'

export type ThemeIslandProp = {
  name: string
  type: ThemeIslandPropType
  stringValue?: string
  booleanValue?: boolean
  integerValue?: number
}

export type ThemeIslandDescriptor = {
  id: string
  componentId: string
  props?: ThemeIslandProp[]
}

export type ThemeRenderOutput = {
  htmlSegments: string[]
  islands?: ThemeIslandDescriptor[]
  seo?: Record<string, unknown>
}

export type ThemeRenderNode =
  | { kind: 'text', value: string }
  | { kind: 'comment', value: string }
  | { kind: 'element', tag: string, attrs: Record<string, string>, children: ThemeRenderNode[] }
  | { kind: 'island', descriptor: ThemeIslandDescriptor, props: Record<string, string | boolean | number> }

type ParseOptions = {
  allowedComponents?: ReadonlySet<string>
}

type LegacyIslandBinding = {
  componentId: string
}

const HTML_NAMESPACE = 'http://www.w3.org/1999/xhtml'
const ISLAND_ATTRIBUTE = 'data-sforum-island'
const MAX_RENDER_NODES = 50_000
const MAX_RENDER_DEPTH = 128
const ID_PATTERN = /^[a-z][a-z0-9_.-]*:[1-9][0-9]*$/
const COMPONENT_PATTERN = /^[a-z][a-z0-9_.-]*$/
const PROP_PATTERN = /^[a-z][a-z0-9-]*$/
const FORBIDDEN_TAGS = new Set([
  'script', 'style', 'iframe', 'object', 'embed', 'svg', 'math', 'base', 'meta', 'link', 'form'
])
const FORBIDDEN_ATTRIBUTES = new Set([
  'style', 'srcdoc', 'is', 'innerhtml', 'outerhtml', 'textcontent', 'key', 'ref', 'ref_for', 'ref_key'
])
const URL_ATTRIBUTES = new Set([
  'action', 'background', 'cite', 'data', 'formaction', 'href', 'longdesc', 'manifest', 'ping',
  'poster', 'profile', 'src', 'srcset', 'usemap'
])

export class ThemeRenderOutputError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'ThemeRenderOutputError'
  }
}

/**
 * 将后端 ThemeCompiler 的 HTML skeleton 与 typed islands 合并为稳定 AST。
 * parse5 在 Nitro 与浏览器使用相同的 HTML5 规则，避免 SSR/hydration 分叉。
 */
export function parseThemeRenderOutput(input: unknown, options: ParseOptions = {}): ThemeRenderNode[] {
  if (!isObject(input) || !Array.isArray(input.htmlSegments) || !Array.isArray(input.islands ?? [])) {
    throw new ThemeRenderOutputError('invalid render output')
  }

  const segments = input.htmlSegments
  if (!segments.length || segments.some(segment => typeof segment !== 'string')) {
    throw new ThemeRenderOutputError('invalid html segments')
  }

  const descriptors = normalizeDescriptors(input.islands ?? [], options.allowedComponents)
  const context = createParseContext(descriptors)
  const nodes = segments.flatMap(segment => parseHTMLFragment(segment, context, 0))
  assertAllIslandsConsumed(context)
  return nodes
}

/** P13 前的 legacy L1 兼容层；同样走 HTML AST，不再用正则拆岛。 */
export function parseLegacyThemeHTML(
  source: string,
  bindings: Readonly<Record<string, LegacyIslandBinding>>
): ThemeRenderNode[] {
  if (typeof source !== 'string' || !source.trim()) {
    return []
  }
  const context = createParseContext(new Map(), bindings)
  return parseHTMLFragment(source, context, 0)
}

export function renderThemeRenderNodes(
  nodes: readonly ThemeRenderNode[],
  resolveIsland: (componentId: string) => Component | undefined
): VNodeChild[] {
  const renderNode = (node: ThemeRenderNode): VNodeChild => {
    if (node.kind === 'text') {
      return node.value
    }
    if (node.kind === 'comment') {
      return createCommentVNode(node.value)
    }
    if (node.kind === 'island') {
      const component = resolveIsland(node.descriptor.componentId)
      if (!component) {
        return createCommentVNode('sforum-island-unavailable')
      }
      return h(component, node.props)
    }
    return h(node.tag, node.attrs, node.children.map(renderNode))
  }

  return nodes.map(renderNode)
}

type ParseContext = {
  descriptors: Map<string, ThemeIslandDescriptor>
  consumed: Set<string>
  legacyBindings?: Readonly<Record<string, LegacyIslandBinding>>
  nodeCount: number
}

function createParseContext(
  descriptors: Map<string, ThemeIslandDescriptor>,
  legacyBindings?: Readonly<Record<string, LegacyIslandBinding>>
): ParseContext {
  return { descriptors, consumed: new Set(), legacyBindings, nodeCount: 0 }
}

function parseHTMLFragment(source: string, context: ParseContext, depth: number): ThemeRenderNode[] {
  const fragment = parseFragment(source)
  return fragment.childNodes.map(node => convertNode(node, context, depth))
}

function convertNode(
  node: DefaultTreeAdapterTypes.ChildNode,
  context: ParseContext,
  depth: number
): ThemeRenderNode {
  context.nodeCount++
  if (context.nodeCount > MAX_RENDER_NODES || depth > MAX_RENDER_DEPTH) {
    throw new ThemeRenderOutputError('render tree limit exceeded')
  }
  if (node.nodeName === '#text' && 'value' in node) {
    return { kind: 'text', value: node.value }
  }
  if (node.nodeName === '#comment' && 'data' in node) {
    return { kind: 'comment', value: node.data }
  }
  if (node.nodeName === '#documentType') {
    throw new ThemeRenderOutputError('document type is not allowed in page output')
  }
  if (!('tagName' in node) || !('attrs' in node) || !('childNodes' in node)) {
    throw new ThemeRenderOutputError('unsupported HTML node')
  }
  if (node.namespaceURI !== HTML_NAMESPACE) {
    throw new ThemeRenderOutputError('foreign content is not allowed')
  }

  const tag = node.tagName.toLowerCase()
  if (FORBIDDEN_TAGS.has(tag)) {
    throw new ThemeRenderOutputError(`forbidden element ${tag}`)
  }

  const islandAttribute = node.attrs.find(attribute => attribute.name.toLowerCase() === ISLAND_ATTRIBUTE)
  if (islandAttribute) {
    return convertTypedIsland(node, tag, islandAttribute.value, context)
  }

  const legacyBinding = context.legacyBindings?.[tag]
  if (legacyBinding) {
    return convertLegacyIsland(node, legacyBinding)
  }
  if (tag.startsWith('sf-')) {
    throw new ThemeRenderOutputError(`unknown legacy island ${tag}`)
  }
  if (tag === 'template') {
    throw new ThemeRenderOutputError('unbound template element')
  }

  const attrs = normalizeHTMLAttributes(node.attrs)
  return {
    kind: 'element',
    tag,
    attrs,
    children: node.childNodes.map(child => convertNode(child, context, depth + 1))
  }
}

function convertTypedIsland(
  node: DefaultTreeAdapterTypes.Element,
  tag: string,
  id: string,
  context: ParseContext
): ThemeRenderNode {
  const content = tag === 'template' && 'content' in node
    ? (node as DefaultTreeAdapterTypes.Template).content.childNodes
    : node.childNodes
  if (tag !== 'template' || node.attrs.length !== 1 || content.length !== 0) {
    throw new ThemeRenderOutputError('forged island placeholder')
  }
  const descriptor = context.descriptors.get(id)
  if (!descriptor || context.consumed.has(id)) {
    throw new ThemeRenderOutputError('unknown or duplicate island placeholder')
  }
  context.consumed.add(id)
  return { kind: 'island', descriptor, props: normalizeIslandProps(descriptor.props ?? []) }
}

function convertLegacyIsland(
  node: DefaultTreeAdapterTypes.Element,
  binding: LegacyIslandBinding
): ThemeRenderNode {
  if (node.childNodes.some(child => child.nodeName !== '#text' || !('value' in child) || child.value.trim() !== '')) {
    throw new ThemeRenderOutputError('legacy island must be empty')
  }
  const descriptor: ThemeIslandDescriptor = {
    id: `legacy:${binding.componentId}`,
    componentId: binding.componentId,
    props: []
  }
  return { kind: 'island', descriptor, props: normalizeHTMLAttributes(node.attrs) }
}

function normalizeDescriptors(
  input: unknown[],
  allowedComponents?: ReadonlySet<string>
): Map<string, ThemeIslandDescriptor> {
  const result = new Map<string, ThemeIslandDescriptor>()
  for (const value of input) {
    if (!isObject(value) || typeof value.id !== 'string' || typeof value.componentId !== 'string'
      || !ID_PATTERN.test(value.id) || !COMPONENT_PATTERN.test(value.componentId)
      || !Array.isArray(value.props ?? [])) {
      throw new ThemeRenderOutputError('invalid island descriptor')
    }
    if (allowedComponents && !allowedComponents.has(value.componentId)) {
      throw new ThemeRenderOutputError('unknown island component')
    }
    if (result.has(value.id)) {
      throw new ThemeRenderOutputError('duplicate island descriptor')
    }
    const descriptor: ThemeIslandDescriptor = {
      id: value.id,
      componentId: value.componentId,
      props: value.props as ThemeIslandProp[]
    }
    // Validate typed values before any component receives them.
    normalizeIslandProps(descriptor.props ?? [])
    result.set(descriptor.id, descriptor)
  }
  return result
}

function normalizeIslandProps(input: ThemeIslandProp[]): Record<string, string | boolean | number> {
  const result: Record<string, string | boolean | number> = {}
  for (const prop of input) {
    if (!isObject(prop) || typeof prop.name !== 'string' || !PROP_PATTERN.test(prop.name)
      || Object.hasOwn(result, prop.name)) {
      throw new ThemeRenderOutputError('invalid island prop')
    }
    switch (prop.type) {
      case 'string':
      case 'url':
        if (!hasOnlyZeroValues(prop, 'stringValue')
          || (prop.stringValue !== undefined && typeof prop.stringValue !== 'string')) {
          throw new ThemeRenderOutputError('invalid island string prop')
        }
        result[prop.name] = prop.stringValue ?? ''
        if (prop.type === 'url' && !isSafeURL(result[prop.name] as string)) {
          throw new ThemeRenderOutputError('invalid island URL prop')
        }
        break
      case 'boolean':
        if (!hasOnlyZeroValues(prop, 'booleanValue')
          || (prop.booleanValue !== undefined && typeof prop.booleanValue !== 'boolean')) {
          throw new ThemeRenderOutputError('invalid island boolean prop')
        }
        result[prop.name] = prop.booleanValue ?? false
        break
      case 'integer':
        if (!hasOnlyZeroValues(prop, 'integerValue')
          || (prop.integerValue !== undefined && !Number.isSafeInteger(prop.integerValue))) {
          throw new ThemeRenderOutputError('invalid island integer prop')
        }
        result[prop.name] = prop.integerValue ?? 0
        break
      default:
        throw new ThemeRenderOutputError('unknown island prop type')
    }
  }
  return result
}

function hasOnlyZeroValues(
  prop: ThemeIslandProp,
  active: 'stringValue' | 'booleanValue' | 'integerValue'
): boolean {
  return (active === 'stringValue' || prop.stringValue === undefined || prop.stringValue === '')
    && (active === 'booleanValue' || prop.booleanValue === undefined || prop.booleanValue === false)
    && (active === 'integerValue' || prop.integerValue === undefined || prop.integerValue === 0)
}

function normalizeHTMLAttributes(
  input: DefaultTreeAdapterTypes.Element['attrs']
): Record<string, string> {
  const result: Record<string, string> = {}
  for (const attribute of input) {
    const name = attribute.name.toLowerCase()
    if (!name || name === ISLAND_ATTRIBUTE || name.startsWith('on') || name.startsWith('v-')
      || name.startsWith(':') || name.startsWith('@') || name.startsWith('#')
      || FORBIDDEN_ATTRIBUTES.has(name) || Object.hasOwn(result, name)) {
      throw new ThemeRenderOutputError(`forbidden attribute ${name}`)
    }
    if (URL_ATTRIBUTES.has(name) && !isSafeURL(attribute.value)) {
      throw new ThemeRenderOutputError(`forbidden URL attribute ${name}`)
    }
    result[attribute.prefix ? `${attribute.prefix}:${name}` : name] = attribute.value
  }
  return result
}

function isSafeURL(value: string): boolean {
  const canonical = value.replace(/[\u0000-\u0020\u007f]/g, '').toLowerCase()
  return !['javascript:', 'vbscript:', 'data:', 'file:'].some(scheme => canonical.includes(scheme))
}

function assertAllIslandsConsumed(context: ParseContext) {
  if (context.consumed.size !== context.descriptors.size) {
    throw new ThemeRenderOutputError('island descriptor is missing its placeholder')
  }
}

function isObject(value: unknown): value is Record<string, any> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}
