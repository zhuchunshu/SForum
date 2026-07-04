<script setup lang="ts">
definePageMeta({ middleware: 'admin' })

type Role = {
  id: number
  key: string
  alias: string
  description: string
  isSystem: boolean
  isDefault: boolean
  isDeletable: boolean
  isEnabled: boolean
}

const { t } = useI18n()
const apiBaseUrl = useRuntimeConfig().public.apiBaseUrl as string
const requestHeaders = import.meta.server ? useRequestHeaders(['cookie']) : undefined
const { data: roles, pending, error, refresh } = await useFetch<Role[]>(`${apiBaseUrl}/roles`, {
  credentials: 'include',
  headers: requestHeaders,
  default: () => []
})

useSeoMeta({
  title: t('admin.roles.metaTitle')
})
</script>

<template>
  <main class="page-shell">
    <section class="forum-board" aria-labelledby="admin-roles-title">
      <div class="roles-header">
        <div>
          <p class="eyebrow">
            {{ t('admin.roles.eyebrow') }}
          </p>
          <h1 id="admin-roles-title">
            {{ t('admin.roles.title') }}
          </h1>
          <p class="intro">
            {{ t('admin.roles.intro') }}
          </p>
        </div>

        <SFButton variant="ghost" :loading="pending" @click="refresh()">
          {{ t('admin.roles.refresh') }}
        </SFButton>
      </div>

      <SFAlert
        v-if="error"
        class="roles-alert"
        :title="t('admin.roles.loadFailed')"
        variant="danger"
        compact
      />

      <div v-else-if="roles.length" class="roles-table-wrap">
        <table class="roles-table">
          <caption>{{ t('admin.roles.caption') }}</caption>
          <thead>
            <tr>
              <th scope="col">
                {{ t('admin.roles.key') }}
              </th>
              <th scope="col">
                {{ t('admin.roles.alias') }}
              </th>
              <th scope="col">
                {{ t('admin.roles.description') }}
              </th>
              <th scope="col">
                {{ t('admin.roles.status') }}
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="role in roles" :key="role.id">
              <td>
                <code>{{ role.key }}</code>
              </td>
              <td>{{ role.alias }}</td>
              <td>{{ role.description || t('admin.roles.noDescription') }}</td>
              <td>
                <div class="role-badges">
                  <SFBadge v-if="role.isSystem" variant="info">
                    {{ t('admin.roles.system') }}
                  </SFBadge>
                  <SFBadge v-if="role.isDefault" variant="primary">
                    {{ t('admin.roles.default') }}
                  </SFBadge>
                  <SFBadge v-if="role.isDeletable" variant="neutral">
                    {{ t('admin.roles.custom') }}
                  </SFBadge>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <SFAlert
        v-else
        class="roles-alert"
        :title="t('admin.roles.empty')"
        variant="info"
        compact
      />
    </section>
  </main>
</template>

<style scoped>
.roles-header {
  display: flex;
  gap: 20px;
  align-items: flex-start;
  justify-content: space-between;
}

.roles-alert {
  margin-top: 24px;
}

.roles-table-wrap {
  overflow-x: auto;
  margin-top: 28px;
  border: 1px solid var(--sf-border);
  border-radius: 8px;
}

.roles-table {
  width: 100%;
  min-width: 720px;
  border-collapse: collapse;
  background: #ffffff;
  font-size: 14px;
}

.roles-table caption {
  width: 1px;
  height: 1px;
  overflow: hidden;
  position: absolute;
  white-space: nowrap;
}

.roles-table th,
.roles-table td {
  padding: 14px 16px;
  border-bottom: 1px solid var(--sf-border-light);
  text-align: left;
  vertical-align: top;
}

.roles-table th {
  color: var(--sf-fg-tertiary);
  font-size: 12px;
  font-weight: 800;
  text-transform: uppercase;
}

.roles-table tr:last-child td {
  border-bottom: 0;
}

.roles-table code {
  color: var(--sf-accent);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 13px;
  font-weight: 700;
}

.role-badges {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

@media (max-width: 760px) {
  .roles-header {
    display: block;
  }

  .roles-header .sf-button {
    margin-top: 20px;
  }
}
</style>
