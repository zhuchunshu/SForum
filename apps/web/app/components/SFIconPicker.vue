<script setup lang="ts">
type IconCollectionId = 'tabler' | 'nuxt'
type IconSource = 'preset' | 'custom'

type IconPreset = {
  name: string
  label: string
  keywords: string[]
}

type IconCollection = {
  id: IconCollectionId
  prefix: string
  label: string
  description: string
  icons: IconPreset[]
}

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

const tablerIcons: IconPreset[] = [
  icon('i-tabler-layout-dashboard', 'layout-dashboard', ['dashboard', 'admin', '控制台', '仪表盘']),
  icon('i-tabler-settings', 'settings', ['settings', 'config', '设置']),
  icon('i-tabler-adjustments-horizontal', 'adjustments-horizontal', ['options', 'sliders', '配置']),
  icon('i-tabler-user-circle', 'user-circle', ['user', 'profile', '用户']),
  icon('i-tabler-user-cog', 'user-cog', ['admin', 'permission', '用户设置']),
  icon('i-tabler-users', 'users', ['group', 'members', '用户组']),
  icon('i-tabler-shield-lock', 'shield-lock', ['security', 'permission', '权限']),
  icon('i-tabler-message-circle', 'message-circle', ['forum', 'topic', '帖子']),
  icon('i-tabler-bell', 'bell', ['notice', 'notification', '通知']),
  icon('i-tabler-search', 'search', ['find', 'query', '搜索']),
  icon('i-tabler-tags', 'tags', ['tag', 'label', '标签']),
  icon('i-tabler-folder', 'folder', ['category', 'board', '版块']),
  icon('i-tabler-pin', 'pin', ['pinned', 'top', '置顶']),
  icon('i-tabler-star', 'star', ['featured', 'favorite', '精华']),
  icon('i-tabler-heart', 'heart', ['like', 'reaction', '喜欢']),
  icon('i-tabler-photo', 'photo', ['image', 'media', '图片']),
  icon('i-tabler-file-text', 'file-text', ['document', 'article', '文档']),
  icon('i-tabler-book', 'book', ['docs', 'guide', '指南']),
  icon('i-tabler-calendar', 'calendar', ['date', 'schedule', '日程']),
  icon('i-tabler-mail', 'mail', ['email', 'inbox', '邮件']),
  icon('i-tabler-database', 'database', ['data', 'storage', '数据库']),
  icon('i-tabler-chart-bar', 'chart-bar', ['analytics', 'stats', '统计']),
  icon('i-tabler-world', 'world', ['site', 'global', '站点']),
  icon('i-tabler-language', 'language', ['locale', 'i18n', '语言']),
  icon('i-tabler-plug-connected', 'plug-connected', ['integration', 'provider', '集成']),
  icon('i-tabler-terminal-2', 'terminal-2', ['system', 'developer', '终端'])
]

const nuxtIcons: IconPreset[] = [
  icon('i-lucide-layout-dashboard', 'layout-dashboard', ['dashboard', 'admin', '控制台', '仪表盘']),
  icon('i-lucide-settings-2', 'settings-2', ['settings', 'config', '设置']),
  icon('i-lucide-sliders-horizontal', 'sliders-horizontal', ['options', 'controls', '配置']),
  icon('i-lucide-user-cog', 'user-cog', ['admin', 'permission', '用户设置']),
  icon('i-lucide-contact', 'contact', ['user', 'profile', '用户']),
  icon('i-lucide-users', 'users', ['group', 'members', '用户组']),
  icon('i-lucide-shield-check', 'shield-check', ['security', 'permission', '权限']),
  icon('i-lucide-message-square-text', 'message-square-text', ['forum', 'topic', '帖子']),
  icon('i-lucide-list-checks', 'list-checks', ['checklist', 'matrix', '清单']),
  icon('i-lucide-bell', 'bell', ['notice', 'notification', '通知']),
  icon('i-lucide-search', 'search', ['find', 'query', '搜索']),
  icon('i-lucide-tag', 'tag', ['tag', 'label', '标签']),
  icon('i-lucide-house', 'house', ['home', 'forum', '首页']),
  icon('i-lucide-lock-keyhole', 'lock-keyhole', ['security', 'locked', '锁定']),
  icon('i-lucide-key-round', 'key-round', ['secret', 'token', '密钥']),
  icon('i-lucide-palette', 'palette', ['theme', 'color', '主题']),
  icon('i-lucide-image', 'image', ['image', 'media', '图片']),
  icon('i-lucide-link', 'link', ['url', 'external', '链接']),
  icon('i-lucide-database', 'database', ['data', 'storage', '数据库']),
  icon('i-lucide-languages', 'languages', ['locale', 'i18n', '语言']),
  icon('i-lucide-plug', 'plug', ['integration', 'provider', '集成']),
  icon('i-lucide-save', 'save', ['persist', '保存']),
  icon('i-lucide-refresh-cw', 'refresh-cw', ['reload', 'sync', '刷新'])
]

const collections = computed<IconCollection[]>(() => [
  {
    id: 'tabler',
    prefix: 'i-tabler-',
    label: t('components.iconPicker.collections.tabler.label'),
    description: t('components.iconPicker.collections.tabler.description'),
    icons: tablerIcons
  },
  {
    id: 'nuxt',
    prefix: 'i-lucide-',
    label: t('components.iconPicker.collections.nuxt.label'),
    description: t('components.iconPicker.collections.nuxt.description'),
    icons: nuxtIcons
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

const filteredIcons = computed(() => {
  const normalizedQuery = query.value.trim().toLowerCase()
  if (!normalizedQuery) {
    return activeCollection.value.icons
  }

  return activeCollection.value.icons.filter((item) => {
    const searchText = [item.name, item.label, ...item.keywords].join(' ').toLowerCase()
    return searchText.includes(normalizedQuery)
  })
})

watch(() => props.modelValue, (value) => {
  const nextCollection = collectionFromName(value)
  if (nextCollection) {
    activeCollectionId.value = nextCollection
  }
})

function icon(name: string, label: string, keywords: string[]): IconPreset {
  return { name, label, keywords }
}

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

function normalizeIconName(value: string, fallbackPrefix: string) {
  const rawValue = value.trim().toLowerCase()
  if (!rawValue) {
    return ''
  }

  if (rawValue.startsWith('i-')) {
    return rawValue
  }

  if (/^[a-z0-9-]+:[a-z0-9-]+$/.test(rawValue)) {
    const [collection, name] = rawValue.split(':')
    return `i-${collection}-${name}`
  }

  if (/^[a-z0-9][a-z0-9-]*$/.test(rawValue)) {
    return `${fallbackPrefix}${rawValue}`
  }

  return rawValue
}

function collectionFromName(value?: string): IconCollectionId | undefined {
  const name = value?.trim().toLowerCase()
  if (!name) {
    return undefined
  }

  if (name.startsWith('i-tabler-') || name.startsWith('tabler:')) {
    return 'tabler'
  }

  return 'nuxt'
}

function isNuxtIconName(value: string) {
  return value.startsWith('i-') && value.length > 2
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

      <div class="sf-icon-picker__grid" role="listbox" :aria-label="activeCollection.label">
        <button
          v-for="item in filteredIcons"
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

      <p v-if="filteredIcons.length === 0" class="sf-icon-picker__empty">
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
