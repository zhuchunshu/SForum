<script setup lang="ts">
import type { NuxtError } from '#app'
import { resolveErrorPageContent } from '~/utils/errorPage'
import { useNotFoundPageResolve } from '~/composables/useNotFoundPageResolve'
import {
  exactThemeIdentityForPageResolve,
  PAGE_RESOLVE_REASON,
  type PageResolveReason
} from '~/utils/pageResolve'
import SFErrorPageContent from './components/SFErrorPageContent.vue'
import SFNotFoundPage from './components/SFNotFoundPage.vue'

const props = defineProps<{
  error: NuxtError
}>()
const nuxtError = computed(() => props.error)
const isNotFound = computed(() => Number(nuxtError.value?.statusCode) === 404)
const { t } = useI18n()
const localeHead = useLocaleHead({ dir: true, lang: true, seo: false })
const {
  siteName,
  resolvedAppearanceTheme,
  refresh: refreshWebOptions
} = useWebOptions()
const {
  refresh: refreshAuthSession,
  status: authStatus,
  lastRefreshError: authRefreshError
} = useAuthSession()
const themeSkin = useActiveThemeSkin()
const startupTimeout = import.meta.dev ? 1000 : 1200
const hasServerSession = import.meta.server
  && /(?:^|;\s*)sforum_session=/.test(useRequestHeaders(['cookie']).cookie || '')
const notFoundContent = computed(() => resolveErrorPageContent(404))
const notFoundTitle = computed(() => t(notFoundContent.value.titleKey, { siteName: siteName.value }))
const notFoundDescription = computed(() => t(notFoundContent.value.descriptionKey, { siteName: siteName.value }))

if (isNotFound.value) {
  useSeoMeta({
    title: notFoundTitle,
    description: notFoundDescription,
    robots: 'noindex,nofollow'
  })

  // Nuxt 的硬导航错误文档不会经过 app.vue，404 必须在根错误页补齐当前主题 L0。
  useHead(() => {
    const htmlAttrs: Record<string, string | undefined> = {
      ...localeHead.value.htmlAttrs,
      'data-sforum-theme': resolvedAppearanceTheme.value.dataTheme
    }
    if (resolvedAppearanceTheme.value.style) {
      htmlAttrs.style = [htmlAttrs.style, resolvedAppearanceTheme.value.style]
        .filter(Boolean)
        .join('; ')
    }
    return {
      htmlAttrs,
      link: [
        ...(localeHead.value.link || []),
        ...themeSkin.links.value.map(href => ({
          rel: 'stylesheet',
          href,
          key: `sforum-theme-skin:${href}`,
          'data-sforum-theme-skin': '1'
        }))
      ],
      meta: localeHead.value.meta
    }
  })
}

// 先确认 system.not_found 可由当前主题解析；失败时不得再启动依赖同一故障 API 的 chrome 请求。
const notFoundResolve = isNotFound.value ? useNotFoundPageResolve() : null
function enterCoreEmergency(reason: PageResolveReason, error?: unknown) {
  themeSkin.clear({ resetIdentity: true })
  notFoundResolve!.useCoreFallback(reason)
  if (error) {
    console.error('[SForum] 404 theme shell unavailable; using Core emergency page', error)
  }
}

async function prepareNotFoundPage() {
  try {
    // SSR 已序列化完整结果时，hydration 必须直接复用同一 provider/artifact/head。
    if (import.meta.client && notFoundResolve!.data.value) {
      return
    }
    if (import.meta.client) {
      // 客户端新进入错误边界时先提交完整 Core，不保留上一页的主题 identity/CSS。
      themeSkin.clear({ resetIdentity: true })
    }

    const resolved = await notFoundResolve!.refresh({ deferCommit: true })
    if (resolved.provider === 'core' || resolved.fallback) {
      themeSkin.clear({ resetIdentity: true })
      notFoundResolve!.commit(resolved)
      if (notFoundResolve!.failure.value) {
        console.error(
          '[SForum] 404 page resolve unavailable; using Core emergency page',
          notFoundResolve!.failure.value
        )
      }
      return
    }

    const expectedIdentity = exactThemeIdentityForPageResolve(resolved)
    if (!expectedIdentity) {
      enterCoreEmergency(PAGE_RESOLVE_REASON.artifactMismatch)
      return
    }

    const skinTask = themeSkin.refresh({
      allowRestore: false,
      apply: false,
      expectedIdentity,
      requireLinks: true
    })
    const optionsTask = refreshWebOptions({
      timeout: startupTimeout,
      serverInternal: import.meta.server
    }).then(
      () => ({ ok: true as const }),
      error => ({ ok: false as const, error })
    )
    const authTask = hasServerSession
      ? refreshAuthSession({
          timeout: startupTimeout,
          serverInternal: true
        }).then(() => authStatus.value === 'unavailable'
          ? { ok: false as const, error: authRefreshError.value }
          : { ok: true as const })
      : Promise.resolve({ ok: true as const })

    const [skinResult, optionsResult, authResult] = await Promise.all([
      skinTask,
      optionsTask,
      authTask
    ])
    if (skinResult.status !== 'success') {
      enterCoreEmergency(
        skinResult.status === 'failed' && skinResult.reason === 'request_failed'
          ? PAGE_RESOLVE_REASON.transportUnavailable
          : PAGE_RESOLVE_REASON.artifactMismatch,
        skinResult.error
      )
      return
    }
    if (!optionsResult.ok) {
      enterCoreEmergency(
        PAGE_RESOLVE_REASON.transportUnavailable,
        optionsResult.error
      )
      return
    }
    if (!authResult.ok) {
      enterCoreEmergency(PAGE_RESOLVE_REASON.transportUnavailable, authResult.error)
      return
    }

    // L0 与 L1 在同一同步批次提交，Vue 不会观察到其中任一半完成态。
    if (!themeSkin.commit(skinResult)) {
      enterCoreEmergency(PAGE_RESOLVE_REASON.artifactMismatch)
      return
    }
    notFoundResolve!.commit(resolved)
  } catch (error) {
    enterCoreEmergency(PAGE_RESOLVE_REASON.transportUnavailable, error)
  }
}
const notFoundStartup = isNotFound.value
  ? prepareNotFoundPage()
  : null
if (import.meta.server && notFoundStartup) {
  onServerPrefetch(() => notFoundStartup)
}
</script>

<template>
  <SFNotFoundPage
    v-if="isNotFound"
    :error="nuxtError"
    :resolved-page="notFoundResolve?.data.value"
  />
  <UApp v-else>
    <SFErrorPageContent :error="nuxtError" />
  </UApp>
</template>
