/**
 * @提及编辑器扩展：候选浮层（纯文本插入）+ 已输入提及的内联高亮装饰。
 *
 * 提及语法与后端 `apps/api/app/Models/Forum/mentions.go` 及渲染层
 * `utils/forum/forumMentions.ts` 保持同一契约（边界、token 字符集、64 长度上限），
 * 编辑器里被高亮的 @ 才一定会触发后端的提及通知。
 * 插入结果是普通文本 `@username `：正文存储/解析链路完全复用现有 mention 解析，
 * 不引入新的消息节点，Markdown 序列化无需任何适配。
 * 候选数据由调用方注入（论坛场景是 /composer/references?kind=user），
 * 本模块不直接依赖网络层，方便在纯 Node 测试里使用假 fetch。
 */
import type { Node as PMNode } from '@tiptap/pm/model'
import type { EditorState } from '@tiptap/pm/state'
import { Plugin, PluginKey } from '@tiptap/pm/state'
import { Decoration, DecorationSet } from '@tiptap/pm/view'
import { Extension } from '@tiptap/core'
import { Suggestion } from '@tiptap/suggestion'
import { forumMentionTokens } from '~/utils/forum/forumMentions'

export type SForumMentionSuggestionItem = {
  id: number
  username: string
  displayName: string
  avatarUrl?: string
  avatarAlt?: string
}

export type SForumMentionLabels = {
  /** 候选为空时展示。 */
  empty: string
}

export type SForumMentionOptions = {
  /** 按用户名/显示名前缀回归候选；异常时必须自行吞掉并返回 []。 */
  fetch: (query: string) => Promise<SForumMentionSuggestionItem[]>
  labels: SForumMentionLabels
}

export type SForumMentionRange = { from: number, to: number }

const mentionSuggestionPluginKey = new PluginKey('sforumMentionSuggestion')
const mentionDecorationPluginKey = new PluginKey<DecorationSet>('sforumMentionDecoration')

const MENTION_INLINE_CLASS = 'sf-mention-inline'

function nodeAllowsMention(node: PMNode) {
  return Boolean(node.isText && !node.marks.some(mark => mark.type.spec.code))
}

/**
 * 扫描文档中所有可见文本节点的 @提及 token，返回内联装饰区间。
 * 代码块/行内代码不参与，与后端金标解析语义一致。
 * 纯函数便于在无 DOM 环境下直接构造 schema 文档做行为测试。
 */
export function sforumMentionRanges(doc: PMNode): SForumMentionRange[] {
  const ranges: SForumMentionRange[] = []
  doc.descendants((node, pos) => {
    if (node.type.spec.code) {
      return false
    }
    if (!nodeAllowsMention(node)) {
      return node.isInline ? false : true
    }
    for (const token of forumMentionTokens(node.text || '')) {
      ranges.push({ from: pos + token.start, to: pos + token.start + token.text.length })
    }
    return false
  })
  return ranges
}

function buildMentionDecorations(doc: PMNode): DecorationSet {
  if (!(doc.textContent || '').includes('@')) {
    return DecorationSet.empty
  }
  const decorations = sforumMentionRanges(doc).map(range => Decoration.inline(range.from, range.to, { class: MENTION_INLINE_CLASS }))
  // from 已经在文档内，区间顺序与文档顺序一致，可直接 create。
  return DecorationSet.create(doc, decorations)
}

/** @ 触发是否落在可提及文本里（排除代码块与行内代码）。 */
export function sforumMentionSuggestionAllowed(state: EditorState, pos: number): boolean {
  const clamped = Math.max(0, Math.min(pos, state.doc.content.size))
  const $pos = state.doc.resolve(clamped)
  if (!$pos.parent.isTextblock || $pos.parent.type.spec.code) {
    return false
  }
  // 光标位于文本边界时 $pos.marks() 可能为空；优先取右侧行内节点的 marks。
  const marks = $pos.nodeAfter?.isInline ? $pos.nodeAfter.marks : ($pos.marks() || [])
  return !marks.some(mark => mark.type.spec.code)
}

type MentionMenuProps = {
  items: SForumMentionSuggestionItem[]
  command: (item: SForumMentionSuggestionItem) => void
  clientRect?: (() => DOMRect | null) | null
  text?: string
}

type MentionSuggestionRendererProps = MentionMenuProps & { editor: unknown }

function mentionOptionRow(item: SForumMentionSuggestionItem): HTMLButtonElement {
  const row = document.createElement('button')
  row.type = 'button'
  row.className = 'sf-mention-menu__option'
  row.setAttribute('role', 'option')
  row.dataset.username = item.username

  if (item.avatarUrl) {
    const img = document.createElement('img')
    img.className = 'sf-mention-menu__avatar'
    img.src = item.avatarUrl
    img.alt = item.avatarAlt || ''
    img.loading = 'lazy'
    img.referrerPolicy = 'no-referrer'
    row.appendChild(img)
  } else {
    const badge = document.createElement('span')
    badge.className = 'sf-mention-menu__avatar sf-mention-menu__avatar--initials'
    badge.textContent = (item.displayName || item.username || '?').trim().slice(0, 1).toUpperCase()
    badge.setAttribute('aria-hidden', 'true')
    row.appendChild(badge)
  }

  const labels = document.createElement('span')
  labels.className = 'sf-mention-menu__labels'
  const name = document.createElement('span')
  name.className = 'sf-mention-menu__name'
  name.textContent = item.displayName
  const username = document.createElement('span')
  username.className = 'sf-mention-menu__username'
  username.textContent = `@${item.username}`
  labels.appendChild(name)
  labels.appendChild(username)
  row.appendChild(labels)
  return row
}

