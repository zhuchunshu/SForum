<script setup lang="ts">
/**
 * L1 模板渲染：沙箱 HTML + 允许的宿主岛（sf-*）。
 * 将模板切成 text / island 段，岛映射到 host SF 组件。
 */
const props = defineProps<{
  html: string
  extensionId?: string
  dataSource?: string
  dataRoute?: string
}>()

type Segment =
  | { type: 'html', value: string }
  | { type: 'island', tag: string, attrs: Record<string, string> }

const islandComponents: Record<string, string> = {
  'sf-home-page': 'SFHomePage',
  'sf-navbar': 'SFNavbar',
  'sf-footer': 'SFFooter',
  'sf-extension-widget': 'SFExtensionWidget'
}

const segments = computed(() => parseTemplate(props.html || ''))

const pluginData = ref<unknown>(null)
const pluginDataError = ref(false)

watchEffect(async () => {
  pluginData.value = null
  pluginDataError.value = false
  if (props.dataSource !== 'plugin' || !props.dataRoute || !props.extensionId) {
    return
  }
  try {
    const { request } = useApiClient()
    // 插件数据加载器走扩展路由代理，不绕过权限。
    const path = props.dataRoute.startsWith('/')
      ? props.dataRoute
      : `/${props.dataRoute}`
    pluginData.value = await request(`/extensions/${encodeURIComponent(props.extensionId)}${path}`)
  } catch {
    pluginDataError.value = true
  }
})

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
    const attrRaw = match[2] || ''
    const attrs: Record<string, string> = {}
    const attrRe = /([a-zA-Z0-9:-]+)\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))/g
    let am: RegExpExecArray | null
    while ((am = attrRe.exec(attrRaw)) !== null) {
      const key = am[1]
      if (!key) {
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
        v-html="segment.value"
      />
      <component
        :is="islandName(segment.tag)"
        v-else-if="segment.type === 'island' && islandName(segment.tag)"
        v-bind="segment.attrs"
        :extension-id="segment.attrs['extension-id'] || extensionId"
        :entry="segment.attrs.entry"
        :name="segment.attrs.name"
        :data="pluginData"
      />
      <div
        v-else-if="segment.type === 'island'"
        class="sf-theme-template__unknown-island text-sm text-slate-500"
      >
        Unknown island: {{ segment.tag }}
      </div>
    </template>
  </div>
</template>
