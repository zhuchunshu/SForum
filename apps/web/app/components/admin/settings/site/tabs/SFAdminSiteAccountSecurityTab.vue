<script setup lang="ts">
import type { AdminWebOption } from '~/composables/useWebOptions'
import SFAdminFormFooter from '~/components/admin/SFAdminFormFooter.vue'
import {
  accountSecurityPayload,
  normalizeAccountSecurityForm,
  normalizeAccountSecurityForSave,
  recommendedAccountSecurityForm
} from '~/components/admin/settings/site/models/accountSecurity'
import { useAdminOptionTab } from '~/composables/admin/settings/useAdminOptionTab'
import { useSettingsSection } from '~/composables/settings/useSettingsSection'

const props = defineProps<{ items: AdminWebOption[] }>()
const emit = defineEmits<{ saved: [items: AdminWebOption[]] }>()
const { t } = useI18n()
const toast = useToast()
const section = useSettingsSection()
const saving = section.saving
const { saveOptions } = useAdminOptionTab(items => emit('saved', items))
const form = reactive({ ...recommendedAccountSecurityForm })
const initial = computed(() => normalizeAccountSecurityForm(props.items))
const hasChanges = computed(() => JSON.stringify(form) !== JSON.stringify(initial.value))

watch(() => props.items, resetFromItems, { immediate: true })

function resetFromItems() {
  Object.assign(form, initial.value)
}

async function save() {
  await section.runSave({
    successTitle: t('admin.settings.saved'),
    failureTitle: t('admin.settings.saveFailed'),
    prepare: () => {
      Object.assign(form, normalizeAccountSecurityForSave(form))
    },
    save: () => saveOptions(accountSecurityPayload(form))
  })
}

function resetChanges() {
  resetFromItems()
  toast.add({ color: 'neutral', icon: 'i-lucide-rotate-ccw', title: t('admin.settings.basic.resetAccountSecurityChanges'), duration: 10000 })
}

function restoreRecommended() {
  section.runRestore({
    color: 'neutral',
    title: t('admin.settings.basic.restorePasswordDefaults'),
    apply: () => Object.assign(form, recommendedAccountSecurityForm)
  })
}

function blockNonIntegerKey(event: KeyboardEvent) {
  const allowed = ['Backspace', 'Delete', 'Tab', 'Escape', 'Enter', 'ArrowLeft', 'ArrowRight', 'Home', 'End']
  if (!allowed.includes(event.key) && !event.metaKey && !event.ctrlKey && !/^\d$/.test(event.key)) event.preventDefault()
}
</script>

