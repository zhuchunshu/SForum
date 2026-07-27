import { apiErrorMessage, apiErrorReason } from '../composables/useApiClient'

type RegisterErrorTranslator = (key: string) => string

export function registerErrorMessage(error: unknown, translate: RegisterErrorTranslator) {
  const message = apiErrorMessage(error)
  if (message) {
    return message
  }

  switch (apiErrorReason(error)) {
    case 'auth.session_unavailable':
      return translate('errors.sessionUnavailable')
    case 'human_verification.required':
      return translate('errors.humanVerificationRequired')
    case 'human_verification.invalid':
      return translate('errors.humanVerificationInvalid')
    case 'human_verification.expired':
      return translate('errors.humanVerificationExpired')
    case 'human_verification.replayed':
      return translate('errors.humanVerificationReplayed')
    case 'rate_limit.exceeded':
      return translate('errors.rateLimited')
    case 'auth.external_registration_ticket_invalid':
      return translate('auth.external.reasons.ticketInvalid')
    case 'auth.external_registration_ticket_expired':
      return translate('auth.external.reasons.ticketExpired')
    case 'auth.external_bootstrap_required':
      return translate('auth.external.reasons.bootstrapRequired')
    case 'auth.registration_disabled':
      return translate('auth.external.reasons.registrationDisabled')
    default:
      return translate('errors.registerFailed')
  }
}
