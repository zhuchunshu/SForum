/**
 * 代码语法高亮（仅客户端）。
 *
 * 正文 HTML 由后端 goldmark 渲染 + bluemonday sanitize 后通过 v-html 注入，
 * 已是最终 HTML（含 <pre><code class="language-xxx">）。因此采用 highlight.js
 * 在 SSR 后扫描 DOM 的高亮方式，而不在前端重新解析 markdown。
 *
 * 注册全局指令 v-highlight：挂载到任意包含 <pre><code> 的 v-html 容器上，
 * 指令会在 mounted/updated 时扫描其中的代码块并高亮。
 * 用法：<div class="sf-prose" v-highlight v-html="content" />
 *
 * 仅按需注册常见语言以控制体积；未注册的语言由 hljs 自动检测兜底。
 */
import hljs from 'highlight.js/lib/core'
import bash from 'highlight.js/lib/languages/bash'
import css from 'highlight.js/lib/languages/css'
import go from 'highlight.js/lib/languages/go'
import xml from 'highlight.js/lib/languages/xml' // html/xml
import ini from 'highlight.js/lib/languages/ini' // toml/ini/yaml 邻近
import json from 'highlight.js/lib/languages/json'
import javascript from 'highlight.js/lib/languages/javascript'
import markdown from 'highlight.js/lib/languages/markdown'
import python from 'highlight.js/lib/languages/python'
import ruby from 'highlight.js/lib/languages/ruby'
import rust from 'highlight.js/lib/languages/rust'
import shell from 'highlight.js/lib/languages/shell'
import sql from 'highlight.js/lib/languages/sql'
import typescript from 'highlight.js/lib/languages/typescript'
import yaml from 'highlight.js/lib/languages/yaml'

// 论坛常见语言别名，让 class="language-xxx" 命中已注册语言。
const aliases: Record<string, string> = {
  js: 'javascript',
  ts: 'typescript',
  sh: 'bash',
  shell: 'shell',
  py: 'python',
  rb: 'ruby',
  rs: 'rust',
  yml: 'yaml',
  html: 'xml',
  md: 'markdown',
  toml: 'ini',
}

hljs.registerLanguage('bash', bash)
hljs.registerLanguage('css', css)
hljs.registerLanguage('go', go)
hljs.registerLanguage('xml', xml)
hljs.registerLanguage('ini', ini)
hljs.registerLanguage('json', json)
hljs.registerLanguage('javascript', javascript)
hljs.registerLanguage('markdown', markdown)
hljs.registerLanguage('python', python)
hljs.registerLanguage('ruby', ruby)
hljs.registerLanguage('rust', rust)
hljs.registerLanguage('shell', shell)
hljs.registerLanguage('sql', sql)
hljs.registerLanguage('typescript', typescript)
hljs.registerLanguage('yaml', yaml)

// 高亮容器内的全部 <pre><code> 块。重复高亮会抛错，用 data 属性标记已处理。
function highlightContainer(el: Element | null) {
  if (!el || !(el instanceof Element)) return
  const blocks = el.querySelectorAll<HTMLElement>('pre code')
  blocks.forEach((block) => {
    if (block.dataset.sfHighlighted) return
    // 从 class="language-xxx" 解析语言，映射别名后精确高亮；未命中则自动检测。
    const langClass = [...block.classList].find((c) => c.startsWith('language-'))
    const lang = langClass ? langClass.replace('language-', '') : ''
    const registered = lang ? aliases[lang] ?? lang : ''
    try {
      if (registered && hljs.getLanguage(registered)) {
        const result = hljs.highlight(block.textContent ?? '', { language: registered, ignoreIllegals: true })
        block.innerHTML = result.value
        block.classList.add('hljs')
      } else {
        hljs.highlightElement(block)
        block.classList.add('hljs')
      }
    } catch {
      // 高亮失败时保持原文，不影响正文展示。
    }
    block.dataset.sfHighlighted = '1'
  })
}

export default defineNuxtPlugin((nuxtApp) => {
  nuxtApp.vueApp.directive('highlight', {
    mounted(el: Element) {
      highlightContainer(el)
    },
    updated(el: Element) {
      highlightContainer(el)
    },
    getSSRProps() {
      // 服务端渲染时不输出额外属性，高亮完全在客户端进行。
      return {}
    },
  })
})
