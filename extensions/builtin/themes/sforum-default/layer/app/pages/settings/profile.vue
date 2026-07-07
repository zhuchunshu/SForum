<script setup lang="ts">
definePageMeta({ requiresAuth: true })

const { t } = useI18n()
const { siteName } = useWebOptions()
const profileApi = useProfileApi()

useSForumSeo({
  title: () => `${t('profileSettings.metaTitle')} - ${siteName.value}`,
  description: () => t('profileSettings.metaDescription'),
  type: 'website'
})

const { data: profile, pending } = await useAsyncData(
  'my-profile',
  () => profileApi.getMyProfile(),
  { default: () => null as PublicProfile | null }
)

// 可编辑字段（本地副本，避免直接改服务端数据引用）。
const bio = ref('')
const signature = ref('')
const location = ref('')
const websiteUrl = ref('')

// 首次加载后填充表单。
watchEffect(() => {
  if (profile.value) {
    bio.value = profile.value.profile.bio
    signature.value = profile.value.profile.signature
    location.value = profile.value.profile.location
    websiteUrl.value = profile.value.profile.websiteUrl
  }
})

type SaveState = 'idle' | 'saving' | 'error' | 'success'
const saveState = ref<SaveState>('idle')
const errorMessage = ref('')
const successMessage = ref('')
const fieldErrors = ref<Record<string, string[]>>({})

const canSave = computed(() => saveState.value !== 'saving' && !pending.value)

async function save() {
  if (!canSave.value) {
    return
  }
  saveState.value = 'saving'
  errorMessage.value = ''
  successMessage.value = ''
  fieldErrors.value = {}
  try {
    await profileApi.updateMyProfile({
      bio: bio.value,
      signature: signature.value,
      location: location.value,
      websiteUrl: websiteUrl.value
    })
    saveState.value = 'success'
    successMessage.value = t('profileSettings.saved')
    // 10 秒后自动关闭成功提示。
    setTimeout(() => {
      if (saveState.value === 'success') {
        successMessage.value = ''
      }
    }, 10000)
  } catch (error) {
    saveState.value = 'error'
    errorMessage.value = apiErrorMessage(error) || t('profileSettings.saveFailed')
    fieldErrors.value = apiErrorFields(error)
  }
}
</script>

<template>
  <main class="min-h-screen py-8" style="background-color: var(--sf-surface)">
    <div class="max-w-2xl mx-auto px-4 sm:px-6">
      <h1 class="text-2xl font-bold text-slate-900 mb-6 dark:text-zinc-50">
        {{ t('profileSettings.title') }}
      </h1>

      <!-- 成功提示（自动消失） -->
      <SFAlert
        v-if="successMessage"
        variant="success"
        :title="successMessage"
        class="mb-4"
      />
      <!-- 错误提示（不自动消失） -->
      <SFAlert
        v-if="errorMessage"
        variant="danger"
        :title="errorMessage"
        closable
        class="mb-4"
        @close="errorMessage = ''"
      />

      <SFCard class="p-6 space-y-5">
        <!-- 显示名（只读，展示当前值） -->
        <div v-if="profile">
          <label class="block text-sm font-semibold text-slate-700 mb-2 dark:text-zinc-300">
            {{ t('profileSettings.username') }}
          </label>
          <input
            :value="profile.username"
            type="text"
            class="sf-input w-full opacity-60"
            readonly
          >
          <p class="text-xs text-slate-400 mt-1 dark:text-zinc-500">
            {{ t('profileSettings.usernameHint') }}
          </p>
        </div>

        <!-- 简介 -->
        <div>
          <label class="block text-sm font-semibold text-slate-700 mb-2 dark:text-zinc-300">
            {{ t('profileSettings.bio') }}
          </label>
          <textarea
            v-model="bio"
            rows="3"
            maxlength="500"
            class="sf-input w-full"
            :placeholder="t('profileSettings.bioPlaceholder')"
          />
          <p v-if="fieldErrors.bio" class="text-sm text-red-600 mt-1 dark:text-red-400">
            {{ fieldErrors.bio.join(', ') }}
          </p>
        </div>

        <!-- 个性签名 -->
        <div>
          <label class="block text-sm font-semibold text-slate-700 mb-2 dark:text-zinc-300">
            {{ t('profileSettings.signature') }}
          </label>
          <input
            v-model="signature"
            type="text"
            maxlength="200"
            class="sf-input w-full"
            :placeholder="t('profileSettings.signaturePlaceholder')"
          >
        </div>

        <!-- 位置 -->
        <div>
          <label class="block text-sm font-semibold text-slate-700 mb-2 dark:text-zinc-300">
            {{ t('profileSettings.location') }}
          </label>
          <input
            v-model="location"
            type="text"
            maxlength="100"
            class="sf-input w-full"
            :placeholder="t('profileSettings.locationPlaceholder')"
          >
        </div>

        <!-- 网站 -->
        <div>
          <label class="block text-sm font-semibold text-slate-700 mb-2 dark:text-zinc-300">
            {{ t('profileSettings.website') }}
          </label>
          <input
            v-model="websiteUrl"
            type="url"
            maxlength="200"
            class="sf-input w-full"
            placeholder="https://"
          >
          <p v-if="fieldErrors.websiteUrl" class="text-sm text-red-600 mt-1 dark:text-red-400">
            {{ fieldErrors.websiteUrl.join(', ') }}
          </p>
          <p class="text-xs text-slate-400 mt-1 dark:text-zinc-500">
            {{ t('profileSettings.websiteHint') }}
          </p>
        </div>

        <div class="flex justify-end gap-2 pt-2">
          <SFButton
            variant="primary"
            :disabled="!canSave"
            @click="save"
          >
            {{ saveState === 'saving' ? t('profileSettings.saving') : t('profileSettings.save') }}
          </SFButton>
        </div>
      </SFCard>
    </div>
  </main>
</template>

<style scoped>
.sf-input {
  border: 1px solid #d1d5db;
  border-radius: 0.5rem;
  padding: 0.5rem 0.75rem;
  font-size: 0.95rem;
  background: #ffffff;
  color: #111827;
  outline: none;
  transition: border-color 0.15s;
}
.sf-input:focus {
  border-color: #0f766e;
  box-shadow: 0 0 0 3px rgba(15, 118, 110, 0.12);
}
:global(.dark) .sf-input {
  background: #18181b;
  border-color: #3f3f46;
  color: #f4f4f5;
}
</style>
