<script setup lang="ts">
import type { AdminWebOption } from '~/composables/useWebOptions'
import SFAdminFormFooter from '~/components/admin/SFAdminFormFooter.vue'
import { enabledOptionValue, normalizeEnabledOption } from '~/composables/useWebOptions'
import { adminOptionMap, useAdminOptionTab } from '~/composables/admin/settings/useAdminOptionTab'
import { useSettingsSection } from '~/composables/settings/useSettingsSection'

type RegistrationMode = 'open' | 'invite' | 'approval' | 'closed'

const props = defineProps<{ items: AdminWebOption[] }>()
const emit = defineEmits<{ saved: [items: AdminWebOption[]] }>()
const { t } = useI18n()
const toast = useToast()
const section = useSettingsSection()
const { saveOptions } = useAdminOptionTab(items => emit('saved', items))
const map = computed(() => adminOptionMap(props.items))

const form = reactive({
  registrationMode: 'open' as RegistrationMode,
  requireEmailVerification: false,
  blockPostingUntilVerified: true,
  usernameMinLength: 3,
  usernameMaxLength: 20,
  usernameCharset: 'unicode_letters_numbers' as 'unicode_letters_numbers' | 'ascii',
  usernameReserved: ''
})
const initial = computed(() => {
  const rawMode = (map.value['identity.registration.mode']?.value || 'open').trim()
  const enabled = normalizeEnabledOption(map.value['identity.registration.enabled']?.value, true)
  const registrationMode = (['open', 'invite', 'approval', 'closed'].includes(rawMode) ? rawMode : enabled ? 'open' : 'closed') as RegistrationMode
  return {
    registrationMode,
    requireEmailVerification: normalizeEnabledOption(map.value['identity.registration.require_email_verification']?.value, false),
    blockPostingUntilVerified: normalizeEnabledOption(map.value['identity.registration.block_posting_until_verified']?.value, true),
    usernameMinLength: boundedInteger(map.value['identity.username.min_length']?.value, 3, 2, 32),
    usernameMaxLength: boundedInteger(map.value['identity.username.max_length']?.value, 20, 2, 64),
    usernameCharset: map.value['identity.username.charset']?.value === 'ascii' ? 'ascii' as const : 'unicode_letters_numbers' as const,
    usernameReserved: map.value['identity.username.reserved']?.value || ''
  }
})
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
      form.usernameMinLength = boundedInteger(form.usernameMinLength, 3, 2, 32)
      form.usernameMaxLength = boundedInteger(form.usernameMaxLength, 20, 2, 64)
      if (form.usernameMaxLength < form.usernameMinLength) form.usernameMaxLength = form.usernameMinLength
    },
    save: () => saveOptions([
      { name: 'identity.registration.enabled', value: enabledOptionValue(form.registrationMode === 'open') },
      { name: 'identity.registration.mode', value: form.registrationMode },
      { name: 'identity.registration.require_email_verification', value: enabledOptionValue(form.requireEmailVerification) },
      { name: 'identity.registration.block_posting_until_verified', value: enabledOptionValue(form.blockPostingUntilVerified) },
      { name: 'identity.username.min_length', value: String(form.usernameMinLength) },
      { name: 'identity.username.max_length', value: String(form.usernameMaxLength) },
      { name: 'identity.username.charset', value: form.usernameCharset },
      { name: 'identity.username.reserved', value: form.usernameReserved.trim() }
    ])
  })
}

function resetChanges() {
  resetFromItems()
  toast.add({ color: 'neutral', icon: 'i-lucide-rotate-ccw', title: t('admin.settings.registration.resetChanges'), duration: 10000 })
}

function restoreRecommended() {
  section.runRestore({
    title: t('admin.settings.registration.restoreDefaults'),
    apply: () => Object.assign(form, {
      registrationMode: 'open',
      requireEmailVerification: false,
      blockPostingUntilVerified: true,
      usernameMinLength: 3,
      usernameMaxLength: 20,
      usernameCharset: 'unicode_letters_numbers',
      usernameReserved: 'admin,administrator,system,sforum,root,support,moderator,mod,official,null,undefined'
    })
  })
}

