<script setup lang="ts">
import SFButton from '~/components/SFButton.vue'
import { useContentLinkPolicy } from '~/composables/forum/useContentLinkPolicy'
import {
  decodeOutboundTarget,
  isOffsiteContentHref,
  isTrustedOutboundHost,
  outboundTargetHost
} from '~/utils/forum/forumOutboundLinks'

/**
 * 站外链接中转页（Host 安全面）。
 *
 * 形态契约：
 * - 「无需提示」的三种情况（运营关掉中转 / 命中免中转白名单 / 目标其实就是本站）
 *   在 SSR 阶段直接 302 放行，不渲染中间页；
 * - 其余情况必须由用户显式确认，「继续访问」是普通 <a>（无 JS 也能用），
 *   且为同标签页跳转——新标签页语义已在正文链接上生效；
 * - 页面自身不索引、不外链式转发，只展示目的地域名与完整地址。
 */
const route = useRoute()
const localePath = useLocalePath()
const { t } = useI18n()
const toast = useToast()
const { contentLinkPolicy } = useContentLinkPolicy()
// 站点品牌取后台设置（logo/favicon 与站点名），与导航同一来源，不用硬编码徽章。
const { siteLogoUrl, siteName } = useWebOptions()

const target = computed(() => decodeOutboundTarget(route.query.u))
const targetHost = computed(() => (target.value ? outboundTargetHost(target.value) : ''))
const targetIsSecure = computed(() => Boolean(target.value?.startsWith('https://')))
const currentHost = computed(() => (import.meta.client ? window.location.host : ''))

/**
 * 是否需要停在提示页。默认 true（配置未加载时宁可多一次提示，也不静默放行）。
 */
const requiresNotice = computed(() => {
  if (!target.value) {
    return false
  }
  const policy = contentLinkPolicy.value
  if (!policy.interstitial) {
    return false
  }
  if (isTrustedOutboundHost(targetHost.value, policy.trustedDomains)) {
    return false
  }
  return isOffsiteContentHref(target.value, policy.internalHosts)
})

// 无需提示：服务端直接放行（302），客户端导航到本页时同样立刻跳转。
if (target.value && !requiresNotice.value) {
  await navigateTo(target.value, { external: true, redirectCode: 302 })
}

async function copyTarget() {
  const value = target.value
  if (!value || !import.meta.client) {
    return
  }
  try {
    await navigator.clipboard?.writeText(value)
    toast.add({
      color: 'neutral',
      icon: 'i-lucide-clipboard-check',
      title: t('outbound.copySuccessTitle'),
      duration: 10000
    })
  } catch {
    // 剪贴板被拒绝时保持页面可见，用户仍可手动选中地址复制。
  }
}
</script>

