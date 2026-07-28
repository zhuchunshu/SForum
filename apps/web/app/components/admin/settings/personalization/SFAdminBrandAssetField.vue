<script setup lang="ts">
type BrandAsset = {
  id: number
  publicId: string
  url: string
  contentType: string
  size: number
  width?: number
  height?: number
}

const props = withDefaults(defineProps<{
  context: 'logo' | 'favicon' | 'apple-touch-icon'
  label: string
  urlLabel: string
  attachmentIdLabel: string
  icon: string
  previewShape?: 'wide' | 'square'
  disabled?: boolean
}>(), {
  previewShape: 'square',
  disabled: false
})
const url = defineModel<string>('url', { required: true })
const attachmentId = defineModel<string>('attachmentId', { required: true })
const { t } = useI18n()
const toast = useToast()
const { request } = useApiClient()
const fileInput = ref<HTMLInputElement>()
const uploading = ref(false)
const dragging = ref(false)
const errorMessage = ref('')
const previewUrl = computed(() => url.value.trim())

watch(url, () => {
  errorMessage.value = ''
})

function openPicker() {
  if (props.disabled || uploading.value) return
  if (fileInput.value) fileInput.value.value = ''
  fileInput.value?.click()
}

async function uploadFile(file?: File) {
  if (!file || props.disabled || uploading.value) return
  const isSVG = file.type === 'image/svg+xml' || file.name.toLowerCase().endsWith('.svg')
  if (!file.type.startsWith('image/') && !isSVG) {
    errorMessage.value = t('admin.siteChrome.brand.invalidImage')
    return
  }
  uploading.value = true
  errorMessage.value = ''
  try {
    const body = new FormData()
    body.append('context', props.context)
    body.append('file', file)
    const asset = await request<BrandAsset>('/admin/site/brand-assets', { method: 'POST', body })
    url.value = asset.url
    attachmentId.value = String(asset.id)
    toast.add({
      color: 'success',
      icon: 'i-lucide-image-up',
      title: t('admin.siteChrome.brand.uploaded'),
      duration: 10000
    })
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('admin.siteChrome.brand.uploadFailed')
  } finally {
    uploading.value = false
  }
}

function onDrop(event: DragEvent) {
  dragging.value = false
  void uploadFile(event.dataTransfer?.files?.[0])
}

function updateURL(value: string | number) {
  url.value = String(value)
  attachmentId.value = ''
}

function imageFailed() {
  if (previewUrl.value) errorMessage.value = t('admin.siteChrome.brand.imageLoadFailed')
}

function removeImage() {
  url.value = ''
  attachmentId.value = ''
  errorMessage.value = ''
}
</script>

<template>
  <section class="grid gap-3 border-b border-slate-200 pb-5 last:border-b-0 last:pb-0 dark:border-zinc-800">
    <div>
      <h4 class="text-sm font-semibold text-highlighted">{{ label }}</h4>
      <p class="mt-0.5 text-xs text-muted">{{ t('admin.siteChrome.brand.dropHint') }}</p>
    </div>
    <div
      class="grid min-w-0 gap-3 rounded-md border border-dashed p-3 transition sm:grid-cols-[80px_minmax(0,1fr)]"
      :class="dragging ? 'border-[var(--sf-accent)] bg-teal-50 dark:bg-teal-950/30' : 'border-slate-300 bg-slate-50/70 dark:border-zinc-700 dark:bg-zinc-950/40'"
      @dragenter.prevent="dragging = true"
      @dragover.prevent="dragging = true"
      @dragleave.prevent="dragging = false"
      @drop.prevent="onDrop"
    >
      <button
        type="button"
        class="flex size-20 shrink-0 items-center justify-center overflow-hidden rounded-md border border-slate-200 bg-white text-slate-400 dark:border-zinc-700 dark:bg-zinc-900"
        :disabled="disabled || uploading"
        :aria-label="previewUrl ? t('admin.siteChrome.brand.replace') : t('admin.siteChrome.brand.upload')"
        @click="openPicker"
      >
        <UIcon v-if="uploading" name="i-lucide-loader-circle" class="size-5 animate-spin" />
        <img
          v-else-if="previewUrl"
          :src="previewUrl"
          :alt="label"
          class="max-h-full max-w-full"
          :class="previewShape === 'wide' ? 'object-contain' : 'object-cover'"
          @error="imageFailed"
        >
        <UIcon v-else :name="icon" class="size-6" aria-hidden="true" />
      </button>

      <div class="grid min-w-0 gap-3 md:grid-cols-2">
        <UFormField :label="urlLabel" class="min-w-0">
          <UInput
            :model-value="url"
            icon="i-lucide-link"
            :placeholder="t('admin.siteChrome.brand.urlPlaceholder')"
            :disabled="disabled || uploading"
            class="w-full"
            @update:model-value="updateURL"
          />
        </UFormField>
        <UFormField :label="attachmentIdLabel" class="min-w-0">
          <UInput v-model="attachmentId" :disabled="disabled || uploading" class="w-full font-mono" />
        </UFormField>
        <div class="flex flex-wrap items-center gap-2 md:col-span-2">
          <UButton
            type="button"
            color="primary"
            variant="soft"
            :icon="previewUrl ? 'i-lucide-replace' : 'i-lucide-upload'"
            :loading="uploading"
            :disabled="disabled"
            @click="openPicker"
          >
            {{ previewUrl ? t('admin.siteChrome.brand.replace') : t('admin.siteChrome.brand.upload') }}
          </UButton>
          <UButton
            v-if="previewUrl || attachmentId"
            type="button"
            color="error"
            variant="ghost"
            icon="i-lucide-trash-2"
            :disabled="disabled || uploading"
            @click="removeImage"
          >
            {{ t('admin.siteChrome.brand.remove') }}
          </UButton>
          <span v-if="uploading" class="text-xs text-muted">{{ t('admin.siteChrome.brand.uploading') }}</span>
        </div>
      </div>
    </div>
    <p v-if="errorMessage" class="text-sm text-error" role="alert">{{ errorMessage }}</p>
    <input
      ref="fileInput"
      class="hidden"
      type="file"
      accept="image/jpeg,image/png,image/gif,image/webp,image/svg+xml,.svg"
      :disabled="disabled || uploading"
      @change="uploadFile(($event.target as HTMLInputElement).files?.[0])"
    >
  </section>
</template>
