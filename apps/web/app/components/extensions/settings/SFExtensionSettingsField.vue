<script setup lang="ts">
import type { AdminExtensionSettingValue } from '~/utils/adminExtensions'

const props = defineProps<{
  item: AdminExtensionSettingValue
  modelValue: string
}>()

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const { t } = useI18n()

function secretPlaceholder() {
  if (props.item.type === 'secret' && props.item.secretSet) {
    return t('admin.extensions.dynamic.secretSetPlaceholder')
  }
  return props.item.placeholder
}

// full 占满可用列宽；其余（含省略）保持原先 max-w-xl 的默认控件宽度。
const controlClass = computed(() => (
  props.item.width === 'full' ? 'w-full' : 'w-full max-w-xl'
))
</script>

<template>
  <div class="grid gap-3 px-4 py-4 md:grid-cols-[220px_1fr]">
    <div class="min-w-0">
      <label :for="`extension-setting-${item.key}`" class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
        {{ item.label }}
      </label>
      <p class="mt-1 text-xs leading-5 text-slate-500 dark:text-zinc-400">
        {{ item.description || item.key }}
      </p>
    </div>
    <div class="min-w-0">
      <label
        v-if="item.type === 'boolean'"
        class="inline-flex min-h-10 items-center gap-2 rounded-md border border-slate-200 px-3 text-sm text-slate-700 dark:border-zinc-800 dark:text-zinc-200"
      >
        <input
          :id="`extension-setting-${item.key}`"
          type="checkbox"
          class="size-4 accent-[var(--sf-accent)]"
          :checked="modelValue === 'true'"
          @change="emit('update:modelValue', ($event.target as HTMLInputElement).checked ? 'true' : 'false')"
        >
        <span>{{ t('admin.extensions.dynamic.enabled') }}</span>
      </label>
      <USelect
        v-else-if="item.options?.length"
        :id="`extension-setting-${item.key}`"
        :model-value="modelValue"
        :class="controlClass"
        value-key="value"
        label-key="label"
        :items="item.options"
        :placeholder="item.placeholder"
        @update:model-value="emit('update:modelValue', String($event ?? ''))"
      />
      <UTextarea
        v-else-if="item.type === 'textarea'"
        :id="`extension-setting-${item.key}`"
        :model-value="modelValue"
        :class="controlClass"
        :rows="4"
        autoresize
        :placeholder="item.placeholder"
        @update:model-value="emit('update:modelValue', String($event ?? ''))"
      />
      <UInput
        v-else
        :id="`extension-setting-${item.key}`"
        :model-value="modelValue"
        :class="controlClass"
        :type="item.type === 'secret' ? 'password' : item.type === 'number' ? 'number' : 'text'"
        :placeholder="secretPlaceholder()"
        @update:model-value="emit('update:modelValue', String($event ?? ''))"
      />
      <div class="mt-1 flex flex-wrap items-center gap-2 text-xs text-slate-500 dark:text-zinc-400">
        <span>{{ t('admin.extensions.dynamic.defaultValue', { value: item.recommendedValue || item.default || t('admin.extensions.dynamic.emptyValue') }) }}</span>
        <UBadge v-if="item.type === 'secret' && item.secretSet" color="success" variant="soft" size="sm">
          {{ t('admin.extensions.dynamic.secretConfigured') }}
        </UBadge>
      </div>
    </div>
  </div>
</template>
