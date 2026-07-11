<script setup lang="ts">
import {
  forumAuthorName,
  forumTopicPath,
  forumUserProfilePath,
  type ForumTopicSummary
} from '~/utils/forumTaxonomy'

definePageMeta({ public: true })

const route = useRoute()
const { t } = useI18n()
const localePath = useLocalePath()
const { seoSettings } = useWebOptions()
const topicUrlMode = computed(() => seoSettings.value.topicUrlMode)
const profileApi = useProfileApi()

const username = computed(() => String(route.params.username ?? ''))

const { data: profile, error: profileError } = await useAsyncData(
  () => `forum-profile-${username.value}`,
  () => profileApi.getPublicProfile(username.value),
  { default: () => null as PublicProfile | null }
)

useSForumSeo(computed(() => ({
  type: 'profile',
  path: forumUserProfilePath(username.value),
  title: profile.value?.displayName || profile.value?.username || username.value,
  description: profile.value?.profile.bio || t('profile.metaDescription'),
  public: Boolean(profile.value),
  noindex: !profile.value,
  variables: { authorName: profile.value?.displayName || profile.value?.username || username.value },
  authorName: profile.value?.displayName || profile.value?.username || username.value
})))

const displayName = computed(() => profile.value?.displayName || profile.value?.username || username.value)
const isSelf = computed(() => {
  const { user } = useAuthSession()
  return Boolean(profile.value && user.value?.id === profile.value.userId)
})

function formatDate(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return ''
  }
  return date.toLocaleDateString()
}

function topicAuthor(topic: ForumTopicSummary) {
  return forumAuthorName(topic.author, topic.authorUserId)
}
</script>

<template>
  <main class="min-h-screen py-8" style="background-color: var(--sf-surface)">
    <div class="max-w-3xl mx-auto px-4 sm:px-6">
      <SFCard v-if="profileError && !profile" class="p-10">
        <SFEmptyState
          :title="t('profile.notFound.title')"
          :description="t('profile.notFound.description')"
        />
      </SFCard>

      <template v-else-if="profile">
        <!-- 资料头部 -->
        <SFCard class="p-6 mb-4">
          <div class="flex items-center gap-4 mb-4">
            <SFAvatar :name="displayName" :avatar="profile.profile.avatar" size="lg" />
            <div>
              <h1 class="text-xl font-bold text-slate-900 dark:text-zinc-50">
                {{ displayName }}
              </h1>
              <p class="text-sm text-slate-500 dark:text-zinc-400">@{{ profile.username }}</p>
            </div>
          </div>

          <dl class="grid grid-cols-2 gap-4 text-sm mb-4">
            <div>
              <dt class="text-slate-400 dark:text-zinc-500">{{ t('profile.topicCount') }}</dt>
              <dd class="font-semibold text-slate-800 dark:text-zinc-100">{{ profile.topicCount }}</dd>
            </div>
            <div>
              <dt class="text-slate-400 dark:text-zinc-500">{{ t('profile.commentCount') }}</dt>
              <dd class="font-semibold text-slate-800 dark:text-zinc-100">{{ profile.commentCount }}</dd>
            </div>
            <div>
              <dt class="text-slate-400 dark:text-zinc-500">{{ t('profile.joinedAt') }}</dt>
              <dd class="font-semibold text-slate-800 dark:text-zinc-100">{{ formatDate(profile.joinedAt) }}</dd>
            </div>
            <div v-if="profile.profile.location">
              <dt class="text-slate-400 dark:text-zinc-500">{{ t('profile.location') }}</dt>
              <dd class="font-semibold text-slate-800 dark:text-zinc-100">{{ profile.profile.location }}</dd>
            </div>
          </dl>

          <p v-if="profile.profile.bio" class="text-slate-700 dark:text-zinc-300 mb-2">
            {{ profile.profile.bio }}
          </p>
          <a
            v-if="profile.profile.websiteUrl"
            :href="safeUrl(profile.profile.websiteUrl)"
            target="_blank"
            rel="noopener noreferrer nofollow"
            class="inline-flex items-center gap-1 text-sm text-[#0F766E] hover:underline dark:text-teal-300"
          >
            <UIcon name="i-lucide-link" class="size-3.5" />
            {{ profile.profile.websiteUrl }}
          </a>

          <div v-if="isSelf" class="mt-4 pt-4 border-t border-slate-100 dark:border-zinc-800">
            <SFButton variant="ghost" size="sm" :to="localePath('/settings/profile')">
              <UIcon name="i-lucide-settings" class="size-4" />
              <span>{{ t('profile.editProfile') }}</span>
            </SFButton>
          </div>
        </SFCard>

        <!-- 最近主题 -->
        <section v-if="profile.recentTopics && profile.recentTopics.length">
          <h2 class="text-lg font-bold text-slate-800 mb-3 dark:text-zinc-100">
            {{ t('profile.recentTopics') }}
          </h2>
          <SFCard class="divide-y divide-slate-100 dark:divide-zinc-800">
            <NuxtLink
              v-for="topic in profile.recentTopics"
              :key="topic.id"
              :to="localePath(forumTopicPath(topic, topicUrlMode))"
              class="block p-4 transition hover:bg-slate-50 dark:hover:bg-zinc-900/60"
            >
              <p class="font-medium text-slate-800 dark:text-zinc-100">{{ topic.title }}</p>
              <p class="text-xs text-slate-400 dark:text-zinc-500 mt-1">
                {{ topicAuthor(topic) }} · {{ formatDate(topic.lastActivityAt || topic.createdAt) }} · {{ topic.commentCount }} {{ t('profile.replies') }}
              </p>
            </NuxtLink>
          </SFCard>
        </section>
      </template>
    </div>
  </main>
</template>