function boundedInteger(value: unknown, fallback: number, min: number, max: number) {
  const parsed = Number(value)
  if (!Number.isFinite(parsed)) return fallback
  const normalized = Math.trunc(parsed)
  return normalized >= min && normalized <= max ? normalized : fallback
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
            <h2 class="text-base font-bold">{{ t('admin.settings.registration.title') }}</h2>
            <p class="mt-1 text-xs text-muted">{{ t('admin.settings.registration.description') }}</p>
          </div>
          <UBadge color="neutral" variant="soft" class="font-mono">identity.registration.* / identity.username.*</UBadge>
        </div>
      </template>

      <div class="space-y-5">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <UAlert color="neutral" variant="soft" icon="i-lucide-info" :title="t('admin.settings.registration.recommended')" class="flex-1" />
          <UButton type="button" color="neutral" variant="outline" leading-icon="i-lucide-rotate-ccw" @click="restoreRecommended">{{ t('admin.settings.registration.restoreDefaults') }}</UButton>
        </div>
        <UFormField :label="t('admin.settings.registration.mode')" name="registration-mode">
          <select v-model="form.registrationMode" class="h-11 w-full rounded-md border border-slate-200 bg-white px-3 dark:border-zinc-700 dark:bg-zinc-950">
            <option value="open">{{ t('admin.settings.registration.modes.open') }}</option>
            <option value="invite">{{ t('admin.settings.registration.modes.invite') }}</option>
            <option value="approval">{{ t('admin.settings.registration.modes.approval') }}</option>
            <option value="closed">{{ t('admin.settings.registration.modes.closed') }}</option>
          </select>
          <p class="mt-2 text-xs text-muted">{{ t('admin.settings.registration.modeHint') }}</p>
        </UFormField>
        <div class="grid gap-3 md:grid-cols-2">
          <label class="flex cursor-pointer items-start gap-3 rounded-md border border-slate-200 bg-slate-50 p-3 text-sm dark:border-zinc-800 dark:bg-zinc-950/60">
            <input v-model="form.requireEmailVerification" type="checkbox" class="mt-1 size-4">
            <span><strong class="block">{{ t('admin.settings.registration.requireEmailVerification') }}</strong><span class="mt-1 block text-xs text-muted">{{ t('admin.settings.registration.requireEmailVerificationHint') }}</span></span>
          </label>
          <label class="flex cursor-pointer items-start gap-3 rounded-md border border-slate-200 bg-slate-50 p-3 text-sm dark:border-zinc-800 dark:bg-zinc-950/60">
            <input v-model="form.blockPostingUntilVerified" type="checkbox" class="mt-1 size-4">
            <span><strong class="block">{{ t('admin.settings.registration.blockPostingUntilVerified') }}</strong><span class="mt-1 block text-xs text-muted">{{ t('admin.settings.registration.blockPostingUntilVerifiedHint') }}</span></span>
          </label>
        </div>
        <section class="space-y-4 border-t border-slate-200 pt-4 dark:border-zinc-800">
          <div>
            <h3 class="text-sm font-semibold">{{ t('admin.settings.registration.usernameTitle') }}</h3>
            <p class="mt-1 text-xs text-muted">{{ t('admin.settings.registration.usernameDescription') }}</p>
          </div>
          <div class="grid gap-4 md:grid-cols-2">
            <UFormField :label="t('admin.settings.registration.usernameMinLength')" name="username-min">
              <UInput v-model.number="form.usernameMinLength" type="number" min="2" max="32" required class="w-full" @keydown="blockNonIntegerKey" />
            </UFormField>
            <UFormField :label="t('admin.settings.registration.usernameMaxLength')" name="username-max">
              <UInput v-model.number="form.usernameMaxLength" type="number" min="2" max="64" required class="w-full" @keydown="blockNonIntegerKey" />
            </UFormField>
          </div>
          <UFormField :label="t('admin.settings.registration.usernameCharset')" name="username-charset">
            <select v-model="form.usernameCharset" class="h-11 w-full rounded-md border border-slate-200 bg-white px-3 dark:border-zinc-700 dark:bg-zinc-950">
              <option value="unicode_letters_numbers">{{ t('admin.settings.registration.charsetUnicode') }}</option>
              <option value="ascii">{{ t('admin.settings.registration.charsetAscii') }}</option>
            </select>
          </UFormField>
          <UFormField :label="t('admin.settings.registration.usernameReserved')" :description="t('admin.settings.registration.usernameReservedHint')" name="username-reserved">
            <UTextarea v-model="form.usernameReserved" :rows="3" class="w-full" :placeholder="t('admin.settings.registration.usernameReservedPlaceholder')" />
          </UFormField>
        </section>
      </div>

      <template #footer>
        <SFAdminFormFooter :saving="section.saving.value" :show-unsaved-alert="hasChanges" :submit-text="t('admin.settings.save')" @reset="resetChanges" />
      </template>
    </UCard>
  </form>
</template>