<template>
  <main class="sforum-outbound" data-outbound-body="host">
    <!-- 非法/缺失目标：不展示任何可跳转内容，只给返回入口。 -->
    <section v-if="!target" class="sforum-outbound__panel" data-outbound-state="invalid">
      <UIcon name="i-lucide-unlink" class="sforum-outbound__invalid-icon" />
      <h1 class="sforum-outbound__title">{{ t('outbound.invalidTitle') }}</h1>
      <p class="sforum-outbound__lead">{{ t('outbound.invalidBody') }}</p>
      <div class="sforum-outbound__actions">
        <a class="sf-button sf-button--primary sf-button--md sforum-outbound__action" :href="localePath('/')">
          <span class="sf-button__label">{{ t('outbound.invalidBackLabel') }}</span>
        </a>
      </div>
    </section>

    <template v-else>
      <p class="sforum-outbound__strip" role="status">
        <UIcon name="i-lucide-triangle-alert" class="sforum-outbound__strip-icon" />
        <span class="sforum-outbound__strip-title">{{ t('outbound.stripTitle') }}</span>
        <span class="sforum-outbound__strip-hint">{{ t('outbound.stripHint') }}</span>
      </p>

      <section class="sforum-outbound__panel" data-outbound-state="notice">
        <h1 class="sforum-outbound__title">{{ t('outbound.noticeTitle') }}</h1>
        <p class="sforum-outbound__lead">{{ t('outbound.noticeLead') }}</p>

        <div class="sforum-outbound__compare">
          <div class="sforum-outbound__side sforum-outbound__side--origin">
            <p class="sforum-outbound__side-label">{{ t('outbound.originLabel') }}</p>
            <p class="sforum-outbound__side-host">
              <span class="sforum-outbound__brand" aria-hidden="true">
                <img v-if="siteLogoUrl" :src="siteLogoUrl" alt="" class="sforum-outbound__brand-image">
                <UIcon v-else name="i-tabler-message-circle-filled" class="sforum-outbound__brand-icon" />
              </span>
              {{ siteName }}
            </p>
            <p class="sforum-outbound__side-sub">{{ currentHost || t('outbound.originHostFallback') }}</p>
            <ul class="sforum-outbound__side-list">
              <li>{{ t('outbound.originLinkNote') }}</li>
              <li>{{ t('outbound.originSessionNote') }}</li>
            </ul>
          </div>
          <div class="sforum-outbound__side sforum-outbound__side--destination">
            <p class="sforum-outbound__side-label">{{ t('outbound.destinationLabel') }}</p>
            <p class="sforum-outbound__side-host">
              <span class="sforum-outbound__side-icon" aria-hidden="true">
                <UIcon name="i-lucide-triangle-alert" />
              </span>
              {{ targetHost }}
            </p>
            <p class="sforum-outbound__side-sub">
              <span class="sforum-outbound__chip">{{ t('outbound.externalChip') }}</span>
            </p>
            <ul class="sforum-outbound__side-list">
              <li>{{ t('outbound.destinationRuleNote') }}</li>
              <li>{{ t('outbound.destinationPrivacyNote') }}</li>
            </ul>
          </div>
        </div>

        <p class="sforum-outbound__url">{{ target }}</p>

        <ul class="sforum-outbound__checks">
          <li>
            <UIcon :name="targetIsSecure ? 'i-lucide-shield-check' : 'i-lucide-shield-alert'" :class="['sforum-outbound__check-icon', targetIsSecure ? '' : 'sforum-outbound__check-icon--warn']" />
            {{ targetIsSecure ? t('outbound.checkSecure') : t('outbound.checkInsecure') }}
          </li>
          <li>
            <UIcon name="i-lucide-shield-check" class="sforum-outbound__check-icon" />
            {{ t('outbound.checkNoSession') }}
          </li>
          <li>
            <UIcon name="i-lucide-shield-check" class="sforum-outbound__check-icon" />
            {{ t('outbound.checkNoCredentials') }}
          </li>
        </ul>

        <div class="sforum-outbound__actions">
          <!-- 同标签页跳转；新标签页语义已经在正文链接上生效。 -->
          <a
            class="sf-button sf-button--primary sf-button--md sforum-outbound__action"
            :href="target"
            rel="nofollow noopener noreferrer"
          >
            <span class="sf-button__label">{{ t('outbound.continueLabel') }}</span>
            <span class="sf-button__icon"><UIcon name="i-lucide-external-link" /></span>
          </a>
          <a class="sf-button sf-button--ghost sf-button--md sforum-outbound__action" :href="localePath('/')">
            <span class="sf-button__label">{{ t('outbound.stayLabel') }}</span>
          </a>
          <SFButton variant="ghost" size="md" @click="copyTarget">
            <template #leading><UIcon name="i-lucide-copy" /></template>
            {{ t('outbound.copyLabel') }}
          </SFButton>
        </div>

        <p class="sforum-outbound__foot">{{ t('outbound.footerNote') }}</p>
      </section>
    </template>
  </main>
</template>

<style scoped>
.sforum-outbound {
  width: 100%;
  max-width: 780px;
  margin: 0 auto;
  padding: 20px 16px 48px;
}

/* 警示条：左侧强调条 + 加粗标题，第一眼就能看出「要离开本站了」。 */
.sforum-outbound__strip {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 1rem;
  padding: 0.6875rem 0.875rem;
  border: 1px solid #fde68a;
  border-left: 4px solid var(--sf-warning);
  border-radius: 10px;
  background: var(--sf-warning-soft);
  color: var(--sf-warning);
  font-size: 0.85rem;
}

.sforum-outbound__strip-icon {
  flex: 0 0 auto;
  font-size: 1.1rem;
}

.sforum-outbound__strip-title {
  font-weight: 700;
}

.sforum-outbound__strip-hint {
  margin-left: auto;
  font-weight: 500;
  opacity: 0.85;
}

.sforum-outbound__panel {
  position: relative;
  padding: 1.5rem;
  border: 1px solid var(--sf-border);
  border-top: 3px solid var(--sf-warning);
  border-radius: 16px;
  background: var(--sf-card);
  box-shadow: 0 8px 24px rgb(15 23 42 / 0.06);
}

