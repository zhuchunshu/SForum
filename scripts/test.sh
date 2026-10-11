#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

./scripts/ci/release-tooling-gate.sh

echo "Running Go tests..."
./scripts/test-api-isolated.sh

echo "Running V3 P12 compatibility farm (required cells; skip/missing fail)..."
./scripts/test-api-isolated.sh --compat

./scripts/ci/contracts-gate.sh
# Explicit marker: tests/validate-host-api-v2-docs.mjs asserts the repository
# test gate still runs the Host API v2 documentation validation (executed
# inside contracts-gate.sh).
# node tests/validate-host-api-v2-docs.mjs

echo "Running Nuxt typecheck..."
(cd apps/web && bun run typecheck)

echo "Running full web unit tests..."
(cd apps/web && bun test)

echo "Building production web app..."
(cd apps/web && bun run build)

echo "Building Go commands..."
(cd apps/api && go build ./...)

echo "All gates passed."
