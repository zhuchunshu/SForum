<script setup lang="ts">
import { useSForumSeo } from '~/composables/seo/useSForumSeo'
import { useRegistryPageResolve } from '~/composables/pages/useRegistryPageResolve'
// 子目录组件必须显式导入：Nuxt 自动导入名会带路径前缀，静态契约测试按名字匹配。
import SFRegistryPageBody from '~/components/extensions/public/SFRegistryPageBody.vue'
import SFRegistryPageShell from '~/components/extensions/public/SFRegistryPageShell.vue'

/**
 * 动态公开路由宿主：匹配 Page Registry 的 action=add 贡献。
 * 仅在无其它 Nuxt 文件路由命中时生效（catch-all）。
 * 权威 access/permission 由 API resolve-path 返回 401/403/404，解析与映射见
 * `useRegistryPageResolve`；外壳选择见 `SFRegistryPageShell`。
 */
definePageMeta({
  public: true
})

const route = useRoute()

const requestPath = computed(() => {
  const raw = route.params.sfRegistryPage
  const parts = Array.isArray(raw) ? raw.map(String) : (raw ? [String(raw)] : [])
  // 去掉可选 locale 前缀由 API strip；此处保留完整 path
  return '/' + parts.join('/')
})

const { data, error, status, pending } = await useRegistryPageResolve(requestPath, 'page-resolve-path')

const templateHtml = computed(() => (data.value?.templateHtml || '').trim())

// Page Registry 贡献没有 label 字段；页面可见标题优先从正文首个 h1 提取，
// 没有标题的页面退回路径本身，避免 document.title 出现 page id 噪声。
const pageTitle = computed(() => {
  const heading = templateHtml.value.match(/<h1[^>]*>([\s\S]*?)<\/h1>/i)?.[1]
    ?.replace(/<[^>]*>/g, '').trim()
  return heading || requestPath.value
})

useSForumSeo({
  title: () => pageTitle.value,
  type: 'website',
  noindex: true
})
</script>

<template>
  <!-- 客户端换页的解析窗口：骨架 + 宿主公开 chrome，替代原来的整页空白。 -->
  <div v-if="pending && !data" class="sf-registry-page-pending" aria-busy="true">
    <SFHostPublicChrome>
      <div class="sf-registry-add sf-registry-add--site mx-auto w-full max-w-4xl px-4 py-8">
        <div class="sf-skeleton">
          <div class="sf-skeleton__content">
            <span class="sf-skeleton__line" />
            <span class="sf-skeleton__line" style="width: 62%" />
            <span class="sf-skeleton__line" style="width: 40%" />
          </div>
        </div>
        <div class="sf-skeleton mt-6">
          <div class="sf-skeleton__content">
            <span class="sf-skeleton__line" />
            <span class="sf-skeleton__line" style="width: 55%" />
          </div>
        </div>
      </div>
    </SFHostPublicChrome>
  </div>
  <SFRegistryPageShell
    v-else
    :layout="data?.layout"
    :shell-html="data?.shellHtml"
    :shell-provider="data?.shellProvider"
    :body="{
      loadFailed: Boolean(error && status === 'error'),
      templateHtml,
      renderOutput: data?.renderOutput,
      view: data?.view,
      component: data?.component,
      schemaView: data?.schemaView,
      extensionId: data?.extensionId || data?.provider || '',
      dataSource: data?.dataSource,
      dataRoute: data?.dataRoute,
      loaderData: data?.loaderData,
      loaderError: data?.loaderError,
      fallbackDescription: requestPath
    }"
  />
</template>