.sforum-outbound__invalid-icon {
  font-size: 1.6rem;
  color: var(--sf-accent);
}

.sforum-outbound__title {
  margin: 0;
  font-size: 1.3rem;
  font-weight: 750;
  letter-spacing: -0.015em;
}

.sforum-outbound__lead {
  margin: 0.4rem 0 0;
  color: var(--sf-fg-tertiary);
  font-size: 0.875rem;
}

/* 双栏对比：来源侧用品牌色，目标侧用警示色，颜色本身就在说明问题。 */
.sforum-outbound__compare {
  display: grid;
  grid-template-columns: 1fr 1fr;
  margin-top: 1.25rem;
  border: 1px solid var(--sf-border);
  border-radius: 12px;
  overflow: hidden;
}

.sforum-outbound__side {
  padding: 1rem;
  min-width: 0;
}

.sforum-outbound__side--origin {
  background: var(--sf-accent-soft);
}

.sforum-outbound__side--destination {
  border-left: 1px solid var(--sf-border);
  background: var(--sf-warning-soft);
}

.sforum-outbound__side-label {
  margin: 0 0 0.5rem;
  color: var(--sf-fg-tertiary);
  font-size: 0.6875rem;
  font-weight: 700;
  letter-spacing: 0.08em;
}

.sforum-outbound__side-host {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin: 0;
  font-size: 1rem;
  font-weight: 700;
  overflow-wrap: anywhere;
}

.sforum-outbound__side--destination .sforum-outbound__side-host {
  color: var(--sf-warning);
}

.sforum-outbound__side-sub {
  margin: 0.35rem 0 0;
  color: var(--sf-fg-tertiary);
  font-size: 0.75rem;
  overflow-wrap: anywhere;
}

.sforum-outbound__brand {
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 1.5rem;
  height: 1.5rem;
  overflow: hidden;
  border-radius: 7px;
  color: var(--sf-accent);
}

.sforum-outbound__brand-image {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.sforum-outbound__brand-icon {
  font-size: 1.25rem;
}

.sforum-outbound__side-icon {
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 1.5rem;
  height: 1.5rem;
  border-radius: 7px;
  background: var(--sf-card);
  color: var(--sf-warning);
  font-size: 1rem;
}

.sforum-outbound__chip {
  display: inline-block;
  padding: 0.125rem 0.5rem;
  border: 1px solid #fde68a;
  border-radius: 999px;
  background: var(--sf-card);
  color: var(--sf-warning);
  font-size: 0.6875rem;
  font-weight: 700;
  letter-spacing: 0.02em;
}

.sforum-outbound__side-list {
  margin: 0.7rem 0 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
  color: var(--sf-fg-tertiary);
  font-size: 0.78rem;
}

.sforum-outbound__side-list li::before {
  content: "·";
  margin-right: 0.4rem;
}

.sforum-outbound__url {
  margin: 1rem 0 0;
  padding: 0.6rem 0.7rem;
  border: 1px solid var(--sf-border);
  border-radius: 9px;
  background: var(--sf-muted);
  color: var(--sf-fg-secondary);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.75rem;
  overflow-wrap: anywhere;
}

.sforum-outbound__checks {
  margin: 1rem 0 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  color: var(--sf-fg-secondary);
  font-size: 0.8rem;
}

.sforum-outbound__checks li {
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
}

.sforum-outbound__check-icon {
  flex: 0 0 auto;
  margin-top: 0.2rem;
  color: var(--sf-success);
}

.sforum-outbound__check-icon--warn {
  color: var(--sf-warning);
}

.sforum-outbound__actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.625rem;
  margin-top: 1.25rem;
}

.sforum-outbound__action {
  text-decoration: none;
}

.sforum-outbound__foot {
  margin: 1rem 0 0;
  padding-top: 0.875rem;
  border-top: 1px dashed var(--sf-border);
  color: var(--sf-fg-tertiary);
  font-size: 0.75rem;
}

@media (max-width: 640px) {
  .sforum-outbound__compare {
    grid-template-columns: minmax(0, 1fr);
  }

  .sforum-outbound__side--destination {
    border-left: none;
    border-top: 1px solid var(--sf-border);
  }

  .sforum-outbound__strip-hint,
  .sforum-outbound__side-list {
    display: none;
  }
}
</style>
