<script setup lang="ts">
/**
 * 扩展 add 页面宿主：路径 /x/* 映射到 Page Registry 的 add 贡献。
 * 更通用的任意 path add 在后续可通过 Nitro 中间件重写到此宿主。
 */
definePageMeta({
  public: true
})

const route = useRoute()
const pathParts = computed(() => {
  const raw = route.params.path
  if (Array.isArray(raw)) {
    return raw.map(String)
  }
  if (typeof raw === 'string' && raw) {
    return [raw]
  }
  return []
})
const publicPath = computed(() => '/x/' + pathParts.value.join('/'))

const { data, error } = await useAsyncData(
  () => `page-added:${publicPath.value}`,
  async () => {
    const api = useApiClient()
    // 管理端 added 列表需权限；公开侧用 resolve 无法查 add id。
    // 简化：渲染通用扩展页壳，数据由插件 route 加载。
    return { path: publicPath.value }
  }
)

useSForumSeo({
  title: () => publicPath.value,
  type: 'website',
  noindex: true
})
</script>

<template>
  <main class="sf-public-page sf-public-page__container mx-auto w-full max-w-4xl px-4 py-8">
    <SFAlert
      v-if="error"
      variant="danger"
      title="Failed to load extension page"
    />
    <div
      v-else
      class="space-y-4"
    >
      <h1 class="text-xl font-semibold">
        Extension page
      </h1>
      <p class="text-sm text-slate-500 font-mono">
        {{ data?.path }}
      </p>
      <SFExtensionWidget
        extension-id="host"
        entry=""
        name="placeholder"
      />
    </div>
  </main>
</template>
