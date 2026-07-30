import hljs from 'highlight.js/lib/common'
import dart from 'highlight.js/lib/languages/dart'
import dockerfile from 'highlight.js/lib/languages/dockerfile'
import http from 'highlight.js/lib/languages/http'
import nginx from 'highlight.js/lib/languages/nginx'
import powershell from 'highlight.js/lib/languages/powershell'

hljs.registerLanguage('dart', dart)
hljs.registerLanguage('dockerfile', dockerfile)
hljs.registerLanguage('http', http)
hljs.registerLanguage('nginx', nginx)
hljs.registerLanguage('powershell', powershell)

const languageAliases: Record<string, string> = {
  'c++': 'cpp',
  'c#': 'csharp',
  cs: 'csharp',
  docker: 'dockerfile',
  gql: 'graphql',
  html: 'xml',
  htm: 'xml',
  js: 'javascript',
  jsx: 'javascript',
  kt: 'kotlin',
  md: 'markdown',
  node: 'javascript',
  objc: 'objectivec',
  ps1: 'powershell',
  py: 'python',
  rb: 'ruby',
  rs: 'rust',
  sh: 'bash',
  shell: 'bash',
  svelte: 'xml',
  toml: 'ini',
  ts: 'typescript',
  tsx: 'typescript',
  vue: 'xml',
  yml: 'yaml',
  zsh: 'bash'
}

const languageNames: Record<string, string> = {
  bash: 'Bash',
  c: 'C',
  cpp: 'C++',
  csharp: 'C#',
  css: 'CSS',
  dart: 'Dart',
  diff: 'Diff',
  dockerfile: 'Dockerfile',
  go: 'Go',
  graphql: 'GraphQL',
  html: 'HTML',
  http: 'HTTP',
  ini: 'INI',
  java: 'Java',
  javascript: 'JavaScript',
  json: 'JSON',
  kotlin: 'Kotlin',
  latex: 'LaTeX',
  less: 'Less',
  lua: 'Lua',
  makefile: 'Makefile',
  markdown: 'Markdown',
  nginx: 'Nginx',
  objectivec: 'Objective-C',
  perl: 'Perl',
  php: 'PHP',
  plaintext: 'Plain text',
  powershell: 'PowerShell',
  python: 'Python',
  r: 'R',
  ruby: 'Ruby',
  rust: 'Rust',
  scss: 'SCSS',
  sql: 'SQL',
  svelte: 'Svelte',
  swift: 'Swift',
  toml: 'TOML',
  typescript: 'TypeScript',
  vbnet: 'Visual Basic',
  vue: 'Vue',
  wasm: 'WebAssembly',
  xml: 'XML',
  yaml: 'YAML'
}

const languageMarks: Record<string, string> = {
  bash: 'SH',
  csharp: 'C#',
  cpp: 'C++',
  dockerfile: 'DK',
  graphql: 'GQL',
  html: 'HTML',
  javascript: 'JS',
  markdown: 'MD',
  objectivec: 'OC',
  plaintext: 'TXT',
  powershell: 'PS',
  python: 'PY',
  typescript: 'TS',
  vue: 'VUE',
  wasm: 'WASM',
  xml: 'XML',
  yaml: 'YML'
}

export const supportedCodeLanguageIds = [
  'bash', 'c', 'cpp', 'csharp', 'css', 'dart', 'diff', 'dockerfile', 'go',
  'graphql', 'http', 'ini', 'java', 'javascript', 'json', 'kotlin', 'less',
  'lua', 'makefile', 'markdown', 'nginx', 'objectivec', 'perl', 'php',
  'plaintext', 'powershell', 'python', 'r', 'ruby', 'rust', 'scss', 'shell',
  'sql', 'swift', 'typescript', 'vbnet', 'wasm', 'xml', 'yaml'
] as const

export interface CodeHighlightLabels {
  code: string
  copy: string
  copied: string
  plainText: string
}

export interface CodeHighlightOptions {
  labels: CodeHighlightLabels
  onCopySuccess?: () => void
  onCopyError?: () => void
  writeClipboard?: (text: string) => Promise<void>
}

const copyIcon = '<svg viewBox="0 0 24 24" aria-hidden="true"><rect width="14" height="14" x="8" y="8" rx="2" ry="2"/><path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"/></svg>'
const checkIcon = '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="m5 12 4 4L19 6"/></svg>'

function rawLanguage(block: HTMLElement) {
  const className = [...block.classList].find(value => value.startsWith('language-'))
  return className?.slice('language-'.length).trim().toLowerCase() || ''
}

function registeredLanguage(language: string) {
  const candidate = languageAliases[language] || language
  return candidate && hljs.getLanguage(candidate) ? candidate : ''
}

export function codeLanguageDisplayName(language: string, fallback: string) {
  const normalized = language.trim().toLowerCase()
  const canonical = languageAliases[normalized] || normalized
  if (!normalized) return fallback
  if (canonical === 'plaintext') return fallback
  return languageNames[normalized]
    || languageNames[canonical]
    || normalized.replace(/(^|[-_.])([a-z])/g, (_match, separator: string, letter: string) => `${separator ? ' ' : ''}${letter.toUpperCase()}`)
}

function codeLanguageMark(language: string) {
  const normalized = language.trim().toLowerCase()
  const canonical = languageAliases[normalized] || normalized || 'plaintext'
  return languageMarks[normalized]
    || languageMarks[canonical]
    || languageNames[canonical]?.slice(0, 3).toUpperCase()
    || canonical.slice(0, 3).toUpperCase()
}

