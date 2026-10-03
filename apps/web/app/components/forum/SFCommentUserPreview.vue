<script setup lang="ts">
import { useProfileApi, type AvatarView, type PublicProfile } from '~/composables/profile/useProfileApi'
import { apiErrorStatusCode } from '~/composables/useApiClient'

const props = defineProps<{
  author: string
  username: string
  avatar?: AvatarView | null
  profilePath: string
}>()

const { t } = useI18n()
const profileApi = useProfileApi()
const profileCache = useState<Record<string, PublicProfile>>('profile:comment-user-preview-cache', () => ({}))
const pending = ref(false)
const failed = ref(false)
// 查无此人（404）与「资料暂不可用」分开表达：@提及 常常指向拼错或不存在的用户名。
const notFound = ref(false)

const profile = computed(() => profileCache.value[props.username] || null)
const displayName = computed(() => profile.value?.displayName || props.author)
const displayAvatar = computed(() => profile.value?.profile.avatar || props.avatar)
const bio = computed(() => profile.value?.profile.bio?.trim() || profile.value?.profile.signature?.trim() || '')

async function loadProfile() {
  if (!props.username || profile.value || pending.value) {
    return
  }
  pending.value = true
  failed.value = false
  notFound.value = false
  try {
    const result = await profileApi.getPublicProfile(props.username)
    profileCache.value = { ...profileCache.value, [props.username]: result }
  } catch (error) {
    // 公开资料可能被站点关闭、用户不存在或暂时不可用；保留已有的身份和主页入口。
    failed.value = true
    notFound.value = apiErrorStatusCode(error) === 404
  } finally {
    pending.value = false
  }
}

onMounted(() => {
  void loadProfile()
})
</script>

<template>
  <section
    class="sf-comment-user-preview"
    role="dialog"
    :aria-label="t('topicDetail.userPreview.ariaLabel', { name: displayName })"
    data-testid="comment-user-preview"
    :data-state="pending ? 'loading' : notFound ? 'not-found' : failed ? 'unavailable' : 'ready'"
  >
    <div class="sf-comment-user-preview__identity">
      <SFAvatar :name="displayName" :avatar="displayAvatar" size="md" />
      <span class="sf-comment-user-preview__names">
        <NuxtLink :to="profilePath" class="sf-comment-user-preview__name">
          {{ displayName }}
        </NuxtLink>
        <span class="sf-comment-user-preview__username">@{{ username }}</span>
      </span>
    </div>

    <SFSkeleton v-if="pending" width="88%" height="0.75rem" class="sf-comment-user-preview__skeleton" />
    <p v-else-if="failed" class="sf-comment-user-preview__status">
      {{ notFound ? t('topicDetail.userPreview.notFound') : t('topicDetail.userPreview.unavailable') }}
    </p>
    <p v-else class="sf-comment-user-preview__bio">
      {{ bio || t('topicDetail.userPreview.noBio') }}
    </p>

    <div v-if="profile" class="sf-comment-user-preview__stats">
      <span>{{ t('topicDetail.userPreview.topics', { count: profile.topicCount }) }}</span>
      <span aria-hidden="true">·</span>
      <span>{{ t('topicDetail.userPreview.replies', { count: profile.commentCount }) }}</span>
    </div>

    <NuxtLink :to="profilePath" class="sf-comment-user-preview__primary">
      <UIcon name="i-lucide-user-round" class="size-4" aria-hidden="true" />
      <span>{{ t('topicDetail.userPreview.viewProfile') }}</span>
    </NuxtLink>
  </section>
</template>
