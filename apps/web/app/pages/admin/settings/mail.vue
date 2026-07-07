<script setup lang="ts">
import { enabledOptionValue } from '~/composables/useWebOptions'
import { useAdminPage } from '~/composables/useAdminPage'
import { apiErrorMessage, apiErrorReason } from '~/composables/useApiClient'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminMailSettings'
})

const { t } = useI18n()
const { fetchAdminEnvelope, saveMany } = useWebOptions()
const { request } = useApiClient()

const adminPage = useAdminPage('/settings/mail')

// 邮件选项名称常量。
const NAME_PROVIDER = 'mail.provider'
const NAME_FROM_ADDRESS = 'mail.from_address'
const NAME_FROM_NAME = 'mail.from_name'
const NAME_SMTP_HOST = 'mail.smtp.host'
const NAME_SMTP_PORT = 'mail.smtp.port'
const NAME_SMTP_USERNAME = 'mail.smtp.username'
const NAME_SMTP_PASSWORD = 'mail.smtp.password'
const NAME_SMTP_ENCRYPTION = 'mail.smtp.encryption'

const providerOptions = [
  { label: 'dev_log（开发日志，默认）', value: 'dev_log' },
  { label: 'smtp（SMTP 投递）', value: 'smtp' },
  { label: 'noop（关闭投递）', value: 'noop' }
]
const encryptionOptions = [
  { label: 'STARTTLS（推荐）', value: 'starttls' },
  { label: 'TLS（隐式）', value: 'tls' },
  { label: 'none（明文，不推荐）', value: 'none' }
]

const provider = ref('dev_log')
const fromAddress = ref('noreply@example.com')
const fromName = ref('SForum')
const smtpHost = ref('')
const smtpPort = ref('587')
const smtpUsername = ref('')
const smtpPassword = ref('')
const smtpEncryption = ref('starttls')
const passwordSecretSet = ref(false)

const pending = ref(true)
const saving = ref(false)
const testing = ref(false)
const errorMessage = ref('')
const successMessage = ref('')
const testResult = ref('')

// 从 admin web-options 加载当前值。
async function load() {
  pending.value = true
  errorMessage.value = ''
  try {
    const envelope = await fetchAdminEnvelope()
    const byName = new Map(envelope.data.map((item) => [item.name, item]))
    provider.value = byName.get(NAME_PROVIDER)?.value ?? 'dev_log'
    fromAddress.value = byName.get(NAME_FROM_ADDRESS)?.value ?? 'noreply@example.com'
    fromName.value = byName.get(NAME_FROM_NAME)?.value ?? 'SForum'
    smtpHost.value = byName.get(NAME_SMTP_HOST)?.value ?? ''
    smtpPort.value = byName.get(NAME_SMTP_PORT)?.value ?? '587'
    smtpUsername.value = byName.get(NAME_SMTP_USERNAME)?.value ?? ''
    smtpEncryption.value = byName.get(NAME_SMTP_ENCRYPTION)?.value ?? 'starttls'
    passwordSecretSet.value = byName.get(NAME_SMTP_PASSWORD)?.secretSet ?? false
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('admin.mailSettings.loadFailed')
  } finally {
    pending.value = false
  }
}

await load()

async function save() {
  if (saving.value) {
    return
  }
  saving.value = true
  errorMessage.value = ''
  successMessage.value = ''
  try {
    const items = [
      { name: NAME_PROVIDER, value: provider.value },
      { name: NAME_FROM_ADDRESS, value: fromAddress.value },
      { name: NAME_FROM_NAME, value: fromName.value },
      { name: NAME_SMTP_HOST, value: smtpHost.value },
      { name: NAME_SMTP_PORT, value: smtpPort.value },
      { name: NAME_SMTP_USERNAME, value: smtpUsername.value },
      { name: NAME_SMTP_ENCRYPTION, value: smtpEncryption.value }
    ]
    // 密码为空时不提交（保留已存的密钥）。
    if (smtpPassword.value.trim() !== '') {
      items.push({ name: NAME_SMTP_PASSWORD, value: smtpPassword.value })
    }
    await saveMany(items)
    smtpPassword.value = ''
    successMessage.value = t('admin.mailSettings.saved')
    setTimeout(() => {
      if (successMessage.value) {
        successMessage.value = ''
      }
    }, 10000)
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('admin.mailSettings.saveFailed')
  } finally {
    saving.value = false
  }
}

async function sendTestMail() {
  if (testing.value) {
    return
  }
  testing.value = true
  testResult.value = ''
  errorMessage.value = ''
  try {
    await request('/admin/mail/test', {
      method: 'POST',
      body: { recipient: testRecipient.value }
    })
    testResult.value = t('admin.mailSettings.testSent')
    setTimeout(() => {
      if (testResult.value) {
        testResult.value = ''
      }
    }, 10000)
  } catch (error) {
    testResult.value = apiErrorMessage(error) || t('admin.mailSettings.testFailed')
  } finally {
    testing.value = false
  }
}

const testRecipient = ref(fromAddress.value)
</script>

