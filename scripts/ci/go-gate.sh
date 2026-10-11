#!/usr/bin/env bash
# Go-side CI gate: isolated database tests plus Go command build.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

./scripts/test-api-isolated.sh

echo "Running V3 P12 compatibility farm (required cells; skip/missing fail)..."
./scripts/test-api-isolated.sh --compat

echo "Building Go commands..."
(cd apps/api && go build ./...)
