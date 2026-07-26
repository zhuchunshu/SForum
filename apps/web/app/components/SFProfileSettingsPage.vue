<script setup lang="ts">
/**
 * 宿主 body 岛：forum.settings.profile。主题 L1 挂载；路由页仅 outlet + fail-closed 回退。
 * 三栏 chrome 由 SFSettingsShell 提供，此处仅保留资料表单业务。
 */
import type { ProfileData, PublicProfile } from '~/composables/useProfileApi'
import { safeUrl } from '~/utils/sfUrl'

type ProfileDraft = {
  bio: string
  signature: string
  location: string
  websiteUrl: string
}

type SaveState = 'idle' | 'saving' | 'error' | 'success'

const { t } = useI18n()
const localePath = useLocalePath()
const toast = useToast()
const { siteName, avatarSettings } = useWebOptions()
const { user: authUser, setUser } = useAuthSession()
const { can } = usePermissions()
const { formatDateOnly } = useSiteDateTime()
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

const draft = reactive<ProfileDraft>({
  bio: '',
  signature: '',
  location: '',
  websiteUrl: ''
})
const baseline = ref<ProfileDraft>({
  bio: '',
  signature: '',
  location: '',
  websiteUrl: ''
})
const formReady = ref(false)

const avatarInput = ref<HTMLInputElement | null>(null)
const avatarBusy = ref(false)
const avatarError = ref('')
const saveState = ref<SaveState>('idle')
const errorMessage = ref('')
const successMessage = ref('')
const fieldErrors = ref<Record<string, string[]>>({})
let successTimer: ReturnType<typeof setTimeout> | undefined

const currentAvatar = computed(() => profile.value?.profile.avatar || null)
const displayName = computed(() => profile.value?.displayName || profile.value?.username || '')
const publicProfilePath = computed(() => profile.value ? localePath(`/u/${profile.value.username}`) : undefined)
const joinedLabel = computed(() => profile.value ? formatDateOnly(profile.value.joinedAt) : '')
const avatarAccept = computed(() => avatarSettings.value.allowGif ? 'image/jpeg,image/png,image/gif' : 'image/jpeg,image/png')
const canUploadAvatar = computed(() => avatarSettings.value.allowUpload && can(FORUM_PERMISSIONS.attachmentUpload))
const avatarHint = computed(() => {
  const types = avatarSettings.value.allowGif ? 'JPG / PNG / GIF' : 'JPG / PNG'
  return t('profileSettings.avatarHint', {
    types,
    size: avatarSettings.value.maxSizeKb,
    dimension: avatarSettings.value.maxDimension
  })
})
const avatarCapabilityHint = computed(() => {
  if (!avatarSettings.value.allowUpload) {
    return t('profileSettings.avatarUploadDisabled')
  }
  if (!canUploadAvatar.value) {
    return t('profileSettings.avatarUploadPermissionDenied')
  }
  return avatarHint.value
})

