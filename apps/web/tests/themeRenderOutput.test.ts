import { describe, expect, it } from 'bun:test'
import { createSSRApp, defineComponent, h } from 'vue'
import { renderToString } from 'vue/server-renderer'
import {
  parseLegacyThemeHTML,
  parseThemeRenderOutput,
  renderThemeRenderNodes,
  ThemeRenderOutputError,
  type ThemeRenderNode,
  type ThemeRenderOutput
} from '../app/composables/useThemeRenderOutput'

const allowedComponents = new Set(['forum.component.home_page'])

function output(overrides: Partial<ThemeRenderOutput> = {}): ThemeRenderOutput {
  return {
    htmlSegments: [
      '<section class="theme"><header>Before</header><div class="nested"><template data-sforum-island="forum.component.home_page:1"></template></div><footer>After</footer></section>'
    ],
    islands: [{
      id: 'forum.component.home_page:1',
      componentId: 'forum.component.home_page',
      props: [
        { name: 'label', type: 'string', stringValue: 'Typed island' },
        { name: 'compact', type: 'boolean', booleanValue: true },
        { name: 'page', type: 'integer', integerValue: 2 },
        { name: 'link', type: 'url', stringValue: '/topics' }
      ]
    }],
    ...overrides
  }
}

function findIsland(nodes: readonly ThemeRenderNode[]): Extract<ThemeRenderNode, { kind: 'island' }> | undefined {
  for (const node of nodes) {
    if (node.kind === 'island') return node
    if (node.kind === 'element') {
      const nested = findIsland(node.children)
      if (nested) return nested
    }
  }
}

describe('typed theme render output', () => {
  it('preserves nested island position and typed props during SSR', async () => {
    const nodes = parseThemeRenderOutput(output(), { allowedComponents })
    const HostIsland = defineComponent({
      props: {
        label: String,
        compact: Boolean,
        page: Number,
        link: String
      },
      setup(props) {
        return () => h('a', {
          class: props.compact ? 'compact' : '',
          href: props.link,
          'data-page': props.page
        }, props.label)
      }
    })
    const Root = defineComponent({
      setup() {
        return () => h('main', renderThemeRenderNodes(
          nodes,
          componentId => componentId === 'forum.component.home_page' ? HostIsland : undefined
        ))
      }
    })

    const html = await renderToString(createSSRApp(Root))
    expect(html).toContain('<header>Before</header><div class="nested"><a class="compact" href="/topics" data-page="2">Typed island</a></div><footer>After</footer>')
    expect(html).not.toContain('data-sforum-island')
  })

  it('renders an exact typed island fallback through the component slot', async () => {
    const widgetId = 'core.component.shared.sfextension_widget'
    const nodes = parseThemeRenderOutput(output({
      htmlSegments: [`<template data-sforum-island="${widgetId}:1"></template>`],
      islands: [{
        id: `${widgetId}:1`,
        componentId: widgetId,
        props: [],
        fallbackHtmlSegments: ['<article><h2>Indexable fallback</h2><a href="/topics">Topics</a></article>']
      }]
    }), { allowedComponents: new Set([widgetId]), fallbackComponents: new Set([widgetId]) })
    const Widget = defineComponent({
      setup(_, { slots }) {
        return () => h('section', { 'data-widget': '' }, slots.default?.())
      }
    })
    const Root = defineComponent(() => () => h('main', renderThemeRenderNodes(nodes, () => Widget)))

    const html = await renderToString(createSSRApp(Root))
    expect(html).toContain('<section data-widget><article><h2>Indexable fallback</h2><a href="/topics">Topics</a></article></section>')
  })

  it('parses the legacy compatibility path without flattening nested DOM', async () => {
    const nodes = parseLegacyThemeHTML(
      '<article><div><sf-home-page></sf-home-page></div></article>',
      { 'sf-home-page': { componentId: 'forum.component.home_page' } }
    )
    const HostIsland = defineComponent(() => () => h('p', 'legacy island'))
    const Root = defineComponent(() => () => h('main', renderThemeRenderNodes(nodes, () => HostIsland)))

    expect(await renderToString(createSSRApp(Root))).toContain(
      '<article><div><p>legacy island</p></div></article>'
    )
  })

  it('restores Go omitempty zero values from each declared prop type', () => {
    const nodes = parseThemeRenderOutput(output({
      islands: [{
        id: 'forum.component.home_page:1',
        componentId: 'forum.component.home_page',
        props: [
          { name: 'label', type: 'string' },
          { name: 'link', type: 'url' },
          { name: 'compact', type: 'boolean' },
          { name: 'page', type: 'integer' }
        ]
      }]
    }), { allowedComponents })
    const nestedIsland = findIsland(nodes)

    expect(nestedIsland?.kind).toBe('island')
    expect(nestedIsland?.kind === 'island' ? nestedIsland.props : {}).toEqual({
      label: '',
      link: '',
      compact: false,
      page: 0
    })
  })

  it.each([
    {
      name: 'missing descriptor',
      value: output({ islands: [] })
    },
    {
      name: 'descriptor without placeholder',
      value: output({ htmlSegments: ['<p>plain</p>'] })
    },
    {
      name: 'duplicate placeholder',
      value: output({ htmlSegments: [
        '<template data-sforum-island="forum.component.home_page:1"></template><template data-sforum-island="forum.component.home_page:1"></template>'
      ] })
    },
    {
      name: 'forged placeholder element',
      value: output({ htmlSegments: [
        '<div data-sforum-island="forum.component.home_page:1"></div>'
      ] })
    },
    {
      name: 'unknown component',
      value: output({ islands: [{
        id: 'plugin.component.capture:1',
        componentId: 'plugin.component.capture',
        props: []
      }], htmlSegments: ['<template data-sforum-island="plugin.component.capture:1"></template>'] })
    },
    {
      name: 'event attribute',
      value: output({ htmlSegments: [
        '<img src="/safe.png" onerror="alert(1)"><template data-sforum-island="forum.component.home_page:1"></template>'
      ] })
    },
    {
      name: 'unsafe URL',
      value: output({ htmlSegments: [
        '<a href="javascript:alert(1)">bad</a><template data-sforum-island="forum.component.home_page:1"></template>'
      ] })
    },
    {
      name: 'forbidden form boundary',
      value: output({ htmlSegments: [
        '<form action="/login"></form><template data-sforum-island="forum.component.home_page:1"></template>'
      ] })
    },
    {
      name: 'unsafe island fallback',
      value: output({ islands: [{
        id: 'forum.component.home_page:1',
        componentId: 'forum.component.home_page',
        props: [],
        fallbackHtmlSegments: ['<script>alert(1)</script>']
      }] })
    },
    {
      name: 'invalid typed prop',
      value: output({ islands: [{
        id: 'forum.component.home_page:1',
        componentId: 'forum.component.home_page',
        props: [{ name: 'page', type: 'integer', integerValue: Number.MAX_SAFE_INTEGER + 1 }]
      }] })
    },
    {
      name: 'nonzero field from another prop type',
      value: output({ islands: [{
        id: 'forum.component.home_page:1',
        componentId: 'forum.component.home_page',
        props: [{ name: 'compact', type: 'boolean', stringValue: 'true' }]
      }] })
    }
  ])('fails closed for $name', ({ value }) => {
    expect(() => parseThemeRenderOutput(value, { allowedComponents })).toThrow(ThemeRenderOutputError)
  })
})
