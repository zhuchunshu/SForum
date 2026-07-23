<script setup lang="ts">
import type { PageResolvePayload } from '~/utils/pageResolve'

const props = defineProps<{
  page: string
  resolved?: PageResolvePayload | null
  resolveError?: unknown
}>()

const provider = computed(() => props.resolved?.provider || 'core')
const hasResolved = computed(() => Boolean(props.resolved))
const templateHtml = computed(() => (props.resolved?.templateHtml || '').trim())
const renderOutput = computed(() => props.resolved?.renderOutput)
const useTemplate = computed(() =>
  provider.value !== 'core'
  && Boolean(renderOutput.value || templateHtml.value)
  && !props.resolved?.fallback
)
const showFallbackNotice = computed(() => Boolean(props.resolved?.fallback || props.resolveError))

const useHostPublicChrome = computed(() => {
  if (!hasResolved.value) {
    return false
  }
  if (useTemplate.value) {
    return false
  }
  const id = props.page
  return !id.startsWith('auth.') && id !== 'system.not_found' && id !== 'dev.components'
})
</script>

<template>
  <div
    class="sf-page-outlet"
    :data-page="page"
    :data-provider="provider"
    :data-template="useTemplate ? '1' : '0'"
    :data-host-chrome="useHostPublicChrome ? '1' : '0'"
  >
    <SFSystemThemeTemplate
      v-if="useTemplate && page === 'system.not_found'"
      :html="templateHtml"
      :render-output="renderOutput"
      :extension-id="resolved?.extensionId || provider"
    />
    <SFThemeTemplate
      v-else-if="useTemplate"
      :html="templateHtml"
      :render-output="renderOutput"
      :extension-id="resolved?.extensionId || provider"
      :data-source="resolved?.dataSource"
      :data-route="resolved?.dataRoute"
      :loader-data="resolved?.loaderData"
      :loader-error="resolved?.loaderError"
    >
      <slot />
    </SFThemeTemplate>
    <SFHostPublicChrome v-else-if="useHostPublicChrome">
      <slot />
    </SFHostPublicChrome>
    <slot v-else />
    <p v-if="showFallbackNotice" class="sr-only">
      page registry fallback to core
    </p>
  </div>
</template>