<template>
  <div class="space-y-4">
    <header>
      <h1 class="text-xl font-bold text-slate-900 dark:text-zinc-50">
        <UIcon :name="adminPage.icon" class="mr-2 inline-block size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
        {{ t('admin.mailSettings.title') }}
      </h1>
      <p class="text-sm text-slate-500 mt-1 dark:text-zinc-400">
        {{ t('admin.mailSettings.description') }}
      </p>
    </header>

    <UDashboardToolbar class="border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 rounded-lg px-4 py-2.5 text-slate-500 dark:text-zinc-400">
      <template #left>
        <div class="flex min-w-0 items-center gap-2 text-sm">
          <UIcon name="i-lucide-mail-check" class="size-4" />
          <span class="truncate">{{ t('admin.mailSettings.description') }}</span>
        </div>
      </template>
      <template #right>
        <UButton icon="i-lucide-rotate-cw" color="neutral" variant="subtle" :loading="pending" @click="load">
          {{ t('admin.home.refresh') }}
        </UButton>
      </template>
    </UDashboardToolbar>

    <SFAlert v-if="successMessage" variant="success" :title="successMessage" />
    <SFAlert v-if="errorMessage" variant="danger" :title="errorMessage" closable @close="errorMessage = ''" />

    <SFCard class="p-6 space-y-5">
      <div>
        <label class="block text-sm font-semibold text-slate-700 mb-2 dark:text-zinc-300">
          {{ t('admin.mailSettings.provider') }}
        </label>
        <select v-model="provider" class="sf-input w-full">
          <option v-for="opt in providerOptions" :key="opt.value" :value="opt.value">
            {{ opt.label }}
          </option>
        </select>
        <p class="text-xs text-slate-400 mt-1 dark:text-zinc-500">
          {{ t('admin.mailSettings.providerHint') }}
        </p>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div>
          <label class="block text-sm font-semibold text-slate-700 mb-2 dark:text-zinc-300">
            {{ t('admin.mailSettings.fromAddress') }}
          </label>
          <input v-model="fromAddress" type="email" class="sf-input w-full">
        </div>
        <div>
          <label class="block text-sm font-semibold text-slate-700 mb-2 dark:text-zinc-300">
            {{ t('admin.mailSettings.fromName') }}
          </label>
          <input v-model="fromName" type="text" class="sf-input w-full">
        </div>
      </div>
    </SFCard>

    <SFCard class="p-6 space-y-5">
      <h2 class="text-sm font-bold text-slate-700 uppercase tracking-wide dark:text-zinc-300">
        {{ t('admin.mailSettings.smtpTitle') }}
      </h2>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div>
          <label class="block text-sm font-semibold text-slate-700 mb-2 dark:text-zinc-300">
            {{ t('admin.mailSettings.smtpHost') }}
          </label>
          <input v-model="smtpHost" type="text" class="sf-input w-full" placeholder="smtp.example.com">
        </div>
        <div>
          <label class="block text-sm font-semibold text-slate-700 mb-2 dark:text-zinc-300">
            {{ t('admin.mailSettings.smtpPort') }}
          </label>
          <input v-model="smtpPort" type="text" class="sf-input w-full" placeholder="587">
        </div>
        <div>
          <label class="block text-sm font-semibold text-slate-700 mb-2 dark:text-zinc-300">
            {{ t('admin.mailSettings.smtpUsername') }}
          </label>
          <input v-model="smtpUsername" type="text" class="sf-input w-full">
        </div>
        <div>
          <label class="block text-sm font-semibold text-slate-700 mb-2 dark:text-zinc-300">
            {{ t('admin.mailSettings.smtpPassword') }}
          </label>
          <input v-model="smtpPassword" type="password" class="sf-input w-full" :placeholder="passwordSecretSet ? t('admin.mailSettings.passwordSetHint') : ''">
          <p v-if="passwordSecretSet" class="text-xs text-slate-400 mt-1 dark:text-zinc-500">
            {{ t('admin.mailSettings.passwordPreservedHint') }}
          </p>
        </div>
        <div class="sm:col-span-2">
          <label class="block text-sm font-semibold text-slate-700 mb-2 dark:text-zinc-300">
            {{ t('admin.mailSettings.smtpEncryption') }}
          </label>
          <select v-model="smtpEncryption" class="sf-input w-full">
            <option v-for="opt in encryptionOptions" :key="opt.value" :value="opt.value">
              {{ opt.label }}
            </option>
          </select>
        </div>
      </div>
      <div class="flex justify-end gap-2 pt-2">
        <SFButton variant="primary" :disabled="saving || pending" @click="save">
          {{ saving ? t('admin.mailSettings.saving') : t('admin.mailSettings.save') }}
        </SFButton>
      </div>
    </SFCard>

    <SFCard class="p-6 space-y-4">
      <h2 class="text-sm font-bold text-slate-700 uppercase tracking-wide dark:text-zinc-300">
        {{ t('admin.mailSettings.testTitle') }}
      </h2>
      <p class="text-sm text-slate-500 dark:text-zinc-400">
        {{ t('admin.mailSettings.testDesc') }}
      </p>
      <div class="flex gap-2">
        <input v-model="testRecipient" type="email" class="sf-input flex-1" :placeholder="t('admin.mailSettings.testRecipientPlaceholder')">
        <SFButton variant="secondary" :disabled="testing || !testRecipient" @click="sendTestMail">
          {{ testing ? t('admin.mailSettings.testing') : t('admin.mailSettings.sendTest') }}
        </SFButton>
      </div>
      <SFAlert v-if="testResult" variant="info" :title="testResult" />
    </SFCard>
  </div>
</template>

<style scoped>
.sf-input {
  border: 1px solid #d1d5db;
  border-radius: 0.5rem;
  padding: 0.5rem 0.75rem;
  font-size: 0.95rem;
  background: #ffffff;
  color: #111827;
  outline: none;
  transition: border-color 0.15s;
}
.sf-input:focus {
  border-color: #0f766e;
  box-shadow: 0 0 0 3px rgba(15, 118, 110, 0.12);
}
:global(.dark) .sf-input {
  background: #18181b;
  border-color: #3f3f46;
  color: #f4f4f5;
}
</style>
