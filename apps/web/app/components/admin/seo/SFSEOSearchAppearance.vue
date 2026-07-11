<script setup lang="ts">
import SFSEOImagePicker from './SFSEOImagePicker.vue'

const inheritSiteName = defineModel<boolean>('inheritSiteName', { required: true })
const seoSiteName = defineModel<string>('seoSiteName', { required: true })
const homeTitle = defineModel<string>('homeTitle', { required: true })
const homeDescription = defineModel<string>('homeDescription', { required: true })
const homeKeywords = defineModel<string>('homeKeywords', { required: true })
const homeOGImageUrl = defineModel<string>('homeOGImageUrl', { required: true })
defineProps<{ productSiteName: string, siteUrl: string }>()
const emit = defineEmits<{ restore: [] }>()
const { t } = useI18n()
</script>

<template>
  <div class="grid gap-5 xl:grid-cols-[minmax(0,1fr)_360px]">
    <div class="grid gap-5">
      <div class="flex justify-end"><UButton type="button" color="neutral" variant="outline" icon="i-lucide-rotate-ccw" @click="emit('restore')">{{ t('admin.seo.restoreRecommended') }}</UButton></div>
      <section class="grid gap-4 border-b border-slate-200 pb-5 dark:border-zinc-800">
        <h3 class="text-sm font-bold">{{ t('admin.seo.siteIdentity') }}</h3>
        <label class="flex items-start gap-3">
          <UCheckbox v-model="inheritSiteName" />
          <span><b class="block text-sm">{{ t('admin.seo.inheritSiteName') }}</b><small class="text-slate-500">{{ productSiteName }}</small></span>
        </label>
        <UFormField v-if="!inheritSiteName" :label="t('admin.seo.seoSiteName')">
          <UInput v-model="seoSiteName" icon="i-lucide-building-2" maxlength="120" class="w-full" />
        </UFormField>
      </section>
      <section class="grid gap-4">
        <h3 class="text-sm font-bold">{{ t('admin.seo.homeSearchInfo') }}</h3>
        <UFormField :label="t('admin.seo.homeTitle')"><UInput v-model="homeTitle" maxlength="120" class="w-full" /></UFormField>
        <UFormField :label="t('admin.seo.homeDescription')"><UTextarea v-model="homeDescription" :rows="3" maxlength="320" class="w-full" /></UFormField>
        <UFormField :label="t('admin.seo.homeKeywords')" :help="t('admin.seo.homeKeywordsHelp')"><UInput v-model="homeKeywords" icon="i-lucide-tags" maxlength="200" class="w-full" /></UFormField>
        <SFSEOImagePicker v-model="homeOGImageUrl" context="home-og-image" :label="t('admin.seo.homeSocialImage')" recommended="1200 x 630" />
      </section>
    </div>
    <aside class="h-fit border border-slate-200 bg-slate-50 p-4 dark:border-zinc-800 dark:bg-zinc-950/60 xl:sticky xl:top-4">
      <b class="text-sm">{{ t('admin.seo.searchPreview') }}</b>
      <div class="mt-4 break-words text-lg text-blue-700 dark:text-blue-400">{{ homeTitle || seoSiteName || productSiteName }}</div>
      <div class="mt-1 break-all text-xs text-emerald-700 dark:text-emerald-400">{{ siteUrl }}/</div>
      <p class="mt-2 text-sm leading-6 text-slate-600 dark:text-zinc-300">{{ homeDescription }}</p>
      <img v-if="homeOGImageUrl" :src="homeOGImageUrl" alt="" class="mt-4 aspect-[1200/630] w-full object-cover">
    </aside>
  </div>
</template>
