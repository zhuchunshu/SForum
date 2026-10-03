import {
  mergeAttributes,
  Node,
  type JSONContent,
  type MarkdownParseHelpers,
  type MarkdownRendererHelpers,
  type MarkdownToken,
  type MarkdownTokenizer
} from '@tiptap/vue-3'
import { nodeInputRule, nodePasteRule, type NodeViewRenderer } from '@tiptap/core'
import type {
  EditorProtectedKind,
  EditorProtectedSelectionV1,
  EditorReferenceKind,
  EditorReferenceSelectionV1
} from '~/runtime/editor-extensions/types'

export const SF_SHORTCODE_REF_NODE = 'sforumShortcodeRef'
export const SF_SHORTCODE_BLOCK_NODE = 'sforumShortcodeBlock'
export const SF_SHORTCODE_TEXT_VERSION = 'sforum.shortcode-text@1'

export const SF_SHORTCODE_MAX_NODES = 32
export const SF_SHORTCODE_MAX_DEPTH = 4

export type SFShortcodeResourceKind = 'topic' | 'comment'

export type SFShortcodeAvatarView = {
  kind: 'uploaded' | 'initials' | 'gravatar' | 'static'
  url: string
  attachmentId?: number | null
  alt: string
}

export type SFShortcodeReferencePreview = {
  label: string
  secondaryLabel?: string
  avatar?: SFShortcodeAvatarView | null
  icon?: string
  iconColor?: string
}

type SFShortcodeKind = 'reference' | 'protected'

type SFShortcodeDeclaration = {
  name: string
  id: string
  contractVersion: string
  kind: SFShortcodeKind
  canonicalArgument?: string
  bracketArgument?: string
  commentOnly?: boolean
  fallbackCode: string
  fallbackLabel: string
}

const declarations: SFShortcodeDeclaration[] = [
  { name: 'user', id: 'sforum-shortcodes.user', contractVersion: 'sforum-shortcodes.user@1', kind: 'reference', canonicalArgument: 'userId', bracketArgument: 'user_id', fallbackCode: 'shortcode.user.unavailable', fallbackLabel: '用户引用暂不可用' },
  { name: 'topic', id: 'sforum-shortcodes.topic', contractVersion: 'sforum-shortcodes.topic@1', kind: 'reference', canonicalArgument: 'topicId', bracketArgument: 'topic_id', fallbackCode: 'shortcode.topic.unavailable', fallbackLabel: '主题引用暂不可用' },
  { name: 'comment', id: 'sforum-shortcodes.comment', contractVersion: 'sforum-shortcodes.comment@1', kind: 'reference', canonicalArgument: 'commentId', bracketArgument: 'comment_id', fallbackCode: 'shortcode.comment.unavailable', fallbackLabel: '评论引用暂不可用' },
  { name: 'category', id: 'sforum-shortcodes.category', contractVersion: 'sforum-shortcodes.category@1', kind: 'reference', canonicalArgument: 'categoryId', bracketArgument: 'category_id', fallbackCode: 'shortcode.category.unavailable', fallbackLabel: '分类引用暂不可用' },
  { name: 'friend-links', id: 'sforum-shortcodes.friend-links', contractVersion: 'sforum-shortcodes.friend-links@1', kind: 'reference', fallbackCode: 'shortcode.friend_links.omitted', fallbackLabel: '' },
  { name: 'login', id: 'sforum-shortcodes.login', contractVersion: 'sforum-shortcodes.login@1', kind: 'protected', fallbackCode: 'shortcode.protected.unavailable', fallbackLabel: '受保护内容暂不可用' },
  { name: 'reply', id: 'sforum-shortcodes.reply', contractVersion: 'sforum-shortcodes.reply@1', kind: 'protected', fallbackCode: 'shortcode.protected.unavailable', fallbackLabel: '受保护内容暂不可用' },
  { name: 'only-author', id: 'sforum-shortcodes.only-author', contractVersion: 'sforum-shortcodes.only-author@1', kind: 'protected', commentOnly: true, fallbackCode: 'shortcode.protected.unavailable', fallbackLabel: '受保护内容暂不可用' }
]

