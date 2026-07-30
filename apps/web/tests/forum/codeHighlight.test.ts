import { describe, expect, test } from 'bun:test'
import { Window } from 'happy-dom'
import {
  codeLanguageDisplayName,
  enhanceCodeBlocks,
  supportedCodeLanguageIds
} from '../../app/utils/forum/codeHighlight'

const labels = {
  code: '代码块',
  copy: '复制',
  copied: '已复制',
  plainText: '纯文本'
}

describe('forum code highlighting', () => {
  test('supports the expanded common language catalog and forum aliases', () => {
    expect(supportedCodeLanguageIds).toContain('cpp')
    expect(supportedCodeLanguageIds).toContain('csharp')
    expect(supportedCodeLanguageIds).toContain('dart')
    expect(supportedCodeLanguageIds).toContain('dockerfile')
    expect(supportedCodeLanguageIds).toContain('java')
    expect(supportedCodeLanguageIds).toContain('kotlin')
    expect(supportedCodeLanguageIds).toContain('php')
    expect(supportedCodeLanguageIds).toContain('powershell')
    expect(supportedCodeLanguageIds).toContain('swift')
    expect(codeLanguageDisplayName('vue', labels.plainText)).toBe('Vue')
    expect(codeLanguageDisplayName('tsx', labels.plainText)).toBe('TypeScript')
    expect(codeLanguageDisplayName('toml', labels.plainText)).toBe('TOML')
  })

  test('adds a language header, stable line gutter, highlighting, and copy behavior once', async () => {
    const browser = new Window({ url: 'http://localhost/' })
    const container = browser.document.createElement('div')
    container.innerHTML = '<pre><code class="language-c++">const answer = 42;\nreturn answer;\n</code></pre>'
    let copied = ''
    let copySuccesses = 0

    const options = {
      labels,
      writeClipboard: async (value: string) => { copied = value },
      onCopySuccess: () => { copySuccesses += 1 }
    }
    expect(enhanceCodeBlocks(container as unknown as ParentNode, options)).toBe(1)

    const figure = container.querySelector('.sf-code-block')
    expect(figure?.getAttribute('data-language')).toBe('c++')
    expect(figure?.getAttribute('aria-label')).toBe('C++ 代码块')
    expect(figure?.querySelector('.sf-code-language strong')?.textContent).toBe('C++')
    expect(figure?.querySelector('.sf-code-language__mark')?.textContent).toBe('C++')
    expect(figure?.querySelectorAll('.sf-code-lines > span')).toHaveLength(2)
    expect(figure?.querySelector('code')?.classList.contains('hljs')).toBe(true)
    expect(figure?.querySelector('.hljs-keyword')).not.toBeNull()

    const copyButton = figure?.querySelector<HTMLButtonElement>('.sf-code-copy')
    copyButton?.click()
    await new Promise(resolve => globalThis.setTimeout(resolve, 0))
    expect(copied).toBe('const answer = 42;\nreturn answer;')
    expect(copySuccesses).toBe(1)
    expect(copyButton?.getAttribute('aria-label')).toBe('已复制')

    expect(enhanceCodeBlocks(container as unknown as ParentNode, options)).toBe(0)
    expect(container.querySelectorAll('.sf-code-block')).toHaveLength(1)
  })

  test('shows an unknown explicit language without guessing a different grammar', () => {
    const browser = new Window({ url: 'http://localhost/' })
    const container = browser.document.createElement('div')
    container.innerHTML = '<pre><code class="language-brainfuck">++[&gt;++&lt;-].</code></pre>'

    expect(enhanceCodeBlocks(container as unknown as ParentNode, { labels })).toBe(1)
    expect(container.querySelector('.sf-code-language strong')?.textContent).toBe('Brainfuck')
    expect(container.querySelector('.sf-code-block')?.getAttribute('data-language')).toBe('brainfuck')
    expect(container.querySelector('.hljs-keyword')).toBeNull()
    expect(container.querySelector('code')?.textContent).toBe('++[>++<-].')
  })

  test('keeps an unspecified code block as honest plain text', () => {
    const browser = new Window({ url: 'http://localhost/' })
    const container = browser.document.createElement('div')
    container.innerHTML = '<pre><code>.steps { color: red; }\nreturn ready;</code></pre>'

    expect(enhanceCodeBlocks(container as unknown as ParentNode, { labels })).toBe(1)
    expect(container.querySelector('.sf-code-language strong')?.textContent).toBe('纯文本')
    expect(container.querySelector('.sf-code-language__mark')?.textContent).toBe('TXT')
    expect(container.querySelector('.sf-code-block')?.getAttribute('data-language')).toBe('plaintext')
    expect(container.querySelector('.hljs-keyword')).toBeNull()
    expect(container.querySelector('code')?.textContent).toBe('.steps { color: red; }\nreturn ready;')
  })
})
