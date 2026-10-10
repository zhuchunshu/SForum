<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'
import type { GamificationBadge, GamificationBadgePayload } from '~/utils/admin/adminGamification'
import { ADMIN_GAMIFICATION_KEY, type AdminGamificationState } from '~/composables/admin/useAdminGamification'
import { badgeRarityColors, describeBadgeRule, recommendedBadgeSlugs } from '~/utils/admin/adminGamification'
import { useSettingsSection } from '~/composables/settings/useSettingsSection'

/** 徽章定义：表格 + 编辑抽屉 + 停用（不删除）+ 一键恢复推荐默认。 */
const gamification = inject(ADMIN_GAMIFICATION_KEY) as AdminGamificationState
const { t } = useI18n()
const toast = useToast()
const section = useSettingsSection()

const editing = ref<GamificationBadge | null>(null)
const creating = ref(false)
const search = ref('')

const filtered = computed(() => {
  const keyword = search.value.trim().toLowerCase()
  if (!keyword) return gamification.badges.value ?? []
  return (gamification.badges.value ?? []).filter(item =>
    item.slug.toLowerCase().includes(keyword) || item.name.toLowerCase().includes(keyword)
  )
})

const columns = computed<TableColumn<GamificationBadge>[]>(() => [
  { accessorKey: 'name', header: t('admin.gamification.badges.name') },
  { accessorKey: 'slug', header: t('admin.gamification.badges.slug') },
  { accessorKey: 'rule', header: t('admin.gamification.badges.rule') },
  { accessorKey: 'grantsPoints', header: t('admin.gamification.badges.points') },
  { accessorKey: 'rarity', header: t('admin.gamification.badges.rarity') },
  { id: 'actions', header: t('admin.gamification.badges.actions') }
])

const form = reactive<GamificationBadgePayload>({
  slug: '',
  name: '',
  description: '',
  rarity: 'common',
  rule: { kind: 'threshold', metric: 'topic_count', threshold: 1 },
  icon: '',
  grantsPoints: 0,
  sort: 0
})

const ruleKind = ref<'threshold' | 'manual'>('threshold')
const ruleMetric = ref<'topic_count' | 'comment_count' | 'registered_days'>('topic_count')
const ruleThreshold = ref(1)

const formError = computed(() => {
  if (!form.slug.trim() || !/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(form.slug.trim())) {
    return t('admin.gamification.badges.slugInvalid')
  }
  if (!form.name.trim()) return t('admin.gamification.badges.nameRequired')
  if (ruleKind.value === 'threshold' && ruleThreshold.value < 1) {
    return t('admin.gamification.badges.thresholdInvalid')
  }
  return ''
})

function openCreate() {
  creating.value = true
  editing.value = null
  form.slug = ''
  form.name = ''
  form.description = ''
  form.rarity = 'common'
  form.icon = ''
  form.grantsPoints = 0
  form.sort = 0
  ruleKind.value = 'threshold'
  ruleMetric.value = 'topic_count'
  ruleThreshold.value = 1
}

function openEdit(badge: GamificationBadge) {
  creating.value = false
  editing.value = badge
  form.slug = badge.slug
  form.name = badge.name
  form.description = badge.description
  form.rarity = badge.rarity
  form.icon = badge.icon
  form.grantsPoints = badge.grantsPoints
  form.sort = badge.sort
  ruleKind.value = badge.rule?.kind ?? 'threshold'
  ruleMetric.value = badge.rule?.metric ?? 'topic_count'
  ruleThreshold.value = badge.rule?.threshold ?? 1
}

function closeForm() {
  creating.value = false
  editing.value = null
}

async function submit() {
  if (formError.value) return
  const payload: GamificationBadgePayload = {
    slug: form.slug.trim(),
    name: form.name.trim(),
    description: form.description?.trim() || '',
    rarity: form.rarity,
    icon: form.icon?.trim() || '',
    grantsPoints: Number(form.grantsPoints) || 0,
    sort: Number(form.sort) || 0,
    rule: ruleKind.value === 'manual'
      ? { kind: 'manual' }
      : { kind: 'threshold', metric: ruleMetric.value, threshold: Number(ruleThreshold.value) }
  }
  const ok = await gamification.saveBadge(payload, editing.value?.slug)
  if (ok) closeForm()
}