const declarationsByName = new Map(declarations.map(item => [item.name, item]))
const declarationsByID = new Map(declarations.map(item => [item.id, item]))
const bracketNamePattern = /^[a-z][a-z0-9-]*$/
const bracketKeyPattern = /^[a-z][a-z0-9_]*$/
const referenceRuleSource = String.raw`\[(?:user|topic|comment|category|friend-links|topic-tag)(?:[ \t]+[a-z_]+=(?:"\d+"|\d+))?\]\[\/(?:user|topic|comment|category|friend-links|topic-tag)\]`

type ParsedOpening = {
  declaration: SFShortcodeDeclaration
  attrs: Record<string, unknown>
}

type ShortcodeInspection = {
  activated: boolean
  withinBudgets: boolean
  count: number
  maxDepth: number
}

export function inspectSForumShortcodeMarkdown(
  markdown: string,
  resourceKind: SFShortcodeResourceKind = 'topic'
): ShortcodeInspection {
  const lines = markdown.match(/.*(?:\n|$)/g)?.filter(Boolean) || []
  const opaque = opaqueSourceLines(lines)
  let count = 0
  let maxDepth = 0

  const scan = (start: number, end: number, depth: number): boolean => {
    let activated = false
    for (let index = start; index < end; index += 1) {
      if (opaque[index]) continue
      const line = normalizedSourceLine(lines[index] || '')
      if (line.startsWith('\\[')) continue
      if (parseReferenceLine(line, resourceKind)) {
        count += 1
        maxDepth = Math.max(maxDepth, depth + 1)
        activated = true
        continue
      }
      const opening = parseOpeningLine(line, resourceKind)
      if (!opening || opening.declaration.kind !== 'protected') continue
      const closing = findProtectedClosingLine(lines, opaque, index, end, opening.declaration.name, resourceKind)
      if (closing < 0 || closing === index + 1) continue
      if (!lines.slice(index + 1, closing).join('').trim()) continue
      scan(index + 1, closing, depth + 1)
      count += 1
      maxDepth = Math.max(maxDepth, depth + 1)
      activated = true
      index = closing
    }
    return activated
  }

  const activated = scan(0, lines.length, 0)
  return {
    activated,
    withinBudgets: count <= SF_SHORTCODE_MAX_NODES && maxDepth <= SF_SHORTCODE_MAX_DEPTH,
    count,
    maxDepth
  }
}

// Over-budget textual input remains literal. Escaping only declared opening
// lines prevents Tiptap's native tokenizer from partially activating it.
export function prepareSForumShortcodeMarkdown(
  markdown: string,
  resourceKind: SFShortcodeResourceKind = 'topic'
) {
  const inspection = inspectSForumShortcodeMarkdown(markdown, resourceKind)
  const lines = markdown.match(/.*(?:\n|$)/g)?.filter(Boolean) || []
  const opaque = opaqueSourceLines(lines)
  return lines.map((raw, index) => {
    const newline = raw.endsWith('\n') ? '\n' : ''
    const withoutNewline = raw.slice(0, raw.length - newline.length)
    const indentation = withoutNewline.match(/^[ ]{0,3}/)?.[0] || ''
    const candidate = withoutNewline.slice(indentation.length).replace(/\r$/, '')
    const escapedCandidate = candidate.startsWith('\\[') ? candidate.slice(1) : candidate
    const openingName = declaredOpeningName(escapedCandidate)
    const mustRemainLiteral = candidate.startsWith('\\[')
      || (resourceKind === 'topic' && openingName === 'only-author')
      || (!inspection.withinBudgets && Boolean(openingName) && !opaque[index])
    if (mustRemainLiteral) {
      const source = candidate.startsWith('\\[') ? candidate.slice(1) : candidate
      return `${indentation}&#91;${source.slice(1)}${newline}`
    }
    return raw
  }).join('')
}

