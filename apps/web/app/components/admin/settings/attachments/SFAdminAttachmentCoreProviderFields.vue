<script setup lang="ts">
import type { AttachmentSettings } from '~/utils/attachments/attachmentSettings'

defineProps<{ settings: AttachmentSettings }>()

const { t } = useI18n()
</script>

<template>
  <div v-if="settings.provider === 'local'" class="grid gap-4 md:grid-cols-2">
    <UFormField :label="t('admin.attachments.localRoot')" :help="t('admin.attachments.localRootDescription')" name="attachment-local-root">
      <UInput v-model="settings.local.root" size="lg" icon="i-lucide-folder-tree" class="w-full font-mono" />
    </UFormField>
    <UFormField :label="t('admin.attachments.localPublicPrefix')" :help="t('admin.attachments.fieldHelp.localPublicPrefix')" name="attachment-local-prefix">
      <UInput v-model="settings.local.publicPrefix" size="lg" type="url" icon="i-lucide-folder" class="w-full" />
    </UFormField>
  </div>

  <div v-else-if="settings.provider === 'aliyun_oss'" class="grid gap-4 md:grid-cols-2">
    <UFormField :label="t('admin.attachments.aliyun.endpoint')" :help="t('admin.attachments.fieldHelp.aliyunEndpoint')" name="attachment-aliyun-endpoint">
      <UInput v-model="settings.aliyunOss.endpoint" size="lg" placeholder="https://oss-cn-hangzhou.aliyuncs.com" icon="i-lucide-cloud" class="w-full" />
    </UFormField>
    <UFormField :label="t('admin.attachments.aliyun.bucket')" :help="t('admin.attachments.fieldHelp.aliyunBucket')" name="attachment-aliyun-bucket">
      <UInput v-model="settings.aliyunOss.bucket" size="lg" icon="i-lucide-archive" class="w-full" />
    </UFormField>
    <UFormField :label="t('admin.attachments.aliyun.region')" :help="t('admin.attachments.fieldHelp.aliyunRegion')" name="attachment-aliyun-region">
      <UInput v-model="settings.aliyunOss.region" size="lg" placeholder="cn-hangzhou" icon="i-lucide-map-pin" class="w-full" />
    </UFormField>
    <UFormField :label="t('admin.attachments.aliyun.accessKeyId')" :help="t('admin.attachments.fieldHelp.aliyunAccessKeyId')" name="attachment-aliyun-access-key-id">
      <UInput v-model="settings.aliyunOss.accessKeyId" size="lg" icon="i-lucide-key-round" class="w-full" />
    </UFormField>
    <UFormField :label="t('admin.attachments.aliyun.accessKeySecret')" :help="t('admin.attachments.fieldHelp.secret')" name="attachment-aliyun-access-key-secret">
      <UInput v-model="settings.aliyunOss.accessKeySecret" size="lg" type="password" :placeholder="settings.aliyunOss.accessKeySecretSet ? t('admin.attachments.keepSecret') : ''" icon="i-lucide-lock-keyhole" class="w-full" />
    </UFormField>
  </div>

  <div v-else-if="settings.provider === 'tencent_cos'" class="grid gap-4 md:grid-cols-2">
    <UFormField :label="t('admin.attachments.tencent.region')" :help="t('admin.attachments.fieldHelp.tencentRegion')" name="attachment-tencent-region">
      <UInput v-model="settings.tencentCos.region" size="lg" placeholder="ap-guangzhou" icon="i-lucide-map-pin" class="w-full" />
    </UFormField>
    <UFormField :label="t('admin.attachments.tencent.bucket')" :help="t('admin.attachments.fieldHelp.tencentBucket')" name="attachment-tencent-bucket">
      <UInput v-model="settings.tencentCos.bucket" size="lg" placeholder="example-1250000000" icon="i-lucide-archive" class="w-full" />
    </UFormField>
    <UFormField :label="t('admin.attachments.tencent.secretId')" :help="t('admin.attachments.fieldHelp.tencentSecretId')" name="attachment-tencent-secret-id">
      <UInput v-model="settings.tencentCos.secretId" size="lg" icon="i-lucide-key-round" class="w-full" />
    </UFormField>
    <UFormField :label="t('admin.attachments.tencent.secretKey')" :help="t('admin.attachments.fieldHelp.secret')" name="attachment-tencent-secret-key">
      <UInput v-model="settings.tencentCos.secretKey" size="lg" type="password" :placeholder="settings.tencentCos.secretKeySet ? t('admin.attachments.keepSecret') : ''" icon="i-lucide-lock-keyhole" class="w-full" />
    </UFormField>
    <UFormField :label="t('admin.attachments.tencent.cdnDomain')" :help="t('admin.attachments.fieldHelp.tencentCdnDomain')" name="attachment-tencent-cdn-domain">
      <UInput v-model="settings.tencentCos.cdnDomain" size="lg" type="url" placeholder="https://cdn.example.com" icon="i-lucide-globe" class="w-full" />
    </UFormField>
  </div>

  <div v-else-if="settings.provider === 'ftp'" class="grid gap-4 md:grid-cols-2">
    <UFormField :label="t('admin.attachments.remote.host')" :help="t('admin.attachments.fieldHelp.remoteHost')" name="attachment-ftp-host">
      <UInput v-model="settings.ftp.host" size="lg" icon="i-lucide-server" class="w-full" />
    </UFormField>
    <UFormField :label="t('admin.attachments.remote.port')" :help="t('admin.attachments.fieldHelp.ftpPort')" name="attachment-ftp-port">
      <UInput v-model.number="settings.ftp.port" size="lg" type="number" min="1" max="65535" icon="i-lucide-hash" class="w-full" />
    </UFormField>
    <UFormField :label="t('admin.attachments.remote.username')" :help="t('admin.attachments.fieldHelp.remoteUsername')" name="attachment-ftp-username">
      <UInput v-model="settings.ftp.username" size="lg" icon="i-lucide-user" class="w-full" />
    </UFormField>
    <UFormField :label="t('admin.attachments.remote.password')" :help="t('admin.attachments.fieldHelp.secret')" name="attachment-ftp-password">
      <UInput v-model="settings.ftp.password" size="lg" type="password" :placeholder="settings.ftp.passwordSet ? t('admin.attachments.keepSecret') : ''" icon="i-lucide-lock-keyhole" class="w-full" />
    </UFormField>
    <UFormField :label="t('admin.attachments.remote.rootPath')" :help="t('admin.attachments.fieldHelp.remoteRootPath')" name="attachment-ftp-root-path">
      <UInput v-model="settings.ftp.rootPath" size="lg" icon="i-lucide-folder-tree" class="w-full font-mono" />
    </UFormField>
    <UFormField :label="t('admin.attachments.remote.publicBaseUrl')" :help="t('admin.attachments.fieldHelp.remotePublicBaseUrl')" name="attachment-ftp-public-base-url">
      <UInput v-model="settings.ftp.publicBaseUrl" size="lg" type="url" placeholder="https://files.example.com" icon="i-lucide-link" class="w-full" />
    </UFormField>
    <label class="flex items-start gap-3 rounded-lg border border-slate-200 p-4 dark:border-zinc-800">
      <input v-model="settings.ftp.passive" type="checkbox" class="mt-1 size-4 rounded border-slate-300 text-[var(--sf-accent)] focus:ring-[var(--sf-accent)]" />
      <span>
        <span class="block text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ t('admin.attachments.ftp.passive') }}</span>
        <span class="mt-1 block text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.attachments.fieldHelp.ftpPassive') }}</span>
      </span>
    </label>
    <label class="flex items-start gap-3 rounded-lg border border-slate-200 p-4 dark:border-zinc-800">
      <input v-model="settings.ftp.explicitTls" type="checkbox" class="mt-1 size-4 rounded border-slate-300 text-[var(--sf-accent)] focus:ring-[var(--sf-accent)]" />
      <span>
        <span class="block text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ t('admin.attachments.ftp.explicitTls') }}</span>
        <span class="mt-1 block text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.attachments.fieldHelp.ftpExplicitTls') }}</span>
      </span>
    </label>
  </div>

  <div v-else-if="settings.provider === 'sftp'" class="grid gap-4 md:grid-cols-2">
    <UFormField :label="t('admin.attachments.remote.host')" :help="t('admin.attachments.fieldHelp.remoteHost')" name="attachment-sftp-host">
      <UInput v-model="settings.sftp.host" size="lg" icon="i-lucide-server" class="w-full" />
    </UFormField>
    <UFormField :label="t('admin.attachments.remote.port')" :help="t('admin.attachments.fieldHelp.sftpPort')" name="attachment-sftp-port">
      <UInput v-model.number="settings.sftp.port" size="lg" type="number" min="1" max="65535" icon="i-lucide-hash" class="w-full" />
    </UFormField>
    <UFormField :label="t('admin.attachments.remote.username')" :help="t('admin.attachments.fieldHelp.remoteUsername')" name="attachment-sftp-username">
      <UInput v-model="settings.sftp.username" size="lg" icon="i-lucide-user" class="w-full" />
    </UFormField>
    <UFormField :label="t('admin.attachments.remote.password')" :help="t('admin.attachments.fieldHelp.sftpPassword')" name="attachment-sftp-password">
      <UInput v-model="settings.sftp.password" size="lg" type="password" :placeholder="settings.sftp.passwordSet ? t('admin.attachments.keepSecret') : ''" icon="i-lucide-lock-keyhole" class="w-full" />
    </UFormField>
    <UFormField :label="t('admin.attachments.sftp.privateKey')" :help="t('admin.attachments.fieldHelp.sftpPrivateKey')" name="attachment-sftp-private-key" class="md:col-span-2">
      <UTextarea v-model="settings.sftp.privateKey" size="lg" :rows="4" :placeholder="settings.sftp.privateKeySet ? t('admin.attachments.keepSecret') : ''" class="w-full font-mono text-xs" />
    </UFormField>
    <UFormField :label="t('admin.attachments.sftp.passphrase')" :help="t('admin.attachments.fieldHelp.sftpPassphrase')" name="attachment-sftp-passphrase">
      <UInput v-model="settings.sftp.passphrase" size="lg" type="password" :placeholder="settings.sftp.passphraseSet ? t('admin.attachments.keepSecret') : ''" icon="i-lucide-lock" class="w-full" />
    </UFormField>
    <UFormField :label="t('admin.attachments.remote.rootPath')" :help="t('admin.attachments.fieldHelp.remoteRootPath')" name="attachment-sftp-root-path">
      <UInput v-model="settings.sftp.rootPath" size="lg" icon="i-lucide-folder-tree" class="w-full font-mono" />
    </UFormField>
    <UFormField :label="t('admin.attachments.sftp.hostKeyFingerprint')" :help="t('admin.attachments.fieldHelp.sftpHostKeyFingerprint')" name="attachment-sftp-host-key-fingerprint">
      <UInput v-model="settings.sftp.hostKeyFingerprint" size="lg" placeholder="SHA256:..." icon="i-lucide-fingerprint" class="w-full font-mono" />
    </UFormField>
    <UFormField :label="t('admin.attachments.remote.publicBaseUrl')" :help="t('admin.attachments.fieldHelp.remotePublicBaseUrl')" name="attachment-sftp-public-base-url">
      <UInput v-model="settings.sftp.publicBaseUrl" size="lg" type="url" placeholder="https://files.example.com" icon="i-lucide-link" class="w-full" />
    </UFormField>
  </div>
</template>
