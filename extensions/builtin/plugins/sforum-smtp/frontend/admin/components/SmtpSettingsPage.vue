<script setup lang="ts">
import type { AdminSlotProps } from '@sforum/admin-sdk'
import { computed } from 'vue'
import { useSForumAdminHost } from '@sforum/admin-sdk'

const props = defineProps<AdminSlotProps<'admin.extension.settings.page'>>()
const host = useSForumAdminHost()

const fieldOrder = ['host', 'encryption', 'port', 'username', 'password', 'from_address', 'from_name'] as const
const groupOf: Record<string, 'server' | 'auth' | 'sender'> = {
  host: 'server',
  encryption: 'server',
  port: 'server',
  username: 'auth',
  password: 'auth',
  from_address: 'sender',
  from_name: 'sender'
}

const itemsByKey = computed(() => Object.fromEntries(props.context.items.map(item => [item.key, item])))

const groups = computed(() => {
  const order: Array<'server' | 'auth' | 'sender'> = ['server', 'auth', 'sender']
  return order.map((id) => ({
    id,
    title: host.t(`groups.${id}`),
    keys: fieldOrder.filter(key => groupOf[key] === id && itemsByKey.value[key])
  })).filter(group => group.keys.length)
})

// 与主题设置页一致：字段文案优先 API 已解析的 label/description，host.t 作扩展 locale 回退。
function labelFor(key: string) {
  const fromApi = itemsByKey.value[key]?.label?.trim()
  if (fromApi) {
    return fromApi
  }
  const translated = host.t(`fields.${key}.label`)
  return translated === `fields.${key}.label` ? key : translated
}
function descriptionFor(key: string) {
  const fromApi = itemsByKey.value[key]?.description?.trim()
  if (fromApi) {
    return fromApi
  }
  const translated = host.t(`fields.${key}.description`)
  return translated === `fields.${key}.description` ? '' : translated
}
function valueOf(key: string) {
  return props.context.values[key] ?? itemsByKey.value[key]?.value ?? ''
}
function setValue(key: string, value: string | number) {
  props.context.updateValue(key, String(value ?? ''))
}
function placeholderFor(key: string) {
  const item = itemsByKey.value[key]
  if (item?.type === 'secret' && item.secretSet) {
    return host.t('secretSet')
  }
  return item?.placeholder || ''
}
function encryptionItems() {
  return [
    { value: 'starttls', label: host.t('encryption.starttls') },
    { value: 'tls', label: host.t('encryption.tls') },
    { value: 'none', label: host.t('encryption.none') }
  ]
}
async function onSave() {
  await props.context.save()
  host.toast({ title: host.t('save'), description: host.t('savedHint'), kind: 'success' })
}
</script>

<template>
  <div class="space-y-4">
    <section class="rounded-lg border border-emerald-200 bg-emerald-50/80 p-4 text-sm text-emerald-950 dark:border-emerald-900/60 dark:bg-emerald-950/30 dark:text-emerald-100">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div class="min-w-0">
          <h3 class="text-base font-bold">{{ host.t('recommendedTitle') }}</h3>
          <p class="mt-1 max-w-3xl text-sm text-emerald-800 dark:text-emerald-200">
            {{ host.t('recommendedDescription') }}
          </p>
        </div>
        <UButton
          color="neutral"
          variant="outline"
          leading-icon="i-lucide-mail"
          class="shrink-0 border-slate-200 dark:border-zinc-700"
          @click="context.openMailCenter?.()"
        >
          {{ host.t('openMailCenter') }}
        </UButton>
      </div>
    </section>

    <UAlert
      color="info"
      variant="soft"
      icon="i-lucide-route"
      :title="host.t('flowTitle')"
      :description="host.t('flowDescription')"
    />

    <UCard
      class="border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900"
      :ui="{ footer: 'sticky bottom-0 z-20 border-t border-slate-200 bg-white/95 p-4 backdrop-blur-sm dark:border-zinc-800 dark:bg-zinc-900/95 sm:px-6' }"
    >
      <template #header>
        <div>
          <h2 class="text-base font-bold text-slate-900 dark:text-white">{{ host.t('title') }}</h2>
          <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">{{ host.t('intro') }}</p>
        </div>
      </template>

      <div class="grid max-w-3xl gap-8">
        <section v-for="group in groups" :key="group.id" class="space-y-4">
          <div class="border-b border-slate-200 pb-2 dark:border-zinc-800">
            <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ group.title }}</h3>
          </div>
          <div v-for="key in group.keys" :key="key" class="grid gap-2">
            <UFormField :label="labelFor(key)" :description="descriptionFor(key)" :name="`smtp-${key}`">
              <USelect
                v-if="key === 'encryption'"
                :model-value="valueOf(key)"
                class="w-full"
                value-key="value"
                label-key="label"
                :items="encryptionItems()"
                @update:model-value="setValue(key, $event as string)"
              />
              <UInput
                v-else
                :model-value="valueOf(key)"
                class="w-full"
                :type="key === 'password' ? 'password' : key === 'port' ? 'number' : 'text'"
                :placeholder="placeholderFor(key)"
                @update:model-value="setValue(key, $event as string)"
              />
            </UFormField>
            <div class="flex flex-wrap items-center gap-2 text-xs text-slate-500 dark:text-zinc-400">
              <UBadge color="neutral" variant="outline" class="font-mono">{{ key }}</UBadge>
              <span v-if="itemsByKey[key]?.recommendedValue || itemsByKey[key]?.default">
                {{ host.t('recommended', { value: itemsByKey[key]?.recommendedValue || itemsByKey[key]?.default }) }}
              </span>
              <UBadge v-if="key === 'password' && itemsByKey[key]?.secretSet" color="success" variant="soft">
                {{ host.t('secretConfigured') }}
              </UBadge>
            </div>
          </div>
        </section>
      </div>

      <template #footer>
        <div class="flex w-full items-center justify-between gap-3">
          <span class="text-xs text-slate-500 dark:text-zinc-400">{{ host.t('footerHint') }}</span>
          <div class="flex items-center gap-3">
            <UButton
              type="button"
              color="neutral"
              variant="outline"
              leading-icon="i-lucide-rotate-ccw"
              :loading="context.saving"
              :disabled="context.loading || context.recommendedApplied"
              @click="context.reset()"
            >
              {{ host.t('reset') }}
            </UButton>
            <UButton
              type="button"
              leading-icon="i-lucide-save"
              :loading="context.saving"
              :disabled="context.loading"
              @click="onSave"
            >
              {{ host.t('save') }}
            </UButton>
          </div>
        </div>
      </template>
    </UCard>
  </div>
</template>