/**
 * 候选浮层（原生 DOM，不依赖 tippy/VueRenderer）：
 * 挂载在 body 下，位置由 suggestion 提供的 clientRect 驱动；
 * ↑↓ 移动、Enter/Tab 选中、Esc 关闭、点击选中（mousedown 保编辑器焦点）。
 */
function createMentionMenuRenderer(labels: SForumMentionLabels) {
  let container: HTMLDivElement | null = null
  let items: SForumMentionSuggestionItem[] = []
  let props: MentionSuggestionRendererProps | null = null
  let activeIndex = 0

  function destroy() {
    container?.remove()
    container = null
    props = null
    items = []
    activeIndex = 0
  }

  function reposition() {
    if (!container || !props?.clientRect) {
      return
    }
    const rect = props.clientRect()
    if (!rect) {
      return
    }
    const gap = 6
    const height = container.offsetHeight || 0
    let top = rect.bottom + gap
    // 视口下沿放不下时翻到 @ 上方，避免被遮挡。
    if (top + height > window.innerHeight - 8 && rect.top - gap - height > 8) {
      top = rect.top - gap - height
    }
    const width = container.offsetWidth || 0
    const left = Math.max(8, Math.min(rect.left, window.innerWidth - width - 8))
    container.style.top = `${Math.round(top)}px`
    container.style.left = `${Math.round(left)}px`
  }

  function render() {
    if (!container) {
      return
    }
    container.replaceChildren()
    if (items.length === 0) {
      const empty = document.createElement('div')
      empty.className = 'sf-mention-menu__empty'
      empty.textContent = labels.empty
      container.appendChild(empty)
      return
    }
    if (activeIndex >= items.length) {
      activeIndex = items.length - 1
    }
    const list = document.createElement('div')
    list.className = 'sf-mention-menu__list'
    list.setAttribute('role', 'listbox')
    items.forEach((item, index) => {
      const row = mentionOptionRow(item)
      row.dataset.active = index === activeIndex ? 'true' : 'false'
      row.setAttribute('aria-selected', index === activeIndex ? 'true' : 'false')
      row.addEventListener('mousedown', (event) => {
        // mousedown 阻止默认行为以避免编辑器失焦导致 suggestion 在命令前被销毁。
        event.preventDefault()
        props?.command(item)
      })
      list.appendChild(row)
    })
    container.appendChild(list)
  }

  function selectActive() {
    const item = items[activeIndex]
    if (item && props) {
      props.command(item)
    }
  }

  return {
    onStart(nextProps: MentionSuggestionRendererProps) {
      props = nextProps
      items = nextProps.items || []
      activeIndex = 0
      container = document.createElement('div')
      container.className = 'sf-mention-menu'
      container.setAttribute('data-sforum-mention-menu', '')
      document.body.appendChild(container)
      render()
      reposition()
    },
    onUpdate(nextProps: MentionSuggestionRendererProps) {
      props = nextProps
      items = nextProps.items || []
      render()
      reposition()
    },
    onExit() {
      destroy()
    },
    onKeyDown({ event }: { event: KeyboardEvent }) {
      if (!container) {
        return false
      }
      if (event.key === 'Escape') {
        destroy()
        return true
      }
      if (items.length === 0) {
        return false
      }
      if (event.key === 'ArrowDown') {
        activeIndex = (activeIndex + 1) % items.length
        render()
        return true
      }
      if (event.key === 'ArrowUp') {
        activeIndex = (activeIndex - 1 + items.length) % items.length
        render()
        return true
      }
      if (event.key === 'Enter' || event.key === 'Tab') {
        selectActive()
        return true
      }
      return false
    }
  }
}

export function createSForumMentionExtensions(options: SForumMentionOptions) {
  const suggestion = Extension.create({
    name: 'sforumMentionSuggestion',
    addProseMirrorPlugins() {
      return [
        Suggestion<SForumMentionSuggestionItem>({
          editor: this.editor,
          pluginKey: mentionSuggestionPluginKey,
          char: '@',
          allowSpaces: false,
          // 至少输入 1 个字符才请求候选，避免 @ 刚落就打一次空查询。
          minQueryLength: 1,
          debounce: 150,
          allow: ({ state, range }) => sforumMentionSuggestionAllowed(state, range.from),
          items: async ({ query }) => {
            const trimmed = query.trim()
            if (!trimmed) {
              return []
            }
            try {
              return await options.fetch(trimmed.slice(0, 64))
            } catch {
              return []
            }
          },
          command: ({ editor, range, props }) => {
            editor.chain().focus().insertContentAt(range, `@${props.username} `).run()
          },
          render: () => createMentionMenuRenderer(options.labels)
        })
      ]
    }
  })

  const decorations = Extension.create({
    name: 'sforumMentionDecorations',
    addProseMirrorPlugins() {
      return [
        new Plugin({
          key: mentionDecorationPluginKey,
          state: {
            init: (_config, state) => buildMentionDecorations(state.doc),
            apply: (tr, previous) => (tr.docChanged ? buildMentionDecorations(tr.doc) : previous)
          },
          props: {
            decorations(state) {
              return mentionDecorationPluginKey.getState(state) as DecorationSet
            }
          }
        })
      ]
    }
  })

  return [suggestion, decorations]
}
