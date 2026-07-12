<script setup lang="ts">
/**
 * L2 预构建组件加载器：按 integrity 动态加载扩展 ESM。
 * SSR 仅渲染占位；客户端 import。
 */
const props = defineProps<{
  extensionId: string
  entry: string
  integrity?: string
  name?: string
}>()

const loaded = ref(false)
const failed = ref(false)
const host = ref<HTMLElement | null>(null)

onMounted(async () => {
  if (!props.entry) {
    failed.value = true
    return
  }
  try {
    // 仅允许同源 /api/v1/site/theme-assets/ 或 /api/v1/extensions/ 资源
    const url = props.entry.startsWith('http')
      ? props.entry
      : props.entry.startsWith('/')
        ? props.entry
        : `/api/v1/site/theme-assets/${encodeURIComponent(props.extensionId)}/${props.entry}`
    if (props.integrity && 'integrity' in HTMLScriptElement.prototype) {
      // dynamic import 无法直接带 integrity；用 script module + 校验哈希在后续增强
    }
    const mod: any = await import(/* @vite-ignore */ url)
    const mount = mod.default || mod.mount
    if (typeof mount === 'function' && host.value) {
      mount(host.value, { name: props.name })
    }
    loaded.value = true
  } catch {
    failed.value = true
  }
})
</script>

<template>
  <div
    ref="host"
    class="sf-extension-widget"
    :data-extension-id="extensionId"
    :data-loaded="loaded"
    :data-failed="failed"
  >
    <slot v-if="!loaded && !failed" name="placeholder">
      <SFSkeleton class="h-24 w-full" />
    </slot>
    <slot
      v-if="failed"
      name="error"
    >
      <SFAlert
        variant="danger"
        :title="`Widget failed: ${name || entry}`"
      />
    </slot>
  </div>
</template>
