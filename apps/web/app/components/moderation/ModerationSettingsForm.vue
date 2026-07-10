<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import type { ModerationMode, ModerationSettings } from '~/composables/useModerationApi'

const props = defineProps<{ modelValue: ModerationSettings }>()
const emit = defineEmits<{ updated: [settings: ModerationSettings] }>()
const { t } = useI18n()
const toast = useToast()
const moderationApi = useModerationApi()

const form = reactive<ModerationSettings>({ ...props.modelValue })
const saving = ref(false)
const resetting = ref(false)
const errorMessage = ref('')
const modeOptions: Array<{ value: ModerationMode; label: string; description: string }> = [
  { value: 'off', label: t('admin.moderation.mode.off'), description: t('admin.moderation.mode.offDescription') },
  { value: 'rules', label: t('admin.moderation.mode.rules'), description: t('admin.moderation.mode.rulesDescription') },
  { value: 'all', label: t('admin.moderation.mode.all'), description: t('admin.moderation.mode.allDescription') }
]

watch(() => props.modelValue, value => Object.assign(form, value))

async function saveSettings() {
  saving.value = true
  errorMessage.value = ''
  try {
    const updated = await moderationApi.updateSettings({ ...form })
    Object.assign(form, updated)
    emit('updated', updated)
    toast.add({ color: 'primary', icon: 'i-lucide-check', title: t('admin.moderation.settingsSaved'), duration: 10000 })
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('admin.moderation.settingsSaveFailed')
  } finally {
    saving.value = false
  }
}

async function resetSettings() {
  resetting.value = true
  errorMessage.value = ''
  try {
    const updated = await moderationApi.resetSettings()
    Object.assign(form, updated)
    emit('updated', updated)
    toast.add({ color: 'primary', icon: 'i-lucide-rotate-ccw', title: t('admin.moderation.settingsReset'), duration: 10000 })
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('admin.moderation.settingsResetFailed')
  } finally {
    resetting.value = false
  }
}
</script>

<template>
  <section aria-labelledby="moderation-settings-title" class="border-y border-slate-200 py-5 dark:border-zinc-800">
    <div class="mb-4">
      <h2 id="moderation-settings-title" class="text-base font-semibold text-slate-900 dark:text-zinc-100">{{ t('admin.moderation.settingsTitle') }}</h2>
      <p class="mt-1 text-sm text-slate-500 dark:text-zinc-400">{{ t('admin.moderation.settingsDescription') }}</p>
    </div>

    <SFAlert v-if="errorMessage" variant="danger" :title="errorMessage" closable class="mb-4" @close="errorMessage = ''" />

    <div class="grid gap-3 md:grid-cols-3" role="radiogroup" :aria-label="t('admin.moderation.modeLabel')">
      <button
        v-for="option in modeOptions"
        :key="option.value"
        type="button"
        class="min-h-24 border p-4 text-left transition-colors"
        :class="form.mode === option.value ? 'border-[var(--sf-accent)] bg-teal-50/60 dark:bg-teal-950/20' : 'border-slate-200 bg-white hover:border-slate-300 dark:border-zinc-800 dark:bg-zinc-900'"
        role="radio"
        :aria-checked="form.mode === option.value"
        @click="form.mode = option.value"
      >
        <span class="flex items-center gap-2 font-semibold text-slate-900 dark:text-zinc-100">
          {{ option.label }}
          <SFBadge v-if="option.value === 'rules'" variant="info">{{ t('admin.moderation.recommended') }}</SFBadge>
        </span>
        <span class="mt-2 block text-xs leading-5 text-slate-500 dark:text-zinc-400">{{ option.description }}</span>
      </button>
    </div>

    <div v-if="form.mode === 'rules'" class="mt-5 grid gap-4 md:grid-cols-2">
      <label class="flex items-start gap-3 border border-slate-200 p-4 dark:border-zinc-800">
        <UCheckbox v-model="form.reviewNewUsers" class="mt-0.5" />
        <span class="min-w-0">
          <span class="block text-sm font-semibold text-slate-800 dark:text-zinc-200">{{ t('admin.moderation.reviewNewUsers') }}</span>
          <span class="mt-1 block text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.moderation.reviewNewUsersDescription') }}</span>
          <UInput v-if="form.reviewNewUsers" v-model.number="form.newUserMaxAgeDays" type="number" :min="0" :max="3650" class="mt-3 w-32" />
        </span>
      </label>
      <label class="flex items-start gap-3 border border-slate-200 p-4 dark:border-zinc-800">
        <UCheckbox v-model="form.reviewExternalLinks" class="mt-0.5" />
        <span>
          <span class="block text-sm font-semibold text-slate-800 dark:text-zinc-200">{{ t('admin.moderation.reviewExternalLinks') }}</span>
          <span class="mt-1 block text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.moderation.reviewExternalLinksDescription') }}</span>
        </span>
      </label>
    </div>

    <div class="mt-5 flex flex-wrap items-center justify-between gap-3">
      <p class="max-w-2xl text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.moderation.resetScope') }}</p>
      <div class="flex gap-2">
        <UButton color="neutral" variant="subtle" icon="i-lucide-rotate-ccw" :loading="resetting" :disabled="saving" @click="resetSettings">{{ t('admin.moderation.restoreDefaults') }}</UButton>
        <UButton color="primary" icon="i-lucide-save" :loading="saving" :disabled="resetting" @click="saveSettings">{{ t('admin.moderation.saveSettings') }}</UButton>
      </div>
    </div>
  </section>
</template>