const isDirty = computed(() => (
  draft.bio !== baseline.value.bio
  || draft.signature !== baseline.value.signature
  || draft.location !== baseline.value.location
  || draft.websiteUrl !== baseline.value.websiteUrl
))
const canSave = computed(() => saveState.value !== 'saving' && !pending.value && isDirty.value)
const hasUploadedAvatar = computed(() => Boolean(
  profile.value?.profile.avatarAttachmentId
  || profile.value?.profile.avatar.attachmentId
  || profile.value?.profile.avatar.kind === 'uploaded'
))
const bioCount = computed(() => draft.bio.length)
const signatureCount = computed(() => draft.signature.length)
const publicWebsiteText = computed(() => {
  const value = draft.websiteUrl.trim()
  return value ? value.replace(/^https?:\/\//i, '') : ''
})
const publicWebsiteHref = computed(() => safeUrl(draft.websiteUrl))

watch(profile, (value) => {
  if (!value || formReady.value) {
    return
  }
  setDraftFromProfile(value.profile)
  formReady.value = true
}, { immediate: true })

onBeforeUnmount(() => {
  if (successTimer) {
    clearTimeout(successTimer)
  }
})

function profileValues(data: ProfileData): ProfileDraft {
  return {
    bio: data.bio || '',
    signature: data.signature || '',
    location: data.location || '',
    websiteUrl: data.websiteUrl || ''
  }
}

function setDraftFromProfile(data: ProfileData) {
  const values = profileValues(data)
  Object.assign(draft, values)
  baseline.value = { ...values }
}

function applyProfileUpdate(updated: ProfileData, options: { resetDraft?: boolean } = {}) {
  if (profile.value) {
    profile.value = { ...profile.value, profile: updated }
  }
  if (options.resetDraft) {
    setDraftFromProfile(updated)
  }
}

function syncAuthAvatar(updated: ProfileData) {
  if (!authUser.value) {
    return
  }
  setUser({ ...authUser.value, avatar: updated.avatar })
}

function showSuccess(message: string) {
  successMessage.value = message
  if (successTimer) {
    clearTimeout(successTimer)
  }
  successTimer = setTimeout(() => {
    successMessage.value = ''
  }, 10000)
}

function resetDraft() {
  if (!isDirty.value) {
    return
  }
  Object.assign(draft, baseline.value)
  fieldErrors.value = {}
  errorMessage.value = ''
  successMessage.value = ''
  saveState.value = 'idle'
  toast.add({
    color: 'neutral',
    icon: 'i-lucide-rotate-ccw',
    title: t('profileSettings.resetDone'),
    duration: 10000
  })
}

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
      bio: draft.bio,
      signature: draft.signature,
      location: draft.location,
      websiteUrl: draft.websiteUrl
    })
    applyProfileUpdate(updated, { resetDraft: true })
    saveState.value = 'success'
    showSuccess(t('profileSettings.saved'))
    toast.add({
      color: 'primary',
      icon: 'i-lucide-check',
      title: t('profileSettings.saved'),
      duration: 10000
    })
  } catch (error) {
    saveState.value = 'error'
    errorMessage.value = apiErrorMessage(error) || t('profileSettings.saveFailed')
    fieldErrors.value = apiErrorFields(error)
  }
}

function openAvatarPicker() {
  if (avatarBusy.value || !canUploadAvatar.value) {
    return
  }
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
  if (!canUploadAvatar.value) {
    avatarError.value = t('profileSettings.avatarUploadPermissionDenied')
    return
  }
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
    applyProfileUpdate(updated)
    syncAuthAvatar(updated)
    toast.add({
      color: 'primary',
      icon: 'i-lucide-check',
      title: t('profileSettings.avatarUploaded'),
      duration: 10000
    })
  } catch (error) {
    avatarError.value = apiErrorMessage(error) || t('profileSettings.avatarUploadFailed')
  } finally {
    avatarBusy.value = false
  }
}

async function removeAvatar() {
  if (avatarBusy.value || !hasUploadedAvatar.value) {
    return
  }
  avatarError.value = ''
  avatarBusy.value = true
  try {
    const updated = await profileApi.deleteAvatar()
    applyProfileUpdate(updated)
    syncAuthAvatar(updated)
    toast.add({
      color: 'primary',
      icon: 'i-lucide-check',
      title: t('profileSettings.avatarRemoved'),
      duration: 10000
    })
  } catch (error) {
    avatarError.value = apiErrorMessage(error) || t('profileSettings.avatarRemoveFailed')
  } finally {
    avatarBusy.value = false
  }
}
</script>

