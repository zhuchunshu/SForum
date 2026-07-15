import type {
  PublicFrontendBridgeV1,
  PublicFrontendCleanup,
  PublicFrontendModuleV1
} from './types'
import { PublicFrontendContractError } from './types'

export const PUBLIC_FRONTEND_MOUNT_TIMEOUT_MS = 10_000

export async function mountPublicFrontendModule(
  module: PublicFrontendModuleV1,
  target: HTMLElement,
  bridge: PublicFrontendBridgeV1,
  timeoutMS = PUBLIC_FRONTEND_MOUNT_TIMEOUT_MS
): Promise<PublicFrontendCleanup> {
  if (!Number.isFinite(timeoutMS) || timeoutMS <= 0) {
    throw new PublicFrontendContractError('public component mount timeout is invalid')
  }
  const moduleUnmount = module.unmount
    ? () => module.unmount?.(target, bridge)
    : undefined
  let result: void | PublicFrontendCleanup
  try {
    result = await withMountTimeout(
      Promise.resolve(module.mount(target, bridge)),
      timeoutMS,
      'public component mount timed out'
    )
    if (result !== undefined && typeof result !== 'function') {
      throw new PublicFrontendContractError('public component cleanup must be a function')
    }
  } catch (error) {
    try {
      if (moduleUnmount) {
        await withMountTimeout(
          Promise.resolve(moduleUnmount()),
          timeoutMS,
          'public component unmount timed out'
        )
      }
    } catch {
      // Preserve the original mount/contract failure while restoring Host DOM.
    }
    target.replaceChildren()
    throw error
  }

  const cleanup = typeof result === 'function' ? result : moduleUnmount
  let released = false
  return async () => {
    if (released) return
    released = true
    try {
      if (cleanup) {
        await withMountTimeout(
          Promise.resolve(cleanup()),
          timeoutMS,
          'public component cleanup timed out'
        )
      }
    } finally {
      target.replaceChildren()
    }
  }
}

function withMountTimeout<T>(promise: Promise<T>, timeoutMS: number, message: string): Promise<T> {
  return new Promise((resolve, reject) => {
    const timeout = globalThis.setTimeout(() => reject(new PublicFrontendContractError(message)), timeoutMS)
    promise.then(
      value => {
        globalThis.clearTimeout(timeout)
        resolve(value)
      },
      error => {
        globalThis.clearTimeout(timeout)
        reject(error)
      }
    )
  })
}
