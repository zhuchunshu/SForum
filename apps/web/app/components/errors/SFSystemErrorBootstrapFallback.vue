<script setup lang="ts">
import type { NuxtError } from '#app'

/**
 * 错误页的零依赖兜底。
 *
 * i18n 插件失败时 `app/components/errors` 下的组件全部依赖 useI18n，会二次
 * 崩溃并掩盖原始错误。这里只渲染状态码与一句双语提示，不碰 i18n、主题皮肤
 * 或 API，确保"应用启动失败"本身仍然可读。
 */
const props = defineProps<{
  error?: NuxtError | null
}>()

const statusCode = computed(() => {
  const raw = Number(props.error?.statusCode)
  return Number.isFinite(raw) && raw >= 400 && raw <= 599 ? raw : 500
})

// 这里不能用 i18n 或路由相关组合式（本组件就是它们失败后的兜底）。
// 重试做成真链接：入口 chunk 被拒载时页面没有 JS，按钮点击不会有任何反应。
const requestURL = useRequestURL()
const retryHref = `${requestURL.pathname}${requestURL.search}`
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-default p-6 text-default">
    <div class="w-full max-w-lg rounded-lg border border-default p-8">
      <p class="text-sm font-medium text-muted">
        {{ statusCode }}
      </p>
      <h1 class="mt-2 text-2xl font-semibold">
        页面暂时无法加载
      </h1>
      <p class="mt-1 text-sm text-muted">
        This page could not be loaded.
      </p>
      <p class="mt-4 text-sm text-muted">
        应用初始化失败，错误页已降级为最简渲染。原始错误请查看浏览器控制台。
      </p>
      <p class="mt-1 text-sm text-muted">
        App initialization failed; the error page fell back to a minimal render. Check the browser console for the original error.
      </p>
      <a
        :href="retryHref"
        class="mt-6 inline-flex items-center justify-center rounded-md border border-default px-4 py-2 text-sm font-medium"
      >
        重新加载 / Reload
      </a>
    </div>
  </div>
</template>
