<script setup lang="ts">
import type { PageRegionItem } from '~/composables/pages/usePageRegions'
import { safePageRegionHref, usePageRegionsState } from '~/composables/pages/usePageRegions'
import { forumTopicExtensionLabel } from '~/utils/forum/forumTaxonomy'

/**
 * 标准页面区域出口(forum.page.regions)。
 * 只消费 SFPageOutletResolver 在 SSR 阶段写入的共享状态,自身不发请求;
 * widget 的 CSP 已由外层单点聚合,SFExtensionWidget 挂载时按信任授予权威裁决。
 */
const props = defineProps<{
  page: string
  region: string
}>()

const { t, locale } = useI18n()
const localePath = useLocalePath()
const { request } = useApiClient()
const toast = useToast()

const payload = usePageRegionsState(props.page)
const items = computed(() =>
  payload.value?.regions.find(region => region.id === props.region)?.items ?? []
)

const runningKey = ref('')

function itemLabel(item: PageRegionItem) {
  return forumTopicExtensionLabel(item, String(locale.value || 'zh-CN')) || item.contributionId
}

function linkTo(item: PageRegionItem) {
  return safePageRegionHref(item.href) ? localePath(item.href) : ''
}

function actionProxyPath(item: PageRegionItem) {
  const path = `/extensions/${item.extensionId}${item.path}`
  if (path.includes('://') || path.includes('..')) {
    return ''
  }
  return path
}

async function runRegionAction(item: PageRegionItem) {
  const path = actionProxyPath(item)
  const key = `${item.extensionId}:${item.contributionId}`
  if (!path || runningKey.value) {
    return
  }
  runningKey.value = key
  try {
    await request(path, {
      method: item.method as 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
    })
    toast.add({
      color: 'primary',
      icon: item.icon || 'i-lucide-check',
      title: itemLabel(item),
      duration: 10000
    })
  } catch (error) {
    toast.add({
      color: 'error',
      icon: 'i-lucide-alert-circle',
      title: apiErrorMessage(error) || t('regions.actionFailed'),
      // 错误 Toast 不自动关闭（项目约定）
      duration: 0
    })
  } finally {
    runningKey.value = ''
  }
}
</script>

<template>
  <!-- 空区域整体不渲染,保证零内容零布局影响 -->
  <section
    v-if="items.length"
    class="sf-region-outlet"
    :data-page="page"
    :data-region="region"
    :data-testid="`region-outlet-${region}`"
    :aria-label="t('regions.ariaLabel')"
  >
    <template v-for="item in items" :key="`${item.extensionId}:${item.contributionId}`">
      <NuxtLink
        v-if="item.kind === 'link' && linkTo(item)"
        :to="linkTo(item)"
        class="sf-region-outlet__card sf-region-outlet__card--link"
      >
        <UIcon v-if="item.icon" :name="item.icon" class="size-4" aria-hidden="true" />
        <span>{{ itemLabel(item) }}</span>
      </NuxtLink>
      <button
        v-else-if="item.kind === 'action' && actionProxyPath(item)"
        type="button"
        class="sf-region-outlet__card sf-region-outlet__card--action"
        :disabled="runningKey === `${item.extensionId}:${item.contributionId}`"
        @click="runRegionAction(item)"
      >
        <UIcon v-if="item.icon" :name="item.icon" class="size-4" aria-hidden="true" />
        <span>{{ itemLabel(item) }}</span>
      </button>
      <SFExtensionWidget
        v-else-if="item.kind === 'widget' && item.widget"
        :extension-id="item.widget.extensionId"
        :component-id="item.widget.componentId"
        class="sf-region-outlet__widget"
      />
    </template>
  </section>
</template>
