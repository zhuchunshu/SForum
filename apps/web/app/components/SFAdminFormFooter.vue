<script setup lang="ts">
const { t } = useI18n()

const props = withDefaults(defineProps<{
  saving?: boolean
  disabled?: boolean
  submitText?: string
  resetText?: string
  submitIcon?: string
  resetIcon?: string
  showUnsavedAlert?: boolean
}>(), {
  saving: false,
  disabled: false,
  submitText: undefined,
  resetText: undefined,
  submitIcon: 'i-lucide-save',
  resetIcon: 'i-lucide-rotate-ccw',
  showUnsavedAlert: false
})

const emit = defineEmits<{
  submit: []
  reset: []
}>()
</script>

<template>
  <div class="sticky bottom-0 z-20 mt-8 flex items-center justify-between px-6 py-4 rounded-xl border border-slate-200 dark:border-zinc-800 bg-white/90 dark:bg-zinc-900/90 backdrop-blur-sm shadow-lg transition-all">
    <!-- 左侧插槽或提示语 -->
    <div class="flex items-center gap-2 text-xs sm:text-sm text-slate-500 dark:text-zinc-400">
      <slot name="left">
        <template v-if="showUnsavedAlert">
          <UIcon name="i-lucide-circle-alert" class="size-4 text-amber-500 animate-pulse" />
          <span class="text-amber-600 dark:text-amber-400 font-medium">
            {{ t('admin.form.unsavedChanges') }}
          </span>
        </template>
      </slot>
    </div>

    <!-- 右侧动作按钮区 -->
    <div class="flex items-center gap-3">
      <slot name="actions">
        <!-- 重置按钮 -->
        <UButton
          color="neutral"
          variant="outline"
          :leading-icon="resetIcon"
          :disabled="disabled || saving"
          class="border-slate-200 dark:border-zinc-700 font-medium"
          @click="emit('reset')"
        >
          {{ resetText || t('admin.form.reset') }}
        </UButton>

        <!-- 保存按钮 -->
        <UButton
          type="submit"
          :leading-icon="submitIcon"
          :loading="saving"
          :disabled="disabled"
          class="bg-teal-600 hover:bg-teal-700 dark:bg-teal-500 dark:hover:bg-teal-600 text-white font-semibold"
          @click="emit('submit')"
        >
          {{ submitText || t('admin.form.save') }}
        </UButton>
      </slot>
    </div>
  </div>
</template>
