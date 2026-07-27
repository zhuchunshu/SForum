<script setup lang="ts">
import type { AdminWebOption } from '~/composables/useWebOptions'
import { normalizeTopicUrlMode, recommendedTopicUrlMode } from '~/composables/useWebOptions'
import type { TopicUrlMode } from '~/utils/forum/forumTaxonomy'
import { useAdminOptionForm } from '~/composables/admin/settings/useAdminOptionForm'
import SFAdminSeoTabFrame from '../SFAdminSeoTabFrame.vue'

const props = defineProps<{ items: AdminWebOption[] }>()
const emit = defineEmits<{ saved: [items: AdminWebOption[]] }>()
const { t } = useI18n()
const form = useAdminOptionForm(toRef(props, 'items'), map => ({
  mode: normalizeTopicUrlMode(map['seo.topic_url_mode']?.value || recommendedTopicUrlMode)
}), value => [{ name: 'seo.topic_url_mode', value: value.mode }], () => ({ mode: recommendedTopicUrlMode }), items => emit('saved', items), {
  saved: t('admin.seo.saved'), saveFailed: t('admin.seo.saveFailed'), reset: t('admin.seo.resetChanges'), restored: t('admin.seo.recommendedRestored')
})
const modes = computed<Array<{ value: TopicUrlMode, preview: string, label: string, description: string }>>(() => [
  { value: 'id_slug', preview: '/t/123/hello-world', label: t('admin.seo.topicUrlModeIdSlug'), description: t('admin.seo.topicUrlModeIdSlugHelp') },
  { value: 'id', preview: '/t/123', label: t('admin.seo.topicUrlModeId'), description: t('admin.seo.topicUrlModeIdHelp') },
  { value: 'slug', preview: '/t/hello-world', label: t('admin.seo.topicUrlModeSlug'), description: t('admin.seo.topicUrlModeSlugHelp') }
])
</script>

<template>
  <SFAdminSeoTabFrame tab="permalinks" :dirty="form.hasChanges.value" :saving="form.saving.value" @save="form.save" @reset="form.resetChanges">
    <div class="grid max-w-5xl gap-5">
      <div class="flex justify-end"><UButton type="button" color="neutral" variant="outline" icon="i-lucide-rotate-ccw" @click="form.restoreRecommended">{{ t('admin.seo.restoreRecommended') }}</UButton></div>
      <p class="text-sm text-muted">{{ t('admin.seo.topicUrlModeHelp') }}</p>
      <button v-for="mode in modes" :key="mode.value" type="button" class="flex items-start gap-3 rounded-md border border-slate-200 p-4 text-left dark:border-zinc-800" @click="form.form.mode = mode.value">
        <UIcon :name="form.form.mode === mode.value ? 'i-lucide-circle-check' : 'i-lucide-circle'" class="mt-0.5 size-5" />
        <span><strong class="block text-sm">{{ mode.label }}</strong><code class="mt-1 block text-xs text-muted">{{ mode.preview }}</code><span class="mt-1 block text-xs text-muted">{{ mode.description }}</span></span>
      </button>
      <p v-if="form.form.mode === 'slug'" class="text-xs text-amber-600 dark:text-amber-400">{{ t('admin.seo.topicUrlModeSlugWarning') }}</p>
    </div>
  </SFAdminSeoTabFrame>
</template>
