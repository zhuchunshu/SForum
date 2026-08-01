export type AccountSecurityOption = {
  name: string
  value: string
}

export type AccountSecurityForm = {
  passwordMinLength: number
  passwordMaxLength: number
  passwordRequireLowercase: boolean
  passwordRequireUppercase: boolean
  passwordRequireNumber: boolean
  passwordRequireSymbol: boolean
  sessionsMaxDevices: number
  sessionsKeepDays: number
  loginMaxFailures: number
  loginLockoutMinutes: number
  mailResendCooldownSeconds: number
  mailResendWindowMinutes: number
  mailResendMaxPerTarget: number
  mailResendMaxPerIP: number
}

export const recommendedAccountSecurityForm: AccountSecurityForm = {
  passwordMinLength: 12,
  passwordMaxLength: 128,
  passwordRequireLowercase: false,
  passwordRequireUppercase: false,
  passwordRequireNumber: false,
  passwordRequireSymbol: false,
  sessionsMaxDevices: 5,
  sessionsKeepDays: 30,
  loginMaxFailures: 10,
  loginLockoutMinutes: 15,
  mailResendCooldownSeconds: 30,
  mailResendWindowMinutes: 60,
  mailResendMaxPerTarget: 3,
  mailResendMaxPerIP: 10
}

export function normalizeAccountSecurityForm(items: AccountSecurityOption[]): AccountSecurityForm {
  const options = Object.fromEntries(items.map(item => [item.name, item.value]))
  return {
    passwordMinLength: boundedInteger(options['identity.password.min_length'], 12, 8, 128),
    passwordMaxLength: boundedInteger(options['identity.password.max_length'], 128, 64, 512),
    passwordRequireLowercase: enabledValue(options['identity.password.require_lowercase'], false),
    passwordRequireUppercase: enabledValue(options['identity.password.require_uppercase'], false),
    passwordRequireNumber: enabledValue(options['identity.password.require_number'], false),
    passwordRequireSymbol: enabledValue(options['identity.password.require_symbol'], false),
    sessionsMaxDevices: boundedInteger(options['identity.sessions.max_devices'], 5, 1, 20),
    sessionsKeepDays: boundedInteger(options['identity.sessions.keep_days'], 30, 1, 365),
    loginMaxFailures: boundedInteger(options['identity.login.max_failures'], 10, 0, 50),
    loginLockoutMinutes: boundedInteger(options['identity.login.lockout_minutes'], 15, 0, 1440),
    mailResendCooldownSeconds: boundedInteger(options['identity.mail_resend.cooldown_seconds'], 30, 0, 3600),
    mailResendWindowMinutes: boundedInteger(options['identity.mail_resend.window_minutes'], 60, 1, 1440),
    mailResendMaxPerTarget: boundedInteger(options['identity.mail_resend.max_per_target'], 3, 1, 100),
    mailResendMaxPerIP: boundedInteger(options['identity.mail_resend.max_per_ip'], 10, 1, 1000)
  }
}

export function normalizeAccountSecurityForSave(form: AccountSecurityForm): AccountSecurityForm {
  const normalized = {
    passwordMinLength: boundedInteger(form.passwordMinLength, 12, 8, 128),
    passwordMaxLength: boundedInteger(form.passwordMaxLength, 128, 64, 512),
    passwordRequireLowercase: form.passwordRequireLowercase,
    passwordRequireUppercase: form.passwordRequireUppercase,
    passwordRequireNumber: form.passwordRequireNumber,
    passwordRequireSymbol: form.passwordRequireSymbol,
    sessionsMaxDevices: boundedInteger(form.sessionsMaxDevices, 5, 1, 20),
    sessionsKeepDays: boundedInteger(form.sessionsKeepDays, 30, 1, 365),
    loginMaxFailures: boundedInteger(form.loginMaxFailures, 10, 0, 50),
    loginLockoutMinutes: boundedInteger(form.loginLockoutMinutes, 15, 0, 1440),
    mailResendCooldownSeconds: boundedInteger(form.mailResendCooldownSeconds, 30, 0, 3600),
    mailResendWindowMinutes: boundedInteger(form.mailResendWindowMinutes, 60, 1, 1440),
    mailResendMaxPerTarget: boundedInteger(form.mailResendMaxPerTarget, 3, 1, 100),
    mailResendMaxPerIP: boundedInteger(form.mailResendMaxPerIP, 10, 1, 1000)
  }
  if (normalized.passwordMaxLength < normalized.passwordMinLength) {
    normalized.passwordMaxLength = normalized.passwordMinLength
  }
  return normalized
}

export function accountSecurityPayload(form: AccountSecurityForm): AccountSecurityOption[] {
  return [
    { name: 'identity.password.min_length', value: String(form.passwordMinLength) },
    { name: 'identity.password.max_length', value: String(form.passwordMaxLength) },
    { name: 'identity.password.require_lowercase', value: enabledOption(form.passwordRequireLowercase) },
    { name: 'identity.password.require_uppercase', value: enabledOption(form.passwordRequireUppercase) },
    { name: 'identity.password.require_number', value: enabledOption(form.passwordRequireNumber) },
    { name: 'identity.password.require_symbol', value: enabledOption(form.passwordRequireSymbol) },
    { name: 'identity.sessions.max_devices', value: String(form.sessionsMaxDevices) },
    { name: 'identity.sessions.keep_days', value: String(form.sessionsKeepDays) },
    { name: 'identity.login.max_failures', value: String(form.loginMaxFailures) },
    { name: 'identity.login.lockout_minutes', value: String(form.loginLockoutMinutes) },
    { name: 'identity.mail_resend.cooldown_seconds', value: String(form.mailResendCooldownSeconds) },
    { name: 'identity.mail_resend.window_minutes', value: String(form.mailResendWindowMinutes) },
    { name: 'identity.mail_resend.max_per_target', value: String(form.mailResendMaxPerTarget) },
    { name: 'identity.mail_resend.max_per_ip', value: String(form.mailResendMaxPerIP) }
  ]
}

function enabledValue(value: string | undefined, fallback: boolean) {
  const normalized = value?.trim().toLowerCase()
  if (['enabled', 'true', '1', 'yes', 'on'].includes(normalized || '')) return true
  if (['disabled', 'false', '0', 'no', 'off'].includes(normalized || '')) return false
  return fallback
}

function enabledOption(value: boolean) {
  return value ? 'enabled' : 'disabled'
}

function boundedInteger(value: unknown, fallback: number, min: number, max: number) {
  const parsed = Number(value)
  if (!Number.isFinite(parsed)) return fallback
  const normalized = Math.trunc(parsed)
  return normalized >= min && normalized <= max ? normalized : fallback
}
