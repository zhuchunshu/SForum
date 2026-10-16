<script setup lang="ts">
import { computed, inject, onBeforeUnmount, ref, watch } from 'vue'

import { PLUGIN_ICON_NAME_PATTERN, PLUGIN_ICON_RESOLVER } from '../../icons/resolver'

/**
 * 宿主图标组件。
 *
 * 插件只提供图标名（`i-lucide-*` / `i-tabler-*`），SVG 标记始终由宿主生成并清洗。
 * 宿主未提供 `icons`、名称非法、或解析失败时渲染空装饰元素并只告警一次，
 * 绝不抛错阻塞页面挂载。
 */
const props = withDefaults(defineProps<{
  name?: string
  size?: 'sm' | 'md' | 'lg'
  /** 提供可读标签时图标成为有意义的图像，否则为装饰性元素。 */
  label?: string
}>(), {
  name: '',
  size: 'md',
  label: ''
})

const resolver = inject(PLUGIN_ICON_RESOLVER, undefined)
const markup = ref('')
const warned = ref(false)
let disposed = false
let generation = 0

const a11y = computed(() => props.label
  ? { role: 'img', 'aria-label': props.label }
  : { 'aria-hidden': 'true' })

async function load(name: string) {
  const current = ++generation
  if (!name) {
    markup.value = ''
    return
  }
  if (!PLUGIN_ICON_NAME_PATTERN.test(name)) {
    warnOnce(`SPluginIcon ignored an unsupported icon name: ${name}`)
    markup.value = ''
    return
  }
  if (!resolver) {
    warnOnce('SPluginIcon has no host icon resolver; icons render as empty placeholders')
    markup.value = ''
    return
  }
  try {
    const resolved = await resolver.resolve([name])
    if (disposed || current !== generation) return
    markup.value = resolved[name] || ''
  } catch {
    if (!disposed && current === generation) {
      markup.value = ''
    }
  }
}

function warnOnce(message: string) {
  if (warned.value) return
  warned.value = true
  console.warn(`[sforum/plugin-ui] ${message}`)
}

watch(() => props.name, (name) => void load(name), { immediate: true })
onBeforeUnmount(() => { disposed = true })
</script>

<template>
  <span
    class="splugin-icon"
    :class="`splugin-icon--${size}`"
    v-bind="a11y"
    v-html="markup"
  />
</template>