<template>
  <form class="flex flex-col" @submit.prevent="save">
    <UCard class="border-slate-200 bg-white text-slate-900 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100" :ui="{ footer: 'sticky bottom-0 z-20 bg-white/95 dark:bg-zinc-900/95 backdrop-blur-sm border-t border-slate-200 dark:border-zinc-800 p-4 sm:px-6' }">
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <div>
            <h2 class="text-base font-bold">{{ t('admin.settings.basic.accountSecurityTitle') }}</h2>
            <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.settings.basic.accountSecurityDescription') }}</p>
          </div>
          <UBadge color="neutral" variant="soft" class="border border-slate-200 font-mono dark:border-zinc-800">identity.*</UBadge>
        </div>
      </template>

      <div class="space-y-4">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <UAlert color="neutral" variant="soft" icon="i-lucide-info" :title="t('admin.settings.basic.passwordRecommended')" class="flex-1" />
          <UButton type="button" color="neutral" variant="outline" leading-icon="i-lucide-rotate-ccw" @click="restoreRecommended">{{ t('admin.settings.basic.restorePasswordDefaults') }}</UButton>
        </div>
        <div class="grid gap-4 md:grid-cols-2">
          <UFormField :label="t('admin.settings.basic.passwordMinLength')" name="password-min-length">
            <UInput v-model.number="form.passwordMinLength" size="lg" icon="i-lucide-ruler" type="number" inputmode="numeric" min="8" max="128" step="1" required class="w-full" @keydown="blockNonIntegerKey" />
          </UFormField>
          <UFormField :label="t('admin.settings.basic.passwordMaxLength')" name="password-max-length">
            <UInput v-model.number="form.passwordMaxLength" size="lg" icon="i-lucide-ruler" type="number" inputmode="numeric" min="64" max="512" step="1" required class="w-full" @keydown="blockNonIntegerKey" />
          </UFormField>
        </div>
        <div class="grid gap-3 md:grid-cols-2">
          <label class="flex cursor-pointer items-start gap-3 rounded-md border border-slate-200 bg-slate-50 px-3 py-3 text-sm dark:border-zinc-800 dark:bg-zinc-950/60">
            <input v-model="form.passwordRequireLowercase" type="checkbox" class="mt-1 size-4 rounded border-slate-300 text-[var(--sf-accent)] focus:ring-[var(--sf-accent)]">
            <span class="font-semibold">{{ t('admin.settings.basic.passwordRequireLowercase') }}</span>
          </label>
          <label class="flex cursor-pointer items-start gap-3 rounded-md border border-slate-200 bg-slate-50 px-3 py-3 text-sm dark:border-zinc-800 dark:bg-zinc-950/60">
            <input v-model="form.passwordRequireUppercase" type="checkbox" class="mt-1 size-4 rounded border-slate-300 text-[var(--sf-accent)] focus:ring-[var(--sf-accent)]">
            <span class="font-semibold">{{ t('admin.settings.basic.passwordRequireUppercase') }}</span>
          </label>
          <label class="flex cursor-pointer items-start gap-3 rounded-md border border-slate-200 bg-slate-50 px-3 py-3 text-sm dark:border-zinc-800 dark:bg-zinc-950/60">
            <input v-model="form.passwordRequireNumber" type="checkbox" class="mt-1 size-4 rounded border-slate-300 text-[var(--sf-accent)] focus:ring-[var(--sf-accent)]">
            <span class="font-semibold">{{ t('admin.settings.basic.passwordRequireNumber') }}</span>
          </label>
          <label class="flex cursor-pointer items-start gap-3 rounded-md border border-slate-200 bg-slate-50 px-3 py-3 text-sm dark:border-zinc-800 dark:bg-zinc-950/60">
            <input v-model="form.passwordRequireSymbol" type="checkbox" class="mt-1 size-4 rounded border-slate-300 text-[var(--sf-accent)] focus:ring-[var(--sf-accent)]">
            <span class="font-semibold">{{ t('admin.settings.basic.passwordRequireSymbol') }}</span>
          </label>
        </div>
        <UFormField :label="t('admin.settings.basic.sessionsMaxDevices')" :description="t('admin.settings.basic.sessionsMaxDevicesHint')" name="sessions-max-devices">
          <UInput v-model.number="form.sessionsMaxDevices" size="lg" icon="i-lucide-devices" type="number" inputmode="numeric" min="1" max="20" step="1" required class="w-full max-w-xs" @keydown="blockNonIntegerKey" />
        </UFormField>
        <UFormField :label="t('admin.settings.basic.sessionsKeepDays')" :description="t('admin.settings.basic.sessionsKeepDaysHint')" name="sessions-keep-days">
          <UInput v-model.number="form.sessionsKeepDays" size="lg" icon="i-lucide-calendar-clock" type="number" inputmode="numeric" min="1" max="365" step="1" required class="w-full max-w-xs" @keydown="blockNonIntegerKey" />
        </UFormField>
        <div class="grid gap-4 border-t border-slate-200 pt-4 dark:border-zinc-800 md:grid-cols-2">
          <UFormField :label="t('admin.settings.basic.loginMaxFailures')" :description="t('admin.settings.basic.loginMaxFailuresHint')" name="login-max-failures">
            <UInput v-model.number="form.loginMaxFailures" size="lg" icon="i-lucide-shield-alert" type="number" inputmode="numeric" min="0" max="50" step="1" required class="w-full" @keydown="blockNonIntegerKey" />
          </UFormField>
          <UFormField :label="t('admin.settings.basic.loginLockoutMinutes')" :description="t('admin.settings.basic.loginLockoutMinutesHint')" name="login-lockout-minutes">
            <UInput v-model.number="form.loginLockoutMinutes" size="lg" icon="i-lucide-timer-off" type="number" inputmode="numeric" min="0" max="1440" step="1" required class="w-full" @keydown="blockNonIntegerKey" />
          </UFormField>
        </div>
        <section class="space-y-4 border-t border-slate-200 pt-4 dark:border-zinc-800">
          <div>
            <h3 class="text-sm font-bold">{{ t('admin.settings.basic.mailResendTitle') }}</h3>
            <p class="mt-1 text-xs leading-5 text-slate-500 dark:text-zinc-400">{{ t('admin.settings.basic.mailResendDescription') }}</p>
          </div>
          <UAlert color="neutral" variant="soft" icon="i-lucide-timer-reset" :title="t('admin.settings.basic.mailResendRecommended')" />
          <div class="grid gap-4 md:grid-cols-2">
            <UFormField :label="t('admin.settings.basic.mailResendCooldownSeconds')" :description="t('admin.settings.basic.mailResendCooldownSecondsHint')" name="mail-resend-cooldown-seconds">
              <UInput v-model.number="form.mailResendCooldownSeconds" size="lg" icon="i-lucide-timer" type="number" inputmode="numeric" min="0" max="3600" step="1" required class="w-full" @keydown="blockNonIntegerKey" />
            </UFormField>
            <UFormField :label="t('admin.settings.basic.mailResendWindowMinutes')" :description="t('admin.settings.basic.mailResendWindowMinutesHint')" name="mail-resend-window-minutes">
              <UInput v-model.number="form.mailResendWindowMinutes" size="lg" icon="i-lucide-clock-3" type="number" inputmode="numeric" min="1" max="1440" step="1" required class="w-full" @keydown="blockNonIntegerKey" />
            </UFormField>
            <UFormField :label="t('admin.settings.basic.mailResendMaxPerTarget')" :description="t('admin.settings.basic.mailResendMaxPerTargetHint')" name="mail-resend-max-per-target">
              <UInput v-model.number="form.mailResendMaxPerTarget" size="lg" icon="i-lucide-mail" type="number" inputmode="numeric" min="1" max="100" step="1" required class="w-full" @keydown="blockNonIntegerKey" />
            </UFormField>
            <UFormField :label="t('admin.settings.basic.mailResendMaxPerIP')" :description="t('admin.settings.basic.mailResendMaxPerIPHint')" name="mail-resend-max-per-ip">
              <UInput v-model.number="form.mailResendMaxPerIP" size="lg" icon="i-lucide-network" type="number" inputmode="numeric" min="1" max="1000" step="1" required class="w-full" @keydown="blockNonIntegerKey" />
            </UFormField>
          </div>
        </section>
      </div>

      <template #footer>
        <SFAdminFormFooter :saving="saving" :show-unsaved-alert="hasChanges" :submit-text="t('admin.settings.save')" @reset="resetChanges" />
      </template>
    </UCard>
  </form>
</template>
