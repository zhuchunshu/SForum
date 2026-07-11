<script setup lang="ts">
type SEOAsset = {
  publicId: string
  url: string
  contentType: string
  size: number
  width?: number
  height?: number
}

const props = defineProps<{ context: string, label: string, recommended?: string }>()
const model = defineModel<string>({ default: '' })
const { t } = useI18n()
const toast = useToast()
const { request } = useApiClient()
const fileInput = ref<HTMLInputElement>()
const uploading = ref(false)
const dragging = ref(false)
const errorMessage = ref('')
const pendingUrl = ref(model.value)

watch(model, value => { pendingUrl.value = value })

async function uploadFile(file?: File) {
  if (!file) return
  uploading.value = true
  errorMessage.value = ''
  try {
    const body = new FormData()
    body.append('context', props.context)
    body.append('file', file)
    const asset = await request<SEOAsset>('/admin/seo/assets', { method: 'POST', body })
    model.value = asset.url
    pendingUrl.value = asset.url
    toast.add({ color: 'success', icon: 'i-lucide-image-up', title: t('admin.seo.imageUploaded'), duration: 10000 })
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('admin.seo.imageUploadFailed')
  } finally {
    uploading.value = false
  }
}

function onDrop(event: DragEvent) {
  dragging.value = false
  void uploadFile(event.dataTransfer?.files?.[0])
}

function confirmLoaded() {
  model.value = pendingUrl.value.trim()
  errorMessage.value = ''
}

function imageFailed() {
  if (pendingUrl.value.trim()) errorMessage.value = t('admin.seo.imageLoadFailed')
}

function removeImage() {
  model.value = ''
  pendingUrl.value = ''
  errorMessage.value = ''
}
</script>

<template>
  <UFormField :label="label" :error="errorMessage || undefined">
    <div class="grid gap-3">
      <div
        class="flex min-h-32 items-center justify-center border border-dashed p-4 transition"
        :class="dragging ? 'border-[var(--sf-accent)] bg-teal-50 dark:bg-teal-950/30' : 'border-slate-300 bg-slate-50 dark:border-zinc-700 dark:bg-zinc-950/60'"
        @dragover.prevent="dragging = true"
        @dragleave.prevent="dragging = false"
        @drop.prevent="onDrop"
      >
        <div v-if="uploading" class="flex items-center gap-2 text-sm">
          <UIcon name="i-lucide-loader-circle" class="size-5 animate-spin" />
          {{ t('admin.seo.imageUploading') }}
        </div>
        <div v-else-if="pendingUrl" class="grid w-full gap-3 sm:grid-cols-[160px_1fr]">
          <img :src="pendingUrl" :alt="label" class="aspect-[1200/630] w-full border border-slate-200 object-cover dark:border-zinc-700" @load="confirmLoaded" @error="imageFailed">
          <div class="flex flex-wrap content-center gap-2">
            <UButton type="button" color="neutral" variant="outline" icon="i-lucide-replace" @click="fileInput?.click()">{{ t('admin.seo.imageReplace') }}</UButton>
            <UButton type="button" color="error" variant="soft" icon="i-lucide-trash-2" @click="removeImage">{{ t('admin.seo.imageRemove') }}</UButton>
          </div>
        </div>
        <button v-else type="button" class="flex flex-col items-center gap-2 text-sm" @click="fileInput?.click()">
          <UIcon name="i-lucide-cloud-upload" class="size-7 text-[var(--sf-accent)]" />
          <span>{{ t('admin.seo.imageDrop') }}</span>
          <small v-if="recommended" class="text-slate-500">{{ recommended }}</small>
        </button>
        <input ref="fileInput" class="hidden" type="file" accept="image/jpeg,image/png,image/gif,image/webp" @change="uploadFile(($event.target as HTMLInputElement).files?.[0])">
      </div>
      <UInput v-model="pendingUrl" type="url" icon="i-lucide-link" :placeholder="t('admin.seo.imageUrlPlaceholder')" class="w-full" />
    </div>
  </UFormField>
</template>
