<script setup lang="ts">
import {
  forumAuthorName,
  forumUserProfilePath,
  type ForumTopicContributionEvent,
  type ForumTopicContributionTimeline,
  type ForumUserSummary
} from '~/utils/forumTaxonomy'

/** 单页条数；与 API keyset perPage 对齐 */
const PER_PAGE = 15

const props = defineProps<{
  open: boolean
  topicId: number
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

const { t } = useI18n()
const localePath = useLocalePath()
const { format: formatSiteDateTime } = useSiteDateTime()
const forumApi = useForumApi()

const loading = ref(false)
const errorMessage = ref('')
const items = ref<ForumTopicContributionEvent[]>([])
const hasMore = ref(false)
const currentPage = ref(1)
/** page → 请求该页时使用的 after 游标；第 1 页无 after */
const pageAfterTokens = ref(new Map<number, string>())
const loadedTopicId = ref(0)

const modalOpen = computed({
  get: () => props.open,
  set: (value: boolean) => emit('update:open', value)
})

const canPrev = computed(() => currentPage.value > 1 && !loading.value)
const canNext = computed(() => hasMore.value && !loading.value)

watch(
  () => [props.open, props.topicId] as const,
  async ([open, topicId]) => {
    if (!open || topicId <= 0) {
      return
    }
    // 同一主题再次打开：保留已加载页，避免闪烁
    if (loadedTopicId.value === topicId && items.value.length > 0) {
      return
    }
    await resetAndLoad(topicId)
  }
)

async function resetAndLoad(topicId: number) {
  currentPage.value = 1
  pageAfterTokens.value = new Map()
  items.value = []
  hasMore.value = false
  errorMessage.value = ''
  await loadPage(topicId, 1)
}

/**
 * keyset 分页：第 N 页的 after 来自第 N-1 页响应的 nextCursor。
 * 仅替换当前页 items，不做无限追加。
 */
async function loadPage(topicId: number, page: number) {
  if (page < 1) {
    return
  }
  if (page > 1 && !pageAfterTokens.value.has(page)) {
    return
  }

  loading.value = true
  errorMessage.value = ''
  try {
    const after = page <= 1 ? undefined : pageAfterTokens.value.get(page)
    const result = await forumApi.listTopicContributionTimeline(topicId, {
      after,
      perPage: PER_PAGE
    })
    applyPage(result, page)
    loadedTopicId.value = topicId
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('topicDetail.contributors.loadFailed')
  } finally {
    loading.value = false
  }
}

function applyPage(page: ForumTopicContributionTimeline, pageNo: number) {
  items.value = page.items || []
  hasMore.value = Boolean(page.hasMore)
  currentPage.value = pageNo
  const next = `${page.nextCursor || ''}`.trim()
  if (next) {
    const nextMap = new Map(pageAfterTokens.value)
    nextMap.set(pageNo + 1, next)
    pageAfterTokens.value = nextMap
  }
}

function actorName(actor?: ForumUserSummary) {
  if (!actor) {
    return t('topicDetail.contributors.unknownActor')
  }
  return forumAuthorName(actor, actor.id)
}

function actorTo(actor?: ForumUserSummary) {
  if (!actor?.username) {
    return ''
  }
  return localePath(forumUserProfilePath(actor.username))
}

function operationLabel(operation: string) {
  const key = `topicDetail.contributors.operation.${operation}`
  const label = t(key)
  return label === key ? operation : label
}

function originLabel(origin: string) {
  const key = `topicDetail.contributors.origin.${origin}`
  const label = t(key)
  return label === key ? origin : label
}

function fieldLabel(field: string) {
  const key = `topicDetail.contributors.field.${field}`
  const label = t(key)
  return label === key ? field : label
}

function formatCommittedAt(value: string) {
  return formatSiteDateTime(value)
}

async function goPrev() {
  if (!canPrev.value) {
    return
  }
  await loadPage(props.topicId, currentPage.value - 1)
}

async function goNext() {
  if (!canNext.value) {
    return
  }
  await loadPage(props.topicId, currentPage.value + 1)
}
</script>

<template>
  <UModal
    v-model:open="modalOpen"
    :title="t('topicDetail.contributors.modalTitle')"
    :ui="{ content: 'sm:max-w-3xl' }"
  >
    <template #body>
      <div class="sf-topic-contributors-modal">
        <p class="sf-topic-contributors-modal__hint">
          {{ t('topicDetail.contributors.modalHint') }}
        </p>

        <div class="sf-topic-contributors-modal__scroll">
          <div v-if="loading && !items.length" class="sf-topic-contributors-modal__state">
            {{ t('topicDetail.contributors.loading') }}
          </div>
          <div v-else-if="errorMessage" class="sf-topic-contributors-modal__error" role="alert">
            {{ errorMessage }}
          </div>
          <ol
            v-else-if="items.length"
            class="sf-topic-contributors-modal__list"
            :class="{ 'is-loading': loading }"
            :aria-busy="loading ? 'true' : undefined"
          >
            <li
              v-for="event in items"
              :key="`${event.revisionNo}-${event.committedAt}`"
              class="sf-topic-contributors-modal__item"
            >
              <div class="sf-topic-contributors-modal__avatar">
                <NuxtLink
                  v-if="actorTo(event.actor)"
                  :to="actorTo(event.actor)"
                  :aria-label="actorName(event.actor)"
                >
                  <SFAvatar
                    :name="actorName(event.actor)"
                    :avatar="event.actor?.avatar"
                    size="sm"
                  />
                </NuxtLink>
                <SFAvatar
                  v-else
                  :name="actorName(event.actor)"
                  :avatar="event.actor?.avatar"
                  size="sm"
                />
              </div>
              <div class="sf-topic-contributors-modal__body">
                <div class="sf-topic-contributors-modal__title">
                  <NuxtLink
                    v-if="actorTo(event.actor)"
                    :to="actorTo(event.actor)"
                    class="sf-topic-contributors-modal__name"
                  >
                    {{ actorName(event.actor) }}
                  </NuxtLink>
                  <strong v-else class="sf-topic-contributors-modal__name">
                    {{ actorName(event.actor) }}
                  </strong>
                  <span class="sf-topic-contributors-modal__op">
                    {{ operationLabel(event.operation) }}
                  </span>
                  <span
                    v-if="event.origin === 'staff'"
                    class="sf-topic-contributors-modal__origin"
                  >
                    {{ originLabel(event.origin) }}
                  </span>
                  <span
                    v-if="event.current"
                    class="sf-topic-contributors-modal__current"
                  >
                    {{ t('topicDetail.contributors.current') }}
                  </span>
                  <span
                    v-if="event.redacted"
                    class="sf-topic-contributors-modal__redacted"
                  >
                    {{ t('topicDetail.contributors.redacted') }}
                  </span>
                </div>
                <div class="sf-topic-contributors-modal__meta">
                  <time :datetime="event.committedAt">
                    {{ formatCommittedAt(event.committedAt) }}
                  </time>
                  <span v-if="event.changedFields?.length">
                    ·
                    {{ event.changedFields.map(fieldLabel).join('、') }}
                  </span>
                  <span v-if="event.restoredFromRevisionNo">
                    ·
                    {{ t('topicDetail.contributors.restoredFrom', { n: event.restoredFromRevisionNo }) }}
                  </span>
                  <span class="sf-topic-contributors-modal__rev">
                    ·
                    {{ t('topicDetail.contributors.revisionNo', { n: event.revisionNo }) }}
                  </span>
                </div>
              </div>
            </li>
          </ol>
          <div v-else class="sf-topic-contributors-modal__state">
            {{ t('topicDetail.contributors.empty') }}
          </div>
        </div>

        <!-- keyset 上一页 / 下一页 -->
        <footer
          v-if="items.length || currentPage > 1"
          class="sf-topic-contributors-modal__pager"
        >
          <UButton
            color="neutral"
            variant="soft"
            size="sm"
            icon="i-lucide-chevron-left"
            :disabled="!canPrev"
            :loading="loading && currentPage > 1"
            @click="goPrev"
          >
            {{ t('topicDetail.contributors.prevPage') }}
          </UButton>
          <span class="sf-topic-contributors-modal__page-label">
            {{ t('topicDetail.contributors.pageLabel', { page: currentPage }) }}
            <template v-if="hasMore">
              · {{ t('topicDetail.contributors.hasMore') }}
            </template>
          </span>
          <UButton
            color="neutral"
            variant="soft"
            size="sm"
            trailing-icon="i-lucide-chevron-right"
            :disabled="!canNext"
            :loading="loading && hasMore"
            @click="goNext"
          >
            {{ t('topicDetail.contributors.nextPage') }}
          </UButton>
        </footer>
      </div>
    </template>
  </UModal>
</template>

<style scoped>
.sf-topic-contributors-modal {
  display: grid;
  gap: 12px;
  /* 大弹窗内可滚动列表 + 底部分页固定 */
  min-height: min(52vh, 420px);
}

.sf-topic-contributors-modal__hint {
  margin: 0;
  font-size: 13px;
  line-height: 1.5;
  color: var(--sf-public-text-muted, #64748b);
}

.sf-topic-contributors-modal__scroll {
  min-height: 240px;
  max-height: min(58vh, 560px);
  overflow: auto;
  margin: 0 -4px;
  padding: 0 4px;
  border: 1px solid var(--sf-border-light, #e2e8f0);
  border-radius: 12px;
  background: var(--sf-public-surface, #fff);
}

.sf-topic-contributors-modal__state,
.sf-topic-contributors-modal__error {
  padding: 28px 16px;
  font-size: 14px;
  text-align: center;
  color: var(--sf-public-text-muted, #64748b);
}

.sf-topic-contributors-modal__error {
  color: var(--sf-danger, #dc2626);
}

.sf-topic-contributors-modal__list {
  display: grid;
  gap: 0;
  margin: 0;
  padding: 4px 12px;
  list-style: none;
}

.sf-topic-contributors-modal__list.is-loading {
  opacity: 0.55;
  pointer-events: none;
}

.sf-topic-contributors-modal__item {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 12px;
  padding: 14px 4px;
  border-bottom: 1px solid var(--sf-border-light, #e2e8f0);
}

.sf-topic-contributors-modal__item:last-child {
  border-bottom: 0;
}

.sf-topic-contributors-modal__body {
  min-width: 0;
}

.sf-topic-contributors-modal__title {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 8px;
  font-size: 14px;
  line-height: 1.4;
}

.sf-topic-contributors-modal__name {
  color: var(--sf-public-text, #0f172a);
  font-weight: 750;
  text-decoration: none;
}

a.sf-topic-contributors-modal__name:hover {
  color: var(--sf-accent, #f97316);
}

.sf-topic-contributors-modal__op {
  color: var(--sf-public-text, #0f172a);
  font-weight: 650;
}

.sf-topic-contributors-modal__origin,
.sf-topic-contributors-modal__current,
.sf-topic-contributors-modal__redacted {
  display: inline-flex;
  padding: 1px 7px;
  border-radius: 999px;
  background: var(--sf-public-surface-muted, #f1f5f9);
  color: var(--sf-public-text-muted, #64748b);
  font-size: 11px;
  font-weight: 700;
}

.sf-topic-contributors-modal__redacted {
  background: color-mix(in srgb, #dc2626 12%, transparent);
  color: #b91c1c;
}

.sf-topic-contributors-modal__meta {
  margin-top: 4px;
  font-size: 12px;
  line-height: 1.45;
  color: var(--sf-public-text-muted, #64748b);
}

.sf-topic-contributors-modal__rev {
  white-space: nowrap;
}

.sf-topic-contributors-modal__pager {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding-top: 2px;
}

.sf-topic-contributors-modal__page-label {
  flex: 1 1 auto;
  text-align: center;
  font-size: 13px;
  font-weight: 650;
  color: var(--sf-public-text-muted, #64748b);
}
</style>
