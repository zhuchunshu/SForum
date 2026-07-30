import type { Editor } from '@tiptap/core'
import { apiErrorMessage } from '~/composables/useApiClient'
import {
  addEditorImageUploadPlaceholder,
  findEditorImageUploadPlaceholder,
  removeEditorImageUploadPlaceholder
} from '~/utils/editor/editorImageUpload'

export type EditorImageAttachment = {
  id: number
  publicId: string
  name: string
  contentType: string
  size: number
  url: string
}

type AttachmentUploadPolicy = {
  allowed: boolean
  reason: string
  effectiveMaxFileSizeBytes: number
}

type EditorImageUploadLabels = {
  uploading: string
  invalidType: string
  notAllowed: string
  tooLarge: (fileName: string, maxSize: string) => string
  uploaded: (count: number) => string
  partiallyUploaded: (uploaded: number, failed: number) => string
  failed: string
  positionLost: string
}

let uploadBatchSequence = 0

function isImageFile(file: File) {
  if (file.type === 'image/svg+xml' || file.name.toLowerCase().endsWith('.svg')) {
    return false
  }
  return file.type.startsWith('image/')
    || /\.(?:avif|bmp|gif|heic|heif|jpe?g|png|tiff?|webp)$/i.test(file.name)
}

function formatBytes(bytes: number) {
  if (bytes < 1024 * 1024) {
    return `${Math.max(1, Math.round(bytes / 1024))} KB`
  }
  return `${(bytes / 1024 / 1024).toFixed(bytes < 10 * 1024 * 1024 ? 1 : 0)} MB`
}

export function useEditorImageUpload(labels: EditorImageUploadLabels) {
  const { request } = useApiClient()
  const toast = useToast()
  const pendingUploadCount = ref(0)
  let disposed = false

  onScopeDispose(() => {
    disposed = true
  })

  function errorToast(title: string, description?: string) {
    if (disposed) return
    toast.add({
      color: 'error',
      icon: 'i-lucide-triangle-alert',
      title,
      description,
      duration: 0
    })
  }

  async function uploadOne(file: File) {
    const body = new FormData()
    body.append('file', file)
    return request<EditorImageAttachment>('/attachments', { method: 'POST', body })
  }

  async function uploadImages(editor: Editor, inputFiles: File[], position: number) {
    if (disposed || editor.isDestroyed) return

    const imageFiles = inputFiles.filter(isImageFile)
    const rejectedTypeCount = inputFiles.length - imageFiles.length
    if (imageFiles.length === 0) {
      errorToast(labels.invalidType)
      return
    }

    const id = `editor-image-upload-${Date.now()}-${++uploadBatchSequence}`
    const safePosition = Math.max(0, Math.min(position, editor.state.doc.content.size))
    addEditorImageUploadPlaceholder(editor, {
      id,
      pos: safePosition,
      label: labels.uploading,
      fileCount: imageFiles.length
    })
    pendingUploadCount.value += imageFiles.length

    let insertedCount = 0
    let failureCount = rejectedTypeCount
    let failureDescription = rejectedTypeCount ? labels.invalidType : ''

    try {
      const policy = await request<AttachmentUploadPolicy>('/attachments/upload-policy')
      if (!policy.allowed) {
        failureCount += imageFiles.length
        failureDescription = labels.notAllowed
        return
      }

      const acceptedFiles: File[] = []
      for (const file of imageFiles) {
        if (policy.effectiveMaxFileSizeBytes > 0 && file.size > policy.effectiveMaxFileSizeBytes) {
          failureCount += 1
          failureDescription = labels.tooLarge(file.name, formatBytes(policy.effectiveMaxFileSizeBytes))
        } else {
          acceptedFiles.push(file)
        }
      }

      const results = await Promise.allSettled(acceptedFiles.map(uploadOne))
      const attachments = results.flatMap(result => result.status === 'fulfilled' ? [result.value] : [])
      failureCount += results.length - attachments.length
      const firstFailure = results.find(result => result.status === 'rejected')
      if (firstFailure?.status === 'rejected') {
        failureDescription = apiErrorMessage(firstFailure.reason) || labels.failed
      }

      if (!disposed && !editor.isDestroyed && attachments.length > 0) {
        const insertPosition = findEditorImageUploadPlaceholder(editor, id)
        if (insertPosition == null) {
          failureCount += attachments.length
          failureDescription = labels.positionLost
        } else {
          const inserted = editor.commands.insertContentAt(
            insertPosition,
            attachments.map(attachment => ({
              type: 'image',
              attrs: {
                src: attachment.url,
                alt: attachment.name,
                attachmentId: attachment.id,
                attachmentPublicId: attachment.publicId
              }
            })),
            { updateSelection: false }
          )
          if (inserted) {
            insertedCount = attachments.length
          } else {
            failureCount += attachments.length
            failureDescription = labels.positionLost
          }
        }
      }
    } catch (error) {
      failureCount += imageFiles.length
      failureDescription = apiErrorMessage(error) || labels.failed
    } finally {
      if (!editor.isDestroyed) {
        removeEditorImageUploadPlaceholder(editor, id)
      }
      pendingUploadCount.value = Math.max(0, pendingUploadCount.value - imageFiles.length)

      if (!disposed && insertedCount > 0 && failureCount === 0) {
        toast.add({
          color: 'success',
          icon: 'i-lucide-image-up',
          title: labels.uploaded(insertedCount),
          duration: 10000
        })
      } else if (!disposed && insertedCount > 0) {
        errorToast(labels.partiallyUploaded(insertedCount, failureCount), failureDescription)
      } else if (failureCount > 0) {
        errorToast(labels.failed, failureDescription)
      }
    }
  }

  return {
    pendingUploadCount: readonly(pendingUploadCount),
    uploadImages
  }
}

