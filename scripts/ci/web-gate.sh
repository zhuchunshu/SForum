#!/usr/bin/env bash
# Web-side CI gate: Nuxt typecheck, full web unit tests, and production build.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

if [ ! -d apps/web/node_modules ]; then
  echo "apps/web/node_modules is required; run bun install --frozen-lockfile first." >&2
  exit 1
fi

echo "Running Nuxt typecheck..."
(cd apps/web && bun run typecheck)

echo "Running all web unit tests..."
(cd apps/web && bun test)

echo "Building production web app..."
(cd apps/web && bun run build)
