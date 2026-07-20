#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

echo "Running Go tests..."
(cd apps/api && go test ./...)

echo "Running Protobuf lint and generated SDK drift validation..."
./scripts/proto.sh check

echo "Running Host API v2 SDK documentation drift validation..."
node tests/validate-host-api-v2-docs.mjs

echo "Running OpenAPI reference validation..."
ruby scripts/validate-openapi-refs.rb

echo "Running staged extension management contract validation..."
node tests/validate-staged-extension-contracts.js

echo "Running V3 production trust deployment validation..."
node tests/validate-v3-production-trust.mjs

echo "Running production WebSocket proxy validation..."
node tests/validate-production-websocket-proxy.mjs

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

echo "Running homepage validation..."
node tests/validate-homepage.js

echo "Running public SEO validation..."
bun tests/validate-public-seo.ts

echo "Running SEO workbench validation..."
bun tests/validate-seo-workbench.ts

echo "Running moderation workbench validation..."
bun tests/validate-moderation-workbenches.ts

echo "Running theme runtime validation..."
node tests/validate-theme-runtime.js

echo "Running trusted admin runtime validation..."
node tests/validate-trusted-admin-runtime.js

echo "Running synchronous theme activation validation..."
node tests/validate-theme-activation.js

echo "Running development worker script validation..."
node tests/validate-dev-worker-script.js

echo "Running Signal Garden theme validation..."
node tests/validate-signal-garden-theme.js

echo "Running SF component library validation..."
node tests/validate-sf-components.js

echo "Running Runtime Page Registry offline contracts (+ optional live HTTP smoke)..."
node tests/validate-page-registry-runtime.js

echo "Running V3 platform catalog and traceability validation..."
node tests/validate-v3-p0-catalogs.mjs
