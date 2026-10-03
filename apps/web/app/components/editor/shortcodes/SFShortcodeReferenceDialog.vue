<script setup lang="ts">
import { apiErrorStatusCode } from '~/composables/useApiClient'
import type {
  EditorReferenceDialogResultV1,
  EditorProtectedKind,
  EditorReferenceKind,
  EditorShortcodeKind,
  EditorShortcodeSelectionV1
} from '~/runtime/editor-extensions/types'
import { sforumProtectedKind, sforumReferenceKind, sforumReferenceNumericID } from '~/utils/editor/shortcodes'

type ReferenceOption = {
  id: number
  label: string
  secondaryLabel?: string
  avatar?: import('~/composables/profile/useProfileApi').AvatarView | null
  icon?: string
  iconColor?: string
}
type ReferenceOptionList = { items: ReferenceOption[], hasMore: boolean }

const props = withDefaults(defineProps<{
  open: boolean
  resources: EditorShortcodeKind[]
  selection?: EditorShortcodeSelectionV1
  disabled?: boolean
}>(), {
  selection: undefined,
  disabled: false
})

const emit = defineEmits<{
  'update:open': [open: boolean]
  resolve: [result: EditorReferenceDialogResultV1]
}>()

const { t } = useI18n()
const { request } = useApiClient()
const activeKind = ref<Exclude<EditorReferenceKind, 'friend-links'> | null>(null)
const query = ref('')
const items = ref<ReferenceOption[]>([])
const status = ref<'idle' | 'loading' | 'ready' | 'empty' | 'denied' | 'error'>('idle')
const searchInput = ref<{ inputRef?: HTMLInputElement } | null>(null)
let searchTimer: ReturnType<typeof setTimeout> | undefined
let requestRevision = 0

const currentKind = computed(() => props.selection
  ? (sforumReferenceKind({ id: props.selection.id }) || sforumProtectedKind({ id: props.selection.id }))
  : null)
const currentID = computed(() => props.selection
  ? sforumReferenceNumericID({ id: props.selection.id, arguments: props.selection.arguments })
  : 0)
const resourceItems = computed(() => props.resources.map(kind => ({
  kind,
  label: t(`composer.shortcodes.kinds.${kind}`),
  icon: iconFor(kind)
})))

watch(() => props.open, async open => {
  if (!open) return
  query.value = ''
  items.value = []
  status.value = 'idle'
  const editableKind = currentKind.value
  activeKind.value = editableKind && isSearchableKind(editableKind)
    && props.resources.includes(editableKind)
    ? editableKind
    : null
  if (activeKind.value) {
    await loadOptions()
    await focusSearch()
  }
})

watch(query, () => {
  if (!props.open || !activeKind.value) return
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => void loadOptions(), 250)
})

onBeforeUnmount(() => {
  if (searchTimer) clearTimeout(searchTimer)
})

async function chooseKind(kind: EditorShortcodeKind) {
  if (props.disabled) return
  if (kind === 'friend-links' || isProtectedKind(kind)) {
    finish({ action: 'upsert', kind })
    return
  }
  activeKind.value = kind
  query.value = ''
  await loadOptions()
  await focusSearch()
}

async function focusSearch() {
  await nextTick()
  const candidate = searchInput.value?.inputRef
    || document.querySelector<HTMLInputElement>('[data-shortcode-reference-search] input')
  candidate?.focus()
}

async function loadOptions() {
  const kind = activeKind.value
  if (!kind) return
  const revision = ++requestRevision
  status.value = 'loading'
  try {
    const params = new URLSearchParams({ kind, query: query.value, limit: '12' })
    if (currentKind.value === kind && currentID.value > 0) {
      params.set('selectedId', String(currentID.value))
    }
    const result = await request<ReferenceOptionList>(`/composer/references?${params}`)
    if (revision !== requestRevision) return
    items.value = Array.isArray(result.items) ? result.items : []
    status.value = items.value.length > 0 ? 'ready' : 'empty'
  } catch (error) {
    if (revision !== requestRevision) return
    items.value = []
    const code = apiErrorStatusCode(error)
    status.value = code === 401 || code === 403 ? 'denied' : 'error'
  }
}

function back() {
  requestRevision += 1
  activeKind.value = null
  query.value = ''
  items.value = []
  status.value = 'idle'
}

function finish(result: EditorReferenceDialogResultV1) {
  emit('resolve', result)
  emit('update:open', false)
}

function close() {
  finish({ action: 'cancel' })
}

function select(item: ReferenceOption) {
  if (!activeKind.value || props.disabled) return
  finish({ action: 'upsert', kind: activeKind.value, id: item.id })
}

function isProtectedKind(kind: EditorShortcodeKind): kind is EditorProtectedKind {
  return kind === 'login' || kind === 'reply' || kind === 'only-author'
}

function isSearchableKind(kind: EditorShortcodeKind): kind is Exclude<EditorReferenceKind, 'friend-links'> {
  return !isProtectedKind(kind) && kind !== 'friend-links'
}

function iconFor(kind: EditorShortcodeKind) {
  switch (kind) {
    case 'user': return 'i-lucide-user-round'
    case 'topic': return 'i-lucide-message-square-text'
    case 'comment': return 'i-lucide-message-circle'
    case 'category': return 'i-lucide-folder'
    case 'friend-links': return 'i-lucide-link-2'
    case 'login': return 'i-lucide-log-in'
    case 'reply': return 'i-lucide-message-square-reply'
    case 'only-author': return 'i-lucide-user-round-check'
  }
}
</script>

