<script setup lang="ts">
/**
 * 宿主 body 岛：forum.settings.profile。主题 L1 挂载；路由页仅 outlet + fail-closed 回退。
 */


const { t } = useI18n()
const toast = useToast()
const { siteName, avatarSettings } = useWebOptions()
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
const avatarInput = ref<HTMLInputElement | null>(null)
const avatarBusy = ref(false)
const avatarError = ref('')

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
const currentAvatar = computed(() => profile.value?.profile.avatar || null)
const avatarAccept = computed(() => avatarSettings.value.allowGif ? 'image/jpeg,image/png,image/gif' : 'image/jpeg,image/png')
const avatarHint = computed(() => {
  const types = avatarSettings.value.allowGif ? 'JPG / PNG / GIF' : 'JPG / PNG'
  return t('profileSettings.avatarHint', {
    types,
    size: avatarSettings.value.maxSizeKb,
    dimension: avatarSettings.value.maxDimension
  })
})

async function save() {
  if (!canSave.value) {
    return
  }
  saveState.value = 'saving'
  errorMessage.value = ''
  successMessage.value = ''
  fieldErrors.value = {}
  try {
    const updated = await profileApi.updateMyProfile({
      bio: bio.value,
      signature: signature.value,
      location: location.value,
      websiteUrl: websiteUrl.value
    })
    if (profile.value) {
      profile.value = { ...profile.value, profile: updated }
    }
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

function openAvatarPicker() {
  avatarError.value = ''
  avatarInput.value?.click()
}

async function uploadAvatar(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file || avatarBusy.value) {
    return
  }
  avatarError.value = ''
  if (file.size > avatarSettings.value.maxSizeKb * 1024) {
    avatarError.value = t('profileSettings.avatarTooLarge', { size: avatarSettings.value.maxSizeKb })
    return
  }
  if (!avatarSettings.value.allowGif && file.type === 'image/gif') {
    avatarError.value = t('profileSettings.avatarGifDisabled')
    return
  }
  avatarBusy.value = true
  try {
    const updated = await profileApi.uploadAvatar(file)
    if (profile.value) {
      profile.value = { ...profile.value, profile: updated }
    }
    toast.add({ color: 'success', icon: 'i-lucide-check', title: t('profileSettings.avatarUploaded') })
  } catch (error) {
    avatarError.value = apiErrorMessage(error) || t('profileSettings.avatarUploadFailed')
  } finally {
    avatarBusy.value = false
  }
}

async function removeAvatar() {
  if (avatarBusy.value) {
    return
  }
  avatarError.value = ''
  avatarBusy.value = true
  try {
    const updated = await profileApi.deleteAvatar()
    if (profile.value) {
      profile.value = { ...profile.value, profile: updated }
    }
    toast.add({ color: 'success', icon: 'i-lucide-check', title: t('profileSettings.avatarRemoved') })
  } catch (error) {
    avatarError.value = apiErrorMessage(error) || t('profileSettings.avatarRemoveFailed')
  } finally {
    avatarBusy.value = false
  }
}
</script>

<template>

<main class="sf-public-page min-h-screen py-8">
    <div class="sf-public-page__container sf-public-page__container--form mx-auto px-4 sm:px-6">
      <h1 class="text-2xl font-bold text-slate-900 mb-6 dark:text-zinc-50">
        {{ t('profileSettings.title') }}
      </h1>

      <!-- 账号设置子导航：在资料设置与账号安全间切换 -->
      <div class="flex gap-2 mb-6">
        <NuxtLink
          to="/settings/profile"
          class="inline-flex items-center gap-1.5 rounded-md border border-slate-200 px-3 py-1.5 text-sm font-medium text-slate-600 hover:bg-slate-50 dark:border-zinc-800 dark:text-zinc-400 dark:hover:bg-zinc-900"
        >
          <UIcon name="i-lucide-user" />
          {{ t('profileSettings.title') }}
        </NuxtLink>
        <NuxtLink
          to="/settings/security"
          class="inline-flex items-center gap-1.5 rounded-md border border-slate-200 px-3 py-1.5 text-sm font-medium text-slate-600 hover:bg-slate-50 dark:border-zinc-800 dark:text-zinc-400 dark:hover:bg-zinc-900"
        >
          <UIcon name="i-lucide-shield-check" />
          {{ t('accountSecurity.title') }}
        </NuxtLink>
      </div>

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
        <div v-if="profile" class="flex flex-col gap-4 rounded-lg border border-slate-200 p-4 dark:border-zinc-800 sm:flex-row sm:items-center sm:justify-between">
          <div class="flex min-w-0 items-center gap-4">
            <SFAvatar :name="profile.displayName" :avatar="currentAvatar" size="lg" />
            <div class="min-w-0">
              <h2 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
                {{ t('profileSettings.avatar') }}
              </h2>
              <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
                {{ avatarSettings.allowUpload ? avatarHint : t('profileSettings.avatarUploadDisabled') }}
              </p>
              <p v-if="avatarError" class="mt-2 text-sm text-red-600 dark:text-red-400">
                {{ avatarError }}
              </p>
            </div>
          </div>
          <div class="flex shrink-0 flex-wrap gap-2">
            <input ref="avatarInput" class="hidden" type="file" :accept="avatarAccept" @change="uploadAvatar">
            <SFButton variant="secondary" size="sm" :disabled="!avatarSettings.allowUpload || avatarBusy" :loading="avatarBusy" @click="openAvatarPicker">
              {{ t('profileSettings.avatarUpload') }}
            </SFButton>
            <SFButton v-if="profile.profile.avatarAttachmentId" variant="ghost" size="sm" :disabled="avatarBusy" @click="removeAvatar">
              {{ t('profileSettings.avatarRemove') }}
            </SFButton>
          </div>
        </div>

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