export type SFShortcodeNodeViewHost = {
  disabled: () => boolean
  resolveReference: (kind: Exclude<EditorReferenceKind, 'friend-links'>, id: number) => Promise<SFShortcodeReferencePreview | null>
  editReference: (selection: EditorReferenceSelectionV1) => void
  deleteReference: (selection: EditorReferenceSelectionV1) => void
  editProtected: (selection: EditorProtectedSelectionV1) => void
  deleteProtected: (selection: EditorProtectedSelectionV1) => void
  unwrapProtected: (selection: EditorProtectedSelectionV1) => void
}

export function createSForumShortcodeExtensions(
  resourceKind: SFShortcodeResourceKind = 'topic',
  referenceNodeView?: NodeViewRenderer,
  protectedNodeView?: NodeViewRenderer,
  nodeViewHost?: SFShortcodeNodeViewHost
) {
  return [
    createSForumShortcodeRefExtension(resourceKind, referenceNodeView, nodeViewHost),
    createSForumShortcodeBlockExtension(resourceKind, protectedNodeView, nodeViewHost)
  ]
}

function createSForumShortcodeRefExtension(
  resourceKind: SFShortcodeResourceKind,
  referenceNodeView?: NodeViewRenderer,
  nodeViewHost?: SFShortcodeNodeViewHost
) {
  return Node.create({
    name: SF_SHORTCODE_REF_NODE,
    group: 'block',
    atom: true,
    isolating: true,

    addOptions() {
      return { nodeViewHost }
    },

    addAttributes: shortcodeAttributes,

    parseHTML() {
      return []
    },

    renderHTML({ node, HTMLAttributes }) {
      const declaration = declarationsByID.get(String(node.attrs.id || ''))
      if (!declaration?.fallbackLabel) return ['span', { hidden: 'hidden' }]
      return ['div', mergeAttributes(HTMLAttributes, {
        class: 'sf-editor-fallback',
        'data-fallback': declaration.fallbackCode,
        contenteditable: 'false'
      }), declaration.fallbackLabel]
    },

    renderText({ node }) {
      return declarationsByID.get(String(node.attrs.id || ''))?.fallbackLabel || ''
    },

    markdownTokenName: SF_SHORTCODE_REF_NODE,
    parseMarkdown: parseShortcodeToken,
    markdownTokenizer: referenceTokenizer(resourceKind),
    renderMarkdown: renderReferenceMarkdown,

    addInputRules() {
      return [nodeInputRule({
        find: new RegExp(`^${referenceRuleSource}$`),
        type: this.type,
        getAttributes: match => referenceRuleAttributes(match[0] || '', resourceKind)
      })]
    },

    addPasteRules() {
      return [nodePasteRule({
        find: new RegExp(`^${referenceRuleSource}$`, 'gm'),
        type: this.type,
        getAttributes: match => referenceRuleAttributes(match[0] || '', resourceKind)
      })]
    },

    ...(referenceNodeView ? { addNodeView: () => referenceNodeView } : {})
  })
}

function referenceRuleAttributes(value: string, resourceKind: SFShortcodeResourceKind) {
  return parseReferenceLine(value.trim(), resourceKind)?.attrs || false
}

export function sforumReferenceKind(attrs: Record<string, unknown>): EditorReferenceKind | null {
  const declaration = declarationsByID.get(String(attrs.id || ''))
  if (!declaration || declaration.kind !== 'reference') return null
  return declaration.name as EditorReferenceKind
}

export function sforumReferenceSelection(
  attrs: Record<string, unknown>,
  position: number
): EditorReferenceSelectionV1 | undefined {
  const kind = sforumReferenceKind(attrs)
  if (!kind) return undefined
  return {
    id: String(attrs.id || ''),
    contractVersion: String(attrs.contractVersion || ''),
    arguments: isRecord(attrs.arguments) ? { ...attrs.arguments } : {},
    position
  }
}

