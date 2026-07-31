#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/sforum-compose-production-test.XXXXXX")"

cleanup() {
  rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

"$ROOT_DIR/deploy/scripts/configure-production.sh" \
  --lang en \
  --defaults \
  --version v3.0.0-alpha.9 \
  --root "$ROOT_DIR" \
  --output "$TEMP_DIR/.env.production" >/dev/null

docker compose \
  --env-file "$TEMP_DIR/.env.production" \
  -f "$ROOT_DIR/compose.yaml" \
  -f "$ROOT_DIR/compose.prod.yaml" \
  -f "$ROOT_DIR/compose.release.yaml" \
  --profile tools \
  config --format json > "$TEMP_DIR/compose.json"

ruby -rjson -e '
  env = File.readlines(ARGV.fetch(0), chomp: true).each_with_object({}) do |line, values|
    next if line.empty? || line.start_with?("#")
    key, value = line.split("=", 2)
    values[key] = value
  end
  compose = JSON.parse(File.read(ARGV.fetch(1)))
  services = compose.fetch("services")
  migrate = services.fetch("migrate").fetch("environment")
  required = %w[
    SESSION_HASH_SECRET
    IDENTITY_SUBJECT_HMAC_SECRET
    APP_OPTION_ENC_KEY
    ALTCHA_SECRET
  ]
  required.each do |key|
    actual = migrate.fetch(key)
    expected = env.fetch(key)
    abort "compose-production_test.sh: migrate #{key} was not passed through" unless actual == expected
    abort "compose-production_test.sh: migrate #{key} is empty" if actual.empty?
  end
  abort "compose-production_test.sh: migrate APP_ENV is not production" unless migrate["APP_ENV"] == "production"
  worker = services.fetch("worker")
  abort "compose-production_test.sh: worker must share only the API PID namespace" unless worker["pid"] == "service:api"
  abort "compose-production_test.sh: worker must depend on API startup" unless worker.dig("depends_on", "api", "condition") == "service_started"
' "$TEMP_DIR/.env.production" "$TEMP_DIR/compose.json"

printf 'compose-production_test.sh: all checks passed\n'
