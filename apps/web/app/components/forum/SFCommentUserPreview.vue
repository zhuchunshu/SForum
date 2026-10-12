<script setup lang="ts">
import { useProfileApi, type PublicProfile } from '~/composables/profile/useProfileApi'
import { apiErrorStatusCode } from '~/composables/useApiClient'
import { userDisplayName, type ForumUserRef } from '~/utils/forum/userDisplay'
import SFUserIdentityName from '~/components/forum/SFUserIdentityName.vue'

const props = defineProps<{
  /** 列表场景已有的用户摘要：名字与头像直接消费，网络资料只做增强。 */
  user: ForumUserRef
  /** 传 user.username 之外的显式用户名（提及场景只有 username）。 */
  username?: string
  profilePath: string
}>()

const { t } = useI18n()
const profileApi = useProfileApi()
const profileCache = useState<Record<string, PublicProfile>>('profile:comment-user-preview-cache', () => ({}))
const pending = ref(false)
const failed = ref(false)
// 查无此人（404）与「资料暂不可用」分开表达：@提及 常常指向拼错或不存在的用户名。
const notFound = ref(false)

const previewUsername = computed(() => props.username || props.user?.username || '')
const profile = computed(() => profileCache.value[previewUsername.value] || null)
// 名字与头像优先取调用方传入的 UserSummary（与评论行内展示一致，避免闪变）；
// 网络资料仅在本地没有 displayName 时补齐（例如纯 @username 的提及占位）。
const displayName = computed(() => props.user?.displayName || profile.value?.displayName || previewUsername.value)
const displayAvatar = computed(() => props.user?.avatar || profile.value?.profile.avatar || null)
const bio = computed(() => profile.value?.profile.bio?.trim() || profile.value?.profile.signature?.trim() || '')

async function loadProfile() {
  if (!previewUsername.value || profile.value || pending.value) {
    return
  }
  pending.value = true
  failed.value = false
  notFound.value = false
  try {
    const result = await profileApi.getPublicProfile(previewUsername.value)
    profileCache.value = { ...profileCache.value, [previewUsername.value]: result }
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
      <SFUserIdentityName
        :user="{ username: previewUsername, displayName }"
        :to="profilePath"
        class="sf-comment-user-preview__names sf-comment-user-preview__identity-names"
      />
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
