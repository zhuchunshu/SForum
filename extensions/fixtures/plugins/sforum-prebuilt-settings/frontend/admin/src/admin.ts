import { createApp } from 'vue'
import type { AdminPageBridgeV1 } from '@sforum/admin-sdk'
import { ADMIN_MICRO_FRONTEND_API_VERSION } from '@sforum/admin-sdk'
import '@sforum/plugin-ui/style.css'
import AdminDashboard from './AdminDashboard.vue'

export const apiVersion = ADMIN_MICRO_FRONTEND_API_VERSION

export function mount(target: HTMLElement, bridge: AdminPageBridgeV1) {
  const app = createApp(AdminDashboard, { bridge })
  app.mount(target)
  return () => app.unmount()
}
