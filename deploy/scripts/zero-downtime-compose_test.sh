#!/usr/bin/env bash
# shellcheck disable=SC1003,SC2016
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/sforum-zero-downtime-compose-test.XXXXXX")"

cleanup() {
  rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

"$ROOT_DIR/deploy/scripts/configure-production.sh" \
  --lang en \
  --defaults \
  --version v3.0.0-alpha.10 \
  --root "$ROOT_DIR" \
  --output "$TEMP_DIR/.env.production" >/dev/null

SFORUM_BLUE_VERSION=v3.0.0-alpha.10 \
SFORUM_GREEN_VERSION=v3.0.0-alpha.11 \
docker compose \
  --env-file "$TEMP_DIR/.env.production" \
  -f "$ROOT_DIR/compose.yaml" \
  -f "$ROOT_DIR/compose.prod.yaml" \
  -f "$ROOT_DIR/compose.release.yaml" \
  -f "$ROOT_DIR/compose.zero-downtime.yaml" \
  --profile zero-downtime \
  config --format json > "$TEMP_DIR/compose.json"

ruby -rjson -e '
  services = JSON.parse(File.read(ARGV.fetch(0))).fetch("services")
  slots = %w[blue green]
  edge = services.fetch("edge")
  published = edge.fetch("ports").map { |port| [port["host_ip"], port["published"], port["target"]] }
  expected_ports = [["127.0.0.1", "3000", 3000], ["127.0.0.1", "18080", 18080]]
  abort "edge is not the stable loopback ingress: #{published.inspect}" unless published == expected_ports

  slots.each do |slot|
    api = services.fetch("api-#{slot}")
    web = services.fetch("web-#{slot}")
    worker = services.fetch("worker-#{slot}")
    abort "api-#{slot} published a host port" if api.key?("ports")
    abort "web-#{slot} published a host port" if web.key?("ports")
    abort "web-#{slot} points at the wrong API" unless web.dig("environment", "NUXT_API_INTERNAL_BASE_URL") == "http://api-#{slot}:8080/api/v1"
    abort "worker-#{slot} does not share its API PID namespace" unless worker["pid"] == "service:api-#{slot}"
    abort "worker-#{slot} lacks graceful drain time" unless worker["stop_grace_period"] == "40s"
    abort "worker-#{slot} does not wait for its API" unless worker.dig("depends_on", "api-#{slot}", "condition") == "service_started"
    %w[api worker].each do |kind|
      mounts = services.fetch("#{kind}-#{slot}").fetch("volumes").map { |volume| [volume["source"], volume["target"]] }
      abort "#{kind}-#{slot} lost attachment volume" unless mounts.include?(["attachment_uploads", "/app/storage/app/attachments"])
      abort "#{kind}-#{slot} lost extension volume" unless mounts.include?(["extension_packages", "/var/lib/sforum/extensions"])
    end
  end
  abort "blue API tag mismatch" unless services.dig("api-blue", "image").end_with?(":v3.0.0-alpha.10")
  abort "green API tag mismatch" unless services.dig("api-green", "image").end_with?(":v3.0.0-alpha.11")
' "$TEMP_DIR/compose.json"

"$ROOT_DIR/upgrade.sh" --help | grep -Fq -- "explicitly declared backward-compatible Core migrations" || {
  printf 'zero-downtime-compose_test.sh: help must document guarded online migrations\n' >&2
  exit 1
}

grep -Fq 'io.sforum.migrations.online-safe-check="v1"' "$ROOT_DIR/apps/api/Dockerfile" || {
  printf 'zero-downtime-compose_test.sh: migrator capability label is missing\n' >&2
  exit 1
}

grep -Fq 'sforum-migrate --check-online-safe' "$ROOT_DIR/upgrade.sh" || {
  printf 'zero-downtime-compose_test.sh: updater does not use the online migration guard\n' >&2
  exit 1
}

"$ROOT_DIR/upgrade.sh" --help | grep -Fq -- "default: latest" || {
  printf 'zero-downtime-compose_test.sh: help must document the latest default\n' >&2
  exit 1
}

grep -Fq 'http://127.0.0.1:3000/health' "$ROOT_DIR/upgrade.sh" || {
  printf 'zero-downtime-compose_test.sh: candidate Web probe must use /health\n' >&2
  exit 1
}

grep -Fq 'http://127.0.0.1:${web_port}/health' "$ROOT_DIR/upgrade.sh" || {
  printf 'zero-downtime-compose_test.sh: stable Web probe must use /health\n' >&2
  exit 1
}

if grep -Fq 'http://127.0.0.1:3000/ \' "$ROOT_DIR/upgrade.sh" \
  || grep -Fq 'http://127.0.0.1:${web_port}/"' "$ROOT_DIR/upgrade.sh"; then
  printf 'zero-downtime-compose_test.sh: Web probes must not render the cached homepage\n' >&2
  exit 1
fi

printf 'zero-downtime-compose_test.sh: all checks passed\n'
