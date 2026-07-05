<script setup lang="ts">
import { addCollection } from '@iconify/vue'
import {
  ICON_PICKER_PAGE_SIZE,
  collectionFromName,
  dedupeIconItems,
  isNuxtIconName,
  normalizeIconName,
  type IconCollectionId,
  type IconPickerItem,
  type IconSource
} from '~/utils/iconPicker'

type IconCollection = {
  id: IconCollectionId
  prefix: string
  label: string
  description: string
}

type IconifyCollectionId = 'lucide' | 'tabler'

type IconCatalogResponse = {
  collection: IconCollectionId
  iconifyPrefix: IconifyCollectionId
  page: number
  pageSize: number
  total: number
  hasMore: boolean
  items: IconPickerItem[]
}

type IconifyCollectionPayload = Parameters<typeof addCollection>[0]

const props = withDefaults(defineProps<{
  modelValue?: string
  label?: string
  hint?: string
  id?: string
  disabled?: boolean
  showCustomInput?: boolean
}>(), {
  modelValue: '',
  label: undefined,
  hint: undefined,
  id: undefined,
  disabled: false,
  showCustomInput: true
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
  change: [value: { name: string, collection: IconCollectionId, source: IconSource }]
}>()

const { t } = useI18n()
const generatedId = useId()
const pickerId = computed(() => props.id || generatedId)
const query = ref('')
const customName = ref('')
const activeCollectionId = ref<IconCollectionId>(collectionFromName(props.modelValue) || 'tabler')
const loadedIcons = ref<IconPickerItem[]>([])
const loadedPage = ref(0)
const totalIcons = ref(0)
const loadingIcons = ref(false)
const iconLoadError = ref(false)
const selectedIconRenderKey = ref(0)

let iconLoadRequestId = 0
let searchTimer: ReturnType<typeof setTimeout> | undefined

const collections = computed<IconCollection[]>(() => [
  {
    id: 'tabler',
    prefix: 'i-tabler-',
    label: t('components.iconPicker.collections.tabler.label'),
    description: t('components.iconPicker.collections.tabler.description')
  },
  {
    id: 'nuxt',
    prefix: 'i-lucide-',
    label: t('components.iconPicker.collections.nuxt.label'),
    description: t('components.iconPicker.collections.nuxt.description')
  }
])

const activeCollection = computed<IconCollection>(() => {
  const firstCollection = collections.value[0]!
  return collections.value.find((collection) => collection.id === activeCollectionId.value) || firstCollection
})

const fieldLabel = computed(() => props.label || t('components.iconPicker.label'))
const fieldHint = computed(() => props.hint || t('components.iconPicker.hint'))
const selectedIconName = computed(() => props.modelValue.trim())
const selectedDisplayName = computed(() => selectedIconName.value || t('components.iconPicker.noSelection'))
const normalizedCustomName = computed(() => normalizeIconName(customName.value, activeCollection.value.prefix))
const canUseCustomName = computed(() => isNuxtIconName(normalizedCustomName.value))
const hasMoreIcons = computed(() => loadedIcons.value.length < totalIcons.value)
const iconCountLabel = computed(() => {
  if (totalIcons.value <= 0) {
    return ''
  }

  return t('components.iconPicker.count', {
    shown: loadedIcons.value.length,
    total: totalIcons.value
  })
})

watch(() => props.modelValue, (value) => {
  const nextCollection = collectionFromName(value)
  if (nextCollection) {
    activeCollectionId.value = nextCollection
  }
})

watch([activeCollectionId, query], ([collection], [previousCollection]) => {
  scheduleIconReload(collection === previousCollection ? 180 : 0)
})

watch(selectedIconName, () => {
  void primeSelectedIcon()
})

onMounted(() => {
  void loadIconPage({ reset: true })
  void primeSelectedIcon()
})

onBeforeUnmount(() => {
  if (searchTimer) {
    clearTimeout(searchTimer)
  }
})

function setActiveCollection(id: IconCollectionId) {
  if (!props.disabled) {
    activeCollectionId.value = id
    query.value = ''
  }
}

function selectIcon(name: string, source: IconSource) {
  if (props.disabled) {
    return
  }

  const collection = collectionFromName(name) || activeCollectionId.value
  emit('update:modelValue', name)
  emit('change', { name, collection, source })
}

function selectCustomIcon() {
  if (!canUseCustomName.value) {
    return
  }

  selectIcon(normalizedCustomName.value, 'custom')
}

async function loadIconPage(options: { reset?: boolean } = {}) {
  if (!import.meta.client) {
    return
  }

  if (loadingIcons.value && !options.reset) {
    return
  }

  const requestId = ++iconLoadRequestId
  if (options.reset) {
    loadedIcons.value = []
    loadedPage.value = 0
    totalIcons.value = 0
  }

  loadingIcons.value = true
  iconLoadError.value = false

  try {
    const nextPage = options.reset ? 1 : loadedPage.value + 1
    const response = await $fetch<IconCatalogResponse>(`/api/icon-collections/${activeCollectionId.value}`, {
      query: {
        q: query.value.trim() || undefined,
        page: nextPage,
        pageSize: ICON_PICKER_PAGE_SIZE
      }
    })

    if (requestId !== iconLoadRequestId) {
      return
    }

    await primeIconifyData(response.items.map((item) => item.name))
    if (requestId !== iconLoadRequestId) {
      return
    }

    loadedIcons.value = options.reset
      ? response.items
      : dedupeIconItems([...loadedIcons.value, ...response.items])
    loadedPage.value = response.page
    totalIcons.value = response.total
  } catch {
    if (requestId === iconLoadRequestId) {
      iconLoadError.value = true
    }
  } finally {
    if (requestId === iconLoadRequestId) {
      loadingIcons.value = false
    }
  }
}

async function primeSelectedIcon() {
  if (!selectedIconName.value) {
    return
  }

  await primeIconifyData([selectedIconName.value])
  selectedIconRenderKey.value += 1
}

async function primeIconifyData(iconNames: string[]) {
  if (!import.meta.client || iconNames.length === 0) {
    return
  }

  const grouped = new Map<IconifyCollectionId, Set<string>>()
  for (const iconName of iconNames) {
    const iconifyName = toIconifyName(iconName)
    if (!iconifyName) {
      continue
    }

    const names = grouped.get(iconifyName.collection) || new Set<string>()
    names.add(iconifyName.name)
    grouped.set(iconifyName.collection, names)
  }

  await Promise.all([...grouped.entries()].map(async ([collection, names]) => {
    const payload = await $fetch<IconifyCollectionPayload>(`/api/_nuxt_icon/${collection}.json`, {
      query: {
        icons: [...names].join(',')
      }
    })
    if (payload?.prefix === collection && payload.icons) {
      addCollection(payload)
    }
  }))
}

function toIconifyName(value: string): { collection: IconifyCollectionId, name: string } | undefined {
  const name = value.trim().toLowerCase()
  if (name.startsWith('i-tabler-')) {
    return { collection: 'tabler', name: name.slice('i-tabler-'.length) }
  }
  if (name.startsWith('i-lucide-')) {
    return { collection: 'lucide', name: name.slice('i-lucide-'.length) }
  }
  if (name.startsWith('tabler:')) {
    return { collection: 'tabler', name: name.slice('tabler:'.length) }
  }
  if (name.startsWith('lucide:')) {
    return { collection: 'lucide', name: name.slice('lucide:'.length) }
  }
  return undefined
}

function scheduleIconReload(delay: number) {
  if (!import.meta.client) {
    return
  }

  if (searchTimer) {
    clearTimeout(searchTimer)
  }

  searchTimer = setTimeout(() => {
    void loadIconPage({ reset: true })
  }, delay)
}

function handleGridScroll(event: Event) {
  const target = event.currentTarget
  if (!(target instanceof HTMLElement) || loadingIcons.value || !hasMoreIcons.value) {
    return
  }

  if (target.scrollTop + target.clientHeight >= target.scrollHeight - 64) {
    void loadIconPage()
  }
}

function reloadIcons() {
  void loadIconPage({ reset: true })
}
</script>

<template>
  <div class="sf-icon-picker" :aria-disabled="disabled ? 'true' : undefined">
    <div class="sf-icon-picker__header">
      <label class="sf-icon-picker__label" :for="`${pickerId}-search`">
        {{ fieldLabel }}
      </label>
      <p v-if="fieldHint" class="sf-icon-picker__hint">
        {{ fieldHint }}
      </p>
    </div>

    <div class="sf-icon-picker__surface">
      <div class="sf-icon-picker__preview">
        <span class="sf-icon-picker__preview-icon" aria-hidden="true">
          <UIcon
            v-if="selectedIconName"
            :key="`${selectedIconName}-${selectedIconRenderKey}`"
            :name="selectedIconName"
            class="sf-icon-picker__preview-svg"
          />
          <UIcon
            v-else
            name="i-lucide-image"
            class="sf-icon-picker__preview-svg"
          />
        </span>
        <div class="sf-icon-picker__preview-copy">
          <span class="sf-icon-picker__preview-label">{{ t('components.iconPicker.selected') }}</span>
          <code class="sf-icon-picker__value">{{ selectedDisplayName }}</code>
        </div>
      </div>

      <div class="sf-icon-picker__collections" role="tablist" :aria-label="t('components.iconPicker.collectionLabel')">
        <button
          v-for="collection in collections"
          :key="collection.id"
          type="button"
          class="sf-icon-picker__collection"
          :class="{ 'sf-icon-picker__collection--active': activeCollectionId === collection.id }"
          role="tab"
          :aria-selected="activeCollectionId === collection.id ? 'true' : 'false'"
          :disabled="disabled"
          @click="setActiveCollection(collection.id)"
        >
          <span class="sf-icon-picker__collection-label">{{ collection.label }}</span>
          <span class="sf-icon-picker__collection-description">{{ collection.description }}</span>
        </button>
      </div>

      <div class="sf-icon-picker__search">
        <UIcon name="i-lucide-search" class="sf-icon-picker__search-icon" aria-hidden="true" />
        <input
          :id="`${pickerId}-search`"
          v-model="query"
          class="sf-icon-picker__search-input"
          type="search"
          :placeholder="t('components.iconPicker.searchPlaceholder')"
          :disabled="disabled"
        >
      </div>

      <div class="sf-icon-picker__meta" aria-live="polite">
        <span v-if="iconLoadError" class="sf-icon-picker__status sf-icon-picker__status--error">
          {{ t('components.iconPicker.loadFailed') }}
        </span>
        <span v-else-if="iconCountLabel" class="sf-icon-picker__status">
          {{ iconCountLabel }}
        </span>
        <span v-else class="sf-icon-picker__status">
          {{ t('components.iconPicker.loading') }}
        </span>
        <button
          v-if="iconLoadError"
          type="button"
          class="sf-icon-picker__retry"
          :disabled="disabled || loadingIcons"
          @click="reloadIcons"
        >
          <UIcon name="i-lucide-refresh-cw" class="sf-icon-picker__retry-icon" aria-hidden="true" />
          <span>{{ t('components.iconPicker.retry') }}</span>
        </button>
      </div>

      <div class="sf-icon-picker__grid" role="listbox" :aria-label="activeCollection.label" @scroll.passive="handleGridScroll">
        <button
          v-for="item in loadedIcons"
          :key="item.name"
          type="button"
          class="sf-icon-picker__option"
          :class="{ 'sf-icon-picker__option--selected': selectedIconName === item.name }"
          role="option"
          :aria-selected="selectedIconName === item.name ? 'true' : 'false'"
          :title="item.name"
          :disabled="disabled"
          @click="selectIcon(item.name, 'preset')"
        >
          <UIcon :name="item.name" class="sf-icon-picker__option-icon" aria-hidden="true" />
          <span class="sf-icon-picker__option-label">{{ item.label }}</span>
        </button>
      </div>

      <p v-if="loadingIcons" class="sf-icon-picker__loading">
        <UIcon name="i-lucide-loader-circle" class="sf-icon-picker__loading-icon" aria-hidden="true" />
        <span>{{ t('components.iconPicker.loading') }}</span>
      </p>

      <p v-else-if="loadedIcons.length === 0" class="sf-icon-picker__empty">
        {{ t('components.iconPicker.empty') }}
      </p>

      <div v-if="showCustomInput" class="sf-icon-picker__custom">
        <label class="sf-icon-picker__custom-label" :for="`${pickerId}-custom`">
          {{ t('components.iconPicker.customLabel') }}
        </label>
        <div class="sf-icon-picker__custom-row">
          <input
            :id="`${pickerId}-custom`"
            v-model="customName"
            class="sf-icon-picker__custom-input"
            type="text"
            spellcheck="false"
            :placeholder="t('components.iconPicker.customPlaceholder')"
            :disabled="disabled"
            @keydown.enter.prevent="selectCustomIcon"
          >
          <button
            type="button"
            class="sf-icon-picker__custom-button"
            :disabled="disabled || !canUseCustomName"
            @click="selectCustomIcon"
          >
            <UIcon name="i-lucide-check" class="sf-icon-picker__custom-button-icon" aria-hidden="true" />
            <span>{{ t('components.iconPicker.useCustom') }}</span>
          </button>
        </div>
        <code v-if="normalizedCustomName" class="sf-icon-picker__normalized">
          {{ normalizedCustomName }}
        </code>
      </div>
    </div>
  </div>
</template>