export function sforumReferenceNodeAttributes(kind: EditorReferenceKind, id?: number) {
  const declaration = declarationsByName.get(kind)
  if (!declaration || declaration.kind !== 'reference') return null
  const args = declaration.canonicalArgument && Number.isSafeInteger(id) && Number(id) > 0
    ? { [declaration.canonicalArgument]: Number(id) }
    : {}
  if (declaration.canonicalArgument && Object.keys(args).length === 0) return null
  return shortcodeAttrs(declaration, args)
}

export function sforumReferenceNumericID(attrs: Record<string, unknown>) {
  const kind = sforumReferenceKind(attrs)
  if (!kind || kind === 'friend-links') return 0
  const declaration = declarationsByName.get(kind)
  const args = isRecord(attrs.arguments) ? attrs.arguments : {}
  const value = Number(args[declaration?.canonicalArgument || ''])
  return Number.isSafeInteger(value) && value > 0 ? value : 0
}

export function sforumProtectedKind(attrs: Record<string, unknown>): EditorProtectedKind | null {
  const declaration = declarationsByID.get(String(attrs.id || ''))
  if (!declaration || declaration.kind !== 'protected') return null
  return declaration.name as EditorProtectedKind
}

export function sforumProtectedSelection(
  attrs: Record<string, unknown>,
  position: number
): EditorProtectedSelectionV1 | undefined {
  const kind = sforumProtectedKind(attrs)
  if (!kind) return undefined
  return {
    id: String(attrs.id || ''),
    contractVersion: String(attrs.contractVersion || ''),
    arguments: {},
    position,
    protected: true
  }
}

export function sforumProtectedNodeAttributes(kind: EditorProtectedKind, resourceKind: SFShortcodeResourceKind) {
  const declaration = declarationsByName.get(kind)
  if (!declaration || declaration.kind !== 'protected' || (declaration.commentOnly && resourceKind !== 'comment')) {
    return null
  }
  return shortcodeAttrs(declaration, {})
}

export function inspectSForumShortcodeDocument(document: JSONContent, resourceKind: SFShortcodeResourceKind) {
  let count = 0
  let maxDepth = 0
  let valid = true
  const walk = (nodes: JSONContent[], depth: number, parentType: string) => {
    for (const node of nodes) {
      let nextDepth = depth
      if (node.type === SF_SHORTCODE_REF_NODE || node.type === SF_SHORTCODE_BLOCK_NODE) {
        count += 1
        nextDepth += 1
        maxDepth = Math.max(maxDepth, nextDepth)
        const declaration = declarationsByID.get(String(node.attrs?.id || ''))
        if (!declaration || (declaration.commentOnly && resourceKind !== 'comment')) valid = false
        if (parentType !== 'doc' && parentType !== SF_SHORTCODE_BLOCK_NODE) valid = false
        if (node.type === SF_SHORTCODE_BLOCK_NODE && !shortcodeBodyHasContent(node.content || [])) valid = false
      }
      walk(node.content || [], nextDepth, node.type || '')
    }
  }
  walk(document.content || [], 0, 'doc')
  return { valid: valid && count <= SF_SHORTCODE_MAX_NODES && maxDepth <= SF_SHORTCODE_MAX_DEPTH, count, maxDepth }
}

function shortcodeBodyHasContent(nodes: JSONContent[]): boolean {
  return nodes.some(node => {
    if (node.type === 'text') return Boolean(node.text?.trim())
    if (['image', 'horizontalRule', 'sforumEmoji', SF_SHORTCODE_REF_NODE].includes(node.type || '')) return true
    return shortcodeBodyHasContent(node.content || [])
  })
}

