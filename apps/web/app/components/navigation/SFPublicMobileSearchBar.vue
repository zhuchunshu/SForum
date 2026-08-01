<script setup lang="ts">
import SFSearch from '~/components/SFSearch.vue'

const props = defineProps<{
  modelValue: string
  canCreateTopic: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
  submit: [value: string]
}>()

const { t } = useI18n()
const localePath = useLocalePath()
const searchQuery = computed({
  get: () => props.modelValue,
  set: value => emit('update:modelValue', value)
})
</script>

<template>
  <div class="sf-public-mobile-search-bar">
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
    </div>
  </div>
</template>

<style scoped>
.sf-public-mobile-search-bar {
  display: none;
  border-top: 1px solid #e4e8ef;
  background: #fff;
}

.sf-public-mobile-search-bar__inner {
  display: flex;
  align-items: center;
  gap: 8px;
  max-width: var(--sf-public-container);
  margin: 0 auto;
  padding: 8px 24px;
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

:global(.dark) .sf-public-mobile-search-bar {
  border-top-color: #27272a;
  background: #09090b;
}

@media (max-width: 980px) {
  .sf-public-mobile-search-bar {
    display: block;
    height: 54px;
    border-top: 0;
    background: transparent;
  }

  .sf-public-mobile-search-bar__inner {
    height: 54px;
    max-width: none;
    padding: 10px 12px 8px;
  }

  .sf-public-mobile-search-bar__field {
    min-width: 40px;
    min-height: 40px;
  }

  .sf-public-mobile-search-bar__field :deep(.sf-search__box) {
    min-height: 40px;
  }

  .sf-public-mobile-search-bar__field .sf-search__box {
    min-height: 38px;
    border: 0;
    border-radius: 12px;
    background: var(--sf-public-surface);
    box-shadow: 0 2px 12px rgb(51 56 80 / 0.055);
  }

  .sf-public-mobile-search-bar__compose {
    min-height: 38px;
    border-radius: 10px;
  }
}

@media (max-width: 520px) {
  .sf-public-mobile-search-bar__inner {
    padding: 7px 10px;
  }
}
</style>
