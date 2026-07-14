import { describe, expect, it } from 'bun:test'
import { Window } from 'happy-dom'

const browser = new Window({ url: 'http://127.0.0.1/' })
Object.assign(globalThis, {
  window: browser,
  document: browser.document,
  navigator: browser.navigator,
  Node: browser.Node,
  Element: browser.Element,
  HTMLElement: browser.HTMLElement,
  SVGElement: browser.SVGElement,
  ShadowRoot: browser.ShadowRoot,
  Event: browser.Event
})

const { createSSRApp, defineComponent, h, nextTick, ref } = await import('vue')
const { renderToString } = await import('vue/server-renderer')
const { parseThemeRenderOutput, renderThemeRenderNodes } = await import(
  '../app/composables/useThemeRenderOutput'
)

describe('typed theme render hydration', () => {
  it('hydrates the exact SSR tree and keeps host islands interactive', async () => {
    const nodes = parseThemeRenderOutput({
      htmlSegments: [
        '<article><h1>Indexable title</h1><section><template data-sforum-island="forum.component.home_page:1"></template></section></article>'
      ],
      islands: [{
        id: 'forum.component.home_page:1',
        componentId: 'forum.component.home_page',
        props: [{ name: 'label', type: 'string', stringValue: 'Open' }]
      }]
    }, { allowedComponents: new Set(['forum.component.home_page']) })

    const HostIsland = defineComponent({
      props: { label: { type: String, required: true } },
      setup(props) {
        const count = ref(0)
        return () => h('button', {
          type: 'button',
          'data-host-island': '',
          onClick: () => count.value++
        }, `${props.label} ${count.value}`)
      }
    })
    const Root = defineComponent({
      setup() {
        return () => h('main', renderThemeRenderNodes(nodes, () => HostIsland))
      }
    })

    const serverHTML = await renderToString(createSSRApp(Root))
    const container = browser.document.createElement('div')
    container.innerHTML = serverHTML
    browser.document.body.append(container)
    const beforeHydration = container.innerHTML
    const hydrationMessages: string[] = []
    const originalError = console.error
    const originalWarn = console.warn
    console.error = (...args: unknown[]) => hydrationMessages.push(args.join(' '))
    console.warn = (...args: unknown[]) => hydrationMessages.push(args.join(' '))

    try {
      createSSRApp(Root).mount(container)
      expect(container.innerHTML).toBe(beforeHydration)
      expect(hydrationMessages.join('\n')).not.toMatch(/hydration|mismatch/i)

      const button = container.querySelector<HTMLButtonElement>('[data-host-island]')
      expect(button?.textContent).toBe('Open 0')
      button?.click()
      await nextTick()
      expect(button?.textContent).toBe('Open 1')
    } finally {
      console.error = originalError
      console.warn = originalWarn
      container.remove()
    }
  })
})
