#!/bin/sh
set -eu

# Keep old Compose deployments compatible: they already provide the canonical
# i18n URL through APP_URL or NUXT_PUBLIC_I18N_BASE_URL.
canonical_url="${NUXT_PUBLIC_I18N_BASE_URL:-${APP_URL:-http://127.0.0.1:3000}}"
export NUXT_PUBLIC_I18N_BASE_URL="${NUXT_PUBLIC_I18N_BASE_URL:-$canonical_url}"
export NUXT_PUBLIC_SITE_URL="${NUXT_PUBLIC_SITE_URL:-$canonical_url}"
export NUXT_SITE_URL="${NUXT_SITE_URL:-$canonical_url}"

exec "$@"
