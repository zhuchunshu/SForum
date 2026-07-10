<script setup lang="ts">
import type { ForumCategory } from '~/utils/forumTaxonomy'

const props = withDefaults(defineProps<{
  categories: ForumCategory[]
  selectedCategorySlug: string
  totalTopics: number
  pending?: boolean
}>(), {
  pending: false
})

const emit = defineEmits<{
  'select-category': [slug: string]
}>()

const { t } = useI18n()

function categoryIcon(category: ForumCategory) {
  return category.icon?.startsWith('i-') ? category.icon : 'i-lucide-hash'
}

function selectFromMenu(event: Event) {
  emit('select-category', (event.target as HTMLSelectElement).value)
}
</script>

<template>
  <aside class="sf-home-navigation" :aria-busy="pending">
    <div class="sf-home-navigation__mobile">
      <label class="sf-home-navigation__select-wrap">
        <span class="sf-home-navigation__select-label">{{ t('home.categories') }}</span>
        <span class="sf-home-navigation__select-control">
          <UIcon name="i-lucide-panels-top-left" class="size-4" aria-hidden="true" />
          <select
            class="sf-home-navigation__select"
            :value="selectedCategorySlug"
            @change="selectFromMenu"
          >
            <option value="">{{ t('home.allTopics') }} ({{ totalTopics }})</option>
            <option v-for="category in categories" :key="category.slug" :value="category.slug">
              {{ category.name }} ({{ category.topicCount }})
            </option>
          </select>
          <UIcon name="i-lucide-chevron-down" class="size-4" aria-hidden="true" />
        </span>
      </label>
    </div>

    <nav class="sf-home-navigation__desktop" :aria-label="t('home.categories')">
      <div v-if="pending" class="sf-home-navigation__pending">
        <SFSkeleton v-for="item in 4" :key="item" :lines="1" />
      </div>
      <template v-else>
        <button
          type="button"
          class="sf-home-navigation__item"
          :class="{ 'is-active': !selectedCategorySlug }"
          :aria-pressed="!selectedCategorySlug"
          :title="t('home.allTopics')"
          @click="emit('select-category', '')"
        >
          <UIcon name="i-lucide-messages-square" class="sf-home-navigation__icon" aria-hidden="true" />
          <span class="sr-only">{{ t('home.allTopics') }}</span>
        </button>

        <button
          v-for="category in categories"
          :key="category.slug"
          type="button"
          class="sf-home-navigation__item"
          :class="{ 'is-active': selectedCategorySlug === category.slug }"
          :aria-pressed="selectedCategorySlug === category.slug"
          :title="category.name"
          @click="emit('select-category', category.slug)"
        >
          <UIcon
            :name="categoryIcon(category)"
            class="sf-home-navigation__icon"
            :style="category.iconColor ? { color: category.iconColor } : undefined"
            aria-hidden="true"
          />
          <span class="sr-only">{{ category.name }}</span>
        </button>
      </template>
    </nav>
  </aside>
</template>
