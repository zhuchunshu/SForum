<script setup lang="ts">
import type { NuxtError } from '#app'
import type { PageResolvePayload } from '~/utils/pageResolve'
import NotFoundEmergencyPage from './SFNotFoundEmergencyPage.vue'
import SFPageOutlet from './SFPageOutlet.vue'

const props = defineProps<{
  error: NuxtError
  resolvedPage?: PageResolvePayload | null
}>()

const error = computed(() => props.error)

provideNotFoundPageContext(error)

if (import.meta.server) {
  const event = useRequestEvent()
  if (event) {
    setResponseStatus(event, 404)
    setResponseHeader(event, 'cache-control', 'no-store')
    setResponseHeader(event, 'x-robots-tag', 'noindex,nofollow')
    const routeRules = (event.context as { _nitro?: { routeRules?: { cache?: boolean, swr?: boolean } } })._nitro?.routeRules
    if (routeRules) {
      routeRules.cache = false
      routeRules.swr = false
    }
  }
}

</script>

<template>
  <SFPageOutlet v-if="resolvedPage" page="system.not_found" :resolved-payload="resolvedPage">
    <NotFoundEmergencyPage />
  </SFPageOutlet>
  <!-- 客户端解析期间也只展示完整 Core；候选 L0/L1 一致后才会原子切换。 -->
  <NotFoundEmergencyPage v-else />
</template>
