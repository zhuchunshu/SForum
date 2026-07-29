export type RegistrationMode = 'open' | 'invite' | 'approval' | 'closed'

const registrationModes: RegistrationMode[] = ['open', 'invite', 'approval', 'closed']

// 旧安装可能尚未保存 mode；此时按旧 enabled 开关投影后台选中状态。
export function resolveRegistrationMode(mode: unknown, registrationEnabled: boolean): RegistrationMode {
  const normalized = typeof mode === 'string' ? mode.trim() : ''
  return registrationModes.includes(normalized as RegistrationMode)
    ? normalized as RegistrationMode
    : registrationEnabled ? 'open' : 'closed'
}