async function disable(badge: GamificationBadge) {
  await gamification.disableBadge(badge.slug)
}

async function restoreRecommended() {
  const ok = await gamification.seedDefaults()
  if (ok) {
    toast.add({
      color: 'success',
      icon: 'i-lucide-rotate-ccw',
      title: t('admin.gamification.badges.restoredRecommended'),
      duration: 10000
    })
  }
}

const missingRecommended = computed(() =>
  recommendedBadgeSlugs.filter(slug => !(gamification.badges.value ?? []).some(item => item.slug === slug))
)
</script>

<template>
  <UCard class="border-slate-200 bg-white text-slate-900 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100">
    <template #header>
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="min-w-0">
          <h2 class="text-base font-bold">{{ t('admin.gamification.badges.title') }}</h2>
          <p class="mt-1 text-xs text-muted">{{ t('admin.gamification.badges.description') }}</p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <UButton
            color="neutral"
            variant="outline"
            leading-icon="i-lucide-rotate-ccw"
            :disabled="gamification.saving.value"
            @click="restoreRecommended"
          >
            {{ t('admin.gamification.badges.restoreRecommended') }}
          </UButton>
          <UButton
            leading-icon="i-lucide-plus"
            :disabled="gamification.saving.value || !gamification.pluginEnabled.value"
            class="bg-[var(--sf-accent)] hover:bg-[var(--sf-accent-hover)] font-semibold text-white"
            @click="openCreate"
          >
            {{ t('admin.gamification.badges.create') }}
          </UButton>
        </div>
      </div>
    </template>

    <div class="space-y-4">
      <UAlert
        v-if="missingRecommended.length"
        color="info"
        variant="soft"
        icon="i-lucide-sparkles"
        :title="t('admin.gamification.badges.missingRecommendedTitle')"
        :description="t('admin.gamification.badges.missingRecommended', { count: missingRecommended.length })"
      />

      <UInput
        v-model="search"
        icon="i-lucide-search"
        :placeholder="t('admin.gamification.badges.searchPlaceholder')"
        class="w-full sm:max-w-xs"
      />

      <UTable
        :columns="columns"
        :data="filtered"
        :loading="gamification.pending.value"
        :empty="t('admin.gamification.badges.empty')"
      >
        <template #name-cell="{ row }">
          <div class="flex items-center gap-2">
            <UIcon v-if="row.original.icon" :name="row.original.icon" class="size-4 text-muted" />
            <span class="font-medium">{{ row.original.name }}</span>
            <UBadge v-if="row.original.disabled" color="neutral" variant="soft" size="sm">
              {{ t('admin.gamification.badges.disabledBadge') }}
            </UBadge>
          </div>
        </template>
        <template #slug-cell="{ row }">
          <span class="font-mono text-xs text-muted">{{ row.original.slug }}</span>
        </template>
        <template #rule-cell="{ row }">
          <span class="text-xs">{{ describeBadgeRule(row.original.rule, t) }}</span>
        </template>
        <template #rarity-cell="{ row }">
          <UBadge :color="badgeRarityColors[row.original.rarity]" variant="soft" size="sm">
            {{ t(`admin.gamification.rarity.${row.original.rarity}`) }}
          </UBadge>
        </template>
        <template #actions-cell="{ row }">
          <div class="flex items-center gap-1">
            <UButton
              size="sm"
              color="neutral"
              variant="ghost"
              icon="i-lucide-pencil"
              :aria-label="t('admin.gamification.badges.edit')"
              :title="t('admin.gamification.badges.edit')"
              @click="openEdit(row.original)"
            />
            <UButton
              size="sm"
              color="neutral"
              variant="ghost"
              icon="i-lucide-archive"
              :disabled="row.original.disabled"
              :aria-label="t('admin.gamification.badges.disable')"
              :title="t('admin.gamification.badges.disable')"
              @click="disable(row.original)"
            />
          </div>
        </template>
      </UTable>
    </div>

    <!-- 新建 / 编辑表单 -->
    <template v-if="creating || editing" #footer>
      <form class="space-y-4 border-t border-slate-200 pt-4 dark:border-zinc-800" @submit.prevent="submit">
        <div class="grid gap-4 sm:grid-cols-2">
          <UFormField :label="t('admin.gamification.badges.slug')" name="badge-slug" required>
            <UInput v-model="form.slug" :disabled="!!editing" class="w-full" placeholder="first-topic" />
          </UFormField>
          <UFormField :label="t('admin.gamification.badges.name')" name="badge-name" required>
            <UInput v-model="form.name" class="w-full" />
          </UFormField>
          <UFormField :label="t('admin.gamification.badges.description')" name="badge-description" class="sm:col-span-2">
            <UInput v-model="form.description" class="w-full" />
          </UFormField>
          <UFormField :label="t('admin.gamification.badges.rarity')" name="badge-rarity">
            <USelect
              v-model="form.rarity"
              :items="['common', 'rare', 'epic', 'legendary'].map(value => ({ value, label: t(`admin.gamification.rarity.${value}`) }))"
              class="w-full"
            />
          </UFormField>
          <UFormField :label="t('admin.gamification.badges.icon')" name="badge-icon">
            <UInput v-model="form.icon" class="w-full" placeholder="i-tabler-award" />
          </UFormField>
        </div>

        <div class="grid gap-4 sm:grid-cols-3">
          <UFormField :label="t('admin.gamification.badges.ruleKind')" name="badge-rule-kind">
            <USelect
              v-model="ruleKind"
              :items="[
                { value: 'threshold', label: t('admin.gamification.rule.kindThreshold') },
                { value: 'manual', label: t('admin.gamification.rule.kindManual') }
              ]"
              class="w-full"
            />
          </UFormField>
          <UFormField v-if="ruleKind === 'threshold'" :label="t('admin.gamification.badges.metric')" name="badge-metric">
            <USelect
              v-model="ruleMetric"
              :items="[
                { value: 'topic_count', label: t('admin.gamification.rule.metricTopic') },
                { value: 'comment_count', label: t('admin.gamification.rule.metricComment') },
                { value: 'registered_days', label: t('admin.gamification.rule.metricDays') }
              ]"
              class="w-full"
            />
          </UFormField>
          <UFormField v-if="ruleKind === 'threshold'" :label="t('admin.gamification.badges.threshold')" name="badge-threshold">
            <UInput v-model.number="ruleThreshold" type="number" min="1" class="w-full" />
          </UFormField>
        </div>

        <div class="grid gap-4 sm:grid-cols-2">
          <UFormField :label="t('admin.gamification.badges.points')" name="badge-points">
            <UInput v-model.number="form.grantsPoints" type="number" min="0" class="w-full" />
          </UFormField>
          <UFormField :label="t('admin.gamification.badges.sort')" name="badge-sort">
            <UInput v-model.number="form.sort" type="number" min="0" class="w-full" />
          </UFormField>
        </div>

        <UAlert v-if="formError" color="error" variant="soft" icon="i-lucide-triangle-alert" :title="formError" />

        <div class="flex flex-wrap items-center justify-end gap-2">
          <UButton type="button" color="neutral" variant="outline" :disabled="section.saving.value" @click="closeForm">
            {{ t('admin.form.cancel') }}
          </UButton>
          <UButton
            type="submit"
            leading-icon="i-lucide-save"
            :loading="gamification.saving.value"
            :disabled="!!formError"
            class="bg-[var(--sf-accent)] hover:bg-[var(--sf-accent-hover)] font-semibold text-white"
          >
            {{ t('admin.settings.save') }}
          </UButton>
        </div>
      </form>
    </template>
  </UCard>
</template>
