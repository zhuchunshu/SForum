import {
  adminSurfaceIdempotencyKey,
  type AdminSurfaceCatalog,
  type AdminSurfaceContract,
  type AdminSurfaceInvocation,
  type AdminSurfaceKind
} from '~/utils/admin/adminSurfaces'

export function useAdminSurfaces() {
  const { request } = useApiClient()

  function list(placementId?: string, kind?: AdminSurfaceKind) {
    const query = new URLSearchParams()
    if (placementId) query.set('placementId', placementId)
    if (kind) query.set('kind', kind)
    const suffix = query.size > 0 ? `?${query.toString()}` : ''
    return request<AdminSurfaceCatalog>(`/admin/admin-surfaces${suffix}`)
  }

  function invoke(surface: AdminSurfaceContract, input: Record<string, unknown>) {
    const headers: Record<string, string> = {}
    if (surface.operation === 'command') {
      headers['Idempotency-Key'] = adminSurfaceIdempotencyKey(surface.id)
    }
    return request<AdminSurfaceInvocation>(`/admin/admin-surfaces/${encodeURIComponent(surface.id)}/invoke`, {
      method: 'POST',
      headers,
      body: {
        contractVersion: surface.contractVersion,
        input
      }
    })
  }

  return { list, invoke }
}