function codeLineCount(value: string) {
  const withoutTrailingLine = value.replace(/(?:\r\n|\r|\n)$/, '')
  return Math.max(1, withoutTrailingLine.split(/\r\n|\r|\n/).length)
}

async function writeClipboardText(text: string, document: Document) {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(text)
    return
  }

  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.style.position = 'fixed'
  textarea.style.opacity = '0'
  document.body.append(textarea)
  textarea.select()
  const copied = document.execCommand('copy')
  textarea.remove()
  if (!copied) throw new Error('clipboard unavailable')
}

function highlightBlock(block: HTMLElement) {
  const source = block.textContent || ''
  const specified = rawLanguage(block)
  const registered = registeredLanguage(specified)
  let resolved = specified

  try {
    if (registered) {
      block.innerHTML = hljs.highlight(source, { language: registered, ignoreIllegals: true }).value
      resolved = specified || registered
    } else if (!specified) {
      // 未声明语言时保持纯文本；短片段自动检测容易把日志或流程文字误判成 CSS/INI。
      resolved = 'plaintext'
    }
  } catch {
    resolved = specified || 'plaintext'
  }

  block.classList.add('hljs')
  block.dataset.sfHighlighted = '1'
  return { source, language: resolved || 'plaintext' }
}

function createLanguageLabel(document: Document, language: string, labels: CodeHighlightLabels) {
  const wrapper = document.createElement('span')
  wrapper.className = 'sf-code-language'

  const mark = document.createElement('span')
  mark.className = 'sf-code-language__mark'
  mark.textContent = codeLanguageMark(language)
  mark.setAttribute('aria-hidden', 'true')

  const name = document.createElement('strong')
  name.textContent = codeLanguageDisplayName(language, labels.plainText)
  wrapper.append(mark, name)
  return wrapper
}

function createLineNumbers(document: Document, count: number) {
  const gutter = document.createElement('span')
  gutter.className = 'sf-code-lines'
  gutter.setAttribute('aria-hidden', 'true')
  const fragment = document.createDocumentFragment()
  for (let index = 1; index <= count; index += 1) {
    const line = document.createElement('span')
    line.textContent = String(index)
    fragment.append(line)
  }
  gutter.append(fragment)
  return gutter
}

function createCopyButton(document: Document, source: string, options: CodeHighlightOptions) {
  const button = document.createElement('button')
  button.type = 'button'
  button.className = 'sf-code-copy'
  button.title = options.labels.copy
  button.setAttribute('aria-label', options.labels.copy)
  button.innerHTML = `<span class="sf-code-copy__icon">${copyIcon}</span><span class="sf-code-copy__label">${options.labels.copy}</span>`

  button.addEventListener('click', async () => {
    if (button.dataset.pending === '1') return
    button.dataset.pending = '1'
    button.disabled = true
    try {
      await (options.writeClipboard || ((value: string) => writeClipboardText(value, document)))(source.replace(/(?:\r\n|\r|\n)$/, ''))
      button.classList.add('is-copied')
      button.setAttribute('aria-label', options.labels.copied)
      button.innerHTML = `<span class="sf-code-copy__icon">${checkIcon}</span><span class="sf-code-copy__label">${options.labels.copied}</span>`
      options.onCopySuccess?.()
      globalThis.setTimeout(() => {
        button.classList.remove('is-copied')
        button.title = options.labels.copy
        button.setAttribute('aria-label', options.labels.copy)
        button.innerHTML = `<span class="sf-code-copy__icon">${copyIcon}</span><span class="sf-code-copy__label">${options.labels.copy}</span>`
      }, 1800)
    } catch {
      options.onCopyError?.()
    } finally {
      button.disabled = false
      delete button.dataset.pending
    }
  })
  return button
}

function decorateBlock(block: HTMLElement, source: string, language: string, options: CodeHighlightOptions) {
  if (block.dataset.sfCodeEnhanced === '1') return
  const pre = block.parentElement
  if (!pre || pre.tagName !== 'PRE') return

  const document = block.ownerDocument
  const figure = document.createElement('figure')
  figure.className = 'sf-code-block'
  figure.dataset.language = language
  figure.setAttribute('aria-label', `${codeLanguageDisplayName(language, options.labels.plainText)} ${options.labels.code}`)

  const caption = document.createElement('figcaption')
  caption.className = 'sf-code-head'
  caption.append(
    createLanguageLabel(document, language, options.labels),
    createCopyButton(document, source, options)
  )

  const scroll = document.createElement('div')
  scroll.className = 'sf-code-scroll'
  const lines = createLineNumbers(document, codeLineCount(source))

  pre.replaceWith(figure)
  scroll.append(lines, pre)
  figure.append(caption, scroll)
  pre.classList.add('sf-code-pre')
  block.dataset.sfCodeEnhanced = '1'
}

export function enhanceCodeBlocks(container: ParentNode | null, options: CodeHighlightOptions) {
  if (!container) return 0
  const blocks = container.querySelectorAll<HTMLElement>('pre code')
  let enhanced = 0
  blocks.forEach((block) => {
    if (block.dataset.sfCodeEnhanced === '1') return
    const highlighted = block.dataset.sfHighlighted === '1'
      ? { source: block.textContent || '', language: rawLanguage(block) || 'plaintext' }
      : highlightBlock(block)
    decorateBlock(block, highlighted.source, highlighted.language, options)
    enhanced += 1
  })
  return enhanced
}
