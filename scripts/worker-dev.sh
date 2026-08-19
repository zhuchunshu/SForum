#!/usr/bin/env bash
set -euo pipefail

echo "Standalone Worker is no longer supported. Run ./scripts/api-dev.sh; the API always owns background jobs." >&2
exit 1
