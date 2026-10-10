<script setup lang="ts">
import type { GamificationGrant } from '~/utils/admin/adminGamification'
import { ADMIN_GAMIFICATION_KEY, type AdminGamificationState } from '~/composables/admin/useAdminGamification'

/**
 * 手动授予 / 撤销 / 授予日志。
 * threshold 徽章不由这里授予（引擎按规则自动发），只有 manual 徽章可选。
 */
const gamification = inject(ADMIN_GAMIFICATION_KEY) as AdminGamificationState
const { t } = useI18n()

const userId = ref('')
const badgeSlug = ref('')
const grants = ref<GamificationGrant[]>([])
const loadingLog = ref(false)
const searchedUser = ref<number | null>(null)

const manualBadges = computed(() =>
  (gamification.badges.value ?? []).filter(item => !item.disabled && item.rule?.kind === 'manual')
)

const canGrant = computed(() => {
  const id = Number(userId.value)
  return Number.isInteger(id) && id > 0 && !!badgeSlug.value
})

async function grant() {
  if (!canGrant.value) return
  const ok = await gamification.grantBadge(Number(userId.value), badgeSlug.value)
  if (ok && searchedUser.value === Number(userId.value)) {
    await loadLog()
  }
}

async function loadLog() {
  const id = Number(userId.value)
  if (!Number.isInteger(id) || id <= 0) return
  loadingLog.value = true
  searchedUser.value = id
  grants.value = await gamification.loadUserBadges(id)
  loadingLog.value = false
}

async function revoke(grant: GamificationGrant) {
  const ok = await gamification.revokeBadge(grant.userId, grant.badgeSlug)
  if (ok) await loadLog()
}

function formatTime(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <UCard class="border-slate-200 bg-white text-slate-900 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100">
      <template #header>
        <div class="min-w-0">
          <h2 class="text-base font-bold">{{ t('admin.gamification.grants.title') }}</h2>
          <p class="mt-1 text-xs text-muted">{{ t('admin.gamification.grants.description') }}</p>
        </div>
      </template>

      <div class="space-y-4">
        <UAlert
          color="info"
          variant="soft"
          icon="i-lucide-award"
          :title="t('admin.gamification.grants.hintTitle')"
          :description="t('admin.gamification.grants.hint')"
        />

        <div class="grid gap-4 sm:grid-cols-3">
          <UFormField :label="t('admin.gamification.grants.userId')" name="grant-user" required>
            <UInput v-model="userId" type="number" min="1" class="w-full" placeholder="42" />
          </UFormField>
          <UFormField :label="t('admin.gamification.grants.badge')" name="grant-badge" required>
            <USelect
              v-model="badgeSlug"
              :items="manualBadges.map(item => ({ value: item.slug, label: item.name }))"
              class="w-full"
              :placeholder="t('admin.gamification.grants.selectBadge')"
            />
          </UFormField>
          <div class="flex items-end gap-2">
            <UButton
              leading-icon="i-lucide-gift"
              :disabled="!canGrant || gamification.saving.value"
              class="bg-[var(--sf-accent)] hover:bg-[var(--sf-accent-hover)] font-semibold text-white"
              @click="grant"
            >
              {{ t('admin.gamification.grants.grant') }}
            </UButton>
            <UButton
              color="neutral"
              variant="outline"
              leading-icon="i-lucide-list"
              :loading="loadingLog"
              :disabled="!userId"
              @click="loadLog"
            >
              {{ t('admin.gamification.grants.loadLog') }}
            </UButton>
          </div>
        </div>

        <p v-if="!manualBadges.length" class="text-xs text-muted">
          {{ t('admin.gamification.grants.noManualBadges') }}
        </p>
      </div>
    </UCard>

    <UCard
      v-if="searchedUser !== null"
      class="border-slate-200 bg-white text-slate-900 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100"
    >
      <template #header>
        <div class="min-w-0">
          <h2 class="text-base font-bold">{{ t('admin.gamification.grants.logTitle') }}</h2>
          <p class="mt-1 text-xs text-muted">
            {{ t('admin.gamification.grants.logDescription', { userId: searchedUser }) }}
          </p>
        </div>
      </template>

      <ul class="divide-y divide-slate-200 dark:divide-zinc-800">
        <li v-for="grant in grants" :key="`${grant.badgeSlug}-${grant.grantedAt}`" class="flex flex-wrap items-center justify-between gap-3 py-2.5">
          <div class="flex min-w-0 items-center gap-2">
            <span class="font-mono text-xs">{{ grant.badgeSlug }}</span>
            <UBadge :color="grant.source === 'manual' ? 'warning' : 'neutral'" variant="soft" size="sm">
              {{ t(`admin.gamification.grants.source.${grant.source}`) }}
            </UBadge>
            <span v-if="grant.source === 'manual'" class="text-xs text-muted">
              {{ t('admin.gamification.grants.actor', { id: grant.actorId }) }}
            </span>
          </div>
          <div class="flex items-center gap-2">
            <span class="text-xs text-muted">{{ formatTime(grant.grantedAt) }}</span>
            <UButton
              size="sm"
              color="neutral"
              variant="ghost"
              icon="i-lucide-trash-2"
              :disabled="gamification.saving.value"
              :aria-label="t('admin.gamification.grants.revoke')"
              :title="t('admin.gamification.grants.revoke')"
              @click="revoke(grant)"
            />
          </div>
        </li>
        <li v-if="!grants.length" class="py-6 text-center text-sm text-muted">
          {{ t('admin.gamification.grants.emptyLog') }}
        </li>
      </ul>
    </UCard>
  </div>
</template>