function createSForumShortcodeBlockExtension(
  resourceKind: SFShortcodeResourceKind,
  protectedNodeView?: NodeViewRenderer,
  nodeViewHost?: SFShortcodeNodeViewHost
) {
  return Node.create({
    name: SF_SHORTCODE_BLOCK_NODE,
    group: 'block',
    content: 'block+',
    isolating: true,

    addOptions() {
      return { nodeViewHost }
    },

    addAttributes: shortcodeAttributes,

    parseHTML() {
      return []
    },

    // The fallback deliberately has no content hole. Canonical JSON retains
    // descendants, while client HTML and public previews cannot serialize them.
    renderHTML({ node, HTMLAttributes }) {
      const declaration = declarationsByID.get(String(node.attrs.id || ''))
      return ['div', mergeAttributes(HTMLAttributes, {
        class: 'sf-editor-fallback',
        'data-fallback': declaration?.fallbackCode || 'shortcode.protected.unavailable',
        contenteditable: 'false'
      }), declaration?.fallbackLabel || '受保护内容暂不可用']
    },

    renderText() {
      return '受保护内容暂不可用'
    },

    markdownTokenName: SF_SHORTCODE_BLOCK_NODE,
    parseMarkdown(token: MarkdownToken, helpers: MarkdownParseHelpers) {
      return helpers.createNode(SF_SHORTCODE_BLOCK_NODE, token.attributes, helpers.parseChildren(token.tokens || []))
    },
    markdownTokenizer: protectedTokenizer(resourceKind),
    renderMarkdown: renderProtectedMarkdown,

    ...(protectedNodeView ? { addNodeView: () => protectedNodeView } : {})
  })
}

function shortcodeAttributes() {
  return {
    id: { default: '', rendered: false },
    contractVersion: { default: '', rendered: false },
    arguments: { default: {}, rendered: false }
  }
}

function parseShortcodeToken(token: MarkdownToken, helpers: MarkdownParseHelpers) {
  return helpers.createNode(SF_SHORTCODE_REF_NODE, token.attributes, [])
}

