#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SMTP_DIR="$ROOT_DIR/extensions/builtin/plugins/sforum-smtp/backend"

echo "Building protected built-in plugin: sforum.smtp"
(cd "$SMTP_DIR" && go build -o plugin .)
