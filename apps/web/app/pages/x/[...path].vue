<script setup lang="ts">
import { useSForumSeo } from '~/composables/seo/useSForumSeo'
import { useRegistryPageResolve } from '~/composables/pages/useRegistryPageResolve'
import SFRegistryPageBody from '~/components/extensions/public/SFRegistryPageBody.vue'
import SFRegistryPageShell from '~/components/extensions/public/SFRegistryPageShell.vue'

/**
 * 兼容路径 /x/*：转发到统一 Registry 解析（与根 catch-all 共用同一实现）。
 * 正式 add 路径应使用 manifest 声明的真实 path（由 [...sfRegistryPage] 承接）。
 */
definePageMeta({ public: true })

const route = useRoute()

const requestPath = computed(() => {
  const raw = route.params.path
  const parts = Array.isArray(raw) ? raw.map(String) : (raw ? [String(raw)] : [])
  return '/x/' + parts.join('/')
})

const { data, error, status, pending } = await useRegistryPageResolve(requestPath, 'page-resolve-path-x')

const templateHtml = computed(() => (data.value?.templateHtml || '').trim())

useSForumSeo({
  title: () => data.value?.page?.id || requestPath.value,
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
