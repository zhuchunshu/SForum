#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

echo "Running Go tests..."
(cd apps/api && go test ./...)

echo "Running OpenAPI reference validation..."
ruby scripts/validate-openapi-refs.rb

if [ -d apps/web/node_modules ]; then
  echo "Running Nuxt typecheck..."
  (cd apps/web && bun run typecheck)
  echo "Running admin framework validation..."
  bun tests/validate-admin-framework.ts
else
  echo "Skipping Nuxt typecheck because apps/web/node_modules is missing."
fi

echo "Running identity UI validation..."
node tests/validate-identity-ui.js

echo "Running SF component library validation..."
node tests/validate-sf-components.js
