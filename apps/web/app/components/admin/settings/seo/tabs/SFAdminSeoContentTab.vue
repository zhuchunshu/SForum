<script setup lang="ts">
import type { AdminWebOption } from '~/composables/useWebOptions'
import { enabledOptionValue, normalizeEnabledOption } from '~/composables/useWebOptions'
import { useAdminOptionForm } from '~/composables/admin/settings/useAdminOptionForm'
import SFAdminSeoTabFrame from '../SFAdminSeoTabFrame.vue'
import SFSEOContentTypes from './SFSEOContentTypes.vue'

type ContentType = 'category' | 'tag' | 'topic' | 'profile' | 'static'
type Policy = { titleTemplate: string, descriptionSource: string, defaultImageUrl: string, indexMode: 'index' | 'noindex', includeInSitemap: boolean, schemaType: string }
type ContentForm = { policies: Record<ContentType, Policy> }
const contentTypes: ContentType[] = ['category', 'tag', 'topic', 'profile', 'static']

const props = defineProps<{ items: AdminWebOption[] }>()
const emit = defineEmits<{ saved: [items: AdminWebOption[]] }>()
const { t } = useI18n()
const form = useAdminOptionForm<ContentForm>(
  toRef(props, 'items'),
  map => {
    const defaults = recommendedContentPolicies()
    for (const type of contentTypes) {
      const prefix = `seo.content_type.${type}`
      const policy = defaults[type]
      policy.titleTemplate = map[`${prefix}.title_template`]?.value || policy.titleTemplate
      policy.descriptionSource = map[`${prefix}.description_source`]?.value || policy.descriptionSource
      policy.defaultImageUrl = map[`${prefix}.default_image_url`]?.value || ''
      policy.indexMode = map[`${prefix}.index_mode`]?.value === 'index' ? 'index' : 'noindex'
      policy.includeInSitemap = normalizeEnabledOption(map[`${prefix}.include_in_sitemap`]?.value, policy.includeInSitemap)
      policy.schemaType = map[`${prefix}.schema_type`]?.value || policy.schemaType
    }
    return { policies: defaults }
  },
  value => contentTypes.flatMap(type => {
    const prefix = `seo.content_type.${type}`
    const policy = value.policies[type]
    return [
      { name: `${prefix}.title_template`, value: policy.titleTemplate },
      { name: `${prefix}.description_source`, value: policy.descriptionSource },
      { name: `${prefix}.default_image_url`, value: policy.defaultImageUrl },
      { name: `${prefix}.index_mode`, value: policy.indexMode },
      { name: `${prefix}.include_in_sitemap`, value: enabledOptionValue(policy.includeInSitemap) },
      { name: `${prefix}.schema_type`, value: policy.schemaType }
    ]
  }),
  () => ({ policies: recommendedContentPolicies() }),
  items => emit('saved', items),
  { saved: t('admin.seo.saved'), saveFailed: t('admin.seo.saveFailed'), reset: t('admin.seo.resetChanges'), restored: t('admin.seo.recommendedRestored') }
)

function recommendedContentPolicies(): Record<ContentType, Policy> {
  return {
    category: { titleTemplate: '{categoryName} | {seoSiteName}', descriptionSource: 'category_description,site_default', defaultImageUrl: '', indexMode: 'index', includeInSitemap: true, schemaType: 'CollectionPage' },
    tag: { titleTemplate: '{tagName} | {seoSiteName}', descriptionSource: 'tag_description,site_default', defaultImageUrl: '', indexMode: 'index', includeInSitemap: true, schemaType: 'CollectionPage' },
    topic: { titleTemplate: '{topicTitle} | {seoSiteName}', descriptionSource: 'topic_summary,topic_excerpt,site_default', defaultImageUrl: '', indexMode: 'index', includeInSitemap: true, schemaType: 'DiscussionForumPosting' },
    profile: { titleTemplate: '{authorName} | {seoSiteName}', descriptionSource: 'profile_bio,site_default', defaultImageUrl: '', indexMode: 'noindex', includeInSitemap: false, schemaType: 'ProfilePage' },
    static: { titleTemplate: '{pageTitle} | {seoSiteName}', descriptionSource: 'page_description,site_default', defaultImageUrl: '', indexMode: 'index', includeInSitemap: true, schemaType: 'WebPage' }
  }
}
</script>

<template>
  <SFAdminSeoTabFrame tab="content" :dirty="form.hasChanges.value" :saving="form.saving.value" @save="form.save" @reset="form.resetChanges">
    <SFSEOContentTypes v-model="form.form.policies" @restore="form.restoreRecommended" />
  </SFAdminSeoTabFrame>
</template>
