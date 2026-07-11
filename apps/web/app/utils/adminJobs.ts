export type AdminJobState = 'available' | 'cancelled' | 'completed' | 'discarded' | 'pending' | 'retryable' | 'running' | 'scheduled'
export type AdminJob = { id: number, kind: string, queue: string, state: AdminJobState, attempt: number, maxAttempts: number, priority: number, args: unknown, metadata: unknown, tags: string[], errors: unknown[], attemptedBy: string[], createdAt: string, scheduledAt: string, attemptedAt?: string, finalizedAt?: string }
export type AdminJobQueue = { name: string, pausedAt?: string, updatedAt: string, available: number, running: number, failed: number }
export type AdminJobsOverview = { counts: Record<string, number>, queues: AdminJobQueue[] }
export const ALL_ADMIN_JOBS_FILTER = '__all__'
export const adminJobFilterValue = (value: string) => value === ALL_ADMIN_JOBS_FILTER ? '' : value
export const jobStateColor = (state: AdminJobState) => state === 'completed' ? 'success' : state === 'running' ? 'info' : state === 'discarded' ? 'error' : state === 'retryable' ? 'warning' : state === 'cancelled' ? 'neutral' : 'primary'
export const jobCanRetry = (state: AdminJobState) => ['discarded', 'cancelled', 'completed'].includes(state)
export const jobCanCancel = (state: AdminJobState) => !['cancelled', 'completed', 'discarded'].includes(state)
