<script setup lang="ts">
import type { AdminWebOption } from '~/composables/useWebOptions'
import SFAdminFormFooter from '~/components/admin/SFAdminFormFooter.vue'
import { enabledOptionValue, normalizeEnabledOption } from '~/composables/useWebOptions'
import { adminOptionMap, useAdminOptionTab } from '~/composables/admin/settings/useAdminOptionTab'
import { useSettingsSection } from '~/composables/settings/useSettingsSection'

const props = defineProps<{ items: AdminWebOption[] }>()
const emit = defineEmits<{ saved: [items: AdminWebOption[]] }>()
const { t } = useI18n()
const toast = useToast()
const section = useSettingsSection()
const { saveOptions } = useAdminOptionTab(items => emit('saved', items))
const map = computed(() => adminOptionMap(props.items))

const form = reactive({
  days: 7,
  topicCooldown: 300,
  commentCooldown: 60,
  dailyTopicLimit: 3,
  dailyCommentLimit: 30,
  forbidOutboundLinks: true,
  forbidAttachments: false
})
const initial = computed(() => ({
  days: boundedInteger(map.value['trust.new_user_days']?.value, 7, 0, 365),
  topicCooldown: boundedInteger(map.value['trust.new_user.topic_cooldown_seconds']?.value, 300, 0, 86400),
  commentCooldown: boundedInteger(map.value['trust.new_user.comment_cooldown_seconds']?.value, 60, 0, 86400),
  dailyTopicLimit: boundedInteger(map.value['trust.new_user.daily_topic_limit']?.value, 3, 0, 10000),
  dailyCommentLimit: boundedInteger(map.value['trust.new_user.daily_comment_limit']?.value, 30, 0, 10000),
  forbidOutboundLinks: normalizeEnabledOption(map.value['trust.new_user.forbid_outbound_links']?.value, true),
  forbidAttachments: normalizeEnabledOption(map.value['trust.new_user.forbid_attachments']?.value, false)
}))
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
      form.days = boundedInteger(form.days, 7, 0, 365)
      form.topicCooldown = boundedInteger(form.topicCooldown, 300, 0, 86400)
      form.commentCooldown = boundedInteger(form.commentCooldown, 60, 0, 86400)
      form.dailyTopicLimit = boundedInteger(form.dailyTopicLimit, 3, 0, 10000)
      form.dailyCommentLimit = boundedInteger(form.dailyCommentLimit, 30, 0, 10000)
    },
    save: () => saveOptions([
      { name: 'trust.new_user_days', value: String(form.days) },
      { name: 'trust.new_user.topic_cooldown_seconds', value: String(form.topicCooldown) },
      { name: 'trust.new_user.comment_cooldown_seconds', value: String(form.commentCooldown) },
      { name: 'trust.new_user.daily_topic_limit', value: String(form.dailyTopicLimit) },
      { name: 'trust.new_user.daily_comment_limit', value: String(form.dailyCommentLimit) },
      { name: 'trust.new_user.forbid_outbound_links', value: enabledOptionValue(form.forbidOutboundLinks) },
      { name: 'trust.new_user.forbid_attachments', value: enabledOptionValue(form.forbidAttachments) }
    ])
  })
}

function resetChanges() {
  resetFromItems()
  toast.add({ color: 'neutral', icon: 'i-lucide-rotate-ccw', title: t('admin.settings.newcomers.resetChanges'), duration: 10000 })
}