function referenceTokenizer(resourceKind: SFShortcodeResourceKind): MarkdownTokenizer {
  return {
    name: SF_SHORTCODE_REF_NODE,
    level: 'block',
    start: src => src.search(/^ {0,3}\[[a-z]/m),
    tokenize(src) {
      const raw = firstSourceLine(src)
      if (/^(?: {4,}|\t)/.test(raw)) return undefined
      const parsed = parseReferenceLine(normalizedSourceLine(raw), resourceKind)
      if (!parsed) return undefined
      return { type: SF_SHORTCODE_REF_NODE, raw, attributes: parsed.attrs }
    }
  }
}

function protectedTokenizer(resourceKind: SFShortcodeResourceKind): MarkdownTokenizer {
  return {
    name: SF_SHORTCODE_BLOCK_NODE,
    level: 'block',
    start: src => src.search(/^ {0,3}\[(?:login|reply|only-author)\]/m),
    tokenize(src, _tokens, lexer) {
      const lines = src.match(/.*(?:\n|$)/g)?.filter(Boolean) || []
      if (lines.length < 3) return undefined
      if (/^(?: {4,}|\t)/.test(lines[0] || '')) return undefined
      const opening = parseOpeningLine(normalizedSourceLine(lines[0] || ''), resourceKind)
      if (!opening || opening.declaration.kind !== 'protected') return undefined
      const opaque = opaqueSourceLines(lines)
      const closing = findProtectedClosingLine(lines, opaque, 0, lines.length, opening.declaration.name, resourceKind)
      if (closing <= 1) return undefined
      const raw = lines.slice(0, closing + 1).join('')
      // Goldmark terminates a list before a following heading only after the
      // explicit blank-line boundary; normalize the tokenizer input to that
      // same boundary so exported Markdown is byte-identical.
      // Goldmark requires an explicit blank-line boundary after a list before
      // a following heading. Tiptap's block lexer otherwise keeps the heading
      // in the final list item, producing a different canonical document.
      const body = lines.slice(1, closing).join('').replace(/(^|\n)((?:[-+*]|\d+[.)])\s+[^\n]*\n)\n(?=#{1,6}\s)/g, '$1$2\n\n')
      if (!body.trim()) return undefined
      return {
        type: SF_SHORTCODE_BLOCK_NODE,
        raw,
        attributes: opening.attrs,
        tokens: lexer.blockTokens(body)
      }
    }
  }
}

function renderReferenceMarkdown(node: JSONContent) {
  const declaration = declarationsByID.get(String(node.attrs?.id || ''))
  if (!declaration) return ''
  return `${canonicalOpening(declaration, node.attrs?.arguments)}[/${declaration.name}]`
}

function renderProtectedMarkdown(node: JSONContent, helpers: MarkdownRendererHelpers) {
  const declaration = declarationsByID.get(String(node.attrs?.id || ''))
  if (!declaration) return ''
  const content = helpers.renderChildren(node.content || [], '\n\n')
    // Goldmark's canonical block separation keeps a full blank line between
    // a list and a following heading. Tiptap emits the list/heading boundary
    // with two newlines, so normalize that exact shape before exporting.
    .replace(/(^|\n)((?:[-+*]|\d+[.)])\s+[^\n]*)\n\n(?=#{1,6}\s)/g, '$1$2\n\n\n')
    .trim()
  return `${canonicalOpening(declaration, node.attrs?.arguments)}\n${content}\n[/${declaration.name}]`
}

function canonicalOpening(declaration: SFShortcodeDeclaration, args: Record<string, unknown> = {}) {
  if (!declaration.canonicalArgument || !declaration.bracketArgument) return `[${declaration.name}]`
  const value = args[declaration.canonicalArgument]
  return `[${declaration.name} ${declaration.bracketArgument}="${String(value)}"]`
}

function parseReferenceLine(line: string, resourceKind: SFShortcodeResourceKind): ParsedOpening | null {
  const closingBracket = line.indexOf(']')
  if (closingBracket < 0) return null
  const opening = parseOpeningLine(line.slice(0, closingBracket + 1), resourceKind)
  if (!opening || opening.declaration.kind !== 'reference') return null
  const openingInner = line.slice(1, closingBracket)
  const syntaxName = openingInner.split(/[ \t]/, 1)[0] || ''
  return line.slice(closingBracket + 1) === `[/${syntaxName}]` ? opening : null
}

function parseOpeningLine(line: string, resourceKind: SFShortcodeResourceKind): ParsedOpening | null {
  if (line.length < 3 || !line.startsWith('[') || !line.endsWith(']') || line.startsWith('[/')) return null
  const inner = line.slice(1, -1)
  const split = inner.search(/[ \t]/)
  const syntaxName = split < 0 ? inner : inner.slice(0, split)
  const argumentText = split < 0 ? '' : inner.slice(split + 1).trim()
  if (!bracketNamePattern.test(syntaxName)) return null

  const alias = syntaxName === 'topic-tag'
  const declaration = declarationsByName.get(alias ? 'category' : syntaxName)
  if (!declaration || (declaration.commentOnly && resourceKind === 'topic')) return null
  if (!declaration.canonicalArgument) {
    if (argumentText) return null
    return { declaration, attrs: shortcodeAttrs(declaration, {}) }
  }

  const argument = parseSingleArgument(argumentText)
  const expectedKey = alias ? 'tag_id' : declaration.bracketArgument
  if (!argument || argument.key !== expectedKey) return null
  return {
    declaration,
    attrs: shortcodeAttrs(declaration, { [declaration.canonicalArgument]: argument.value })
  }
}

function declaredOpeningName(line: string) {
  if (!line.startsWith('[') || line.startsWith('[/')) return ''
  const closing = line.indexOf(']')
  if (closing < 2) return ''
  const name = line.slice(1, closing).split(/[ \t]/, 1)[0] || ''
  return declarationsByName.has(name) || name === 'topic-tag' ? name : ''
}

function parseSingleArgument(input: string) {
  const equals = input.indexOf('=')
  if (equals <= 0 || input.slice(equals + 1).includes('=')) return null
  const key = input.slice(0, equals).trim()
  let value = input.slice(equals + 1).trim()
  if (!bracketKeyPattern.test(key) || key.length > 64 || !value) return null
  if (value.startsWith('"') || value.endsWith('"')) {
    if (!(value.startsWith('"') && value.endsWith('"')) || value.length < 2) return null
    value = value.slice(1, -1)
  }
  if (!/^\d+$/.test(value)) return null
  if (!/^\d+$/.test(value)) return null
  try {
    const parsed = BigInt(value)
    if (parsed <= 0n || parsed > 9223372036854775807n) return null
    // Keep safe IDs as numbers for the existing editor contract. Larger
    // signed-int64 IDs stay decimal strings so JavaScript never rounds them.
    return { key, value: parsed <= BigInt(Number.MAX_SAFE_INTEGER) ? Number(parsed) : value }
  } catch {
    return null
  }
}

function shortcodeAttrs(declaration: SFShortcodeDeclaration, args: Record<string, unknown>) {
  return {
    id: declaration.id,
    contractVersion: declaration.contractVersion,
    arguments: args
  }
}

function findProtectedClosingLine(
  lines: string[],
  opaque: boolean[],
  opening: number,
  end: number,
  openingName: string,
  resourceKind: SFShortcodeResourceKind
) {
  const stack = [openingName]
  for (let index = opening + 1; index < end; index += 1) {
    if (opaque[index]) continue
    const line = normalizedSourceLine(lines[index] || '')
    if (line.startsWith('\\[')) continue
    const nested = parseOpeningLine(line, resourceKind)
    if (nested?.declaration.kind === 'protected') {
      stack.push(nested.declaration.name)
      continue
    }
    const closing = line.match(/^\[\/([a-z][a-z0-9-]*)\]$/)?.[1]
    const declaration = closing ? declarationsByName.get(closing) : undefined
    if (!declaration || declaration.kind !== 'protected') continue
    if (stack.at(-1) !== closing) return -1
    const previous = normalizedSourceLine(lines[index - 1] || '')
    if (/^(?:>|(?:[-+*]|\d+[.)])\s+)/.test(previous)) return -1
    stack.pop()
    if (stack.length === 0) return index
  }
  return -1
}

function firstSourceLine(source: string) {
  const newline = source.indexOf('\n')
  return newline < 0 ? source : source.slice(0, newline + 1)
}

function normalizedSourceLine(raw: string) {
  return raw.replace(/\r?\n$/, '').trim()
}

function opaqueSourceLines(lines: string[]) {
  const result = Array.from({ length: lines.length }, () => false)
  let fence = ''
  let htmlClose = ''
  for (let index = 0; index < lines.length; index += 1) {
    const raw = (lines[index] || '').replace(/\r?\n$/, '')
    const trimmed = raw.trim()
    if (fence) {
      result[index] = true
      if (new RegExp(`^ {0,3}${fence[0]}{${fence.length},}\\s*$`).test(raw)) fence = ''
      continue
    }
    if (htmlClose) {
      result[index] = true
      if (trimmed.toLowerCase().includes(htmlClose)) htmlClose = ''
      continue
    }
    const fenceMatch = raw.match(/^ {0,3}(`{3,}|~{3,})/)
    if (fenceMatch) {
      fence = fenceMatch[1] || ''
      result[index] = true
      continue
    }
    if (/^(?: {4,}|\t)/.test(raw)) {
      result[index] = true
      continue
    }
    const htmlMatch = trimmed.toLowerCase().match(/^<(script|style|pre|textarea)(?:\s|>)/)
    if (htmlMatch) {
      htmlClose = `</${htmlMatch[1]}>`
      result[index] = true
      if (trimmed.toLowerCase().includes(htmlClose)) htmlClose = ''
      continue
    }
    if (/^<[^>]+>$/.test(trimmed)) result[index] = true
  }
  return result
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}
