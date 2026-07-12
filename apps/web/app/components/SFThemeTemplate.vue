<script setup lang="ts">
/**
 * L1 模板渲染：仅消费服务端 bluemonday 净化后的 HTML。
 * 前端不再用正则承担安全边界；仅做宿主岛拆分。
 * L2 widget 岛已从 allowlist 移除，不会挂载可执行组件。
 *
 * 用户内容不得以 raw HTML 进入模板；仅核心安全岛可承载业务数据。
 */
const props = defineProps<{
  html: string
  extensionId?: string
  dataSource?: string
  dataRoute?: string
  /** SSR resolve 注入的插件页面数据（唯一数据来源；禁止客户端再请求插件 route） */
  loaderData?: unknown
  loaderError?: string
}>()

type Segment =
  | { type: 'html', value: string }
  | { type: 'island', tag: string, attrs: Record<string, string> }

/** 与后端 allowedHostIslands 对齐；不含 sf-extension-widget */
// 宿主岛映射：仅指向已存在的核心 SF 组件。
// auth 表单类页面的 replace 必须由宿主页保留 mutation 组件（见 SFPageOutlet constrained）。
const islandComponents: Record<string, string> = {
  'sf-home-page': 'SFHomePage',
  'sf-navbar': 'SFNavbar',
  'sf-footer': 'SFFooter',
  'sf-home-navigation': 'SFHomeNavigation'
}

const ALLOWED_ISLANDS = new Set(Object.keys(islandComponents))

const segments = computed(() => parseTemplate(props.html || ''))

// 仅消费宿主 SSR 注入的 loaderData；前端不得自行访问插件 route。
const pluginData = computed(() => props.loaderData ?? null)
const pluginDataError = computed(() => Boolean(props.loaderError))

/**
 * 在已净化 HTML 上拆分宿主岛。安全边界在 API 的 bluemonday；
 * 此处只做结构拆分，并拒绝未知岛 / 事件属性。
 */
function parseTemplate(source: string): Segment[] {
  const out: Segment[] = []
  const re = /<(sf-[a-z0-9-]+)(\s[^>]*)?\s*(?:\/>|>([\s\S]*?)<\/\1>)/gi
  let last = 0
  let match: RegExpExecArray | null
  while ((match = re.exec(source)) !== null) {
    if (match.index > last) {
      out.push({ type: 'html', value: source.slice(last, match.index) })
    }
    const tagName = match[1]
    if (!tagName) {
      continue
    }
    const tag = tagName.toLowerCase()
    if (!ALLOWED_ISLANDS.has(tag)) {
      // 未知岛丢弃（含已禁用的 L2）
      last = match.index + match[0].length
      continue
    }
    const attrRaw = match[2] || ''
    const attrs: Record<string, string> = {}
    const attrRe = /([a-zA-Z0-9:-]+)\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))/g
    let am: RegExpExecArray | null
    while ((am = attrRe.exec(attrRaw)) !== null) {
      const key = (am[1] || '').toLowerCase()
      if (!key || key.startsWith('on') || key === 'style' || key === 'src' || key === 'href') {
        continue
      }
      attrs[key] = am[2] ?? am[3] ?? am[4] ?? ''
    }
    out.push({ type: 'island', tag, attrs })
    last = match.index + match[0].length
  }
  if (last < source.length) {
    out.push({ type: 'html', value: source.slice(last) })
  }
  return out
}

function islandName(tag: string) {
  return islandComponents[tag] || ''
}

/** SSR 与 client 均直接输出服务端已净化 HTML；不做 client-only 二次改写以免 hydration mismatch */
function safeHtml(value: string) {
  return value
}
</script>

<template>
  <div
    class="sf-theme-template"
    :data-extension-id="extensionId || ''"
  >
    <SFAlert
      v-if="pluginDataError"
      variant="danger"
      class="mb-4"
      title="Plugin page data failed to load"
    />
    <template
      v-for="(segment, index) in segments"
      :key="index"
    >
      <div
        v-if="segment.type === 'html'"
        class="sf-theme-template__html"
        v-html="safeHtml(segment.value)"
      />
      <component
        :is="islandName(segment.tag)"
        v-else-if="segment.type === 'island' && islandName(segment.tag)"
        v-bind="segment.attrs"
        :extension-id="segment.attrs['extension-id'] || extensionId"
        :name="segment.attrs.name"
        :data="pluginData"
      />
    </template>
  </div>
</template>