<template>
  <UModal
    :open="open"
    :ui="{ content: 'sm:max-w-lg' }"
    @update:open="value => { if (!value) close() }"
  >
    <template #content>
      <section class="sf-shortcode-dialog" aria-labelledby="sf-shortcode-dialog-title">
        <header class="sf-shortcode-dialog__header">
          <UButton
            v-if="activeKind"
            type="button"
            color="neutral"
            variant="ghost"
            icon="i-lucide-arrow-left"
            :aria-label="t('common.back')"
            :title="t('common.back')"
            @click="back"
          />
          <div class="min-w-0 flex-1">
            <h2 id="sf-shortcode-dialog-title" class="sf-shortcode-dialog__title">
              {{ activeKind ? t(`composer.shortcodes.kinds.${activeKind}`) : t('composer.shortcodes.title') }}
            </h2>
            <p class="sf-shortcode-dialog__description">
              {{ activeKind ? t('composer.shortcodes.searchDescription') : t('composer.shortcodes.chooseDescription') }}
            </p>
          </div>
          <UButton
            type="button"
            color="neutral"
            variant="ghost"
            icon="i-lucide-x"
            :aria-label="t('common.close')"
            :title="t('common.close')"
            @click="close"
          />
        </header>

        <div v-if="!activeKind" class="sf-shortcode-dialog__resources" role="list">
          <button
            v-for="item in resourceItems"
            :key="item.kind"
            type="button"
            class="sf-shortcode-dialog__resource"
            role="listitem"
            :disabled="disabled"
            @click="chooseKind(item.kind)"
          >
            <UIcon :name="item.icon" class="size-5" aria-hidden="true" />
            <span>{{ item.label }}</span>
            <UIcon name="i-lucide-chevron-right" class="ml-auto size-4" aria-hidden="true" />
          </button>
        </div>

        <div v-else class="sf-shortcode-dialog__picker">
          <UInput
            ref="searchInput"
            v-model="query"
            data-shortcode-reference-search
            icon="i-lucide-search"
            size="lg"
            :disabled="disabled"
            :placeholder="t('composer.shortcodes.searchPlaceholder')"
            :aria-label="t('composer.shortcodes.searchLabel', { kind: t(`composer.shortcodes.kinds.${activeKind}`) })"
            autocomplete="off"
          />

          <div class="sf-shortcode-dialog__results" aria-live="polite">
            <div v-if="status === 'loading'" class="sf-shortcode-dialog__state">
              <UIcon name="i-lucide-loader-circle" class="size-5 animate-spin" aria-hidden="true" />
              <span>{{ t('composer.shortcodes.loading') }}</span>
            </div>
            <div v-else-if="status === 'empty'" class="sf-shortcode-dialog__state">
              <UIcon name="i-lucide-search-x" class="size-5" aria-hidden="true" />
              <span>{{ t('composer.shortcodes.empty') }}</span>
            </div>
            <div v-else-if="status === 'denied'" class="sf-shortcode-dialog__state sf-shortcode-dialog__state--error" role="alert">
              <UIcon name="i-lucide-shield-alert" class="size-5" aria-hidden="true" />
              <span>{{ t('composer.shortcodes.denied') }}</span>
            </div>
            <div v-else-if="status === 'error'" class="sf-shortcode-dialog__state sf-shortcode-dialog__state--error" role="alert">
              <UIcon name="i-lucide-triangle-alert" class="size-5" aria-hidden="true" />
              <span>{{ t('composer.shortcodes.loadError') }}</span>
              <UButton type="button" color="neutral" variant="soft" size="xs" @click="loadOptions">
                {{ t('common.retry') }}
              </UButton>
            </div>
            <button
              v-for="item in items"
              v-else
              :key="item.id"
              type="button"
              class="sf-shortcode-dialog__option"
              :disabled="disabled"
              @click="select(item)"
            >
              <SFAvatar
                v-if="activeKind === 'user'"
                :name="item.label"
                :avatar="item.avatar"
                size="sm"
                aria-hidden="true"
              />
              <span
                v-else-if="activeKind === 'category'"
                class="sf-shortcode-dialog__option-icon"
                :style="item.iconColor && /^#[0-9a-f]{6}$/i.test(item.iconColor) ? { color: item.iconColor } : undefined"
                aria-hidden="true"
              >
                <UIcon :name="item.icon?.startsWith('i-') ? item.icon : 'i-lucide-folder'" class="size-4" />
              </span>
              <span class="min-w-0">
                <span class="sf-shortcode-dialog__option-label">{{ item.label }}</span>
                <span v-if="item.secondaryLabel" class="sf-shortcode-dialog__option-secondary">{{ item.secondaryLabel }}</span>
              </span>
              <UIcon name="i-lucide-check" class="size-4 shrink-0" aria-hidden="true" />
            </button>
          </div>
        </div>

        <footer v-if="selection" class="sf-shortcode-dialog__footer">
          <UButton
            type="button"
            color="error"
            variant="soft"
            icon="i-lucide-trash-2"
            :disabled="disabled"
            @click="finish({ action: 'delete' })"
          >
            {{ t('composer.shortcodes.delete') }}
          </UButton>
        </footer>
      </section>
    </template>
  </UModal>
</template>
