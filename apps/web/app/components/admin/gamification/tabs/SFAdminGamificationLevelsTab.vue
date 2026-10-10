<script setup lang="ts">
import type { GamificationLevel } from '~/utils/admin/adminGamification'
import { ADMIN_GAMIFICATION_KEY, type AdminGamificationState } from '~/composables/admin/useAdminGamification'
import { recommendedLevels } from '~/utils/admin/adminGamification'

/** 等级阶梯：min_points → 称号。成长值只升不降，等级查表即得。 */
const gamification = inject(ADMIN_GAMIFICATION_KEY) as AdminGamificationState
const { t } = useI18n()

const creating = ref(false)
const form = reactive({ minPoints: 0, title: '', icon: '' })

const sorted = computed(() =>
  [...(gamification.levels.value ?? [])].sort((a, b) => a.minPoints - b.minPoints)
)

const formError = computed(() => {
  if (!form.title.trim()) return t('admin.gamification.levels.titleRequired')
  if (Number(form.minPoints) < 0) return t('admin.gamification.levels.pointsInvalid')
  return ''
})

const missingRecommended = computed(() =>
  recommendedLevels.filter(level => !sorted.value.some(item => item.minPoints === level.minPoints))
)

function openCreate() {
  creating.value = true
  form.minPoints = 0
  form.title = ''
  form.icon = ''
}

function openEdit(level: GamificationLevel) {
  creating.value = true
  form.minPoints = level.minPoints
  form.title = level.title
  form.icon = level.icon
}

function cancelCreate() {
  creating.value = false
}

// 提交处理器不返回值（表单事件要求 void）。
function onSubmit() {
  void submit()
}

async function submit() {
  if (formError.value) return
  const ok = await gamification.saveLevel({
    minPoints: Number(form.minPoints),
    title: form.title.trim(),
    icon: form.icon.trim()
  })
  if (ok) creating.value = false
}
</script>

<template>
  <UCard class="border-slate-200 bg-white text-slate-900 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100">
    <template #header>
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="min-w-0">
          <h2 class="text-base font-bold">{{ t('admin.gamification.levels.title') }}</h2>
          <p class="mt-1 text-xs text-muted">{{ t('admin.gamification.levels.description') }}</p>
        </div>
        <UButton
          leading-icon="i-lucide-plus"
          :disabled="gamification.saving.value || !gamification.pluginEnabled.value"
          class="bg-[var(--sf-accent)] hover:bg-[var(--sf-accent-hover)] font-semibold text-white"
          @click="openCreate"
        >
          {{ t('admin.gamification.levels.create') }}
        </UButton>
      </div>
    </template>

    <div class="space-y-4">
      <UAlert
        color="info"
        variant="soft"
        icon="i-lucide-info"
        :title="t('admin.gamification.levels.hintTitle')"
        :description="t('admin.gamification.levels.hint')"
      />

      <UAlert
        v-if="missingRecommended.length"
        color="warning"
        variant="soft"
        icon="i-lucide-sparkles"
        :title="t('admin.gamification.levels.missingRecommendedTitle')"
        :description="t('admin.gamification.levels.missingRecommended')"
      />

      <ul class="divide-y divide-slate-200 dark:divide-zinc-800">
        <li
          v-for="level in sorted"
          :key="level.minPoints"
          class="flex items-center justify-between gap-3 py-2.5"
        >
          <div class="flex min-w-0 items-center gap-2">
            <UIcon v-if="level.icon" :name="level.icon" class="size-4 text-muted" />
            <span class="truncate font-medium">{{ level.title }}</span>
          </div>
          <div class="flex items-center gap-2">
            <UBadge color="neutral" variant="soft" class="font-mono">≥ {{ level.minPoints }}</UBadge>
            <UButton
              size="sm"
              color="neutral"
              variant="ghost"
              icon="i-lucide-pencil"
              :aria-label="t('admin.gamification.levels.edit')"
              :title="t('admin.gamification.levels.edit')"
              @click="openEdit(level)"
            />
          </div>
        </li>
        <li v-if="!sorted.length" class="py-6 text-center text-sm text-muted">
          {{ t('admin.gamification.levels.empty') }}
        </li>
      </ul>

      <form v-if="creating" class="space-y-4 border-t border-slate-200 pt-4 dark:border-zinc-800" @submit.prevent="onSubmit">
        <div class="grid gap-4 sm:grid-cols-3">
          <UFormField :label="t('admin.gamification.levels.minPoints')" name="level-min-points" required>
            <UInput v-model.number="form.minPoints" type="number" min="0" class="w-full" />
          </UFormField>
          <UFormField :label="t('admin.gamification.levels.title')" name="level-title" required>
            <UInput v-model="form.title" class="w-full" />
          </UFormField>
          <UFormField :label="t('admin.gamification.levels.icon')" name="level-icon">
            <UInput v-model="form.icon" class="w-full" placeholder="i-tabler-tree" />
          </UFormField>
        </div>
        <UAlert v-if="formError" color="error" variant="soft" icon="i-lucide-triangle-alert" :title="formError" />
        <div class="flex flex-wrap items-center justify-end gap-2">
          <UButton type="button" color="neutral" variant="outline" @click="cancelCreate">
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
    </div>
  </UCard>
</template>
