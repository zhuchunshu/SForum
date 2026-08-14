<script setup lang="ts">
import SFSearch from '~/components/SFSearch.vue'

const props = defineProps<{
  modelValue: string
  canCreateTopic: boolean
  /** 移动端搜索面板是否展开（topbar 搜索图标触发） */
  open: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
  submit: [value: string]
  close: []
}>()

const { t } = useI18n()
const localePath = useLocalePath()
const searchQuery = computed({
  get: () => props.modelValue,
  set: value => emit('update:modelValue', value)
})
</script>

<template>
  <div class="sf-public-mobile-search-bar" :class="{ 'is-open': open }" :hidden="!open">
    <div class="sf-public-mobile-search-bar__inner">
      <SFSearch
        v-model="searchQuery"
        class="sf-public-mobile-search-bar__field"
        :placeholder="t('home.searchPlaceholder')"
        :aria-label="t('nav.search')"
        @submit="emit('submit', $event)"
      />
      <NuxtLink
        :to="canCreateTopic ? localePath('/topics/new') : localePath('/login')"
        class="sf-public-mobile-search-bar__compose"
      >
        <UIcon :name="canCreateTopic ? 'i-lucide-square-pen' : 'i-lucide-log-in'" class="size-4" aria-hidden="true" />
        <span>{{ canCreateTopic ? t('nav.newTopic') : t('nav.login') }}</span>
      </NuxtLink>
      <button
        type="button"
        class="sf-public-mobile-search-bar__cancel"
        :aria-label="t('nav.closeSearch')"
        :title="t('nav.closeSearch')"
        @click="emit('close')"
      >
        <UIcon name="i-lucide-x" class="size-4" aria-hidden="true" />
      </button>
    </div>
  </div>
</template>

<style scoped>
.sf-public-mobile-search-bar {
  display: none;
}

.sf-public-mobile-search-bar.is-open {
  display: block;
  position: fixed;
  z-index: 45;
  top: var(--sf-public-topbar-height, 54px);
  right: 0;
  left: 0;
  border-bottom: 1px solid var(--sf-public-border);
  background: var(--sf-public-surface);
  box-shadow: 0 10px 24px rgb(30 36 56 / 0.08);
}

.sf-public-mobile-search-bar__inner {
  display: flex;
  align-items: center;
  gap: 8px;
  max-width: var(--sf-public-container);
  margin: 0 auto;
  padding: 10px 12px;
}

.sf-public-mobile-search-bar__field {
  flex: 1;
}

.sf-public-mobile-search-bar__compose {
  min-height: 40px;
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border-radius: 7px;
  padding: 0 13px;
  background: var(--sf-accent);
  color: #fff;
  font-size: 12px;
  font-weight: 700;
  text-decoration: none;
}

.sf-public-mobile-search-bar__cancel {
  min-height: 40px;
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  gap: 4px;
  border: 0;
  border-radius: 7px;
  padding: 0 10px;
  background: transparent;
  color: var(--sf-public-text-muted);
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
}

.sf-public-mobile-search-bar__cancel:hover {
  background: var(--sf-public-surface-muted);
  color: var(--sf-public-text);
}

:global(.dark) .sf-public-mobile-search-bar.is-open {
  border-bottom-color: #27272a;
  background: #09090b;
}

@media (max-width: 980px) {
  .sf-public-mobile-search-bar.is-open {
    top: var(--sf-public-topbar-height, 54px);
  }

  .sf-public-mobile-search-bar__inner {
    max-width: none;
    padding: 8px 12px;
  }

  .sf-public-mobile-search-bar__field {
    min-width: 40px;
    min-height: 40px;
  }

  .sf-public-mobile-search-bar__field :deep(.sf-search__box) {
    min-height: 40px;
    border: 0;
    border-radius: 12px;
    background: var(--sf-public-surface-muted);
  }

  .sf-public-mobile-search-bar__compose,
  .sf-public-mobile-search-bar__cancel {
    min-height: 38px;
    border-radius: 10px;
  }
}

@media (max-width: 520px) {
  .sf-public-mobile-search-bar__inner {
    padding: 8px 10px;
  }
}
</style>