<template>
  <SFSettingsShell
    class="sforum-settings-profile"
    data-sforum-island-body="forum.component.settings_profile"
    active="profile"
    title-id="profile-settings-title"
    :title="t('profileSettings.canvasTitle')"
    :description="t('profileSettings.canvasDescription')"
    :rail-label="t('profileSettings.preview.ariaLabel')"
    :rail-open-label="t('profileSettings.preview.open')"
    :public-profile-path="publicProfilePath"
    :show-rail="Boolean(profile)"
  >
    <nav
      v-if="profile"
      class="sforum-settings__section-nav"
      :aria-label="t('profileSettings.sections.ariaLabel')"
    >
      <a href="#profile-settings-identity">{{ t('profileSettings.sections.identity') }}</a>
      <a href="#profile-settings-story">{{ t('profileSettings.sections.story') }}</a>
      <a href="#profile-settings-links">{{ t('profileSettings.sections.links') }}</a>
    </nav>

    <template v-if="pending && !profile">
      <div class="sforum-settings-profile__loading">
        <SFSkeleton class="h-16 w-16 rounded-full" />
        <div class="min-w-0 flex-1 space-y-3">
          <SFSkeleton class="h-7 w-2/3" />
          <SFSkeleton class="h-4 w-full" />
          <SFSkeleton class="h-4 w-3/5" />
        </div>
      </div>
    </template>

    <form
      v-else-if="profile"
      class="sforum-settings-profile__form"
      novalidate
      @submit.prevent="save"
    >
      <SFAlert
        v-if="successMessage"
        variant="success"
        :title="successMessage"
        class="sforum-settings-profile__alert"
      />
      <SFAlert
        v-if="errorMessage"
        variant="danger"
        :title="errorMessage"
        closable
        class="sforum-settings-profile__alert"
        @close="errorMessage = ''"
      />

      <section id="profile-settings-identity" class="sforum-settings-profile__section">
        <div class="sforum-settings-profile__section-heading">
          <h2>{{ t('profileSettings.sections.identity') }}</h2>
          <p>{{ t('profileSettings.sections.identityDescription') }}</p>
        </div>
        <div class="sforum-settings-profile__section-body">
          <div class="sforum-settings-profile__avatar-editor">
            <SFAvatar :name="displayName" :avatar="currentAvatar" size="lg" />
            <div class="sforum-settings-profile__avatar-copy">
              <strong>{{ t('profileSettings.avatarCurrent') }}</strong>
              <p>{{ avatarCapabilityHint }}</p>
              <p v-if="avatarError" class="sforum-settings-profile__field-error">
                {{ avatarError }}
              </p>
              <div class="sforum-settings-profile__actions">
                <input ref="avatarInput" class="hidden" type="file" :accept="avatarAccept" @change="uploadAvatar">
                <SFButton
                  variant="ghost"
                  size="sm"
                  :disabled="!canUploadAvatar || avatarBusy"
                  :loading="avatarBusy"
                  @click="openAvatarPicker"
                >
                  <template #leading>
                    <UIcon name="i-lucide-image-plus" />
                  </template>
                  {{ t('profileSettings.avatarUpload') }}
                </SFButton>
                <SFButton
                  v-if="hasUploadedAvatar"
                  variant="ghost"
                  size="sm"
                  :disabled="avatarBusy"
                  @click="removeAvatar"
                >
                  <template #leading>
                    <UIcon name="i-lucide-trash-2" />
                  </template>
                  {{ t('profileSettings.avatarRemove') }}
                </SFButton>
              </div>
            </div>
          </div>

          <div class="sforum-settings-profile__field sforum-settings-profile__field--spaced">
            <label for="profile-settings-username">{{ t('profileSettings.username') }}</label>
            <div class="sforum-settings-profile__prefix-field">
              <span>{{ t('profileSettings.profilePathPrefix') }}</span>
              <input id="profile-settings-username" :value="profile.username" type="text" readonly>
            </div>
            <p>{{ t('profileSettings.usernameHint') }}</p>
          </div>
        </div>
      </section>

      <section id="profile-settings-story" class="sforum-settings-profile__section">
        <div class="sforum-settings-profile__section-heading">
          <h2>{{ t('profileSettings.sections.story') }}</h2>
          <p>{{ t('profileSettings.sections.storyDescription') }}</p>
        </div>
        <div class="sforum-settings-profile__section-body">
          <div class="sforum-settings-profile__field">
            <div class="sforum-settings-profile__field-topline">
              <label for="profile-settings-bio">{{ t('profileSettings.bio') }}</label>
              <span>{{ bioCount }} / 500</span>
            </div>
            <textarea
              id="profile-settings-bio"
              v-model="draft.bio"
              rows="4"
              maxlength="500"
              :class="{ 'is-invalid': fieldErrors.bio }"
              :placeholder="t('profileSettings.bioPlaceholder')"
              :aria-describedby="fieldErrors.bio ? 'profile-settings-bio-error' : undefined"
            />
            <p v-if="fieldErrors.bio" id="profile-settings-bio-error" class="sforum-settings-profile__field-error">
              {{ fieldErrors.bio.join(', ') }}
            </p>
          </div>

          <div class="sforum-settings-profile__field">
            <div class="sforum-settings-profile__field-topline">
              <label for="profile-settings-signature">{{ t('profileSettings.signature') }}</label>
              <span>{{ signatureCount }} / 200</span>
            </div>
            <input
              id="profile-settings-signature"
              v-model="draft.signature"
              type="text"
              maxlength="200"
              :class="{ 'is-invalid': fieldErrors.signature }"
              :placeholder="t('profileSettings.signaturePlaceholder')"
              :aria-describedby="fieldErrors.signature ? 'profile-settings-signature-error' : undefined"
            >
            <p v-if="fieldErrors.signature" id="profile-settings-signature-error" class="sforum-settings-profile__field-error">
              {{ fieldErrors.signature.join(', ') }}
            </p>
            <p>{{ t('profileSettings.signaturePublicHint') }}</p>
          </div>
        </div>
      </section>

      <section id="profile-settings-links" class="sforum-settings-profile__section">
        <div class="sforum-settings-profile__section-heading">
          <h2>{{ t('profileSettings.sections.links') }}</h2>
          <p>{{ t('profileSettings.sections.linksDescription') }}</p>
        </div>
        <div class="sforum-settings-profile__section-body">
          <div class="sforum-settings-profile__field-grid">
            <div class="sforum-settings-profile__field">
              <label for="profile-settings-location">{{ t('profileSettings.location') }}</label>
              <input
                id="profile-settings-location"
                v-model="draft.location"
                type="text"
                maxlength="100"
                :class="{ 'is-invalid': fieldErrors.location }"
                :placeholder="t('profileSettings.locationPlaceholder')"
                :aria-describedby="fieldErrors.location ? 'profile-settings-location-error' : undefined"
              >
              <p v-if="fieldErrors.location" id="profile-settings-location-error" class="sforum-settings-profile__field-error">
                {{ fieldErrors.location.join(', ') }}
              </p>
            </div>

            <div class="sforum-settings-profile__field">
              <label for="profile-settings-website">{{ t('profileSettings.website') }}</label>
              <input
                id="profile-settings-website"
                v-model="draft.websiteUrl"
                type="url"
                maxlength="200"
                :class="{ 'is-invalid': fieldErrors.websiteUrl }"
                :placeholder="t('profileSettings.websitePlaceholder')"
                :aria-describedby="fieldErrors.websiteUrl ? 'profile-settings-website-error profile-settings-website-hint' : 'profile-settings-website-hint'"
              >
              <p v-if="fieldErrors.websiteUrl" id="profile-settings-website-error" class="sforum-settings-profile__field-error">
                {{ fieldErrors.websiteUrl.join(', ') }}
              </p>
              <p id="profile-settings-website-hint">
                {{ t('profileSettings.websiteHint') }}
              </p>
            </div>
          </div>
        </div>
      </section>

      <footer class="sforum-settings-profile__footer">
        <p>{{ isDirty ? t('profileSettings.unsavedFooter') : t('profileSettings.savedFooter') }}</p>
        <div class="sforum-settings-profile__footer-actions">
          <SFButton
            variant="ghost"
            type="button"
            :disabled="!isDirty || saveState === 'saving'"
            @click="resetDraft"
          >
            <template #leading>
              <UIcon name="i-lucide-rotate-ccw" />
            </template>
            {{ t('profileSettings.resetChanges') }}
          </SFButton>
          <SFButton
            variant="primary"
            type="submit"
            :disabled="!canSave"
            :loading="saveState === 'saving'"
          >
            <template #leading>
              <UIcon name="i-lucide-save" />
            </template>
            {{ saveState === 'saving' ? t('profileSettings.saving') : t('profileSettings.save') }}
          </SFButton>
        </div>
      </footer>
    </form>

    <template #rail>
      <SFProfileSettingsPreview
        v-if="profile"
        :profile="profile"
        :display-name="displayName"
        :avatar="currentAvatar"
        :bio="draft.bio"
        :location="draft.location"
        :website-text="publicWebsiteText"
        :website-href="publicWebsiteHref"
        :joined-label="joinedLabel"
        :dirty="isDirty"
        :public-profile-path="publicProfilePath || localePath('/')"
        show-scope
      />
    </template>
  </SFSettingsShell>
</template>

<style src="~/assets/css/sforum-profile-settings.css" lang="css"></style>
