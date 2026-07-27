<script setup lang="ts">
/**
 * 公共登录/注册壳的第三方入口。
 * 仅渲染 Host catalog 中已对当前 operation 有效激活的提供方。
 * 展示名/图标来自 catalog（插件声明 → Host 解析），Core 不硬编码供应商品牌。
 */

import type { AuthProviderOperation, PublicAuthProvider } from '~/composables/identity/useAuthProviders'
import { authProviderDisplayMeta } from '~/composables/identity/useAuthProviders'

const props = defineProps<{
  providers: PublicAuthProvider[]
  operation: AuthProviderOperation
  /** 启动中的 provider id；用于禁用按钮与文案。 */
  startingId?: string
  disabled?: boolean
}>()

const emit = defineEmits<{
  start: [provider: PublicAuthProvider]
}>()

const { t } = useI18n()

const rows = computed(() =>
  props.providers.map(provider => ({
    provider,
    meta: authProviderDisplayMeta(provider, t('auth.providers.genericName'))
  }))
)

const hasProviders = computed(() => rows.value.length > 0)

function buttonLabel(meta: ReturnType<typeof authProviderDisplayMeta>, providerId: string) {
  if (props.startingId === providerId) {
    return t('auth.providers.starting')
  }
  // Host 壳通用句式；{name} 来自插件 catalog.label。
  return t('auth.providers.continueWith', { name: meta.label })
}
</script>

<template>
  <div v-if="hasProviders" class="auth-providers" data-testid="auth-provider-buttons">
    <div class="auth-providers__divider" role="separator">
      <span>{{ t('auth.providers.divider') }}</span>
    </div>
    <div class="auth-providers__list">
      <button
        v-for="row in rows"
        :key="row.provider.id"
        type="button"
        class="auth-provider-btn"
        :disabled="disabled || Boolean(startingId)"
        :aria-busy="startingId === row.provider.id ? 'true' : undefined"
        :data-provider-id="row.provider.id"
        @click="emit('start', row.provider)"
      >
        <UIcon :name="row.meta.icon" class="auth-provider-btn__icon" aria-hidden="true" />
        <span>{{ buttonLabel(row.meta, row.provider.id) }}</span>
      </button>
    </div>
  </div>
</template>

<style scoped>
.auth-providers {
  margin-top: 18px;
}

.auth-providers__divider {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 14px;
  color: var(--sf-fg-tertiary);
  font-size: 12px;
  font-weight: 600;
}

.auth-providers__divider::before,
.auth-providers__divider::after {
  content: '';
  flex: 1;
  height: 1px;
  background: var(--sf-border);
}

.auth-providers__list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.auth-provider-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  width: 100%;
  min-height: 40px;
  padding: 0 14px;
  border: 1px solid var(--sf-border);
  border-radius: 7px;
  background: var(--sf-card);
  color: var(--sf-fg);
  font-size: 14px;
  font-weight: 600;
  font-family: inherit;
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s, box-shadow 0.15s;
}

.auth-provider-btn:hover:not(:disabled) {
  border-color: var(--sf-fg-tertiary);
  background: var(--sf-muted);
}

.auth-provider-btn:focus-visible {
  outline: none;
  border-color: var(--sf-accent);
  box-shadow: 0 0 0 3px var(--sf-accent-focus);
}

.auth-provider-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.auth-provider-btn__icon {
  width: 18px;
  height: 18px;
  flex: 0 0 18px;
}

@media (max-width: 720px) {
  .auth-provider-btn {
    min-height: 44px;
  }
}
</style>