function restoreRecommended() {
  section.runRestore({
    title: t('admin.settings.newcomers.restoreDefaults'),
    apply: () => Object.assign(form, {
      days: 7,
      topicCooldown: 300,
      commentCooldown: 60,
      dailyTopicLimit: 3,
      dailyCommentLimit: 30,
      forbidOutboundLinks: true,
      forbidAttachments: false
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
          <div><h2 class="text-base font-bold">{{ t('admin.settings.newcomers.title') }}</h2><p class="mt-1 text-xs text-muted">{{ t('admin.settings.newcomers.description') }}</p></div>
          <UBadge color="neutral" variant="soft" class="font-mono">trust.new_user.*</UBadge>
        </div>
      </template>
      <div class="space-y-5">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <UAlert color="neutral" variant="soft" icon="i-lucide-info" :title="t('admin.settings.newcomers.recommended')" class="flex-1" />
          <UButton type="button" color="neutral" variant="outline" leading-icon="i-lucide-rotate-ccw" @click="restoreRecommended">{{ t('admin.settings.newcomers.restoreDefaults') }}</UButton>
        </div>
        <UFormField :label="t('admin.settings.newcomers.days')" :description="t('admin.settings.newcomers.daysHint')" name="trust-days">
          <UInput v-model.number="form.days" type="number" min="0" max="365" required class="w-full max-w-xs" @keydown="blockNonIntegerKey" />
        </UFormField>
        <div class="grid gap-4 md:grid-cols-2">
          <UFormField :label="t('admin.settings.newcomers.topicCooldown')" name="trust-topic-cooldown">
            <UInput v-model.number="form.topicCooldown" type="number" min="0" max="86400" required class="w-full" @keydown="blockNonIntegerKey" /><p class="mt-2 text-xs text-muted">{{ t('admin.settings.newcomers.zeroUnlimited') }}</p>
          </UFormField>
          <UFormField :label="t('admin.settings.newcomers.commentCooldown')" name="trust-comment-cooldown">
            <UInput v-model.number="form.commentCooldown" type="number" min="0" max="86400" required class="w-full" @keydown="blockNonIntegerKey" /><p class="mt-2 text-xs text-muted">{{ t('admin.settings.newcomers.zeroUnlimited') }}</p>
          </UFormField>
          <UFormField :label="t('admin.settings.newcomers.dailyTopicLimit')" name="trust-daily-topic">
            <UInput v-model.number="form.dailyTopicLimit" type="number" min="0" max="10000" required class="w-full" @keydown="blockNonIntegerKey" /><p class="mt-2 text-xs text-muted">{{ t('admin.settings.newcomers.zeroUnlimited') }}</p>
          </UFormField>
          <UFormField :label="t('admin.settings.newcomers.dailyCommentLimit')" name="trust-daily-comment">
            <UInput v-model.number="form.dailyCommentLimit" type="number" min="0" max="10000" required class="w-full" @keydown="blockNonIntegerKey" /><p class="mt-2 text-xs text-muted">{{ t('admin.settings.newcomers.zeroUnlimited') }}</p>
          </UFormField>
        </div>
        <div class="grid gap-3 md:grid-cols-2">
          <label class="flex cursor-pointer items-start gap-3 rounded-md border border-slate-200 bg-slate-50 p-3 text-sm dark:border-zinc-800 dark:bg-zinc-950/60">
            <input v-model="form.forbidOutboundLinks" type="checkbox" class="mt-1 size-4"><span><strong class="block">{{ t('admin.settings.newcomers.forbidOutboundLinks') }}</strong><span class="mt-1 block text-xs text-muted">{{ t('admin.settings.newcomers.forbidOutboundLinksHint') }}</span></span>
          </label>
          <label class="flex cursor-pointer items-start gap-3 rounded-md border border-slate-200 bg-slate-50 p-3 text-sm dark:border-zinc-800 dark:bg-zinc-950/60">
            <input v-model="form.forbidAttachments" type="checkbox" class="mt-1 size-4"><span><strong class="block">{{ t('admin.settings.newcomers.forbidAttachments') }}</strong><span class="mt-1 block text-xs text-muted">{{ t('admin.settings.newcomers.forbidAttachmentsHint') }}</span></span>
          </label>
        </div>
      </div>
      <template #footer>
        <SFAdminFormFooter :saving="section.saving.value" :show-unsaved-alert="hasChanges" :submit-text="t('admin.settings.save')" @reset="resetChanges" />
      </template>
    </UCard>
  </form>
</template>
